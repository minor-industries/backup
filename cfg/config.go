package cfg

type BackupTarget struct {
	AwsAccessKeyId     string `toml:"aws_access_key_id"`
	AwsSecretAccessKey string `toml:"aws_secret_access_key"`
	ResticRepository   string `toml:"restic_repository"`
	ResticPassword     string `toml:"restic_password"`
	CACertPath         string `toml:"ca_cert_path"`
}

type BackupConfig struct {
	ResticPath string         `toml:"restic_path"`
	SourceHost string         `toml:"source_host"`
	Targets    []BackupTarget `toml:"targets"`
}
