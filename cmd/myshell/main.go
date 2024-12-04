package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var d debugger = debugger{enabled: false}

var builtIns = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
}

func isExec(file os.FileMode) bool {
	return file.Perm()&0o111 != 0 // bytewise AND against 0111 bitmask
}

func isSymlink(file os.DirEntry) bool {
	return file.Type()&os.ModeSymlink != 0
}

func findExecs() ([]string, error) {
	pathEnv, ok := os.LookupEnv("PATH")
	if !ok || pathEnv == "" {
		return nil, fmt.Errorf("PATH environment variable is not set")
	}
	paths := strings.Split(pathEnv, ":")
	d.print("paths: ", strings.Join(paths, " "))
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
			continue
			// return nil, fmt.Errorf("failed to read dir: %s", err)
		}
		for _, dir := range dirs {
			info, _ := dir.Info()
			if dir.Type().IsRegular() || isSymlink(dir) {
				if isExec(info.Mode()) {
					execs = append(execs, path.Join(p, dir.Name()))
				}
			}
		}
	}
	d.print(execs)
	return execs, nil
}

func getExec(execName string, execs []string) (string, bool) {
	for _, ex := range execs {
		if filepath.Base(ex) == execName {
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
		if inLen < 2 {
			fmt.Println("type: missing operand")
			return nil
		}
		execs, err := findExecs()
		d.print(execs)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		cmd := in[1]
		if _, isBuiltin := builtIns[cmd]; isBuiltin {
			fmt.Println(cmd, "is a shell builtin")
		} else if exec, found := getExec(cmd, execs); found {
			fmt.Println(cmd, "is", exec)
		} else {
			fmt.Printf("%s: not found\n", cmd)
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
