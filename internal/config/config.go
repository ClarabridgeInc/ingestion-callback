package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// Config dependencies for the application
type Config struct {
	SQS SQSConfig `yaml:"sqs"`
}

type SQSConfig struct {
	QueueName string `yaml:"queue_name"`
}

// ReadFromPaths reads from the provided paths and returns a populated Config struct
func ReadFromPaths(paths ...string) (*Config, error) {
	cfg := &Config{}

	for _, path := range paths {
		if err := validatePath(path); err != nil {
			return nil, err
		}

		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
			return nil, err
		}

		file.Close()
	}

	return cfg, nil
}

// validatePath just makes sure that the path provided is a file
// that can be read
func validatePath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}
