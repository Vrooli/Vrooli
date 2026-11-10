// Package repohealth is DBM's read-only view of what is wrong inside a backup
// repository directory on a filesystem destination, and the narrow, recorded
// exceptions an operator may grant for specific unreadable leftovers.
//
// Everything here observes. Nothing renames, deletes, repairs or writes to a
// repository. The one thing this package persists is the operator's decision
// record (an exception for one exact path), which lives under DBM's own state
// root and never inside the repository.
package repohealth

import (
	"errors"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"syscall"
)

// EntryClass says what a repository directory entry is, judged by its name and
// position only (an unreadable entry cannot be opened to find out more).
type EntryClass string

const (
	// ClassLeftoverTemp is an unfinished upload: kopia writes a blob to
	// "<blob>.f.tmp.<random>" and renames it to "<blob>.f" once complete. A
	// temp name therefore never carries committed repository content.
	ClassLeftoverTemp EntryClass = "leftover_temp"
	// ClassRepositoryBlob is a committed blob or repository configuration file.
	// Losing access to one can make snapshots unrestorable.
	ClassRepositoryBlob EntryClass = "repository_blob"
	// ClassOther is anything that fits neither shape (a directory that cannot be
	// listed, an unexpected name). It is never excusable.
	ClassOther EntryClass = "other"
)

// BlobKind is the role of a blob, from the repository's prefix directory.
type BlobKind string

const (
	BlobSessionMarker BlobKind = "session_marker"
	BlobDataPack      BlobKind = "data_pack"
	BlobMetadataPack  BlobKind = "metadata_pack"
	BlobIndex         BlobKind = "index"
	BlobOther         BlobKind = "other"
)

// Problem is why an entry could not be read.
type Problem string

const (
	ProblemIO         Problem = "io_error"
	ProblemPermission Problem = "permission_denied"
	ProblemOther      Problem = "other"
)

// Entry is one directory entry the scan could not read.
type Entry struct {
	// RelPath is slash-separated and relative to the repository directory, so a
	// record stays valid if the drive is re-mounted somewhere else.
	RelPath  string     `json:"rel_path"`
	Class    EntryClass `json:"class"`
	BlobKind BlobKind   `json:"blob_kind"`
	Problem  Problem    `json:"problem"`
	Detail   string     `json:"detail,omitempty"`
}

var (
	// "<blob id>.f.tmp.<hex>": kopia's filesystem upload temp name.
	leftoverTempName = regexp.MustCompile(`^[0-9a-z]+(?:-[0-9a-z]+)*\.f\.tmp\.[0-9a-f]{8,64}$`)
	// "<blob id>.f": a committed blob.
	committedBlobName = regexp.MustCompile(`^[0-9a-z]+(?:-[0-9a-z]+)*\.f$`)
)

// ClassifyName judges a repository-relative path by its shape alone.
func ClassifyName(rel string) (EntryClass, BlobKind) {
	rel = path.Clean(strings.ReplaceAll(rel, `\`, "/"))
	base := path.Base(rel)
	kind := blobKindFor(rel)
	switch {
	case leftoverTempName.MatchString(base):
		return ClassLeftoverTemp, kind
	case committedBlobName.MatchString(base):
		return ClassRepositoryBlob, kind
	default:
		return ClassOther, kind
	}
}

// blobKindFor reads the blob's role from its prefix directory (p/, q/, s/, x/,
// n/). Files at the repository root (kopia.repository.f, xe40.f) are
// configuration or epoch markers.
func blobKindFor(rel string) BlobKind {
	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return BlobOther
	}
	switch parts[0] {
	case "p":
		return BlobDataPack
	case "q":
		return BlobMetadataPack
	case "s":
		return BlobSessionMarker
	case "x", "n":
		return BlobIndex
	default:
		return BlobOther
	}
}

// ProblemFor classifies a read error. A bad sector, a half-written NTFS
// directory entry and a dropped USB link all surface as EIO.
func ProblemFor(err error) Problem {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, syscall.EIO) || strings.Contains(strings.ToLower(err.Error()), "input/output error"):
		return ProblemIO
	case errors.Is(err, fs.ErrPermission):
		return ProblemPermission
	default:
		return ProblemOther
	}
}

// NewEntry builds the Entry for an unreadable path.
func NewEntry(rel string, err error) Entry {
	rel = path.Clean(strings.ReplaceAll(rel, `\`, "/"))
	class, kind := ClassifyName(rel)
	e := Entry{RelPath: rel, Class: class, BlobKind: kind, Problem: ProblemFor(err)}
	if err != nil {
		e.Detail = err.Error()
	}
	return e
}
