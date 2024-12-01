package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

var builtIns = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
}

func isExec(mode os.FileMode) bool {
	return mode&0111 != 0 // bytewise AND against 0111 bitmask
}

func findExecs() ([]string, error) {
	PATH := os.Getenv("PATH")
	paths := strings.Split(PATH, ":")
	execs := make([]string, 0)

	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
			// return nil, fmt.Errorf("failed to open %s: %s", p, err)
		}
		dirs, err := f.ReadDir(-1)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read dir: %s", err)
		}
		for _, dir := range dirs {
			info, _ := dir.Info()
			if dir.Type().IsRegular() && isExec(info.Mode()) {
				execs = append(execs, path.Join(p, dir.Name()))
			}
		}
	}
	return execs, nil
}

func getExec(exec string, execs []string) (string, bool) {
	for _, ex := range execs {
		if strings.HasSuffix(ex, exec) {
			return ex, true
		}
	}
	return "", false
}

// REPL is Read, Eval and Print Loop function that reads user
// input, prints the result and wait for the next input.
func REPL() (err error) {
	// Wait for user input
	input, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read input: %s", err)
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
		execs, err := findExecs()
		if err != nil {
			fmt.Println(err)
			return nil
		}
		if inLen < 2 {
			fmt.Println("type: missing operand")
			return nil
		}
		if _, exist := builtIns[in[1]]; exist {
			fmt.Println(in[1], "is a shell builtin")
		} else if exec, exist := getExec(in[1], execs); exist {
			fmt.Println(in[1], "is", exec)
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
