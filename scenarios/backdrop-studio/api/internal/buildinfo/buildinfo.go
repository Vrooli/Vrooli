// Package buildinfo answers one question: is the API you are talking to built
// from the source tree you are reading?
//
// Two audits in two days reached a false conclusion from a stale binary — a
// feature was declared missing because the running process predated it. The
// cost is not just wasted time: the second audit's written findings were wrong,
// and wrong findings become someone's plan. So the fingerprint is not a
// diagnostic nicety, it is a precondition the integration lane asserts before
// it renders anything.
//
// The fingerprint is a content hash over the API's Go sources and its seed
// data. Both sides compute it the same way — the server at start, the lane from
// the working tree — so a mismatch means exactly one thing: rebuild.
package buildinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Fingerprint is computed once at process start and reported on the health
// payload. It is a variable rather than a constant because it is derived from
// the source tree the process was started from, which a linker flag cannot
// know for a `go run` in development.
var fingerprint = func() string {
	root, err := SourceRoot()
	if err != nil {
		return ""
	}
	sum, err := Compute(root)
	if err != nil {
		return ""
	}
	return sum
}()

// Fingerprint reports this process's build fingerprint, or "" when the source
// tree was not resolvable (a container image that ships no sources). An empty
// value is honest — it says "cannot prove freshness" rather than claiming a
// match.
func Fingerprint() string { return fingerprint }

// SourceRoot resolves the api/ directory this process was built from.
// BACKDROP_STUDIO_API_SOURCE wins so a deployment can point at a mounted copy;
// otherwise the working directory is walked upward for the api module.
func SourceRoot() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("BACKDROP_STUDIO_API_SOURCE")); explicit != "" {
		return explicit, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if isAPIModule(dir) {
			return dir, nil
		}
		candidate := filepath.Join(dir, "api")
		if isAPIModule(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("buildinfo: no backdrop-studio api module found above the working directory")
		}
		dir = parent
	}
}

func isAPIModule(dir string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(raw), "module backdrop-studio")
}

// Compute hashes every input that changes what the API does: Go sources, the
// embedded catalog seed, and the SQL schema. Test files are excluded because a
// test edit does not change the running server's behaviour, and including them
// would make the lane demand a rebuild after every test change.
func Compute(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "testdata", "node_modules", "coverage", "tmp":
				return fs.SkipDir
			case "integration":
				// The integration lane is build-tagged and never linked into
				// the server. Hashing it would make every edit to the lane
				// demand a rebuild of the thing it tests, which trains a reader
				// to ignore the freshness failure — the one signal that must
				// stay meaningful.
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".sql") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("buildinfo: walk %s: %w", root, err)
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("buildinfo: no source files under %s", root)
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return "", relErr
		}
		// The path is hashed alongside the bytes so moving a file changes the
		// fingerprint even when its contents do not.
		fmt.Fprintf(h, "%s\x00", filepath.ToSlash(rel))
		file, openErr := os.Open(path)
		if openErr != nil {
			return "", openErr
		}
		if _, copyErr := io.Copy(h, file); copyErr != nil {
			_ = file.Close()
			return "", copyErr
		}
		if closeErr := file.Close(); closeErr != nil {
			return "", closeErr
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
