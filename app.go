package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"io"
	"net/http"
	"os"
)

const ExitCodeMainError = 1

func runApp(out io.Writer, listenAndServe func(string, http.Handler) error) error {
	envFilename := ""
	if _, err := os.Stat(".env"); err == nil {
		envFilename = ".env"
	}

	var opt *redis.Options
	config, err := loadConfig(envFilename)
	if err == nil {
		opt, err = redis.ParseURL(config.redisDsn)
	}

	if err != nil {
		return err
	}

	redisClient := redis.NewClient(opt)

	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		fmt.Fprintf(out, "Failed to connect to redisClient: %s\n", err.Error())
	}

	storage := NewStorage(redisClient, context.Background())

	gin.SetMode(gin.ReleaseMode)
	return listenAndServe(
		config.listenAddress,
		setupRouter(out, storage),
	)
}

func handleExitError(errStream io.Writer, err error) int {
	if err != nil {
		_, _ = fmt.Fprintln(errStream, err)
	}

	if err != nil {
		return ExitCodeMainError
	}

	return 0
}
