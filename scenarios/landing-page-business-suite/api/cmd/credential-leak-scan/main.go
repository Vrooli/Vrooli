package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"landing-page-business-suite-api/internal/credentialleak"
)

func main() {
	flags := flag.NewFlagSet("credential-leak-scan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "LPBS service manifest (metadata only)")
	artifacts := flags.String("artifacts", "", "downloaded Test Genie artifact directory")
	if flags.Parse(os.Args[1:]) != nil || flags.NArg() != 0 || *manifest == "" || *artifacts == "" {
		fmt.Fprintln(os.Stderr, "usage: credential-leak-scan --manifest <service.json> --artifacts <directory>")
		os.Exit(2)
	}
	os.Exit(scan(*manifest, *artifacts, os.Stdout))
}

func scan(manifestPath, artifactPath string, output io.Writer) int {
	families, err := credentialleak.ReadManifestFamilies(manifestPath)
	if err != nil || credentialleak.ValidateFamilies(families) != nil {
		fmt.Fprintln(output, `{"status":"failed","failure_reason":"credential_inventory_unavailable"}`)
		return 2
	}
	scanner, err := credentialleak.NewScanner(credentialleak.SyntheticCanaries(families))
	if err != nil {
		fmt.Fprintln(output, `{"status":"failed","failure_reason":"scanner_unavailable"}`)
		return 2
	}
	result := scanner.ScanTestGenieArtifactDirectory(artifactPath)
	data, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintln(output, `{"status":"failed","failure_reason":"result_encoding_failed"}`)
		return 2
	}
	if _, err := output.Write(append(data, '\n')); err != nil {
		return 2
	}
	if result.Status != "clean" {
		return 1
	}
	return 0
}
