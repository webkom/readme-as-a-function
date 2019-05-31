package main

import (
	"fmt"
	readmeHandler "github.com/webkom/readme-as-a-function/pkg/handler"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Listening on :8000")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		defer func() {
			logger.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
		}()
		input, _ := ioutil.ReadAll(r.Body)
		fmt.Fprintf(w, readmeHandler.Handle(input))
	})
	http.ListenAndServe(":8000", nil)
}
