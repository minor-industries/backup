package restic_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/minor-industries/backup/cfg"
	"github.com/minor-industries/backup/keychain"
	"github.com/minor-industries/backup/restic"
	"github.com/stretchr/testify/require"
)

func deleteProfileIfExists(profileName string) {
	err := keychain.DeleteProfile(profileName)
	if err != nil && err != keychain.ErrorItemNotFound {
		fmt.Printf("Failed to delete profile %s: %v\n", profileName, err)
	}
}

func TestRunWithKeychainProfiles(t *testing.T) {
	profiles, err := listProfiles()
	require.NoError(t, err)

	for _, profile := range profiles {
		if profile == "profileA" || profile == "profileB" {
			deleteProfileIfExists(profile)
		}
	}

	tmpDir, err := ioutil.TempDir("", "restic-integration-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	err = os.Mkdir(srcDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(srcDir, "testfile.txt")
	err = ioutil.WriteFile(testFile, []byte("Hello, Restic!"), 0644)
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

	err = keychain.NewProfile("profileB", &keychain.Profile{
		ResticRepository: backupDirB,
		ResticPassword:   "passwordB",
	})
	require.NoError(t, err)

	opts := &cfg.BackupConfig{
		ResticPath: "restic",
		SourceHost: "localhost",
		KeychainProfiles: []cfg.KeychainProfile{
			{Profile: "profileA"},
			{Profile: "profileB"},
		},
	}

	var messages []any
	callback := func(msg any) error {
		messages = append(messages, msg)
		return nil
	}

	err = restic.Run(opts, srcDir, callback)
	require.NoError(t, err)

	require.NotEmpty(t, messages)

	for _, msg := range messages {
		fmt.Printf("Received message: %#v\n", msg)
	}
}
