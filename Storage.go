package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v9"
	scoreApi "github.com/kneu-messenger-pigeon/score-api"
	"sort"
	"strconv"
	"time"
)

type StorageInterface interface {
	getDisciplineScoreResultsByStudentId(studentId int) (scoreApi.DisciplineScoreResults, error)
	getDisciplineScoreResultByStudentId(studentId int, disciplineId int) (scoreApi.DisciplineScoreResult, error)
}

type Storage struct {
	redis             *redis.Client
	year              int
	lessonTypes       map[int]scoreApi.LessonType
	scoreRatingLoader ScoreRatingLoaderInterface
}

const IsAbsentScoreValue = float64(-999999)

func (storage *Storage) getDisciplineScoreResultsByStudentId(studentId int) (scoreApi.DisciplineScoreResults, error) {
	semester, disciplineIds, err := storage.getStudentDisciplinesIdsForLastSemester(studentId)

	if err != nil {
		return nil, err
	}

	disciplineScoreResults := make([]scoreApi.DisciplineScoreResult, len(disciplineIds))
	for i, disciplineId := range disciplineIds {
		disciplineScoreResults[i] = scoreApi.DisciplineScoreResult{
			Discipline: scoreApi.Discipline{
				Id:   disciplineId,
				Name: storage.getDisciplineName(disciplineId),
			},
			ScoreRating: storage.scoreRatingLoader.load(storage.year, semester, disciplineId, studentId),
		}
	}

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

func (storage *Storage) getStudentDisciplinesIdsForLastSemester(studentId int) (int, []int, error) {
	var semester int
	var stringIds []string
	var err error

	stringIds = make([]string, 0)
	for semester = 2; semester >= 1; semester-- {
		studentDisciplinesKey := fmt.Sprintf("%d:%d:student_disciplines:%d", storage.year, semester, studentId)
		stringIds, err = storage.redis.SMembers(context.Background(), studentDisciplinesKey).Result()
		if err != nil && err != redis.Nil {
			return 0, nil, err
		}

		if err == nil && len(stringIds) != 0 {
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

		if err != nil && err != redis.Nil {
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

	var lessonIdCompacted string

	scores := make([]scoreApi.Score, len(rawScores))

	lessonsKeys := make([]string, len(rawScores))
	lessonIds := make([]string, len(rawScores))
	lessonsValues := make([]string, len(rawScores))
	lessonsSortMap := make(map[int]int)

	i := 0
	for lessonIdCompacted = range rawScores {
		lessonsKeys[i] = lessonIdCompacted
		i++
	}

	sort.Strings(lessonsKeys)
	for i, lessonIdCompacted = range lessonsKeys {
		lessonIds[i], _ = parseLessonIdAndHalfToString(lessonIdCompacted)
	}

	var lessonId int
	for i, v := range storage.redis.HMGet(context.Background(), disciplineKey, lessonIds...).Val() {
		lessonsValues[i] = v.(string)
		lessonId, _ = strconv.Atoi(lessonIds[i])
		lessonsSortMap[lessonId], _ = strconv.Atoi(lessonsValues[i][0:6])
	}

	for i, lessonIdCompacted = range lessonsKeys {
		if i < len(lessonsValues) {
			scores[i].Lesson, scores[i].LessonHalf = storage.makeLesson(lessonIdCompacted, lessonsValues[i])
		}

		score, _ := strconv.ParseFloat(rawScores[lessonsKeys[i]], 10)
		if IsAbsentScoreValue == score {
			scores[i].IsAbsent = true
		} else {
			scores[i].Score = float32(score)
		}

		i++
	}

	// Sort by date
	sort.SliceStable(scores, func(i, j int) bool {
		return lessonsSortMap[scores[i].Lesson.Id] < lessonsSortMap[scores[j].Lesson.Id]
	})

	return scores
}

func (storage *Storage) makeLesson(lessonIdCompacted string, lessonValue string) (lesson scoreApi.Lesson, lessonHalf int) {
	var lessonTypeId int
	lesson.Id, lessonHalf = parseLessonIdAndHalfToInt(lessonIdCompacted)
	lesson.Date, lessonTypeId = parseLessonValueString(lessonValue)
	lesson.Type = storage.lessonTypes[lessonTypeId]

	return
}

func parseLessonIdAndHalfToString(lessonIdCompacted string) (string, string) {
	return lessonIdCompacted[:len(lessonIdCompacted)-2], lessonIdCompacted[len(lessonIdCompacted)-1:]
}

func parseLessonIdAndHalfToInt(lessonIdCompact string) (id int, half int) {
	idString, halfString := parseLessonIdAndHalfToString(lessonIdCompact)
	id, _ = strconv.Atoi(idString)
	half, _ = strconv.Atoi(halfString)

	return id, half
}

func parseLessonValueString(lessonString string) (dateString string, typeId int) {
	if len(lessonString) >= 7 {
		typeId, _ = strconv.Atoi(lessonString[6:])
		dateString = lessonString[4:6] + "." + lessonString[2:4] + ".20" + lessonString[0:2]
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

		if storage.year == 0 && len(storage.lessonTypes) == 0 {
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
