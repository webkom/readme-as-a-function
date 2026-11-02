package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/webkom/readme-as-a-function/pkg/handler"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read standard input: %s", err.Error())
	}

	fmt.Println(handler.Handle(input))
}
