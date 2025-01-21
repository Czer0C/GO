package main

import (
	"fmt"
	"log"
	"net/http"
	"project/database"
	"project/handlers"

	"github.com/gorilla/mux"
)

func main() {
	database.Connect()

	router := mux.NewRouter()
	router.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	router.HandleFunc("/users/{id}", handlers.GetUser).Methods("GET")
	router.HandleFunc("/users", handlers.CreateUser).Methods("POST")
	router.HandleFunc("/users/{id}", handlers.UpdateUser).Methods("PUT")
	router.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")

	fmt.Println("Server running on port 8080!")
	log.Fatal(http.ListenAndServe(":8080", router))
}
