package commandtree

import (
	"flag"
	"io"
)

// NewFlagSet creates a compatibility parser for legacy commands while they
// migrate to the structured command-tree argument model. Keeping the parser
// policy here prevents each command family from inventing its own error/output
// behavior.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}
