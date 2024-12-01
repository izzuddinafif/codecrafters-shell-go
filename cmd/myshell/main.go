package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

// REPL is Read, Eval and Print Loop that reads user input,
// prints the result and wait for the next input.
func REPL() (err error) {
	// Wait for user input
	input, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		return err
	}
	in := strings.Fields(string(input))
	inLen := len(input)
	switch in[0] {
	case "exit":
		if inLen > 1 {
			if in[1] == "0" {
				os.Exit(0)
			}
		}
	default:
		fmt.Printf("%s: not found\n", input)
	}
	return nil
}

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")
		err := REPL()
		if err != nil {
			log.Println(err)
			break
		}
	}
}
