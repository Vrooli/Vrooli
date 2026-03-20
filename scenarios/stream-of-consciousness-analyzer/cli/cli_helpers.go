package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// newFlagSet creates a flag set with ContinueOnError — use when custom flags are needed.
func newFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ContinueOnError)
}

// cmdFlags creates a named flag set with --json support and parses the args.
func (a *App) cmdFlags(name string, args []string) (*flag.FlagSet, *bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, nil, err
	}
	return fs, jsonOut, nil
}

// requireArg returns a usage error if no positional args are present.
func requireArg(fs *flag.FlagSet, usage string) error {
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: %s", usage)
	}
	return nil
}

// unmarshalBody decodes JSON body into v, wrapping parse errors consistently.
func unmarshalBody(body []byte, v interface{}) error {
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

// getResource performs a GET request and either prints raw JSON or calls formatFn.
func (a *App) getResource(path string, jsonOut *bool, formatFn func([]byte) error) error {
	body, err := a.core.APIClient.Get(a.apiPath(path), nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	return formatFn(body)
}

// postResource performs a POST request and either prints raw JSON or calls formatFn.
func (a *App) postResource(path string, payload interface{}, jsonOut *bool, formatFn func([]byte) error) error {
	body, err := a.core.APIClient.Request("POST", a.apiPath(path), nil, payload)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	return formatFn(body)
}

// putResource performs a PUT request and either prints raw JSON or calls formatFn.
func (a *App) putResource(path string, payload interface{}, jsonOut *bool, formatFn func([]byte) error) error {
	body, err := a.core.APIClient.Request("PUT", a.apiPath(path), nil, payload)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	return formatFn(body)
}

// deleteResource performs a DELETE request and prints a confirmation.
func (a *App) deleteResource(path string, label string, id string) error {
	if _, err := a.core.APIClient.Request("DELETE", a.apiPath(path), nil, nil); err != nil {
		return err
	}
	fmt.Printf("Deleted %s %s\n", label, id)
	return nil
}
