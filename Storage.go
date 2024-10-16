package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	scoreApi "github.com/kneu-messenger-pigeon/score-api"
	"github.com/redis/go-redis/v9"
	"sort"
	"strconv"
	"sync"
	"time"
)

type StorageInterface interface {
	getDisciplineScoreResultsByStudentId(studentId int) (scoreApi.DisciplineScoreResults, error)
	getDisciplineScoreResultByStudentId(studentId int, disciplineId int) (scoreApi.DisciplineScoreResult, error)
	getDisciplineScore(studentId int, disciplineId int, lessonId int) (scoreApi.DisciplineScore, error)
}

type Storage struct {
	redis             *redis.Client
	year              int
	lessonTypes       map[int]scoreApi.LessonType
	scoreRatingLoader ScoreRatingLoaderInterface
}

const IsAbsentScoreValue = float32(-999999)
const DisciplineAmountThresholdForSemesterSwitch = 2

func (storage *Storage) getDisciplineScoreResultsByStudentId(studentId int) (scoreApi.DisciplineScoreResults, error) {
	semester, disciplineIds, err := storage.getStudentDisciplinesIdsForLastSemester(studentId)

	if err != nil {
		return nil, err
	}

	disciplineScoreResults := make([]scoreApi.DisciplineScoreResult, len(disciplineIds))

	wg := sync.WaitGroup{}
	wg.Add(len(disciplineIds))

	for _index := range disciplineIds {
		go func(index int) {
			disciplineId := disciplineIds[index]
			disciplineScoreResults[index] = scoreApi.DisciplineScoreResult{
				Discipline: scoreApi.Discipline{
					Id:   disciplineId,
					Name: storage.getDisciplineName(disciplineId),
				},
				ScoreRating: storage.scoreRatingLoader.load(storage.year, semester, disciplineId, studentId),
			}
			wg.Done()
		}(_index)
	}

	wg.Wait()

	return disciplineScoreResults, nil
}

func (storage *Storage) getDisciplineScoreResultByStudentId(studentId int, disciplineId int) (scoreApi.DisciplineScoreResult, error) {
	semester, err := storage.getSemesterByStudentIdAndDisciplineId(studentId, disciplineId)

	if err != nil {
		return scoreApi.DisciplineScoreResult{}, err
	}

	if semester == 0 {
		return scoreApi.DisciplineScoreResult{}, nil
	}

	return scoreApi.DisciplineScoreResult{
		Discipline: scoreApi.Discipline{
			Id:   disciplineId,
			Name: storage.getDisciplineName(disciplineId),
		},
		ScoreRating: storage.scoreRatingLoader.load(storage.year, semester, disciplineId, studentId),
		Scores:      storage.getScores(semester, disciplineId, studentId),
	}, nil
}

func (storage *Storage) getDisciplineScore(studentId int, disciplineId int, lessonId int) (scoreApi.DisciplineScore, error) {
	semester, err := storage.getSemesterByStudentIdAndDisciplineId(studentId, disciplineId)

	if err != nil {
		return scoreApi.DisciplineScore{}, err
	}

	if semester == 0 {
		return scoreApi.DisciplineScore{}, nil
	}

	return scoreApi.DisciplineScore{
		Discipline: scoreApi.Discipline{
			Id:   disciplineId,
			Name: storage.getDisciplineName(disciplineId),
		},
		Score: storage.getScore(semester, disciplineId, studentId, lessonId),
	}, nil
}

func (storage *Storage) getStudentDisciplinesIdsForLastSemester(studentId int) (int, []int, error) {
	var semester int
	var stringIds []string
	var err error

	stringIds = make([]string, 0)
	for semester = 2; semester >= 1; semester-- {
		studentDisciplinesKey := fmt.Sprintf("%d:%d:student_disciplines:%d", storage.year, semester, studentId)
		stringIds, err = storage.redis.SMembers(context.Background(), studentDisciplinesKey).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return 0, nil, err
		}

		if err == nil && len(stringIds) >= 2 {
			break
		}
	}

	ids := make([]int, len(stringIds))
	for index, stringId := range stringIds {
		ids[index], _ = strconv.Atoi(stringId)
	}
	return semester, ids, nil
}

func (storage *Storage) getSemesterByStudentIdAndDisciplineId(studentId int, disciplineId int) (int, error) {
	for semester := 2; semester >= 1; semester-- {
		studentDisciplinesKey := fmt.Sprintf("%d:%d:student_disciplines:%d", storage.year, semester, studentId)

		isMember, err := storage.redis.SIsMember(context.Background(), studentDisciplinesKey, disciplineId).Result()

		if err != nil && !errors.Is(err, redis.Nil) {
			return 0, err
		} else if isMember {
			return semester, nil
		}
	}

	return 0, nil
}

func (storage *Storage) getScores(semester int, disciplineId int, studentId int) []scoreApi.Score {
	studentDisciplineScoresKey := fmt.Sprintf("%d:%d:scores:%d:%d", storage.year, semester, studentId, disciplineId)
	disciplineKey := fmt.Sprintf("%d:%d:lessons:%d", storage.year, semester, disciplineId)

	rawScores := storage.redis.HGetAll(context.Background(), studentDisciplineScoresKey).Val()
	if len(rawScores) == 0 {
		return make([]scoreApi.Score, 0)
	}

	var lessonId int

	lessons := make(map[int]string)
	for lessonIdString, lessonValue := range storage.redis.HGetAll(context.Background(), disciplineKey).Val() {
		lessonId, _ = strconv.Atoi(lessonIdString)
		lessons[lessonId] = lessonValue
	}

	var lessonTypeId int
	var lessonDate time.Time
	var lessonHalf int
	var scoreValue *float32
	var exists bool

	scoresMap := make(map[int]*scoreApi.Score, len(rawScores))

	for lessonIdCompacted, scoreString := range rawScores {
		lessonId, lessonHalf = parseLessonIdAndHalf(lessonIdCompacted)

		if _, exists = scoresMap[lessonId]; !exists {
			lessonDate, lessonTypeId = parseLessonValueString(lessons[lessonId])
			scoresMap[lessonId] = &scoreApi.Score{
				Lesson: scoreApi.Lesson{
					Id:   lessonId,
					Date: lessonDate,
					Type: storage.lessonTypes[lessonTypeId],
				},
			}
		}

		scoreValue = parseFloat(scoreString)
		if IsAbsentScoreValue == *scoreValue {
			scoresMap[lessonId].IsAbsent = true
		} else if lessonHalf == 1 {
			scoresMap[lessonId].FirstScore = scoreValue
		} else if lessonHalf == 2 {
			scoresMap[lessonId].SecondScore = scoreValue
		}
	}

	scores := make([]scoreApi.Score, len(scoresMap))
	i := 0
	for _, score := range scoresMap {
		scores[i] = *score
		i++
	}

	// Sort by date
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Lesson.Date.Equal(scores[j].Lesson.Date) {
			return scores[i].Lesson.Id < scores[j].Lesson.Id
		} else {
			return scores[i].Lesson.Date.Before(scores[j].Lesson.Date)
		}
	})

	return scores
}

func (storage *Storage) getScore(semester int, disciplineId int, studentId int, lessonId int) scoreApi.Score {
	ctx := context.Background()
	studentDisciplineScoresKey := fmt.Sprintf("%d:%d:scores:%d:%d", storage.year, semester, studentId, disciplineId)
	disciplineLessonsKey := fmt.Sprintf("%d:%d:lessons:%d", storage.year, semester, disciplineId)

	lessonValue := storage.redis.HGet(ctx, disciplineLessonsKey, strconv.Itoa(lessonId)).Val()

	if lessonValue == "" {
		deletedLessonKey := fmt.Sprintf(
			"%d:%d:deleted-lessons:%d:%d",
			storage.year, semester, disciplineId, lessonId,
		)

		lessonValue = storage.redis.Get(ctx, deletedLessonKey).Val()
	}

	if lessonValue == "" {
		return scoreApi.Score{}
	}

	lessonDate, lessonTypeId := parseLessonValueString(lessonValue)

	score := scoreApi.Score{
		Lesson: scoreApi.Lesson{
			Id:   lessonId,
			Date: lessonDate,
			Type: storage.lessonTypes[lessonTypeId],
		},
	}

	lessonIdPrefix := strconv.Itoa(lessonId) + ":"
	rawScores := storage.redis.HMGet(ctx, studentDisciplineScoresKey, lessonIdPrefix+"1", lessonIdPrefix+"2").Val()

	var scoreValue *float32
	for lessonHalf, scoreString := range rawScores {
		if scoreString != nil {
			scoreValue = parseFloat(scoreString)
			if IsAbsentScoreValue == *scoreValue {
				score.IsAbsent = true
			} else if lessonHalf == 0 {
				score.FirstScore = scoreValue
			} else if lessonHalf == 1 {
				score.SecondScore = scoreValue
			}
		}
	}

	return score
}

func parseFloat(input interface{}) *float32 {
	f64, _ := strconv.ParseFloat(input.(string), 10)
	f32 := float32(f64)
	return &f32
}

func parseLessonIdAndHalf(lessonIdCompacted string) (lessonId int, lessonHalf int) {
	lessonId, _ = strconv.Atoi(lessonIdCompacted[:len(lessonIdCompacted)-2])
	lessonHalf, _ = strconv.Atoi(lessonIdCompacted[len(lessonIdCompacted)-1:])

	return
}

func parseLessonValueString(lessonString string) (date time.Time, typeId int) {
	if len(lessonString) >= 7 {
		typeId, _ = strconv.Atoi(lessonString[6:])
		year, _ := strconv.Atoi(lessonString[0:2])
		month, _ := strconv.Atoi(lessonString[2:4])
		day, _ := strconv.Atoi(lessonString[4:6])

		date = time.Date(2000+year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	}
	return
}

func (storage *Storage) getDisciplineName(disciplineId int) string {
	return storage.redis.HGet(
		context.Background(),
		fmt.Sprintf("%d:discipline:%d", storage.year, disciplineId), "name",
	).Val()
}

func (storage *Storage) periodicallyUpdateGeneralData(ctx context.Context) {
	var year int
	var lessonTypesJSON []byte
	var lessonTypes []scoreApi.LessonType

	for ctx.Err() == nil {
		year, _ = storage.redis.Get(context.Background(), "currentYear").Int()
		if year >= 2022 {
			storage.year = year
		}

		lessonTypesJSON, _ = storage.redis.Get(context.Background(), "lessonTypes").Bytes()
		if len(lessonTypesJSON) > 1 && json.Unmarshal(lessonTypesJSON, &lessonTypes) == nil {
			storage.lessonTypes = makeLessonTypesMap(&lessonTypes)
		}

		if storage.year == 0 || len(storage.lessonTypes) == 0 {
			time.Sleep(time.Minute)
		} else {
			time.Sleep(time.Hour * 12)
		}
	}
}

func makeLessonTypesMap(lessonTypesSlice *[]scoreApi.LessonType) map[int]scoreApi.LessonType {
	lessonTypesMap := map[int]scoreApi.LessonType{}
	for _, lessonType := range *lessonTypesSlice {
		lessonTypesMap[lessonType.Id] = lessonType
	}

	return lessonTypesMap
}

func NewStorage(redis *redis.Client, ctx context.Context) *Storage {
	storage := &Storage{
		redis: redis,
		scoreRatingLoader: &ScoreRatingLoader{
			redis: redis,
		},
	}

	go storage.periodicallyUpdateGeneralData(ctx)

	return storage
}
