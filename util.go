package pcfg

import (
	"log"
)

// checkAndFatal check err. If err != nil, trigger log.Fatal
func checkAndFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// assert check exp, if exp == false, panic with message
func assert(exp bool, message string) {
	if !exp {
		log.Fatal(message)
	}
}