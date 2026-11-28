package support

import (
	"errors"
	"flag"
	"sort"
	"sync"
	"sync/atomic"
)

// FlagDoc describes one flag of a command's flag set.
type FlagDoc struct {
	Name    string
	Usage   string
	Default string
	Bool    bool
}

// ErrFlagsDescribed is what ParseFlags returns while a command's flags are being
// described, so the command stops before it does anything.
var ErrFlagsDescribed = errors.New("flags described")

var (
	describeMu   sync.Mutex
	describeHook atomic.Pointer[func(*flag.FlagSet)]
)

// DescribeFlags reads the flags a legacy command defines, from the flag set the
// command itself builds, so help cannot drift from what the command accepts.
// It runs the command with no arguments; every command defines and parses its
// flags before it does anything else, and ParseFlags stops it there. It returns
// no flags for a command that never parses any.
func DescribeFlags(run func(args []string) error) []FlagDoc {
	describeMu.Lock()
	defer describeMu.Unlock()
	var docs []FlagDoc
	hook := func(fs *flag.FlagSet) {
		fs.VisitAll(func(f *flag.Flag) {
			_, isBool := f.Value.(interface{ IsBoolFlag() bool })
			docs = append(docs, FlagDoc{Name: f.Name, Usage: f.Usage, Default: f.DefValue, Bool: isBool})
		})
	}
	describeHook.Store(&hook)
	defer describeHook.Store(nil)
	_ = run(nil)
	sort.SliceStable(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	return docs
}
