package main

import (
	"io"
	"log"
	"net/http"
	"os"

	readmeHandler "github.com/webkom/readme-as-a-function/pkg/handler"
)

func main() {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Listening on :8000")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		defer func() {
			logger.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
		}()
		input, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, readmeHandler.Handle(input))
	})
	logger.Fatal(http.ListenAndServe(":8000", nil))
}
