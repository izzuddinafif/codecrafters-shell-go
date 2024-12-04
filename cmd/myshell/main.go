package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var d debugger = debugger{enabled: true}

// isExec checks if a file is executable by checking if it's a regular file
// and if any execute bit is set when masking with 0111 (binary 000000111).
// The bit mask 0111 checks owner(100), group(010), and other(001) execute
// permissions by performing a bitwise AND with the file's permission bits.
func isExec(file os.FileMode) bool {
	return file.IsRegular() && file.Perm()&0o111 != 0
}

// getExec searches for an executable in the system PATH and returns its full path.
// It checks each directory in PATH for a file matching execName that has execute
// permissions. Returns an error if PATH is not set, executable is not found, or
// encounters permissions/IO errors.
func getExec(execName string) (string, error) {
	pathEnv, ok := os.LookupEnv("PATH")
	if !ok || pathEnv == "" {
		return "", fmt.Errorf("PATH environment variable is not set")
	}
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	d.print("paths: ", strings.Join(paths, " "))

	for _, dir := range paths {
		fullPath := filepath.Join(dir, execName)
		d.printf("looking for %s in %s", filepath.Base(fullPath), fullPath)

		info, err := os.Stat(fullPath)
		if err == nil {
			if !info.IsDir() && isExec(info.Mode()) {
				return fullPath, nil
			}
		} else if !os.IsNotExist(err) {
			// Some other error occured
			return "", err
		} else {
			// only continue if the error is os.ErrNotExist
			continue
		}
	}
	return "", fmt.Errorf("%s: not found", execName)
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
		if inLen == 2 {
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
		cmd := in[1]
		if _, isBuiltin := builtIns[cmd]; isBuiltin {
			fmt.Println(cmd, "is a shell builtin")
		} else if exec, err := getExec(cmd); err == nil {
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
