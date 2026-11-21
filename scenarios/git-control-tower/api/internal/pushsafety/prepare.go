package pushsafety

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Mapping struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
}
type Artifact struct {
	Remote            string    `json:"remote"`
	Branch            string    `json:"branch"`
	Source            string    `json:"source"`
	State             string    `json:"state"`
	Message           string    `json:"message"`
	Fingerprint       string    `json:"fingerprint"`
	Head              string    `json:"head"`
	Base              string    `json:"base"`
	Candidate         string    `json:"candidate"`
	OriginalBundle    string    `json:"original_bundle"`
	RepairedBundle    string    `json:"repaired_bundle"`
	Mappings          []Mapping `json:"mappings"`
	Paths             []string  `json:"paths"`
	SignaturesRemoved bool      `json:"signatures_removed"`
	OriginalDigest    string    `json:"original_sha256"`
	RepairedDigest    string    `json:"repaired_sha256"`
}

// ArtifactStore owns durable state outside the source checkout. An existing
// operation is never overwritten or re-executed on a retry. A preparing record
// after process interruption remains explicit; callers can inspect its files.
type ArtifactStore interface {
	Begin(context.Context, string) (string, *Artifact, error)
	Save(string, Artifact) error
	Digest(context.Context, string) (string, error)
}
type DiskStore struct{ Root string }

func (s DiskStore) Begin(ctx context.Context, key string) (string, *Artifact, error) {
	if len(key) != 64 || !validOID(key) {
		return "", nil, errors.New("invalid recovery identity")
	}
	if !filepath.IsAbs(s.Root) {
		return "", nil, errors.New("recovery storage must be absolute")
	}
	if e := os.MkdirAll(s.Root, 0o700); e != nil {
		return "", nil, e
	}
	dir := filepath.Join(s.Root, key)
	if e := os.Mkdir(dir, 0o700); e != nil {
		if !os.IsExist(e) {
			return "", nil, e
		}
		a, e := s.readRecord(key)
		if e != nil {
			return "", nil, errors.New("recovery reservation exists without a valid readable record; inspect storage before retrying")
		}
		checked, e := s.LoadContext(ctx, a.Source, key)
		return dir, &checked, e
	}
	return dir, nil, nil
}

func (s DiskStore) Save(dir string, a Artifact) error {
	b, e := json.MarshalIndent(a, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".recovery-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	if e = os.Rename(name, filepath.Join(dir, "recovery.json")); e != nil {
		return e
	}
	d, e := os.Open(dir)
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}

// Prepare only writes in an independently owned artifact directory. Source
// files/index/refs are never writers' arguments. A full copy and verified bundle
// protect committed history; live/untracked/ignored files are NOT backup claims.
func Prepare(ctx context.Context, c Commands, store ArtifactStore, source string, r Report) (a Artifact, err error) {
	if !r.CanPrepare || r.State != "blocked" || r.Fingerprint == "" {
		return a, errors.New("this inspection does not permit automatic preparation")
	}
	dir, existing, e := store.Begin(ctx, r.Fingerprint)
	if e != nil {
		return a, e
	}
	if existing != nil {
		if existing.Source != "" && existing.Source != source {
			return a, errors.New("recovery record belongs to another repository")
		}
		return *existing, nil
	}
	a = Artifact{State: "preparing", Source: source, Remote: r.Remote, Branch: r.Branch, Fingerprint: r.Fingerprint, Head: r.Head, Base: r.Base, Message: "Preparation writes only isolated artifacts. Live working files and the index are not backed up."}
	if e = store.Save(dir, a); e != nil {
		return a, e
	}
	defer func() {
		if err != nil {
			a.State = "failed"
			a.Message = "Preparation did not complete. Original repository unchanged. Inspect retained artifacts before retrying."
			if saveErr := store.Save(dir, a); saveErr != nil {
				err = fmt.Errorf("%w; could not persist failure: %v", err, saveErr)
			}
		}
	}()
	for _, f := range r.Files {
		if f.Blocked {
			for _, p := range f.Paths {
				a.Paths = appendUnique(a.Paths, p)
			}
		}
	}
	if len(a.Paths) == 0 {
		return a, errors.New("no exact paths are available for recovery")
	}
	isolated := filepath.Join(dir, "repository.git")
	if _, e = c.Run(ctx, dir, nil, "clone", "--bare", "--no-local", "--no-hardlinks", "--", source, isolated); e != nil {
		return a, e
	}
	if _, e = c.Run(ctx, isolated, nil, "cat-file", "-e", r.Head+"^{commit}"); e != nil {
		return a, errors.New("captured source commit is missing from isolated copy; refresh inspection")
	}
	if _, e = c.Run(ctx, isolated, nil, "update-ref", "refs/heads/recovery-original", r.Head); e != nil {
		return a, e
	}
	a.OriginalBundle = filepath.Join(dir, "original.bundle")
	if _, e = c.Run(ctx, isolated, nil, "bundle", "create", a.OriginalBundle, "refs/heads/recovery-original"); e != nil {
		return a, e
	}
	if _, e = c.Run(ctx, isolated, nil, "bundle", "verify", a.OriginalBundle); e != nil {
		return a, e
	}
	a.OriginalDigest, e = store.Digest(ctx, a.OriginalBundle)
	if e != nil {
		return a, e
	}
	// Restore the original bundle into another empty repository. This checks that
	// the backup is independently usable, rather than relying on source objects.
	check := filepath.Join(dir, "restore-check.git")
	if _, e = c.Run(ctx, dir, nil, "clone", "--bare", "--", a.OriginalBundle, check); e != nil {
		return a, e
	}
	if _, e = c.Run(ctx, check, nil, "fsck", "--full", "--no-reflogs"); e != nil {
		return a, e
	}
	if _, e = c.Run(ctx, check, nil, "cat-file", "-e", r.Head+"^{commit}"); e != nil {
		return a, e
	}
	parent := r.Base
	for _, commit := range r.Commits {
		if _, e = c.Run(ctx, isolated, nil, "read-tree", commit); e != nil {
			return a, e
		}
		var indexInput strings.Builder
		for _, path := range a.Paths {
			indexInput.WriteString("0 " + strings.Repeat("0", len(r.Head)) + "\t" + path + "\x00")
		}
		input := []byte(indexInput.String())
		if _, e = c.Run(ctx, isolated, input, "update-index", "-z", "--index-info"); e != nil {
			return a, e
		}
		tree, e := gitString(ctx, c, isolated, "write-tree")
		if e != nil || !validOID(tree) {
			return a, errors.New("cannot write replacement tree")
		}
		raw, e := c.Run(ctx, isolated, nil, "cat-file", "commit", commit)
		if e != nil {
			return a, e
		}
		rewritten, signed, e := rewriteCommit(raw, tree, parent)
		if e != nil {
			return a, e
		}
		a.SignaturesRemoved = a.SignaturesRemoved || signed
		newOID, e := c.Run(ctx, isolated, rewritten, "hash-object", "-t", "commit", "-w", "--stdin")
		if e != nil {
			return a, e
		}
		replacement := strings.TrimSpace(string(newOID))
		if !validOID(replacement) {
			return a, errors.New("replacement commit identity is invalid")
		}
		// Independently enumerate original and replacement trees. Every entry
		// outside the explicitly approved path set must match, including mode/OID.
		oldTree, e := c.Run(ctx, isolated, nil, "ls-tree", "-r", "-z", commit)
		if e != nil {
			return a, e
		}
		newTree, e := c.Run(ctx, isolated, nil, "ls-tree", "-r", "-z", replacement)
		if e != nil {
			return a, e
		}
		if e = verifyTrees(oldTree, newTree, a.Paths); e != nil {
			return a, e
		}
		a.Mappings = append(a.Mappings, Mapping{commit, replacement})
		parent = replacement
	}
	a.Candidate = parent
	if _, e = c.Run(ctx, isolated, nil, "update-ref", "refs/heads/recovery-candidate", parent); e != nil {
		return a, e
	}
	objects, e := c.Run(ctx, isolated, nil, "rev-list", "--objects", "--no-object-names", r.Base+".."+parent)
	if e != nil {
		return a, e
	}
	meta, e := c.Run(ctx, isolated, objects, "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if e != nil {
		return a, e
	}
	expected := map[string]bool{}
	for _, oid := range strings.Fields(string(objects)) {
		expected[oid] = true
	}
	for _, line := range strings.Split(strings.TrimSpace(string(meta)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || !expected[fields[0]] {
			return a, errors.New("candidate object metadata is incomplete or unexpected")
		}
		delete(expected, fields[0])
	}
	if len(expected) > 0 {
		return a, errors.New("candidate object sizes are incomplete")
	}
	if e = verifySizes(meta, r.Limit); e != nil {
		return a, e
	}
	a.RepairedBundle = filepath.Join(dir, "repaired.bundle")
	if _, e = c.Run(ctx, isolated, nil, "bundle", "create", a.RepairedBundle, "refs/heads/recovery-candidate"); e != nil {
		return a, e
	}
	if _, e = c.Run(ctx, isolated, nil, "bundle", "verify", a.RepairedBundle); e != nil {
		return a, e
	}
	a.RepairedDigest, e = store.Digest(ctx, a.RepairedBundle)
	if e != nil {
		return a, e
	}
	// Independently restore the candidate as well; bundle verify alone does
	// not establish that its packed objects can actually be restored.
	repairedCheck := filepath.Join(dir, "repaired-restore-check.git")
	if _, e = c.Run(ctx, dir, nil, "clone", "--bare", "--", a.RepairedBundle, repairedCheck); e != nil {
		return a, e
	}
	if _, e = c.Run(ctx, repairedCheck, nil, "fsck", "--full", "--no-reflogs"); e != nil {
		return a, e
	}
	restored, e := gitString(ctx, c, repairedCheck, "rev-parse", "--verify", "refs/heads/recovery-candidate")
	if e != nil || restored != a.Candidate {
		return a, errors.New("restored candidate identity does not match")
	}
	for _, bundle := range []struct{ path, digest string }{{a.OriginalBundle, a.OriginalDigest}, {a.RepairedBundle, a.RepairedDigest}} {
		digest, e := store.Digest(ctx, bundle.path)
		if e != nil {
			return a, e
		}
		if digest != bundle.digest {
			return a, errors.New("bundle changed during restore verification")
		}
	}
	a.State = "prepared"
	a.Message = "Replacement commits verified; active branch unchanged. Backup covers committed history only. Before applying, pause shared-checkout writers, capture working files and index, and recheck the live remote. Add an exact ignore rule and verify packaging. No push was performed."
	if e = store.Save(dir, a); e != nil {
		return a, e
	}
	return a, nil
}

func rewriteCommit(raw []byte, tree, parent string) ([]byte, bool, error) {
	header, body, ok := strings.Cut(string(raw), "\n\n")
	if !ok {
		return nil, false, errors.New("invalid commit")
	}
	lines := strings.Split(header, "\n")
	out := []string{}
	skip := false
	signed := false
	parents := 0
	for _, line := range lines {
		if strings.HasPrefix(line, " ") {
			if !skip {
				out = append(out, line)
			}
			continue
		}
		skip = false
		switch {
		case strings.HasPrefix(line, "tree "):
			out = append(out, "tree "+tree)
		case strings.HasPrefix(line, "parent "):
			parents++
			out = append(out, "parent "+parent)
		case strings.HasPrefix(line, "gpgsig ") || strings.HasPrefix(line, "gpgsig-sha256 "):
			skip = true
			signed = true
		case strings.HasPrefix(line, "mergetag "):
			return nil, false, errors.New("merge-tagged history requires manual review")
		default:
			out = append(out, line)
		}
	}
	if parents != 1 {
		return nil, false, errors.New("only linear commits can be prepared")
	}
	return []byte(strings.Join(out, "\n") + "\n\n" + body), signed, nil
}

func verifyTrees(old, new []byte, paths []string) error {
	drop := map[string]bool{}
	for _, p := range paths {
		drop[p] = true
	}
	filter := func(raw []byte, replacement bool) (string, error) {
		out := []string{}
		for _, s := range strings.Split(string(raw), "\x00") {
			if s == "" {
				continue
			}
			_, p, ok := strings.Cut(s, "\t")
			if !ok {
				return "", errors.New("invalid tree record")
			}
			if drop[p] {
				if replacement {
					return "", errors.New("removed path remains in candidate")
				}
				continue
			}
			out = append(out, s)
		}
		return strings.Join(out, "\x00"), nil
	}
	a, e := filter(old, false)
	if e != nil {
		return e
	}
	b, e := filter(new, true)
	if e != nil {
		return e
	}
	if a != b {
		return errors.New("candidate changed content outside the approved paths")
	}
	return nil
}
