package config

var (
	Config Configs
)

type Configs struct {
	LogInfo LogConfigs `yaml:"logInfo"`
}
type LogConfigs struct {
	Level        string `yaml:"level"`
	StdOut       bool   `yaml:"stdOut"`
	MaxSize      int    `yaml:"maxSize"`
	MaxBackups   int    `yaml:"maxBackups"`
	MaxAges      int    `yaml:"maxAges"`
	Compress     bool   `yaml:"compress"`
	ServiceName  string `yaml:"serviceName"`
	InstanceName string `yaml:"instanceName"`
}
