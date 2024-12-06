package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type debugger struct {
	enabled bool
}

func (d debugger) print(a ...interface{}) {
	if d.enabled {
		log.Print(a...)
	}
}

func (d debugger) printf(format string, a ...interface{}) {
	if d.enabled {
		log.Printf(format, a...)
	}
}

var builtIns = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
}

// TODO: implement this type
type command struct {
	name     string
	args     []string
	internal bool
	path     string
	err      error
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}

func newCommand() *command {
	return &command{
		name:     "",
		args:     []string{},
		internal: false,
		path:     "",
		err:      nil,

		// Default to standard input, output, and error
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

func (cmd *command) execute() error {

	if cmd.internal {
		switch cmd.name {
		case "exit":
			if len(cmd.args) > 1 {
				fmt.Println("exit: too many arguments")
				return nil
			}
			if len(cmd.args) == 2 {
				code, err := strconv.Atoi(cmd.args[0])
				if err != nil || code > 255 || code < 0 {
					fmt.Println("exit: invalid argument")
					return nil
				}
				os.Exit(code)
			}
			os.Exit(0)
		case "echo":
			echoed := strings.Join(cmd.args, " ")
			fmt.Println(echoed)
		case "type":
			if len(cmd.args) < 1 {
				fmt.Println("type: missing operand")
				return nil
			}
			c := cmd.args[0]
			if _, isBuiltin := builtIns[c]; isBuiltin {
				fmt.Println(c, "is a shell builtin")
			} else if path, err := getCmdPath(c); err == nil {
				fmt.Println(c, "is", path)
			} else if err == os.ErrNotExist {
				fmt.Printf("%s: not found\n", c)
			} else {
				fmt.Println("error:", err)
			}
		}
	} else {

	}

	return nil
}
