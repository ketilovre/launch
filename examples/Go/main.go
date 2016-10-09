package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Print("Serving hello!")
		fmt.Fprint(w, "Hello, World!")
	})

	log.Printf("Starting a server on port %v", 5000)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
