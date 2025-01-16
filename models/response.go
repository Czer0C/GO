package models

type Response struct {
	Data        []User `json:"data"`
	Total       int    `json:"total"`
	ExecuteTime string `json:"excute_time"`
}
