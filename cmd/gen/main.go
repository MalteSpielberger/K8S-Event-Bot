package main

import "github.com/golangee/i18n"

func main() {
	if err := i18n.Bundle(); err != nil {
		panic(err)
	}
}
