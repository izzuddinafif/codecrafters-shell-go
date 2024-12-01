package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

var builtIns = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
}

// REPL is Read, Eval and Print Loop that reads user input,
// prints the result and wait for the next input.
func REPL() (err error) {
	// Wait for user input
	input, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		fmt.Println("failed to read input:", err)
		return err
	}
	in := strings.Fields(string(input))
	inLen := len(in)
	if inLen == 0 {
		return nil // ignore empty input
	}
	switch in[0] {
	case "exit":
		if inLen > 2 {
			fmt.Println("exit: too many arguments", inLen)
			return nil
		}
		if inLen == 2 || in[1] != "0" {
			code, err := strconv.Atoi(in[1])
			if err != nil || code > 255 || code < 0 {
				fmt.Println("exit: invalid argument")
				return nil
			}
			os.Exit(code)
		}
		os.Exit(0)
	case "echo":
		echoed := strings.Join(in[1:], " ")
		fmt.Println(echoed)
	case "type":
		if inLen < 2 {
			fmt.Println("type: missing operand")
			return nil
		}
		if _, exist := builtIns[in[1]]; exist {
			fmt.Println(in[1], "is a shell builtin")
		} else {
			fmt.Printf("%s: not found\n", in[1])
		}
	default:
		fmt.Printf("%s: not found\n", strings.Join(in, " "))
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
