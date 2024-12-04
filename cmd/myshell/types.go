package main

import (
	"io"
	"log"
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
	external bool
	err      error
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}
