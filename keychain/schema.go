package keychain

type Profile struct {
	AwsAccessKeyID     string `json:"AWS_ACCESS_KEY_ID,omitempty"`
	AwsSecretAccessKey string `json:"AWS_SECRET_ACCESS_KEY,omitempty"`
	ResticRepository   string `json:"RESTIC_REPOSITORY,omitempty"`
	ResticPassword     string `json:"RESTIC_PASSWORD,omitempty"`
}
