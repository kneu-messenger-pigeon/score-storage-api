package main

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	redisDsn      string
	listenAddress string
}

func loadConfig(envFilename string) (Config, error) {
	if envFilename != "" {
		err := godotenv.Load(envFilename)
		if err != nil {
			return Config{}, errors.New(fmt.Sprintf("Error loading %s file: %s", envFilename, err))
		}
	}
	config := Config{
		redisDsn:      os.Getenv("REDIS_DSN"),
		listenAddress: os.Getenv("LISTEN"),
	}

	if config.redisDsn == "" {
		return Config{}, errors.New("empty REDIS_DSN")
	}

	if config.listenAddress == "" {
		return Config{}, errors.New("empty LISTEN")
	}

	return config, nil
}
