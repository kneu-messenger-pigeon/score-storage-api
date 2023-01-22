package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	scoreApi "github.com/kneu-messenger-pigeon/score-api"
	"strconv"
)

type ScoreRatingLoaderInterface interface {
	load(year int, semester int, disciplineId int, studentId int) scoreApi.ScoreRating
}

type ScoreRatingLoader struct {
	redis *redis.Client
}

func (loader *ScoreRatingLoader) load(year int, semester int, disciplineId int, studentId int) (scoreRating scoreApi.ScoreRating) {
	ctx := context.Background()
	disciplineTotalsKey := fmt.Sprintf("%d:%d:totals:%d", year, semester, disciplineId)
	studentKey := strconv.Itoa(studentId)

	scoreRating.StudentsCount = int(loader.redis.ZCard(ctx, disciplineTotalsKey).Val())
	scoreRating.Rating = scoreRating.StudentsCount
	total := loader.redis.ZScore(ctx, disciplineTotalsKey, studentKey).Val()
	scoreRating.Total = float32(total)

	if total > 0 {
		// rating position is amount of students with Total greater than in current student
		scoreRating.Rating = int(loader.redis.ZCount(
			ctx, disciplineTotalsKey,
			"("+strconv.FormatFloat(total, 'f', -1, 64),
			"+inf",
		).Val()) + 1
	}

	var score []redis.Z
	opt := &redis.ZRangeBy{
		Min:    "0.1",
		Max:    "100",
		Offset: 0,
		Count:  1,
	}

	// MIN: ZRANGE 2022:1:totals:194229 0.1 100 BYSCORE LIMIT 0 1 WITHSCORES
	score = loader.redis.ZRangeByScoreWithScores(ctx, disciplineTotalsKey, opt).Val()
	if len(score) != 0 {
		scoreRating.MinTotal = float32(score[0].Score)
	}

	// MAX: ZRANGE 2022:1:totals:194229 100 0.1 BYSCORE REV LIMIT 0 1 WITHSCORES
	score = loader.redis.ZRevRangeByScoreWithScores(ctx, disciplineTotalsKey, opt).Val()
	if len(score) != 0 {
		scoreRating.MaxTotal = float32(score[0].Score)
	}

	return scoreRating
}
