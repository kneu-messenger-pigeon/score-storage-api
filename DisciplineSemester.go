package main

type DisciplineSemester struct {
	Semester     int
	DisciplineId int
}

type DisciplineSemesters []DisciplineSemester

func (disciplines DisciplineSemesters) Has(disciplineId int) bool {
	for _, discipline := range disciplines {
		if discipline.DisciplineId == disciplineId {
			return true
		}
	}
	return false
}
