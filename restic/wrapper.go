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

func RunConsole(
	opts *cfg.BackupConfig,
	chdir string,
	backupPaths []string,
) error {
	if len(backupPaths) == 0 {
		return errors.New("no backup paths given")
	}

	allTargets, err := loadProfilesAndCheckTargets(opts, nil)
	if err != nil {
		return errors.Wrap(err, "check targets")
	}

	for _, target := range allTargets {
		if err := BackupOneConsole(opts, &target, chdir, backupPaths); err != nil {
			return errors.Wrap(err, "backup one")
		}
	}

	return nil
}

func Run(
	opts *cfg.BackupConfig,
	chdir string,
	backupPaths []string,
	callback func(any) error,
) error {
	if len(backupPaths) == 0 {
		return errors.New("no backup paths given")
	}

	allTargets, err := loadProfilesAndCheckTargets(opts, callback)
	if err != nil {
		return errors.Wrap(err, "check targets")
	}

	for _, target := range allTargets {
		if err := BackupOne(opts, &target, chdir, backupPaths, callback); err != nil {
			return errors.Wrap(err, "backup one")
		}
	}

	return nil
}

func loadProfilesAndCheckTargets(
	opts *cfg.BackupConfig,
	callback func(any) error,
) ([]cfg.BackupTarget, error) {
	// check all targets with stats command before starting backup
	for _, target := range opts.Targets {
		if _, err := Stats(opts, &target); err != nil {
			return nil, errors.Wrap(err, "check target")
		}
	}

	// load keychain profiles
	profiles := make([]*keychain.Profile, len(opts.KeychainProfiles))
	profileTargets := make([]cfg.BackupTarget, len(opts.KeychainProfiles))
	for i, p := range opts.KeychainProfiles {
		if callback != nil {
			if err := callback(StartBackup{KeychainProfile: p.Profile}); err != nil {
				return nil, errors.Wrap(err, "callback")
			}
		}

		var err error
		profiles[i], err = keychain.LoadProfile(p.Profile)
		if err != nil {
			return nil, errors.Wrap(err, "load keychain profile")
		}

		profileTargets[i] = cfg.BackupTarget{
			AwsAccessKeyId:     profiles[i].AwsAccessKeyID,
			AwsSecretAccessKey: profiles[i].AwsSecretAccessKey,
			ResticRepository:   profiles[i].ResticRepository,
			ResticPassword:     profiles[i].ResticPassword,
		}
	}

	// check keychain profiles with stats command before starting
	for _, target := range profileTargets {
		if _, err := Stats(opts, &target); err != nil {
			return nil, errors.Wrap(err, "check profile")
		}
	}

	allTargets := append(opts.Targets, profileTargets...)
	return allTargets, nil
}

func BackupOneConsole(
	opts *cfg.BackupConfig,
	target *cfg.BackupTarget,
	chdir string,
	backupPaths []string,
) error {
	masked, err := maskPassword(target.ResticRepository)
	if err != nil {
		return errors.Wrap(err, "mask repo password")
	}

	fmt.Println("starting backup to:", masked)

	args := []string{
		os.ExpandEnv(opts.ResticPath),
		"backup",
	}

	if opts.SourceHost != "" {
		args = append(args, "--host", opts.SourceHost)
	}

	args = append(args, backupPaths...)

	cmd := exec.Command(args[0], args[1:]...)

	if chdir != "" {
		cmd.Dir = chdir
	}

	addEnv(target, cmd)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	return errors.Wrap(err, "run")
}

func BackupOne(
	opts *cfg.BackupConfig,
	target *cfg.BackupTarget,
	chdir string,
	backupPaths []string,
	callback func(any) error,
) error {
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
	}

	if opts.SourceHost != "" {
		args = append(args, "--host", opts.SourceHost)
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
