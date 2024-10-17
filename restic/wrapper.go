package restic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/minor-industries/backup/cfg"
	"github.com/minor-industries/backup/keychain"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
)

type ResticMessage struct {
	MessageType string `json:"message_type"`
}

type ResticStatus struct {
	MessageType  string   `json:"message_type"`
	PercentDone  float64  `json:"percent_done"`
	TotalFiles   int      `json:"total_files"`
	FilesDone    int      `json:"files_done"`
	TotalBytes   int64    `json:"total_bytes"`
	BytesDone    int64    `json:"bytes_done"`
	CurrentFiles []string `json:"current_files"`
}

type ResticSummary struct {
	MessageType         string  `json:"message_type"`
	FilesNew            int     `json:"files_new"`
	FilesChanged        int     `json:"files_changed"`
	FilesUnmodified     int     `json:"files_unmodified"`
	DirsNew             int     `json:"dirs_new"`
	DirsChanged         int     `json:"dirs_changed"`
	DirsUnmodified      int     `json:"dirs_unmodified"`
	DataBlobs           int     `json:"data_blobs"`
	TreeBlobs           int     `json:"tree_blobs"`
	DataAdded           int64   `json:"data_added"`
	DataAddedPacked     int64   `json:"data_added_packed"`
	TotalFilesProcessed int     `json:"total_files_processed"`
	TotalBytesProcessed int64   `json:"total_bytes_processed"`
	TotalDuration       float64 `json:"total_duration"`
	SnapshotID          string  `json:"snapshot_id"`
}

type ResticInitialized struct {
	MessageType string `json:"message_type"`
	ID          string `json:"id"`
	Repository  string `json:"repository"` // TODO: should mask passwords in these messages
}

type ResticStats struct {
	TotalSize      int `json:"total_size"`
	TotalFileCount int `json:"total_file_count"`
	SnapshotsCount int `json:"snapshots_count"`
}

type StartBackup struct {
	Repository      string `json:"repository,omitempty"`
	KeychainProfile string `json:"keychain_profile,omitempty"`
}

func decodeResticMessage(data []byte) (any, error) {
	var shim ResticMessage
	if err := json.Unmarshal(data, &shim); err != nil {
		return nil, err
	}

	switch shim.MessageType {
	case "status":
		var status ResticStatus
		if err := json.Unmarshal(data, &status); err != nil {
			return nil, err
		}
		return status, nil
	case "summary":
		var summary ResticSummary
		if err := json.Unmarshal(data, &summary); err != nil {
			return nil, err
		}
		return summary, nil
	case "initialized":
		var initialized ResticInitialized
		if err := json.Unmarshal(data, &initialized); err != nil {
			return nil, err
		}
		return initialized, nil
	default:
		return nil, fmt.Errorf("unknown message type %s", shim.MessageType)
	}
}

func QuantizeFilter(callback func(msg any) error) func(msg any) error {
	lastQuantum := -1.0

	return func(msg any) error {
		switch msg := msg.(type) {
		case ResticStatus:
			currentQuantum := float64(int(msg.PercentDone*10)) / 10.0
			if currentQuantum > lastQuantum {
				lastQuantum = currentQuantum
				return callback(msg)
			}
			return nil
		case StartBackup, ResticSummary:
			lastQuantum = -1.0
			return callback(msg)
		default:
			return callback(msg)
		}
	}
}

func LogMessages(callback func(msg string) error) func(msg any) error {
	return QuantizeFilter(func(msg any) error {
		switch msg := msg.(type) {
		case StartBackup:
			if msg.KeychainProfile != "" {
				return callback(fmt.Sprintf("loading keychain profile: %s", msg.KeychainProfile))
			}
			if msg.Repository != "" {
				return callback(fmt.Sprintf("starting backup: %s", msg.Repository))
			}
		case ResticStatus:
			return callback(fmt.Sprintf("progress: %.1f%%", msg.PercentDone*100))
		case ResticSummary:
			return callback("backup done")
		default:
			return callback("unknown message type")
		}
		return nil
	})
}

func Run(
	opts *cfg.BackupConfig,
	chdir string,
	backupPaths []string,
	callback func(any) error,
) error {
	// check all targets with stats command before starting backup
	for _, target := range opts.Targets {
		if _, err := Stats(opts, &target); err != nil {
			return errors.Wrap(err, "check target")
		}
	}

	// load keychain profiles
	profiles := make([]*keychain.Profile, len(opts.KeychainProfiles))
	profileTargets := make([]*cfg.BackupTarget, len(opts.KeychainProfiles))
	for i, p := range opts.KeychainProfiles {
		if err := callback(StartBackup{KeychainProfile: p.Profile}); err != nil {
			return errors.Wrap(err, "callback")
		}

		var err error
		profiles[i], err = keychain.LoadProfile(p.Profile)
		if err != nil {
			return errors.Wrap(err, "load keychain profile")
		}

		profileTargets[i] = &cfg.BackupTarget{
			AwsAccessKeyId:     profiles[i].AwsAccessKeyID,
			AwsSecretAccessKey: profiles[i].AwsSecretAccessKey,
			ResticRepository:   profiles[i].ResticRepository,
			ResticPassword:     profiles[i].ResticPassword,
		}
	}

	// check keychain profiles with stats command before starting
	for _, target := range profileTargets {
		if _, err := Stats(opts, target); err != nil {
			return errors.Wrap(err, "check profile")
		}
	}

	for _, target := range opts.Targets {
		if err := BackupOne(opts, &target, chdir, backupPaths, callback); err != nil {
			return errors.Wrap(err, "backup one")
		}
	}

	for i, p := range opts.KeychainProfiles {
		if err := callback(StartBackup{KeychainProfile: p.Profile}); err != nil {
			return errors.Wrap(err, "callback")
		}

		if err := BackupOne(opts, profileTargets[i], chdir, backupPaths, callback); err != nil {
			return errors.Wrap(err, "backup one")
		}
	}

	return nil
}

func BackupOne(
	opts *cfg.BackupConfig,
	target *cfg.BackupTarget,
	chdir string,
	backupPaths []string,
	callback func(any) error,
) error {
	if len(backupPaths) == 0 {
		return errors.New("no backup paths given")
	}

	masked, err := maskPassword(target.ResticRepository)
	if err != nil {
		return errors.Wrap(err, "mask repo password")
	}

	if err := callback(StartBackup{Repository: masked}); err != nil {
		return errors.Wrap(err, "callback")
	}

	args := []string{
		os.ExpandEnv(opts.ResticPath),
		"backup",
		"--json",
		"--host",
		opts.SourceHost,
	}

	args = append(args, backupPaths...)

	cmd := exec.Command(args[0], args[1:]...)

	if chdir != "" {
		cmd.Dir = chdir
	}

	return streamingResticCommand(target, cmd, callback)
}

func InitRepo(
	opts *cfg.BackupConfig,
	target *cfg.BackupTarget,
	callback func(any) error,
) error {
	cmd := exec.Command(opts.ResticPath, "init", "--json")
	return streamingResticCommand(target, cmd, callback)
}

func Stats(
	opts *cfg.BackupConfig,
	target *cfg.BackupTarget,
) (*ResticStats, error) {
	cmd := exec.Command(opts.ResticPath, "stats", "--json")
	addEnv(target, cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "run (output: %s)", string(output))
	}

	var stats ResticStats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return &stats, nil
}

func streamingResticCommand(
	target *cfg.BackupTarget,
	cmd *exec.Cmd,
	callback func(any) error,
) error {
	addEnv(target, cmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "get stdout pipe")
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "get stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "start restic")
	}

	errCh := make(chan error)
	defer close(errCh)
	numProcs := 0
	stderrCh := make(chan string)
	defer close(stderrCh)

	numProcs++
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Bytes()

			msg, err := decodeResticMessage(line)
			if err != nil {
				errCh <- errors.Wrap(err, "decode restic message")
				return
			}

			if err := callback(msg); err != nil {
				errCh <- errors.Wrap(err, "callback returned error")
				return
			}
		}

		errCh <- nil
	}()

	numProcs++
	go func() {
		var stderrBuffer bytes.Buffer
		io.Copy(&stderrBuffer, stderrPipe)
		stderrCh <- stderrBuffer.String()
		errCh <- nil
	}()

	numProcs++
	go func() {
		err := cmd.Wait()
		stderr := <-stderrCh
		if err != nil {
			errCh <- errors.Wrapf(err, "restic command failed, stderr: %s", stderr)
			return
		}
		errCh <- nil
	}()

	allErrors := make([]error, numProcs)
	for i := 0; i < numProcs; i++ {
		allErrors[i] = <-errCh
	}

	for _, err := range allErrors {
		if err != nil {
			return err
		}
	}

	return nil
}

func addEnv(target *cfg.BackupTarget, cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+target.AwsAccessKeyId,
		"AWS_SECRET_ACCESS_KEY="+target.AwsSecretAccessKey,
		"RESTIC_REPOSITORY="+target.ResticRepository,
		"RESTIC_PASSWORD="+target.ResticPassword,
	)
	if target.CACertPath != "" {
		cmd.Env = append(cmd.Env, "RESTIC_CACERT="+os.ExpandEnv(target.CACertPath))
	}
}
