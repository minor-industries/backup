package restic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/minor-industries/backup/cfg"
	"github.com/minor-industries/backup/keychain"
	"github.com/pkg/errors"
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
				return callback(fmt.Sprint("loading keychain profile:", msg.KeychainProfile))
			}
			if msg.Repository != "" {
				return callback(fmt.Sprint("starting backup:", msg.Repository))
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
	for _, target := range opts.Targets {
		if err := BackupOne(opts, &target, chdir, backupPaths, callback); err != nil {
			return errors.Wrap(err, "backup one")
		}
	}

	for _, p := range opts.KeychainProfiles {
		if err := callback(StartBackup{KeychainProfile: p.Profile}); err != nil {
			return errors.Wrap(err, "callback")
		}

		profile, err := keychain.LoadProfile(p.Profile)
		if err != nil {
			return errors.Wrap(err, "load keychain profile")
		}
		if err := BackupOne(opts, &cfg.BackupTarget{
			AwsAccessKeyId:     profile.AwsAccessKeyID,
			AwsSecretAccessKey: profile.AwsSecretAccessKey,
			ResticRepository:   profile.ResticRepository,
			ResticPassword:     profile.ResticPassword,
		}, chdir, backupPaths, callback); err != nil {
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

	return resticCmd(target, cmd, callback)
}

func InitRepo(
	opts *cfg.BackupConfig,
	target *cfg.BackupTarget,
	callback func(any) error,
) error {
	cmd := exec.Command(opts.ResticPath, "init", "--json")
	return resticCmd(target, cmd, callback)
}

func resticCmd(
	target *cfg.BackupTarget,
	cmd *exec.Cmd,
	callback func(any) error,
) error {
	cmd.Env = append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+target.AwsAccessKeyId,
		"AWS_SECRET_ACCESS_KEY="+target.AwsSecretAccessKey,
		"RESTIC_REPOSITORY="+target.ResticRepository,
		"RESTIC_PASSWORD="+target.ResticPassword,
	)
	if target.CACertPath != "" {
		cmd.Env = append(cmd.Env, "RESTIC_CACERT="+os.ExpandEnv(target.CACertPath))
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "get stdout pipe")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "start restic")
	}

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Bytes()

		msg, err := decodeResticMessage(line)
		if err != nil {
			return errors.Wrap(err, "decode restic message")
		}

		if err := callback(msg); err != nil {
			return errors.Wrap(err, "callback returned error")
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "read from restic stdout")
	}

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "wait for restic command")
	}

	return nil
}
