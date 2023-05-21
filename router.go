package main

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

func setupRouter(out io.Writer, storage StorageInterface) *gin.Engine {
	apiController := &ApiController{
		out:     out,
		storage: storage,
	}

	r := gin.New()
	r.GET("/v1/students/:student_id/disciplines", apiController.getStudentDisciplines)
	r.GET("/v1/students/:student_id/disciplines/:discipline_id", apiController.getStudentDiscipline)
	r.GET("/v1/students/:student_id/disciplines/:discipline_id/scores/:lesson_id", apiController.getStudentDisciplineScore)

	r.GET("/healthcheck", func(c *gin.Context) {
		c.String(http.StatusOK, "health")
	})

	return r
}
