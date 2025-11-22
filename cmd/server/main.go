package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dewitt/dewitt-blog/internal/handler"
)

const postFile = "plan.md"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	h := handler.New(postFile)
	http.HandleFunc("/", h)

	log.Printf("Listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
