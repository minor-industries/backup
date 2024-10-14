package restic_test

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/minor-industries/backup/cfg"
	"github.com/minor-industries/backup/keychain"
	"github.com/minor-industries/backup/restic"
	"github.com/stretchr/testify/require"
)

func TestRunWithKeychainProfiles(t *testing.T) {
	profiles, err := keychain.ListProfiles()
	require.NoError(t, err)

	for _, profile := range profiles {
		if profile == "profileA" || profile == "profileB" {
			err := keychain.DeleteProfile(profile)
			require.NoError(t, err)
		}
	}

	tmpDir, err := os.MkdirTemp("", "restic-integration-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	err = os.Mkdir(srcDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(srcDir, "testfile.txt")
	err = os.WriteFile(testFile, []byte("Hello, Restic!"), 0644)
	require.NoError(t, err)

	backupDirA := filepath.Join(tmpDir, "backup", "a")
	backupDirB := filepath.Join(tmpDir, "backup", "b")

	err = os.MkdirAll(backupDirA, 0755)
	require.NoError(t, err)

	err = os.MkdirAll(backupDirB, 0755)
	require.NoError(t, err)

	err = keychain.NewProfile("profileA", &keychain.Profile{
		ResticRepository: backupDirA,
		ResticPassword:   "passwordA",
	})
	require.NoError(t, err)
	defer func() {
		err := keychain.DeleteProfile("profileA")
		require.NoError(t, err)
	}()

	err = keychain.NewProfile("profileB", &keychain.Profile{
		ResticRepository: backupDirB,
		ResticPassword:   "passwordB",
	})
	defer func() {
		err := keychain.DeleteProfile("profileB")
		require.NoError(t, err)
	}()

	opts := &cfg.BackupConfig{
		ResticPath: "restic",
		SourceHost: "localhost",
		KeychainProfiles: []cfg.KeychainProfile{
			{Profile: "profileA"},
			{Profile: "profileB"},
		},
	}

	callback := restic.QuantizeFilter(func(a any) error {
		marshal, err := json.Marshal(a)
		if err != nil {
			return errors.Wrap(err, "marshal")
		}
		fmt.Println(string(marshal))
		return nil
	})

	err = restic.InitRepo(opts, &cfg.BackupTarget{
		ResticRepository: backupDirA,
		ResticPassword:   "passwordA",
	}, callback)
	require.NoError(t, err)

	err = restic.InitRepo(opts, &cfg.BackupTarget{
		ResticRepository: backupDirB,
		ResticPassword:   "passwordB",
	}, callback)
	require.NoError(t, err)

	err = restic.Run(opts, srcDir, callback)
	require.NoError(t, err)
}
