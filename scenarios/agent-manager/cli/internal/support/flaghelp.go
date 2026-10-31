package support

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// describeFlagsArg asks a flag-parsing handler to report its flag set instead
// of running. It cannot be typed as a real argument.
const describeFlagsArg = "\x00describe-flags"

type flagSetDescription struct{ fs *flag.FlagSet }

func (flagSetDescription) Error() string { return "flag set described" }

// ParseFlags parses a handler's flags. Handlers whose command help should list
// their flags call it instead of cliutil.ParseInterspersed, so the help is
// generated from the same flag definitions the handler parses with.
func ParseFlags(fs *flag.FlagSet, args []string) error {
	if len(args) == 1 && args[0] == describeFlagsArg {
		return flagSetDescription{fs}
	}
	return cliutil.ParseInterspersed(fs, args)
}

// FlagHelp returns the Options block for a ParseFlags-based handler, for the
// command's HelpText (cliapp prints flags from Args only for RunCtx commands,
// and these handlers parse their own arguments). prefix is the dispatcher path
// to the handler, for example the subcommand name.
func FlagHelp(dispatch CommandFunc, prefix ...string) string {
	var description flagSetDescription
	if dispatch == nil {
		return ""
	}
	if err := dispatch(append(append([]string{}, prefix...), describeFlagsArg)); !errors.As(err, &description) {
		return ""
	}
	var out strings.Builder
	out.WriteString("Options:\n")
	description.fs.VisitAll(func(f *flag.Flag) {
		synopsis := "--" + f.Name
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); !ok || !b.IsBoolFlag() {
			synopsis += " <value>"
		}
		usage := f.Usage
		if def := f.DefValue; def != "" && def != "0" && def != "false" {
			usage += " (default: " + def + ")"
		}
		fmt.Fprintf(&out, "  %-30s %s\n", synopsis, usage)
	})
	return strings.TrimRight(out.String(), "\n")
}
