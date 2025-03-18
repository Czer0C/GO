package main

import (
	"project/jobs"
	"project/utils"
)

func main() {

	limit := 10
	t1 := utils.ISOToUnix("2025-03-09T00:00:00Z")
	t2 := utils.ISOToUnix("2025-03-10T00:00:00Z")
	rateLimit := 5
	delaySeconds := 5

	jobs.GetOpmsDataPipeline(
		limit,
		t1,
		t2,
		rateLimit,
		delaySeconds,
		"custom_fan_data.csv",
	)

}

// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"project/database"
// 	"project/handlers"

// 	"github.com/gorilla/mux"
// )

// func main() {
// 	database.Connect()

// 	router := mux.NewRouter()
// 	router.HandleFunc("/users", handlers.GetUsers).Methods("GET")
// 	router.HandleFunc("/users", handlers.CreateUser).Methods("POST")
// 	router.HandleFunc("/users/{id}", handlers.UpdateUser).Methods("PUT")
// 	router.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")

// 	fmt.Println("Server running on port 8080")
// 	log.Fatal(http.ListenAndServe(":8000", router))
// }

// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"project/database"
// 	"project/handlers"

// 	"github.com/gorilla/mux"
// )

// func main() {
// 	database.Connect()

// 	router := mux.NewRouter()
// 	router.HandleFunc("/users", handlers.GetUsers).Methods("GET")
// 	router.HandleFunc("/users/{id}", handlers.GetUser).Methods("GET")
// 	router.HandleFunc("/users", handlers.CreateUser).Methods("POST")
// 	router.HandleFunc("/users/{id}", handlers.UpdateUser).Methods("PUT")
// 	router.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")

// 	fmt.Println("Server running on port 8080!")
// 	log.Fatal(http.ListenAndServe(":8080", router))
// }
