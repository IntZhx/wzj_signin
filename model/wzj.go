package model

type SignResultData struct {
	SignRank    int `json:"signRank"`
	StudentRank int `json:"studentRank"`
}

type SignData struct {
	CourseID  int    `json:"courseId"`
	SignID    int    `json:"signId"`
	IsGPS     int    `json:"isGPS"`
	IsQR      int    `json:"isQR"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	StartYear int    `json:"startYear"`
	Term      string `json:"term"`
	Cover     string `json:"cover"`
}