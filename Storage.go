package main

import (
	"github.com/go-redis/redis/v9"
)

type StorageInterface interface {
}

type Storage struct {
	redis redis.UniversalClient
}
