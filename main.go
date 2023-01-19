package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	"io"
	"net/http"
	"os"
)

const ExitCodeMainError = 1

type httpInterface interface {
	ListenAndServe(addr string, handler http.Handler) error
}

func main() {
	os.Exit(handleExitError(os.Stderr, runApp(os.Stdout, http.ListenAndServe)))
}

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

	redis := redis.NewClient(opt)

	_, err = redis.Ping(context.Background()).Result()
	if err != nil {
		fmt.Fprintf(out, "Failed to connect to redis: %s\n", err.Error())
	}

	//	gin.SetMode(gin.ReleaseMode)
	return listenAndServe(
		config.listenAddress,
		setupRouter(out, NewStorage(redis)),
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
