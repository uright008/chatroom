package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	UI       UIConfig
}

type UIConfig struct {
	Title      string `toml:"title"`
	PageTitle  string `toml:"page_title"`
}

type ServerConfig struct {
	Port           string `toml:"port"`
	MaxHistory     int    `toml:"max_history"`
	UploadDir      string `toml:"upload_dir"`
	MaxUploadSize  int    `toml:"max_upload_size"`
}

type DatabaseConfig struct {
	Driver string `toml:"driver"`
	DSN    string `toml:"dsn"`
}

func InitConfig(path string) (*Config, error) {
	var cfg Config

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefaultConfig(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func createDefaultConfig(path string) error {
	if err := os.MkdirAll("config", 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	defaultCfg := Config{
		Server: ServerConfig{
			Port:          ":8080",
			MaxHistory:    100,
			UploadDir:     "./static/uploads",
			MaxUploadSize: 10,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./chatroom.db",
		},
		UI: UIConfig{
			Title:      "我的聊天室",
			PageTitle:  "欢迎来到我的聊天室",
		},
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(defaultCfg); err != nil {
		return fmt.Errorf("failed to encode default config: %w", err)
	}

	return nil
}