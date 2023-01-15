package main

import (
	"github.com/go-redis/redis/v9"
)

type StorageInterface interface {
}

type Storage struct {
	redis redis.UniversalClient
}

type DisciplineWithScores struct {
	id          int
	name        string
	scoreRating DisciplineScoreRating
	scores      []Score `json:,omitempty`
}

type DisciplineScoreRating struct {
	total         float32
	minTotal      float32
	maxTotal      float32
	rating        int
	studentsCount int
}

type Score struct {
	lesson struct {
		id     int
		date   string
		typeId int
	}
	lessonHalf uint
	score      float32
	isAbsent   bool
}
