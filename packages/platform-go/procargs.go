package platform

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
)

// errMalformedProcArgs reports an argument block that does not have the
// KERN_PROCARGS2 layout. It is an error rather than an empty result so a
// caller cannot mistake an unreadable process for one with no environment.
var errMalformedProcArgs = errors.New("platform: malformed process argument block")

// procArgs is the decoded KERN_PROCARGS2 block of one process.
type procArgs struct {
	Executable string
	Args       []string
	Env        map[string]string
}

// parseProcArgs2 decodes the block darwin returns for sysctl kern.procargs2:
// a native-endian int32 argc, the executable path NUL-terminated, NUL
// padding, argc NUL-terminated argv strings, then NUL-terminated environment
// entries ending at an empty string (the kernel's "apple" strings follow and
// are not environment).
//
// The parser has no build tag so it is tested on every host. The block is the
// process's original stack image: a process that rewrites its title (Postgres
// does) can overwrite argv and environ in place, so the environment may be
// incomplete while the executable path, which precedes argv, stays intact.
func parseProcArgs2(data []byte) (procArgs, error) {
	if len(data) < 4 {
		return procArgs{}, errMalformedProcArgs
	}
	// darwin runs only on little-endian hardware (amd64, arm64).
	argc := binary.LittleEndian.Uint32(data[:4])
	rest := data[4:]
	executable, rest, ok := cutNUL(rest)
	if !ok || executable == "" {
		return procArgs{}, errMalformedProcArgs
	}
	for len(rest) > 0 && rest[0] == 0 {
		rest = rest[1:]
	}
	// Every argument needs at least its terminating NUL, which bounds argc by
	// the remaining bytes and keeps a corrupt count from sizing an allocation.
	if uint64(argc) > uint64(len(rest)) {
		return procArgs{}, errMalformedProcArgs
	}
	args := make([]string, 0, argc)
	for i := uint32(0); i < argc; i++ {
		arg, next, found := cutNUL(rest)
		if !found {
			return procArgs{}, errMalformedProcArgs
		}
		args = append(args, arg)
		rest = next
	}
	env := make(map[string]string)
	for len(rest) > 0 {
		entry, next, found := cutNUL(rest)
		// An unterminated tail is a truncated read; keep what was complete.
		if !found || entry == "" {
			break
		}
		if key, value, cut := strings.Cut(entry, "="); cut && key != "" {
			env[key] = value
		}
		rest = next
	}
	return procArgs{Executable: executable, Args: args, Env: env}, nil
}

func cutNUL(data []byte) (string, []byte, bool) {
	index := bytes.IndexByte(data, 0)
	if index < 0 {
		return "", data, false
	}
	return string(data[:index]), data[index+1:], true
}
