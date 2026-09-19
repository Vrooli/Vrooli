package pushsafety

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Digest streams a regular bundle without following a final symlink. The
// context bounds large reads; no file or Git object is written by status reads.
func (s DiskStore) Digest(ctx context.Context, path string) (string, error) {
	info, e := os.Lstat(path)
	if e != nil {
		return "", e
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("bundle is not a regular file")
	}
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil {
		return "", e
	}
	if !os.SameFile(info, opened) {
		return "", errors.New("bundle changed while opening")
	}
	h := sha256.New()
	buf := make([]byte, 128*1024)
	for {
		if e = ctx.Err(); e != nil {
			return "", e
		}
		n, readErr := f.Read(buf)
		if n > 0 {
			_, _ = h.Write(buf[:n])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	after, e := f.Stat()
	if e != nil {
		return "", e
	}
	if after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return "", errors.New("bundle changed during integrity check")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s DiskStore) readRecord(key string) (Artifact, error) {
	if !filepath.IsAbs(s.Root) || len(key) != 64 || !validOID(key) {
		return Artifact{}, errors.New("invalid recovery identity or storage root")
	}
	dir := filepath.Join(s.Root, key)
	info, e := os.Lstat(dir)
	if e != nil {
		return Artifact{}, e
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Artifact{}, errors.New("recovery directory is not a regular directory")
	}
	path := filepath.Join(dir, "recovery.json")
	info, e = os.Lstat(path)
	if e != nil {
		return Artifact{}, e
	}
	if !info.Mode().IsRegular() || info.Size() > 1024*1024 {
		return Artifact{}, errors.New("invalid recovery record file")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return Artifact{}, e
	}
	var a Artifact
	if e = json.Unmarshal(data, &a); e != nil {
		return a, e
	}
	if a.Fingerprint != key {
		return a, errors.New("recovery record identity mismatch")
	}
	return a, nil
}

// An empty identity discovers the repository's most recently updated record,
// independently of current HEAD or remote access. Exact IDs remain attachable.
// Incomplete discovery is an error, never an assertion that no recovery exists.
func (s DiskStore) latest(ctx context.Context, source string) (string, error) {
	dir, e := os.Open(s.Root)
	if os.IsNotExist(e) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	defer dir.Close()
	entries, e := dir.ReadDir(1001)
	if e != nil && e != io.EOF {
		return "", e
	}
	if len(entries) > 1000 {
		return "", errors.New("recovery discovery exceeds 1000 entries; use an exact operation ID")
	}
	key := ""
	var newest time.Time
	for _, entry := range entries {
		if e = ctx.Err(); e != nil {
			return "", e
		}
		if len(entry.Name()) != 64 || !validOID(entry.Name()) {
			continue
		}
		a, err := s.readRecord(entry.Name())
		if err != nil {
			return "", errors.New("recovery discovery is incomplete; a retained record is unreadable; use a known operation ID")
		}
		if a.Source != source {
			continue
		}
		info, err := os.Stat(filepath.Join(s.Root, entry.Name(), "recovery.json"))
		if err != nil {
			return "", err
		}
		if key == "" || info.ModTime().After(newest) || (info.ModTime().Equal(newest) && entry.Name() > key) {
			key = entry.Name()
			newest = info.ModTime()
		}
	}
	return key, nil
}

func (s DiskStore) Load(source, key string) (Artifact, error) {
	return s.LoadContext(context.Background(), source, key)
}

func (s DiskStore) LoadContext(ctx context.Context, source, key string) (Artifact, error) {
	if key == "" {
		var e error
		key, e = s.latest(ctx, source)
		if e != nil {
			return Artifact{}, e
		}
		if key == "" {
			return Artifact{State: "absent", Message: "No retained preparation was found for this repository."}, nil
		}
	}
	a, e := s.readRecord(key)
	if os.IsNotExist(e) {
		// A reserved directory without a readable record may be an interrupted
		// write. Do not erase that distinction or allow an implicit restart.
		if _, dirErr := os.Lstat(filepath.Join(s.Root, key)); dirErr == nil {
			return Artifact{}, errors.New("recovery reservation exists without a readable record; inspect retained storage")
		}
		return Artifact{State: "absent", Message: "No preparation record exists for this operation ID."}, nil
	}
	if e != nil {
		return Artifact{}, e
	}
	if a.Source != source {
		return Artifact{}, errors.New("recovery record belongs to another repository")
	}
	if a.State == "preparing" {
		a.Message = "Preparation is running or was interrupted. Retained evidence is unresolved, not verified. Recheck this operation ID; a server restart requires inspection of retained artifacts."
		return a, nil
	}
	if a.State != "prepared" {
		return a, nil
	}
	if len(a.OriginalDigest) != 64 || len(a.RepairedDigest) != 64 {
		a.State = "unverified"
		a.Message = "Legacy preparation has no saved bundle integrity evidence. Files are retained, but verification is unavailable. Nothing has been applied."
		return a, nil
	}
	for _, b := range []struct{ path, name, digest string }{{a.OriginalBundle, "original.bundle", a.OriginalDigest}, {a.RepairedBundle, "repaired.bundle", a.RepairedDigest}} {
		if b.path != filepath.Join(s.Root, key, b.name) {
			a.State = "damaged"
			a.Message = "Bundle location does not match this operation's storage. Retained files were not changed."
			return a, nil
		}
		digest, err := s.Digest(ctx, b.path)
		if err != nil || !strings.EqualFold(digest, b.digest) {
			a.State = "damaged"
			a.Message = "A recovery bundle is missing, changed, or unreadable. Verification failed; do not use these artifacts. Retained evidence was not changed."
			if ctx.Err() != nil {
				a.State = "unverified"
				a.Message = "Bundle verification did not finish within this request. Retained artifacts are unchanged; recheck this operation ID."
			}
			return a, nil
		}
	}
	a.Message = "Both bundles match the bytes independently restore-checked during preparation. This checks committed-history artifacts only; current source and remote eligibility still require verification. Nothing has been applied."
	return a, nil
}

// CheckCurrent reports current eligibility separately from persisted preparation.
// No status read rewrites the historical operation record.
func CheckCurrent(a Artifact, r Report) Artifact {
	if a.State != "prepared" {
		return a
	}
	if !r.Complete {
		a.State = "unverified"
		a.Message = "Bundle integrity checked. The current source or live remote could not be verified. Retained artifacts remain available; refresh before considering application."
		return a
	}
	if r.Fingerprint != a.Fingerprint {
		a.State = "stale"
		a.Message = "Bundle integrity checked, but the source or push destination changed since preparation. The old operation remains available. Prepare a fresh preview; nothing has been applied."
		return a
	}
	a.Message = "Both bundle contents checked against independently restored originals. Source and live destination match the reviewed snapshot at this check. This does not back up uncommitted work or authorize application. Nothing has been applied or pushed."
	return a
}
