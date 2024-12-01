package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

var builtIns = make(map[string]bool)

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
		} else {
			os.Exit(0)
		}
	case "echo":
		echoed := strings.Join(in[1:], " ")
		fmt.Println(echoed)
	case "type":
		if _, exist := builtIns[in[1]]; exist {
			fmt.Println(in[1], "is a shell builtin")
		} else {
			fmt.Printf("%s: not found\n", in[1])
		}
	default:
		fmt.Printf("%s: not found\n", input)
	}
	return nil
}

func main() {
	cmd := []string{"exit", "echo", "type"}
	for _, v := range cmd {
		builtIns[v] = true
	}

	for {
		fmt.Fprint(os.Stdout, "$ ")
		err := REPL()
		if err != nil {
			log.Println(err)
			break
		}
	}
}
