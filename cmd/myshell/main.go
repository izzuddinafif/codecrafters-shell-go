/*
This is Afif's Implementation of Shell.
*/
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	TAB = 9

	// Keep in mind !!!
	ENTER_LF = 10
	ENTER_CR = 13
	// Added LF (Unix-like systems) since I only used CR (13) but the tester won't
	// detect it since their env uses LF instead of CR. Wasted lots of hours here :(
	// thanks DeepSeek-R1 for detecting this issue.

	BACKSPACE = 127
	CTRL_C    = 3
	CTRL_D    = 4
)

var CONTROL = map[int]string{
	CTRL_C: "CTRL_C",
	CTRL_D: "CTRL_D",
}

type execCache struct {
	exec      []string  // exec names
	timestamp time.Time // cache creation time
}

// TODO: implement exec autocompletion with cache
func newExecCache() *execCache {
	return &execCache{
		exec:      listAllExecInPath(),
		timestamp: time.Now(),
	}
}

var execs execCache = *newExecCache()

type debugger struct {
	enabled bool
}

var d debugger = debugger{enabled: false}

func (d debugger) print(a ...interface{}) {
	if d.enabled {
		fmt.Fprint(os.Stderr, "[DEBUG] ")
		for i, v := range a {
			fmt.Fprint(os.Stderr, v)
			if i < len(a) {
				fmt.Fprint(os.Stderr, " ")
			}
		}
		fmt.Fprint(os.Stderr, "\r\n")
	}
}

func (d debugger) printf(format string, a ...interface{}) {
	if d.enabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\r\n", a...) // Add \r
	}
}

// Maps provide constant-time complexity (O(1)) for key lookups
// instead of slices or array that require linear-time complexity (O(n))
var builtIns = map[string]struct{}{
	// using empty struct to save memory,
	// because it takes 0 bytes but still a valid map key
	"exit": {},
	"echo": {},
	"type": {},
	"pwd":  {},
	"cd":   {},
}

type shell struct {
	oldState    *term.State  // Terminal state to restore
	inputBuffer bytes.Buffer // Current user input
	stdinFD     int          // Needed for term operations
}

// Initialize with terminal setup
func newShell() (*shell, error) {
	stdinFd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return nil, err
	}
	return &shell{
		oldState: oldState,
		stdinFD:  stdinFd,
	}, nil
}

func (s *shell) printPrompt() {
	fmt.Fprint(os.Stdout, "\r$ ")
}

func (s *shell) readInput() string {
	s.inputBuffer.Reset()
	var tabCount int
	var input string
	for {
		var buf [1]byte // read per char input
		n, err := os.Stdin.Read(buf[:])
		if err != nil || n == 0 {
			break
		}
		char := buf[0]
		var matchCount int

		if char == TAB {
			tabCount++
			if s.inputBuffer.Len() > 0 {
				var matches []string
				str := strings.Fields(s.inputBuffer.String())
				substring := str[len(str)-1]
				// d.print(fmt.Print(substring[len(substring)-1]))
				for k := range builtIns {
					if strings.HasPrefix(k, substring) {
						matchCount++
						matches = append(matches, k)
					}
				}
				for _, ex := range execs.exec {
					if strings.HasPrefix(ex, substring) {
						// d.print("match found: ", ex, "\r\n")
						if !slices.Contains(matches, ex) { // not exactly the most efficient but we'll take it for now :/
							matchCount++
							matches = append(matches, ex)
						}
					}
				}
				if matchCount > 1 {
					d.print("more than 1 match found")
					d.printf("%v", matches)
					d.print(tabCount)

					if tabCount < 2 {
						fmt.Print("\a")
					} else if tabCount >= 2 {
						slices.Sort(matches)
						fmt.Printf("\r\n%s\n\r", strings.Join(matches, "  "))
					}
					s.redrawLine()
					continue
				} else if matchCount == 1 {
					s.inputBuffer.Truncate(s.inputBuffer.Len() - len(substring))
					s.inputBuffer.WriteString(matches[0] + " ")
					tabCount = 0
				} else if matchCount == 0 {
					fmt.Print("\a")
				}
			}
			// Handle both LF (10) and CR (13)
		} else if char == ENTER_CR || char == ENTER_LF {
			input = s.inputBuffer.String()
			s.inputBuffer.Reset()
			fmt.Print("\r\n")
			break
		} else if char == BACKSPACE {
			if s.inputBuffer.Len() > 0 {
				fmt.Print("\b \b")
				s.inputBuffer.Truncate(s.inputBuffer.Len() - 1)
			}
			continue
		} else if _, exists := CONTROL[int(char)]; exists {
			if handled := s.handleControlChars(char); handled {
				continue // Skip further processing for this character
			}
		} else {
			s.inputBuffer.Write(buf[:])
		}

		// This approach rewrites the buffer each time we type a char. Apparently this is the standard.
		// Silly me tried to track the cursor and insert/delete char in place :')
		s.redrawLine()
	}
	return input
}

func (s *shell) redrawLine() {
	fmt.Print("\r\x1b[K")                      // Move to start + clear line
	fmt.Printf("$ %s", s.inputBuffer.String()) // Rewrite
	fmt.Print("\x1b[?25h")                     // Ensure cursor visibility
}

func (s *shell) executeCommand(cmd *command) {
	if cmd.internal {
		cmd.err = cmd.execute(s) // builtins use raw mode
		if cmd.err != nil {
			fmt.Fprint(cmd.stderr, cmd.err, "\r\n")
		}
	} else {
		d.print("executing external command")
		term.Restore(s.stdinFD, s.oldState) // set to cooked mode
		defer term.MakeRaw(s.stdinFD)       // restore raw mode
		cmd.err = cmd.execute(s)
		if cmd.err != nil {
			// fmt.Fprint(cmd.stdout, cmd.err, "\r\n") // ignore for now
		}
	}
}

// parseInput reads user input, split it into a command and arguments,
// then determines if the command is built-in or external, if it's external,
// gets the command's path via getCmdPath. Handles quoting via handleArgs.
func (s *shell) parseInput(readString string) (*command, error) {
	cmd := newCommand()
	if len(readString) == 0 {
		return cmd, nil
	}
	input := strings.TrimLeft(readString, " \t")

	var parts []string
	if input[0] == '"' || input[0] == '\'' {
		ind := strings.Index(input[1:], string(input[0]))
		if ind == -1 {
			return cmd, fmt.Errorf("missing closing quote")
		}
		d.print(input)
		parts = strings.SplitN(input[1:], string(input[0]), 2)
	} else {
		parts = strings.SplitN(input, " ", 2)
	}

	cmd.name = parts[0]
	d.print(parts)

	if len(parts) > 1 {
		args := parts[1]
		args = strings.TrimLeft(args, " \t")
		cmd.args, cmd.err = handleArgs(cmd, args)
		if cmd.err != nil {
			return cmd, cmd.err
		}
	}

	_, cmd.internal = builtIns[cmd.name]
	if !cmd.internal {
		if strings.Contains(cmd.name, "/") {
			cmd.path = cmd.name
		} else {
			cmd.path, cmd.err = getCmdPath(cmd.name)
			if cmd.err != nil {
				return cmd, cmd.err
			}
		}
	}
	return cmd, nil
}

func (s *shell) handleControlChars(char byte) bool {
	switch char {
	case CTRL_C:
		fmt.Print("^C")
		fmt.Print("\r\n$ ")
		s.inputBuffer.Reset()
		return true
	case CTRL_D:
		if s.inputBuffer.Len() == 0 {
			fmt.Print("\r\n")
			s.exitShell(0)
		}
		return true
	default:
		return false
	}
}

// Shell's main loop
func (s *shell) run() {
	for {
		s.printPrompt()
		input := s.readInput()
		cmd, err := s.parseInput(input)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				d.print("if it doesnt exist, it should be here")
				fmt.Fprintf(cmd.stderr, "%s: command not found\r\n", cmd.name)
			} else {
				fmt.Fprintf(cmd.stderr, "%v\r\n", cmd.err)
			}
			continue
		}
		if cmd.name == "" {
			continue
		}
		s.executeCommand(cmd)
	}
}

func (s *shell) exitShell(code int) {
	term.Restore(s.stdinFD, s.oldState) // restore the terminal when done
	// fmt.Print("Have a good one!👋\r\n")
	os.Exit(code)
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

func (cmd *command) execute(s *shell) error {
	// handle internal command

	if cmd.internal {
		switch cmd.name {
		case "exit":
			if len(cmd.args) > 1 {
				return fmt.Errorf("exit: too many arguments")
			}
			if len(cmd.args) == 1 {
				code, err := strconv.Atoi(cmd.args[0])
				if err != nil || code > 255 || code < 0 {
					return fmt.Errorf("exit: invalid argument")
				}
				s.exitShell(code)
			}
			s.exitShell(0)
		case "echo":
			echoed := strings.Join(cmd.args, " ")
			// d.print("echoed: ", echoed)
			fmt.Fprintf(cmd.stdout, "%s\r\n", echoed)
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
			if len(cmd.args) > 1 {
				return fmt.Errorf("cd: too many arguments")
			}
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
		// d.printf("cmd stdout: %v", c.Stdout)
		c.Stderr = cmd.stderr
		// d.printf("cmd stderr: %v", c.Stderr)

		if err := c.Run(); err != nil {
			// fmt.Fprintf(cmd.stderr, "%v\r\n", err)
			return err
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

func getPath() ([]string, error) {
	pathEnv, ok := os.LookupEnv("PATH")
	if !ok || pathEnv == "" {
		return nil, fmt.Errorf("PATH environment variable is not set")
	}
	return strings.Split(pathEnv, string(os.PathListSeparator)), nil
	// d.print("paths: ", strings.Join(paths, " "))
}

func listAllExecInPath() []string {
	var exec []string
	paths, err := getPath()
	if err != nil {
		fmt.Printf("%v\r\n", err)
		return nil
	}
	for _, dir := range paths {
		files, err := os.ReadDir(dir)
		if err != nil {
			// fmt.Printf("%v\r\n", err)
			// ignore for now
		}
		for _, file := range files {
			// info, err := file.Info()
			if err != nil {
				d.print(err, "\r\n")
			}
			if exist := slices.Contains(exec, file.Name()); !exist /* && !strings.ContainsRune(file.Name(), '.') */ {
				exec = append(exec, file.Name())
			}
		}
	}
	// d.print("hey we found:", len(exec), "\r\n")
	return exec
}

// getCmdPath searches for an executable in the system PATH and returns its full path.
// It checks each directory in PATH for a file matching execName that has execute
// permissions. Returns an error if PATH is not set, executable is not found, or
// encounters permissions/IO errors.
func getCmdPath(execName string) (string, error) {

	paths, err := getPath()
	if err != nil {
		fmt.Printf("%v\r\n", err)
	}

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

func isEscapableChar(char byte, inSingleQuote bool, inDoubleQuote bool) bool {
	if inSingleQuote {
		// Inside single quotes, nothing is escapable
		return false
	} else if inDoubleQuote {

		// Inside double quotes, single quotes are not escapable
		return char == '\\' || char == '$' || char == '"' || char == '\n' || char == '`'
	} else {
		return char == '\\' || char == '$' || char == '"' || char == '\n' || char == '`' || char == '\''
	}
}

// TODO: Add better handling for missing closing single quote (newline support)
// handleArgs splits a string of arguments into a slice, preserving quoted
// sections as single arguments. Returns an error if there is missing closing
// quote, for now.
func handleArgs(cmd *command, args string) ([]string, error) {
	var argsList []string
	var buf strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	isEscaped := false

	for i, c := range args {
		// d.print("switching: ", string(c))
		switch {
		// handle output redirections
		case c == '>' || (c == '1' && len(args) > i+1 && args[i+1] == '>') || (c == '2' && len(args) > i+1 && args[i+1] == '>'):
			var uses1, uses2, appending bool
			if c == '1' && len(args) > i+1 && args[i+1] == '>' {
				uses1 = true
				if len(args) > i+2 && args[i+2] == '>' {
					appending = true
					d.print("1>>")
				} else {

					d.print("1>")
				}
			} else if c == '2' && len(args) > i+1 && args[i+1] == '>' {
				uses2 = true
				if len(args) > i+2 && args[i+2] == '>' {
					appending = true
					d.print("2>>")
				} else {

					d.print("2>")
				}
			} else if c == '>' && len(args) > i+1 && args[i+1] == '>' {
				appending = true
				d.print(">")
			}
			if inDoubleQuote || inSingleQuote {
				buf.WriteRune(c)
			} else {
				d.print("redirecting: ", argsList)
				if ((appending && !(uses1 || uses2)) || (uses1 && !appending) || (uses2 && !appending)) && len(args) <= i+2 {
					return nil, fmt.Errorf("invalid redirection: no specified target")
				} else if (uses1 || uses2) && appending && len(args) <= i+3 {
					return nil, fmt.Errorf("invalid redirection: no specified target")
				} else if len(args) <= i+1 {
					return nil, fmt.Errorf("invalid redirection: no specified target")
				}
				if buf.Len() > 0 {
					argsList = append(argsList, buf.String())
					buf.Reset()
				}
				var target string
				if (appending && !(uses1 || uses2)) || (uses1 && !appending) || (uses2 && !appending) {
					target = filepath.Clean(strings.TrimSpace(args[i+2:]))
					d.print("target use or append: ", target)
				} else if (uses1 || uses2) && appending {
					target = filepath.Clean(strings.TrimSpace(args[i+3:]))
					d.print("target use and append: ", target)
				} else {
					target = filepath.Clean(strings.TrimSpace(args[i+1:]))
					d.print("target: ", target)
				}
				var descriptor int
				var err error
				if uses1 || uses2 {
					descriptor, err = strconv.Atoi(string(c))
					if err != nil {
						return nil, err
					}
				} else {
					descriptor = 1
				}

				err = redirect(cmd, target, descriptor, appending)
				if err != nil {
					return nil, err
				}
				return argsList, nil
			}
		case c == '"':
			if inDoubleQuote {
				if isEscaped {
					isEscaped = false
					buf.WriteRune(c)
				} else {
					inDoubleQuote = false
					// only append if there is a space after the closing quote
					if len(args) > i+1 && args[i+1] == ' ' || i == len(args)-1 {
						d.print("appending inside quote: ", buf.String())
						argsList = append(argsList, buf.String())
						d.print(argsList)
						buf.Reset()
					}
				}
			} else if inSingleQuote {
				buf.WriteRune(c)

			} else {
				if isEscaped {
					isEscaped = false
					buf.WriteRune(c)

				} else {
					inDoubleQuote = true
				}
			}
		case c == '\\':
			if inDoubleQuote {
				if inSingleQuote {
					buf.WriteRune(c)

				} else if isEscaped {
					isEscaped = false
					buf.WriteRune(c)

				} else if len(args) > i+1 && isEscapableChar(args[i+1], inSingleQuote, inDoubleQuote) {
					isEscaped = true
					d.print("encountering an escape backslash", string(args[i+1]))
				} else {
					buf.WriteRune(c)

				}
			} else if inSingleQuote {
				buf.WriteRune(c)

			} else {
				if len(args) > i+1 && (isEscapableChar(args[i+1], inSingleQuote, inDoubleQuote) || args[i+1] == ' ') {
					isEscaped = true
				}
			}
		case c == '\'':
			if inDoubleQuote {
				buf.WriteRune(c)
			} else if inSingleQuote {
				inSingleQuote = false
				// only append if there is a space after the closing quote
				if len(args) > i+1 && args[i+1] == ' ' || i == len(args)-1 {
					d.print("appending inside quote: ", buf.String())
					argsList = append(argsList, buf.String())
					d.print(argsList)
					buf.Reset()
				}
			} else if isEscaped {
				isEscaped = false
				buf.WriteRune(c)
			} else {
				inSingleQuote = true
			}
		case c == ' ':
			if inDoubleQuote || inSingleQuote {
				// d.print("writing space")
				buf.WriteRune(c)

			} else if !inSingleQuote && !inDoubleQuote {
				if isEscaped {
					isEscaped = false
					buf.WriteRune(c)

				} else if buf.Len() > 0 {
					// d.print("appending outside quote: ", buf.String())
					argsList = append(argsList, buf.String())
					buf.Reset()
				}
			}
		default:
			// d.print("writing: ", string(c))
			buf.WriteRune(c)

		}
	}
	if inSingleQuote || inDoubleQuote {
		d.print(argsList)
		return nil, fmt.Errorf("missing closing quote")
	}
	if buf.Len() > 0 {
		argsList = append(argsList, buf.String())
	}
	return argsList, nil
}

// redirect redirects a command's stdout to a chosen file
func redirect(cmd *command, target string, desc int, appending bool) (err error) {
	var f *os.File
	d.print("opening: ", target)
	if appending {
		f, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
	} else {
		f, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
	}

	switch desc {
	case 1:
		cmd.stdout = f
	case 2:
		cmd.stderr = f
	}
	return nil
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

// !!! DEPRECATED !!!
// handleInterrupt handles interrupt signal with custom behaviour
// func handleInterrupt() {
// 	sigChan := make(chan os.Signal, 1)

// 	// listen for ctrl+c keystroke
// 	signal.Notify(sigChan, os.Interrupt)

// 	go func() {
// 		for range sigChan {
// 			fmt.Fprintln(os.Stdout)
// 			fmt.Fprint(os.Stdout, "$ ")
// 		}
// 	}()
// }

func main() {

	sh, err := newShell()
	if err != nil {
		panic(err)
	}
	defer term.Restore(sh.stdinFD, sh.oldState) // Safety net

	sh.run()
}
