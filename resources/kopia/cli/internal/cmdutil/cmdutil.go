// Package cmdutil holds the small, shared command-handler helpers used by every
// resource-kopia command group (flag-set construction, JSON output, name
// validation). Centralizing them keeps the per-group handlers thin and uniform.
package cmdutil

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
)

// NewFlagSet returns a ContinueOnError flag set that discards its own usage
// output (the App layer renders help), named for error messages.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// Parse parses args, returning a nil error for the help sentinel so help is not
// treated as a failure.
func Parse(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	return nil
}

// RepeatedFlag collects a string flag that may be supplied more than once, in
// declaration order. Register with fs.Var(&rf, "tags", "..."); each occurrence
// appends one value. Used for kopia snapshot --tags key:value pairs.
type RepeatedFlag []string

// String renders the collected values (flag.Value).
func (r *RepeatedFlag) String() string { return strings.Join(*r, ",") }

// Set appends one occurrence (flag.Value).
func (r *RepeatedFlag) Set(v string) error {
	*r = append(*r, v)
	return nil
}

// RequireName validates a required --name flag.
func RequireName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	return nil
}

// WriteJSON marshals v as indented JSON to w with a trailing newline.
func WriteJSON(w io.Writer, v any) error {
	data, err := cliout.MarshalIndent(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// EnsureTrailingNewline returns b with exactly one trailing newline so kopia's
// raw output prints cleanly regardless of whether it already ended in one.
func EnsureTrailingNewline(b []byte) []byte {
	trimmed := strings.TrimRight(string(b), "\n")
	return []byte(trimmed + "\n")
}
