package main

import (
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

// func handleError(err error, msg string) {
// 	if err != nil {
// 		if msg != "" {
// 			d.printf("%s: %v", msg, err)
// 		} else {
// 			d.print(err)
// 		}
// 	}
// }
