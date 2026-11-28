// Command vrooli-gocacheprog is the host's GOCACHEPROG helper. Every go
// command on an activated host starts one as its build cache; it serves the
// shared GOCACHE store and records which outputs its go command holds, so the
// store can be reclaimed while builds run without deleting an output a build
// still needs. See internal/gocache for the protocol.
package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/gocache"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Fprintln(os.Stderr, "vrooli-gocacheprog takes no arguments; cmd/go starts it through GOCACHEPROG (see `vrooli host go-cache status`)")
		os.Exit(2)
	}
	err := gocache.Serve(gocache.ServeOptions{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Getenv: os.Getenv})
	if err != nil {
		fmt.Fprintln(os.Stderr, "vrooli-gocacheprog:", err)
		os.Exit(1)
	}
}
