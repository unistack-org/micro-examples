package main

func newConfig(name, version string) *Config {
	return &Config{
		App:      &AppConfig{},
		Logger:   &LoggerConfig{},
		Server:   &ServerConfig{Name: name, Version: version},
		Vault:    &VaultConfig{},
		Meter:    &MeterConfig{},
		Tracer:   &TracerConfig{Service: name},
		Database: &DatabaseConfig{},
		Broker:   &BrokerConfig{},
	}
}

type Config struct {
	App      *AppConfig      `yaml:"app"`
	Logger   *LoggerConfig   `yaml:"logger"`
	Server   *ServerConfig   `yaml:"server"`
	Vault    *VaultConfig    `yaml:"vault"`
	Meter    *MeterConfig    `yaml:"meter"`
	Database *DatabaseConfig `yaml:"database"`
	Tracer   *TracerConfig   `yaml:"tracer"`
	Broker   *BrokerConfig   `yaml:"broker"`
}

type LoggerConfig struct {
	Level string `yaml:"level"`
}

type ServerConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"-"`
	Address string `yaml:"address" default:":9090"`
}

type AppConfig struct {
	Topic string `yaml:"topic" default:"pubsub"`
}

type TracerConfig struct {
	Metadata  map[string]string `yaml:"tags" env:"JAEGER_TAGS"`
	Service   string            `yaml:"-"`
	Collector string            `yaml:"collector" env:"JAEGER_ENDPOINT"`
	AgentHost string            `yaml:"agent_host" env:"JAEGER_AGENT_HOST,OTEL_EXPORTER_JAEGER_AGENT_HOST" default:"localhost"`
	AgentPort string            `yaml:"agent_port" env:"JAEGER_AGENT_PORT,OTEL_EXPORTER_JAEGER_AGENT_PORT" default:"6832"`
}

type VaultConfig struct {
	Address string `yaml:"address" env:"VAULT_ADDR" default:"http://localhost:54321"`
	Path    string `yaml:"path" env:"VAULT_PATH"`
	Token   string `yaml:"token" env:"VAULT_TOKEN"`
}

type MeterConfig struct {
	Address string `yaml:"address" default:":8080"`
}

type DatabaseConfig struct {
	Migrate string   `yaml:"migrate" flag:"name=migrate,desc='database migrations run mode',default='up'"`
	Dsn     []string `yaml:"dsn"`
}

type BrokerConfig struct {
	Login   string   `yaml:"login"`
	Passw   string   `yaml:"passw"`
	Address []string `yaml:"address"`
}
