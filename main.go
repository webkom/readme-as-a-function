package main

import (
	"fmt"
	"github.com/webkom/readme-as-a-function/pkg/handler"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read standard input: %s", err.Error())
	}

	fmt.Println(handler.Handle(input))
}
