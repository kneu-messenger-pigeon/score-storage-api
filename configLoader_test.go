package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var expectedConfig = Config{
	redisDsn:      "REDIS:6379",
	listenAddress: ":8080",
}

func TestLoadConfigFromEnvVars(t *testing.T) {
	t.Run("FromEnvVars", func(t *testing.T) {
		_ = os.Setenv("REDIS_DSN", expectedConfig.redisDsn)
		_ = os.Setenv("LISTEN", ":8080")

		config, err := loadConfig("")

		assert.NoErrorf(t, err, "got unexpected error %s", err)
		assertConfig(t, expectedConfig, config)
		assert.Equalf(t, expectedConfig, config, "Expected for %v, actual: %v", expectedConfig, config)
	})

	t.Run("FromFile", func(t *testing.T) {
		var envFileContent string

		envFileContent += fmt.Sprintf("REDIS_DSN=%s\n", expectedConfig.redisDsn)
		envFileContent += fmt.Sprintf("LISTEN=%s\n", expectedConfig.listenAddress)

		testEnvFilename := "TestLoadConfigFromFile.env"
		err := os.WriteFile(testEnvFilename, []byte(envFileContent), 0644)
		defer os.Remove(testEnvFilename)
		assert.NoErrorf(t, err, "got unexpected while write file %s error %s", testEnvFilename, err)

		config, err := loadConfig(testEnvFilename)

		assert.NoErrorf(t, err, "got unexpected error %s", err)
		assertConfig(t, expectedConfig, config)
		assert.Equalf(t, expectedConfig, config, "Expected for %v, actual: %v", expectedConfig, config)
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		_ = os.Setenv("REDIS_DSN", "")
		_ = os.Setenv("LISTEN", "")

		config, err := loadConfig("")

		assert.Error(t, err, "loadConfig() should exit with error, actual error is nil")

		assert.Emptyf(
			t, config.redisDsn,
			"Expected for empty config.secondaryDekanatDbDSN, actual %s", config.redisDsn,
		)

		_ = os.Setenv("REDIS_DSN", "dummy-redis")

		config, err = loadConfig("")

		assert.Error(t, err, "loadConfig() should exit with error, actual error is nil")
		assert.Emptyf(
			t, config.listenAddress,
			"Expected for empty config.listenAddress, actual %s", config.listenAddress,
		)

	})

	t.Run("NotExistConfigFile", func(t *testing.T) {
		_ = os.Setenv("REDIS_DSN", "")
		_ = os.Setenv("LISTEN", ":8080")

		config, err := loadConfig("not-exists.env")

		assert.Error(t, err, "loadConfig() should exit with error, actual error is nil")
		assert.Equalf(
			t, "Error loading not-exists.env file: open not-exists.env: no such file or directory", err.Error(),
			"Expected for not exist file error, actual: %s", err.Error(),
		)
		assert.Emptyf(
			t, config.redisDsn,
			"Expected for empty config.kafkaHost, actual %s", config.redisDsn,
		)
	})
}

func assertConfig(t *testing.T, expected Config, actual Config) {
	assert.Equal(t, expected.redisDsn, actual.redisDsn)
}
