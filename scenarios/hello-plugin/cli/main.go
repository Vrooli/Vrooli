package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("hello-plugin: use hello or status")
		return
	}
	switch os.Args[1] {
	case "hello":
		fs := flag.NewFlagSet("hello", flag.ExitOnError)
		name := fs.String("name", "agent", "name to greet")
		_ = fs.Parse(os.Args[2:])
		fmt.Printf("hello, %s!\n", *name)
	case "status":
		fmt.Println("ok")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}
