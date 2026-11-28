package deletiongrant

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Disposition is what the host's declarations say deleting a path loses.
type Disposition string

const (
	// Regenerable: an owner declares the data rebuildable, or storage-manager
	// reaps it on its own.
	Regenerable Disposition = "regenerable"
	// OwnerConfirms: the location is declared, but only its owner can say
	// whether deleting it is safe.
	OwnerConfirms Disposition = "owner_confirms"
	// Protected: an owner declares the data durable; it does not come back.
	Protected Disposition = "protected"
	// Undeclared: nothing on the host declares the location, so nothing will
	// rebuild it.
	Undeclared Disposition = "undeclared"
)

// Declaration is what the host's declarations say about one path.
type Declaration struct {
	Disposition Disposition
	// Owner is who declares the location ("agent-manager"); empty when nobody
	// does.
	Owner string
	// Source and Root name the governing declaration and the root it covers,
	// for the technical details.
	Source, Root, Reason string
	// IsRoot is set when the path is a protected root itself, which no grant
	// can open; such a request is refused.
	IsRoot bool
}

// Declarations reads the host's declarations once and answers every path of
// one request from that read: reading them per path made a large request take
// minutes. An error means what a path holds is unknown, and the request is
// refused rather than asked without it.
type Declarations func() (func(path string) Declaration, error)

// PathInfo is one requested path as the operator sees it: what it is, how big,
// how recently anything in it changed, and what deleting it loses.
type PathInfo struct {
	Path string
	Declaration
	Exists bool
	Dir    bool
	Bytes  int64
	Files  int
	// Partial means measuring stopped at the request's budget; Bytes and Files
	// are then a floor and Changed may miss newer files.
	Partial bool
	// Changed is the newest modification time of anything in the path.
	Changed time.Time
}

// measureBudget bounds the walk one request may do, so a request naming a huge
// tree still answers the agent promptly; what is not walked is shown as "at
// least".
type measureBudget struct {
	entries  int
	deadline time.Time
}

func defaultBudget() *measureBudget {
	// The request clock may be a caller seam (tests, replay). It may not decide
	// how long real I/O runs: a clock behind wall time would end the walk before
	// the first entry and report every path as an empty floor.
	return &measureBudget{entries: 400_000, deadline: time.Now().Add(8 * time.Second)}
}

func (b *measureBudget) spend() bool {
	if b.entries <= 0 || time.Now().After(b.deadline) {
		return false
	}
	b.entries--
	return true
}

// inspect describes every path. Symbolic links are measured as links, never
// followed, so a link to a large tree is not counted as that tree.
func inspect(paths []string, declare func(string) Declaration, budget *measureBudget) []PathInfo {
	out := make([]PathInfo, 0, len(paths))
	for _, p := range paths {
		info := PathInfo{Path: p, Declaration: Declaration{Disposition: Undeclared}}
		if declare != nil {
			info.Declaration = declare(p)
		}
		measure(&info, budget)
		out = append(out, info)
	}
	return out
}

func measure(info *PathInfo, budget *measureBudget) {
	root, err := os.Lstat(info.Path)
	if err != nil {
		return
	}
	info.Exists, info.Dir, info.Changed = true, root.IsDir(), root.ModTime()
	if !root.IsDir() {
		info.Bytes, info.Files = root.Size(), 1
		return
	}
	pending := []string{info.Path}
	for len(pending) > 0 {
		dir := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		f, err := os.Open(dir)
		if err != nil {
			info.Partial = true
			continue
		}
		for {
			entries, readErr := f.ReadDir(256)
			for _, entry := range entries {
				if !budget.spend() {
					info.Partial = true
					_ = f.Close()
					return
				}
				child, statErr := entry.Info()
				if statErr != nil {
					info.Partial = true
					continue
				}
				if child.ModTime().After(info.Changed) {
					info.Changed = child.ModTime()
				}
				switch {
				case child.IsDir():
					pending = append(pending, filepath.Join(dir, entry.Name()))
				case child.Mode().IsRegular():
					info.Bytes += child.Size()
					info.Files++
				}
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) && !errors.Is(readErr, fs.ErrNotExist) {
					info.Partial = true
				}
				break
			}
		}
		_ = f.Close()
	}
}
