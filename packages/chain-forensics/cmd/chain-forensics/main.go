package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	chainforensics "github.com/vrooli/chain-forensics"
)

const exampleID = "us-v-1hq3-public-forfeiture"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "chain-forensics:", err)
		os.Exit(2)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage(stdout)
		return nil
	}
	if len(args) < 3 || args[0] != "case" {
		printUsage(stderr)
		return errors.New("expected `case inspect` or `case packet`")
	}
	if args[2] != exampleID {
		return fmt.Errorf("unknown bundled case %q", args[2])
	}
	c, err := chainforensics.ExampleCase()
	if err != nil {
		return err
	}
	switch args[1] {
	case "inspect":
		if len(args) != 4 || args[3] != "--json" {
			return errors.New("usage: chain-forensics case inspect " + exampleID + " --json")
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(c)
	case "packet":
		if len(args) != 5 || args[3] != "--out" || strings.TrimSpace(args[4]) == "" {
			return errors.New("usage: chain-forensics case packet " + exampleID + " --out <new-local-file>")
		}
		packet, err := chainforensics.MarshalPrivatePacket(c)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(args[4], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("create private packet without overwriting: %w", err)
		}
		if _, err = f.Write(packet); err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil {
			_ = os.Remove(args[4])
			return fmt.Errorf("write private packet: %w", err)
		}
		if closeErr != nil {
			_ = os.Remove(args[4])
			return fmt.Errorf("close private packet: %w", closeErr)
		}
		_, err = fmt.Fprintln(stdout, args[4])
		return err
	default:
		return fmt.Errorf("unknown case command %q", args[1])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Chain Forensics package CLI (offline published case)")
	fmt.Fprintln(w, "  chain-forensics case inspect "+exampleID+" --json")
	fmt.Fprintln(w, "  chain-forensics case packet "+exampleID+" --out <new-local-file>")
}
