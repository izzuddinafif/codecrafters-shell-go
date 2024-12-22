/*
This is Afif's Implementation of Shell.
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
)

type debugger struct {
	enabled bool
}

var d debugger = debugger{enabled: false}

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

var builtIns = map[string]struct{}{
	"exit": {},
	"echo": {},
	"type": {},
	"pwd":  {},
	"cd":   {},
}

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
	// handle internal command
	if cmd.internal {
		switch cmd.name {
		case "exit":
			if len(cmd.args) > 1 {
				return fmt.Errorf("exit: too many arguments")
			}
			if len(cmd.args) == 2 {
				code, err := strconv.Atoi(cmd.args[0])
				if err != nil || code > 255 || code < 0 {
					return fmt.Errorf("exit: invalid argument")
				}
				os.Exit(code)
			}
			os.Exit(0)
		case "echo":
			echoed := strings.Join(cmd.args, " ")
			fmt.Println(echoed)
		case "type":
			if len(cmd.args) < 1 {
				return fmt.Errorf("type: missing operand")
			}
			for _, c := range cmd.args {
				if _, isBuiltin := builtIns[c]; isBuiltin {
					fmt.Println(c, "is a shell builtin")
				} else if path, err := getCmdPath(c); err == nil {
					fmt.Println(c, "is", path)
				} else if err == os.ErrNotExist {
					return fmt.Errorf("%v: not found", c)
				} else {
					return fmt.Errorf("type: %v", err)
				}
			}
		case "pwd":
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("pwd: %v", err)
			}
			fmt.Println(wd)
		case "cd":
			cd := func(dir string) error {
				err := os.Chdir(dir)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("cd: %v: No such file or directory", dir)
					} else {
						// other error occurs
						return fmt.Errorf("cd: %s: %v", dir, err)
					}
				}
				return nil
			}
			// handle tilde or empty args
			if len(cmd.args) == 0 {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				return cd(homeDir)
			}
			if strings.HasPrefix(cmd.args[0], "~") {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				targetDir := strings.TrimPrefix(cmd.args[0], "~")
				dir := filepath.Join(homeDir, targetDir)
				dir = filepath.Clean(dir)
				return cd(dir)
			}

			dir := cmd.args[0]
			return cd(dir)
		}

		// handle external command
	} else {
		c := exec.Command(cmd.name, cmd.args...)
		c.Stdin = cmd.stdin
		c.Stdout = cmd.stdout
		c.Stderr = cmd.stderr

		cmd.err = c.Run()
		if cmd.err != nil {
			return fmt.Errorf("%s: %v", cmd.name, cmd.err)
		}
	}

	return nil
}

// isExec checks if a file is executable by checking if it's a regular file
// and if any execute bit is set when masking with 0111 (binary 000000111).
// The bit mask 0111 checks owner(100), group(010), and other(001) execute
// permissions by performing a bitwise AND with the file's permission bits.
func isExec(file os.FileMode) bool {
	return file.IsRegular() && file.Perm()&0o111 != 0
}

// getCmdPath searches for an executable in the system PATH and returns its full path.
// It checks each directory in PATH for a file matching execName that has execute
// permissions. Returns an error if PATH is not set, executable is not found, or
// encounters permissions/IO errors.
func getCmdPath(execName string) (string, error) {
	pathEnv, ok := os.LookupEnv("PATH")
	if !ok || pathEnv == "" {
		return "", fmt.Errorf("PATH environment variable is not set")
	}
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	// d.print("paths: ", strings.Join(paths, " "))

	for _, dir := range paths {
		fullPath := filepath.Join(dir, execName)
		// d.printf("looking for %s in %s", filepath.Base(fullPath), filepath.Dir(fullPath))

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
	return "", os.ErrNotExist
}

// TODO: Add better handling for missing closing single quote (newline support)
// handleArgs splits a string of arguments into a slice, preserving quoted
// sections as single arguments. Returns an error if there is missing closing
// quote, for now.
func handleArgs(args string) ([]string, error) {
	var argsList []string
	var buf strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	isEscaped := false

	for _, c := range args {
		d.print("switching: ", string(c))
		switch {
		case c == '"':
			if inDoubleQuote {
				if isEscaped {
					isEscaped = false
					buf.WriteRune(c)
				} else {
					inDoubleQuote = false
					argsList = append(argsList, buf.String())
					buf.Reset()
				}
			} else {
				inDoubleQuote = true
			}
		case c == '\\':
			if inDoubleQuote {
				if isEscaped {
					isEscaped = false
					buf.WriteRune(c)
				} else {
					isEscaped = true
				}
			}
		// case c == ''
		case c == '\'':
			if inDoubleQuote {
				isEscaped = false
				buf.WriteRune(c)
			} else if inSingleQuote {
				inSingleQuote = false
				d.print("appending inside quote: ", buf.String())
				argsList = append(argsList, buf.String())
				buf.Reset()
			} else {
				inSingleQuote = true
			}
		case c == ' ':
			if inDoubleQuote || inSingleQuote {
				d.print("writing space")
				buf.WriteRune(c)
			} else if buf.Len() > 0 && !inSingleQuote && !inDoubleQuote {
				d.print("appending outside quote: ", buf.String())
				argsList = append(argsList, buf.String())
				buf.Reset()
			}
		default:
			d.print("writing: ", string(c))
			buf.WriteRune(c)
		}
	}
	if inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("missing closing quote")
	}
	if buf.Len() > 0 {
		argsList = append(argsList, buf.String())
	}
	return argsList, nil
}

// parseUserInput reads user input, split it into a command and arguments,
// then determines if the command is built-in or external, if it's external,
// gets the command's path via getCmdPath. Handles quoting via handleArgs.
func parseUserInput() (*command, error) {
	cmd := newCommand()

	readBytes, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		if err == io.EOF {
			return cmd, io.EOF
		}
		return cmd, fmt.Errorf("failed to read input: %s", err)
	}
	if len(readBytes) == 0 {
		return cmd, nil
	}
	readString := string(readBytes)
	input := strings.TrimLeft(readString, " \t")
	parts := strings.SplitN(input, " ", 2)
	cmd.name = parts[0]

	if len(parts) > 1 {
		args := parts[1]
		args = strings.TrimLeft(args, " \t")
		cmd.args, cmd.err = handleArgs(args)
		if cmd.err != nil {
			return cmd, cmd.err
		}
	}

	_, cmd.internal = builtIns[cmd.name]
	if !cmd.internal {
		cmd.path, cmd.err = getCmdPath(cmd.name)
		if cmd.err != nil {
			return cmd, cmd.err
		}
	}
	return cmd, nil
}

// !!! DEPRECATED !!!
// REPL is Read, Eval and Print Loop function that reads user
// input, prints the result and wait for the next input.
/*
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
		} else if exec, err := getCmdPath(cmd); err == nil {
			fmt.Println(cmd, "is", exec)
		} else {
			fmt.Printf("type: %s: not found\n", cmd)
		}
	default:
		fmt.Printf("%s: not found\n", strings.Join(in, " "))
	}
	return nil
}
*/

// REPLv2 reimplements the former version with the addition of
// type command struct integration.
func REPLv2() {
	cmd, err := parseUserInput()
	if err != nil {
		if err == io.EOF {
			fmt.Println("\nHave a good one!👋")
			os.Exit(0) // exit when ctrl+d is pressed
		} else if err == os.ErrNotExist {
			fmt.Fprintf(cmd.stderr, "%s: command not found\n", cmd.name)
			return
		}
		fmt.Fprintln(cmd.stderr, err)
		return
	}
	if len(cmd.name) == 0 {
		return
	}

	cmd.err = cmd.execute()
	if cmd.err != nil {
		fmt.Fprintln(cmd.stderr, cmd.err)
	}
}

// handleInterrupt handles interrupt signal with custom behaviour
func handleInterrupt() {
	sigChan := make(chan os.Signal, 1)

	// listen for ctrl+c keystroke
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		for range sigChan {
			fmt.Fprintln(os.Stdout)
			fmt.Fprint(os.Stdout, "$ ")
		}
	}()
}

func main() {
	handleInterrupt() // set up ctrl+c handling
	for {
		fmt.Fprint(os.Stdout, "$ ")
		REPLv2()
	}
}
