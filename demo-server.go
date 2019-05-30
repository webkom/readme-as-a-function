package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		input, _ := ioutil.ReadAll(r.Body)
		fmt.Fprintf(w, Handle(input))
	})
	http.ListenAndServe(":8000", nil)
}
