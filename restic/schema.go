package restic

import (
	"encoding/json"
	"fmt"
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
