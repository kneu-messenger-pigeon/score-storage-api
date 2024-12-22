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

	t.Run("testPeriodicallyUpdateGeneralDataRun_EmptyRedis", func(t *testing.T) {
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
	t.Parallel()

	t.Run("success-mixed-semester", func(t *testing.T) {
		t.Parallel()

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
					Id:   200,
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

		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).SetVal([]string{
			"100",
			"110",
			"200", // emualte that discipline was in both semester
		})
		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).SetVal([]string{
			"200",
		})

		oldDate := time.Now().Add(-MaxSemesterUpdatedInterval - time.Hour)

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:100"
		discipline2SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:110"

		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)
		discipline2SemesterUpdatedAtValue := "1" + strconv.FormatInt(oldDate.Unix(), 10)

		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)
		redisMock.ExpectGet(discipline2SemesterUpdatedAtKey).SetVal(discipline2SemesterUpdatedAtValue)

		redisMock.ExpectHGet("2026:discipline:100", "name").SetVal(expectedResults[0].Discipline.Name)
		redisMock.ExpectHGet("2026:discipline:200", "name").SetVal(expectedResults[1].Discipline.Name)

		scoreRatingLoader := NewMockScoreRatingLoaderInterface(t)
		scoreRatingLoader.On("load", 2026, 1, 100, 1100).Return(expectedResults[0].ScoreRating)
		scoreRatingLoader.On("load", 2026, 2, 200, 1100).Return(expectedResults[1].ScoreRating)

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

	t.Run("success_second_semester", func(t *testing.T) {
		t.Parallel()

		lessonTypes := GetTestLessonTypes()

		expectedResults := scoreApi.DisciplineScoreResults{
			scoreApi.DisciplineScoreResult{
				Discipline: scoreApi.Discipline{
					Id:   200,
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
					Id:   204,
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

			scoreApi.DisciplineScoreResult{
				Discipline: scoreApi.Discipline{
					Id:   210,
					Name: "Іноваційно-інвестиційний менеджмент",
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

		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1100"
		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1100"

		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).SetVal([]string{
			"200",
			"204",
			"210",
		})

		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).RedisNil()

		redisMock.ExpectHGet("2026:discipline:200", "name").SetVal(expectedResults[0].Discipline.Name)
		redisMock.ExpectHGet("2026:discipline:204", "name").SetVal(expectedResults[1].Discipline.Name)
		redisMock.ExpectHGet("2026:discipline:210", "name").SetVal(expectedResults[2].Discipline.Name)

		scoreRatingLoader := NewMockScoreRatingLoaderInterface(t)
		scoreRatingLoader.On("load", 2026, 2, 200, 1100).Return(expectedResults[0].ScoreRating)
		scoreRatingLoader.On("load", 2026, 2, 204, 1100).Return(expectedResults[1].ScoreRating)
		scoreRatingLoader.On("load", 2026, 2, 210, 1100).Return(expectedResults[2].ScoreRating)

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

		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).RedisNil()
		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).RedisNil()

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

	t.Run("redis_error_first_semester", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedError := errors.New("expected error")

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1100"

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

	t.Run("redis_error_second_semester", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedError := errors.New("expected error")

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1100"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1100"

		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).RedisNil()
		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).SetErr(expectedError)

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

	t.Run("redis_error_get_discipline_updated_at", func(t *testing.T) {
		t.Parallel()

		lessonTypes := GetTestLessonTypes()

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(false)

		studentDisciplinesKeySemester1 := "2026:1:student_disciplines:1100"
		studentDisciplinesKeySemester2 := "2026:2:student_disciplines:1100"

		redisMock.ExpectSMembers(studentDisciplinesKeySemester1).SetVal([]string{
			"100",
			"110",
			"200", // emualte that discipline was in both semester
		})
		redisMock.ExpectSMembers(studentDisciplinesKeySemester2).SetVal([]string{
			"200",
		})

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:100"
		discipline2SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:110"

		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)

		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)
		redisMock.ExpectGet(discipline2SemesterUpdatedAtKey).SetErr(assert.AnError)

		storage := Storage{
			redis:       redisClient,
			year:        2026,
			lessonTypes: lessonTypes,
		}

		actualResults, err := storage.getDisciplineScoreResultsByStudentId(1100)

		assert.Nil(t, actualResults)
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)

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
					FirstScore:  floatPointer(4.5),
					SecondScore: floatPointer(2),
					IsAbsent:    false,
				},
				{
					Lesson: scoreApi.Lesson{
						Id:   247,
						Date: time.Date(2023, time.Month(2), 12, 0, 0, 0, 0, time.Local),
						Type: lessonTypes[1],
					},
					FirstScore: floatPointer(1),
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

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:199"
		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)

		redisMock.ExpectHGet("2026:discipline:199", "name").SetVal(expectedResult.Discipline.Name)

		studentDisciplineScoresKey := "2026:1:scores:1200:199"
		disciplineKey := "2026:1:lessons:199"

		redisMock.ExpectHGetAll(studentDisciplineScoresKey).SetVal(map[string]string{
			"245:1": "4.5",
			"255:2": strconv.FormatFloat(float64(IsAbsentScoreValue), 'f', -1, 64),
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

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:199"
		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)

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

	t.Run("discipline_never_updated", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		disciplineId := 850

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:850"
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).RedisNil()

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

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:850"
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetErr(expectedError)

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

func TestGetDisciplineScore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedResult := scoreApi.DisciplineScore{
			Discipline: scoreApi.Discipline{
				Id:   199,
				Name: "Капітал!",
			},
			Score: scoreApi.Score{
				Lesson: scoreApi.Lesson{
					Id:   245,
					Date: time.Date(2023, time.Month(2), 12, 0, 0, 0, 0, time.Local),
					Type: lessonTypes[1],
				},
				FirstScore:  floatPointer(4.5),
				SecondScore: floatPointer(2),
				IsAbsent:    false,
			},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:199"
		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)

		redisMock.ExpectHGet("2026:discipline:199", "name").SetVal(expectedResult.Discipline.Name)

		studentDisciplineScoresKey := "2026:1:scores:1200:199"
		disciplineLessonsKey := "2026:1:lessons:199"
		redisMock.ExpectHGet(disciplineLessonsKey, "245").SetVal("2302121")

		expectedScoreValues := make([]interface{}, 2)
		expectedScoreValues[0] = "4.5"
		expectedScoreValues[1] = "2"

		redisMock.ExpectHMGet(studentDisciplineScoresKey, "245:1", "245:2").SetVal(expectedScoreValues)

		storage := Storage{
			redis:       redisClient,
			year:        2026,
			lessonTypes: lessonTypes,
		}

		actualResult, err := storage.getDisciplineScore(
			1200, expectedResult.Discipline.Id, expectedResult.Score.Lesson.Id,
		)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, actualResult)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("absent", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedResult := scoreApi.DisciplineScore{
			Discipline: scoreApi.Discipline{
				Id:   199,
				Name: "Капітал!",
			},
			Score: scoreApi.Score{
				Lesson: scoreApi.Lesson{
					Id:   245,
					Date: time.Date(2023, time.Month(2), 12, 0, 0, 0, 0, time.Local),
					Type: lessonTypes[1],
				},
				FirstScore:  nil,
				SecondScore: nil,
				IsAbsent:    true,
			},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:199"
		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)

		redisMock.ExpectHGet("2026:discipline:199", "name").SetVal(expectedResult.Discipline.Name)

		studentDisciplineScoresKey := "2026:1:scores:1200:199"
		disciplineLessonsKey := "2026:1:lessons:199"

		redisMock.ExpectHGet(disciplineLessonsKey, "245").SetVal("2302121")

		expectedScoreValues := make([]interface{}, 2)
		expectedScoreValues[0] = strconv.FormatFloat(float64(IsAbsentScoreValue), 'f', 0, 64)

		redisMock.ExpectHMGet(studentDisciplineScoresKey, "245:1", "245:2").SetVal(expectedScoreValues)

		storage := Storage{
			redis:       redisClient,
			year:        2026,
			lessonTypes: lessonTypes,
		}

		actualResult, err := storage.getDisciplineScore(
			1200, expectedResult.Discipline.Id, expectedResult.Score.Lesson.Id,
		)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, actualResult)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("deleted_lesson", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedResult := scoreApi.DisciplineScore{
			Discipline: scoreApi.Discipline{
				Id:   199,
				Name: "Капітал!",
			},
			Score: scoreApi.Score{},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:199"
		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)

		redisMock.ExpectHGet("2026:discipline:199", "name").SetVal(expectedResult.Discipline.Name)

		disciplineLessonsKey := "2026:1:lessons:199"
		redisMock.ExpectHGet(disciplineLessonsKey, "245").RedisNil()

		disciplineDeletedLessonsKey := "2026:1:deleted-lessons:199:245"
		redisMock.ExpectGet(disciplineDeletedLessonsKey).RedisNil()

		storage := Storage{
			redis:       redisClient,
			year:        2026,
			lessonTypes: lessonTypes,
		}

		actualResult, err := storage.getDisciplineScore(
			1200, expectedResult.Discipline.Id, 245,
		)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, actualResult)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("deleted_score", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedResult := scoreApi.DisciplineScore{
			Discipline: scoreApi.Discipline{
				Id:   199,
				Name: "Капітал!",
			},
			Score: scoreApi.Score{
				Lesson: scoreApi.Lesson{
					Id:   245,
					Date: time.Date(2023, time.Month(2), 12, 0, 0, 0, 0, time.Local),
					Type: lessonTypes[1],
				},
			},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:199"
		discipline1SemesterUpdatedAtValue := "1" + strconv.FormatInt(time.Now().Unix(), 10)
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetVal(discipline1SemesterUpdatedAtValue)

		redisMock.ExpectHGet("2026:discipline:199", "name").SetVal(expectedResult.Discipline.Name)

		disciplineLessonsKey := "2026:1:lessons:199"
		redisMock.ExpectHGet(disciplineLessonsKey, "245").RedisNil()

		disciplineDeletedLessonsKey := "2026:1:deleted-lessons:199:245"
		redisMock.ExpectGet(disciplineDeletedLessonsKey).SetVal("2302121")

		studentDisciplineScoresKey := "2026:1:scores:1200:199"
		expectedScoreValues := make([]interface{}, 2)
		redisMock.ExpectHMGet(studentDisciplineScoresKey, "245:1", "245:2").SetVal(expectedScoreValues)

		storage := Storage{
			redis:       redisClient,
			year:        2026,
			lessonTypes: lessonTypes,
		}

		actualResult, err := storage.getDisciplineScore(
			1200, expectedResult.Discipline.Id, 245,
		)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, actualResult)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("discipline_never_updated", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedResult := scoreApi.DisciplineScore{
			Discipline: scoreApi.Discipline{
				Id:   199,
				Name: "Капітал!",
			},
			Score: scoreApi.Score{},
		}

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:199"
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).RedisNil()

		storage := Storage{
			redis:       redisClient,
			year:        2026,
			lessonTypes: lessonTypes,
		}

		actualResult, err := storage.getDisciplineScore(
			1200, expectedResult.Discipline.Id, 245,
		)

		assert.NoError(t, err)
		assert.Equal(t, scoreApi.DisciplineScore{}, actualResult)
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("redis_error", func(t *testing.T) {
		lessonTypes := GetTestLessonTypes()

		expectedError := errors.New("expected error")

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.MatchExpectationsInOrder(true)

		disciplineId := 850

		discipline1SemesterUpdatedAtKey := "2026:discipline_semester_updated_at:850"
		redisMock.ExpectGet(discipline1SemesterUpdatedAtKey).SetErr(expectedError)

		storage := Storage{
			redis:             redisClient,
			year:              2026,
			lessonTypes:       lessonTypes,
			scoreRatingLoader: NewMockScoreRatingLoaderInterface(t),
		}

		actualResult, actualErr := storage.getDisciplineScore(1200, disciplineId, 245)

		assert.Error(t, actualErr)
		assert.Equal(t, expectedError, actualErr)
		assert.Equal(t, scoreApi.DisciplineScore{}, actualResult)

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

func floatPointer(value float32) *float32 {
	return &value
}
