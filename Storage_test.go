package main

import (
	"context"
	"errors"
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
		redisMock.MatchExpectationsInOrder(false)

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

	t.Run("emptyDisciplines", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1100"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1100"

		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).RedisNil()
		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).RedisNil()

		storage := Storage{
			redis:             redisClient,
			year:              2026,
			lessonTypes:       lessonTypes,
			scoreRatingLoader: NewMockScoreRatingLoaderInterface(t),
		}

		actualResults, err := storage.getDisciplineScoreResultsByStudentId(1100)

		assert.Equal(t, scoreApi.DisciplineScoreResults{}, actualResults)
		assert.NoError(t, err)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("redis_error", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedError := errors.New("expected error")

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1100"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1100"

		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).RedisNil()
		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).SetErr(expectedError)

		storage := Storage{
			redis:             redisClient,
			year:              2026,
			lessonTypes:       lessonTypes,
			scoreRatingLoader: NewMockScoreRatingLoaderInterface(t),
		}

		actualResults, err := storage.getDisciplineScoreResultsByStudentId(1100)

		assert.Nil(t, actualResults)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)

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
						Date: time.Date(2023, time.Month(2), 12, 0, 0, 0, 0, time.Local),
						Type: lessonTypes[1],
					},
					FirstScore:  4.5,
					SecondScore: 2,
					IsAbsent:    false,
				},
				{
					Lesson: scoreApi.Lesson{
						Id:   247,
						Date: time.Date(2023, time.Month(2), 12, 0, 0, 0, 0, time.Local),
						Type: lessonTypes[1],
					},
					FirstScore: 1,
					IsAbsent:   false,
				},
				{
					Lesson: scoreApi.Lesson{
						Id:   255,
						Date: time.Date(2023, time.Month(2), 14, 0, 0, 0, 0, time.Local),
						Type: lessonTypes[15],
					},
					IsAbsent: true,
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
			"245:2": "2",
			"247:1": "1",
		})

		redisMock.ExpectHGetAll(disciplineKey).SetVal(map[string]string{
			"255": "23021415",
			"245": "2302121",
			"247": "2302121",
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

	t.Run("noScores", func(t *testing.T) {
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
			Scores: []scoreApi.Score{},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1200"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1200"

		redisMock.ExpectSIsMember(studentDisciplinesKeySemester2, expectedResult.Discipline.Id).RedisNil()
		redisMock.ExpectSIsMember(studentDisciplinesKeySemester1, expectedResult.Discipline.Id).SetVal(true)

		redisMock.ExpectHGet("2026:discipline:199", "name").SetVal(expectedResult.Discipline.Name)

		studentDisciplineScoresKey := "2026:1:scores:1200:199"
		redisMock.ExpectHGetAll(studentDisciplineScoresKey).RedisNil()

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

	t.Run("student_has_not_discipline", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		disciplineId := 850

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1200"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1200"

		redisMock.ExpectSIsMember(studentDisciplinesKeySemester2, disciplineId).RedisNil()
		redisMock.ExpectSIsMember(studentDisciplinesKeySemester1, disciplineId).RedisNil()

		storage := Storage{
			redis:             redisClient,
			year:              2026,
			lessonTypes:       lessonTypes,
			scoreRatingLoader: NewMockScoreRatingLoaderInterface(t),
		}

		actualResult, err := storage.getDisciplineScoreResultByStudentId(1200, disciplineId)

		assert.NoError(t, err)
		assert.Equal(t, scoreApi.DisciplineScoreResult{}, actualResult)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("redis_error", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedError := errors.New("expected error")

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		disciplineId := 850

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1200"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1200"

		redisMock.ExpectSIsMember(studentDisciplinesKeySemester2, disciplineId).RedisNil()
		redisMock.ExpectSIsMember(studentDisciplinesKeySemester1, disciplineId).SetErr(expectedError)

		storage := Storage{
			redis:             redisClient,
			year:              2026,
			lessonTypes:       lessonTypes,
			scoreRatingLoader: NewMockScoreRatingLoaderInterface(t),
		}

		actualResult, actualErr := storage.getDisciplineScoreResultByStudentId(1200, disciplineId)

		assert.Error(t, actualErr)
		assert.Equal(t, expectedError, actualErr)
		assert.Equal(t, scoreApi.DisciplineScoreResult{}, actualResult)

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
