package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func ReadJsonConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	configFilePath := filepath.Join(home, ".gatorconfig.json")
	file, err := os.Open(configFilePath)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c *Config) SetUser(name string) error {
	c.CurrentUserName = name
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configFilePath := filepath.Join(home, ".gatorconfig.json")
	file, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) GetUser() (string, error) {
	if c.CurrentUserName == "" {
		return "", nil
	}
	return c.CurrentUserName, nil
}
