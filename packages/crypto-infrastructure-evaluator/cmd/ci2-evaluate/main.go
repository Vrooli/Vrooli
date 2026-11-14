package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vrooli/crypto-infrastructure-evaluator"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: ci2-evaluate <private-packet.json>")
		os.Exit(2)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fail(err)
	}
	defer f.Close()
	r, err := evaluator.EvaluatePacket(f)
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, "ci2-evaluate:", err); os.Exit(1) }
