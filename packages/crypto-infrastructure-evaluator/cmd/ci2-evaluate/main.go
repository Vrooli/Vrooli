package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vrooli/crypto-infrastructure-evaluator"
)

func main() {
	seeded := len(os.Args) == 3 && os.Args[1] == "--seeded"
	if len(os.Args) != 2 && !seeded {
		fmt.Fprintln(os.Stderr, "usage: ci2-evaluate [--seeded] <private-packet.json>")
		os.Exit(2)
	}
	path := os.Args[len(os.Args)-1]
	f, err := os.Open(path)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	var r any
	if seeded {
		r, err = evaluator.EvaluatePacketSeededSuite(f)
	} else {
		r, err = evaluator.EvaluatePacket(f)
	}
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
