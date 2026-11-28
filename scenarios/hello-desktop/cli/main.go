package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	help := flag.Bool("help", false, "Show Hello Desktop CLI help")
	flag.Parse()
	if *help {
		fmt.Println("hello-desktop — desktop delivery fixture")
		return
	}
	if flag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n", flag.Args())
		os.Exit(2)
	}
}
