package utils

import (
	"github.com/joho/godotenv"
)

func GetTokenIOT() string {
	envFile, _ := godotenv.Read(".env")

	envFileToken := envFile["TOKEN"]

	var MOCK_TOKEN = envFileToken

	return MOCK_TOKEN
}
