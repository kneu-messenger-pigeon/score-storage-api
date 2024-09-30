package main

import (
	"github.com/go-redis/redismock/v9"
	scoreApi "github.com/kneu-messenger-pigeon/score-api"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScoreRatingLoader(t *testing.T) {
	t.Run("success", func(t *testing.T) {

		expectedScoreRating := scoreApi.ScoreRating{
			Total:         17.5,
			StudentsCount: 25,
			Rating:        8,
			MinTotal:      10,
			MaxTotal:      20,
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		disciplineTotalsKey := "2023:2:totals:300"
		studentKey := "1200"

		redisMock.ExpectZCard(disciplineTotalsKey).SetVal(25)
		redisMock.ExpectZScore(disciplineTotalsKey, studentKey).SetVal(17.5)

		redisMock.ExpectZCount(disciplineTotalsKey, "(17.5", "+inf").SetVal(7)

		opt := &redis.ZRangeBy{
			Min:    "0.1",
			Max:    "100",
			Offset: 0,
			Count:  1,
		}

		redisMock.ExpectZRangeByScoreWithScores(disciplineTotalsKey, opt).SetVal([]redis.Z{
			{
				Score:  10,
				Member: "1500",
			},
		})

		redisMock.ExpectZRevRangeByScoreWithScores(disciplineTotalsKey, opt).SetVal([]redis.Z{
			{
				Score:  20,
				Member: "1580",
			},
		})

		scoreRatingLoader := ScoreRatingLoader{
			redis: redisClient,
		}

		actualScoreRating := scoreRatingLoader.load(2023, 2, 300, 1200)

		assert.NoError(t, redisMock.ExpectationsWereMet())
		assert.Equal(t, expectedScoreRating, actualScoreRating)
	})

	t.Run("zero_total", func(t *testing.T) {
		expectedScoreRating := scoreApi.ScoreRating{
			Total:         0,
			StudentsCount: 25,
			Rating:        25,
			MinTotal:      10,
			MaxTotal:      20,
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		disciplineTotalsKey := "2023:2:totals:300"
		studentKey := "1200"

		redisMock.ExpectZCard(disciplineTotalsKey).SetVal(25)
		redisMock.ExpectZScore(disciplineTotalsKey, studentKey).RedisNil()

		opt := &redis.ZRangeBy{
			Min:    "0.1",
			Max:    "100",
			Offset: 0,
			Count:  1,
		}
		redisMock.ExpectZRangeByScoreWithScores(disciplineTotalsKey, opt).SetVal([]redis.Z{
			{
				Score:  10,
				Member: "1500",
			},
		})

		redisMock.ExpectZRevRangeByScoreWithScores(disciplineTotalsKey, opt).SetVal([]redis.Z{
			{
				Score:  20,
				Member: "1580",
			},
		})

		scoreRatingLoader := ScoreRatingLoader{
			redis: redisClient,
		}

		actualScoreRating := scoreRatingLoader.load(2023, 2, 300, 1200)

		assert.NoError(t, redisMock.ExpectationsWereMet())
		assert.Equal(t, expectedScoreRating, actualScoreRating)
	})
}
