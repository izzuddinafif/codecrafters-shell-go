package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	// Uncomment this block to pass the first stage
	fmt.Fprint(os.Stdout, "$ ")

	// Wait for user input
	input, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: not found\n", input)

}
