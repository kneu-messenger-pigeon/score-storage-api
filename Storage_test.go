package main

import (
	"context"
	"github.com/go-redis/redismock/v9"
	scoreApi "github.com/kneu-messenger-pigeon/score-api"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	t.Run("testPeriodicallyUpdateGeneralDataRun", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		expectedYear := 2025
		expectedLessonTypes := map[int]scoreApi.LessonType{
			1: {
				Id:        1,
				ShortName: "Тст",
				LongName:  "Тест",
			},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		redisMock.ExpectGet("currentYear").SetVal(strconv.Itoa(expectedYear))
		redisMock.ExpectGet("lessonTypes").SetVal(`[{"id":1,"shortName":"Тст","longName":"Тест"}]`)

		storage := NewStorage(redisClient, ctx)
		time.Sleep(time.Millisecond)
		cancel()

		assert.Equal(t, expectedYear, storage.year)
		assert.Equal(t, expectedLessonTypes, storage.lessonTypes)

		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("testPeriodicallyUpdateGeneralDataRun_EmptyReids", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		redisMock.ExpectGet("currentYear").RedisNil()
		redisMock.ExpectGet("lessonTypes").RedisNil()

		storage := NewStorage(redisClient, ctx)
		time.Sleep(time.Millisecond)
		cancel()

		assert.Empty(t, storage.year)
		assert.Empty(t, storage.lessonTypes)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

}

func TestStorageGetDisciplineScoreResultsByStudentId(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedResults := scoreApi.DisciplineScoreResults{
			scoreApi.DisciplineScoreResult{
				Discipline: scoreApi.Discipline{
					Id:   100,
					Name: "Капітал!",
				},
				ScoreRating: scoreApi.ScoreRating{
					Total:         17,
					StudentsCount: 25,
					Rating:        8,
					MinTotal:      10,
					MaxTotal:      20,
				},
			},
			scoreApi.DisciplineScoreResult{
				Discipline: scoreApi.Discipline{
					Id:   110,
					Name: "Гроші та лихварство",
				},
				ScoreRating: scoreApi.ScoreRating{
					Total:         12,
					StudentsCount: 25,
					Rating:        12,
					MinTotal:      7,
					MaxTotal:      17,
				},
			},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1100"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1100"

		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).RedisNil()
		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).SetVal([]string{
			"100",
			"110",
		})

		redisMock.ExpectHGet("2026:discipline:100", "name").SetVal(expectedResults[0].Discipline.Name)
		redisMock.ExpectHGet("2026:discipline:110", "name").SetVal(expectedResults[1].Discipline.Name)

		scoreRatingLoader := NewMockScoreRatingLoaderInterface(t)
		scoreRatingLoader.On("load", 2026, 1, 100, 1100).Return(expectedResults[0].ScoreRating)
		scoreRatingLoader.On("load", 2026, 1, 110, 1100).Return(expectedResults[1].ScoreRating)

		storage := Storage{
			redis:             redisClient,
			year:              2026,
			lessonTypes:       lessonTypes,
			scoreRatingLoader: scoreRatingLoader,
		}

		actualResults, err := storage.getDisciplineScoreResultsByStudentId(1100)

		assert.Equal(t, expectedResults, actualResults)
		assert.NoError(t, err)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})
}

func TestGetDisciplineScoreResultByStudentId(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedResult := scoreApi.DisciplineScoreResult{
			Discipline: scoreApi.Discipline{
				Id:   199,
				Name: "Капітал!",
			},
			ScoreRating: scoreApi.ScoreRating{
				Total:         17,
				StudentsCount: 25,
				Rating:        8,
				MinTotal:      10,
				MaxTotal:      20,
			},
			Scores: []scoreApi.Score{
				{
					Lesson: scoreApi.Lesson{
						Id:   245,
						Date: "12.02.2023",
						Type: lessonTypes[1],
					},
					LessonHalf: 1,
					Score:      4.5,
					IsAbsent:   false,
				},
				{
					Lesson: scoreApi.Lesson{
						Id:   255,
						Date: "14.02.2023",
						Type: lessonTypes[15],
					},
					LessonHalf: 2,
					Score:      0,
					IsAbsent:   true,
				},
			},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1200"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1200"

		redisMock.ExpectSIsMember(studentDisciplinesKeySemester2, expectedResult.Discipline.Id).RedisNil()
		redisMock.ExpectSIsMember(studentDisciplinesKeySemester1, expectedResult.Discipline.Id).SetVal(true)

		redisMock.ExpectHGet("2026:discipline:199", "name").SetVal(expectedResult.Discipline.Name)

		studentDisciplineScoresKey := "2026:1:scores:1200:199"
		disciplineKey := "2026:1:lessons:199"

		redisMock.ExpectHGetAll(studentDisciplineScoresKey).SetVal(map[string]string{
			"245:1": "4.5",
			"255:2": strconv.FormatFloat(IsAbsentScoreValue, 'f', -1, 64),
		})

		redisMock.ExpectHMGet(disciplineKey, "245", "255").SetVal([]interface{}{
			"2302121",
			"23021415",
		})

		scoreRatingLoader := NewMockScoreRatingLoaderInterface(t)
		scoreRatingLoader.On("load", 2026, 1, expectedResult.Discipline.Id, 1200).Return(expectedResult.ScoreRating)

		storage := Storage{
			redis:             redisClient,
			year:              2026,
			lessonTypes:       lessonTypes,
			scoreRatingLoader: scoreRatingLoader,
		}

		actualResult, err := storage.getDisciplineScoreResultByStudentId(1200, expectedResult.Discipline.Id)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, actualResult)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})
}

func GetTestLessonTypes() map[int]scoreApi.LessonType {
	return map[int]scoreApi.LessonType{
		1: {
			Id:        1,
			ShortName: "ПрЗн",
			LongName:  "Практичне зан.",
		},
		15: {
			Id:        15,
			ShortName: "МК",
			LongName:  "Модульний контроль.",
		},
	}
}