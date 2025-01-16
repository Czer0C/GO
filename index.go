package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, Go Web Server!")
}

func main() {
	http.HandleFunc("/", handler) // Route to handler
	fmt.Println("Starting server at :8080")
	http.ListenAndServe(":8080", nil) // Start server
}
