package gocache

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// HashSize is the size of cmd/go action and output IDs (SHA-256).
const HashSize = 32

const (
	hexSize = HashSize * 2
	// entrySize is the exact size of a cmd/go action entry:
	// "v1 <hex action> <hex output> <size padded to 20> <unixnano padded to 20>\n".
	entrySize = 2 + 1 + hexSize + 1 + hexSize + 1 + 20 + 1 + 20 + 1
	// kindAction and kindOutput are the cmd/go file suffixes for action index
	// entries and output bodies.
	kindAction = 'a'
	kindOutput = 'd'
	// tempMarker names this package's in-progress writes. cmd/go ignores any
	// name that does not end in -a or -d, so a crashed write is never served.
	tempMarker = ".vrooli-tmp-"
)

// ID is a cmd/go action or output ID.
type ID [HashSize]byte

func (id ID) String() string { return hex.EncodeToString(id[:]) }

func idFromBytes(b []byte) (ID, bool) {
	var id ID
	if len(b) != HashSize {
		return id, false
	}
	copy(id[:], b)
	return id, true
}

func idFromHex(s string) (ID, bool) {
	var id ID
	if len(s) != hexSize {
		return id, false
	}
	if _, err := hex.Decode(id[:], []byte(s)); err != nil {
		return id, false
	}
	return id, true
}

// Entry is the value stored for an action ID.
type Entry struct {
	Output ID
	Size   int64
	Time   time.Time
}

// errEntryInvalid reports an action entry that cmd/go would also reject; a
// lookup treats it as a miss.
var errEntryInvalid = errors.New("invalid cache entry")

func encodeEntry(action ID, entry Entry) []byte {
	return fmt.Appendf(nil, "v1 %x %x %20d %20d\n", action[:], entry.Output[:], entry.Size, entry.Time.UnixNano())
}

// decodeEntry applies the same validation cmd/go's DiskCache.get applies, so
// this package serves exactly the entries the native cache would serve.
func decodeEntry(action ID, data []byte) (Entry, error) {
	if len(data) != entrySize {
		return Entry{}, errEntryInvalid
	}
	if data[0] != 'v' || data[1] != '1' || data[2] != ' ' || data[3+hexSize] != ' ' || data[3+hexSize+1+hexSize] != ' ' || data[3+hexSize+1+hexSize+1+20] != ' ' || data[entrySize-1] != '\n' {
		return Entry{}, errEntryInvalid
	}
	gotAction, ok := idFromHex(string(data[3 : 3+hexSize]))
	if !ok || gotAction != action {
		return Entry{}, errEntryInvalid
	}
	rest := data[3+hexSize+1:]
	output, ok := idFromHex(string(rest[:hexSize]))
	if !ok {
		return Entry{}, errEntryInvalid
	}
	rest = rest[hexSize+1:]
	size, err := parsePadded(rest[:20])
	if err != nil {
		return Entry{}, errEntryInvalid
	}
	nanos, err := parsePadded(rest[21:41])
	if err != nil {
		return Entry{}, errEntryInvalid
	}
	return Entry{Output: output, Size: size, Time: time.Unix(0, nanos)}, nil
}

func parsePadded(field []byte) (int64, error) {
	i := 0
	for i < len(field) && field[i] == ' ' {
		i++
	}
	value, err := strconv.ParseInt(string(field[i:]), 10, 64)
	if err != nil {
		return 0, err
	}
	if value < 0 {
		return 0, errEntryInvalid
	}
	return value, nil
}

// parseEntryName reports the ID and kind of a cmd/go cache file name.
func parseEntryName(name string) (ID, byte, bool) {
	if len(name) != hexSize+2 || name[hexSize] != '-' {
		return ID{}, 0, false
	}
	kind := name[hexSize+1]
	if kind != kindAction && kind != kindOutput {
		return ID{}, 0, false
	}
	id, ok := idFromHex(name[:hexSize])
	return id, kind, ok
}

// Store is one GOCACHE directory in cmd/go's native layout.
type Store struct {
	Root string
}

// OpenStore validates that root is an absolute, existing cache directory.
func OpenStore(root string) (Store, error) {
	if root == "" || root == "off" {
		return Store{}, fmt.Errorf("GOCACHE is unset or disabled")
	}
	if !filepath.IsAbs(root) {
		return Store{}, fmt.Errorf("GOCACHE %q is not absolute", root)
	}
	info, err := os.Stat(root)
	if err != nil {
		return Store{}, fmt.Errorf("open Go build cache: %w", err)
	}
	if !info.IsDir() {
		return Store{}, fmt.Errorf("Go build cache %s is not a directory", root)
	}
	return Store{Root: filepath.Clean(root)}, nil
}

func shardName(b byte) string { return fmt.Sprintf("%02x", b) }

func (s Store) shardDir(id ID) string { return filepath.Join(s.Root, shardName(id[0])) }

func (s Store) entryPath(id ID, kind byte) string {
	return filepath.Join(s.shardDir(id), id.String()+"-"+string(kind))
}
