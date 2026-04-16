package main

import (
	"log"
	"os"
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("tech-tree-designer: %v", err)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
