package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Indexer walks a configured filesystem root for component manifests
// under components/*/component.json and upserts the manifest plus its
// version folders into the Repository. A final DeleteMissing call
// removes rows whose manifests no longer exist, so deleted components
// leave the registry without manual intervention.
//
// The header contract (req CR-002, captured in PRD.md):
//
//	/**
//	 * @libraryId   react-component-library:Button
//	 * @displayName Button
//	 * @description Primary call-to-action button.
//	 * @version     1.0.0
//	 * @tags        ["form", "interactive"]
//	 * @deps        {"react": "^18"}
//	 * @warning     DO NOT REMOVE THIS HEADER...
//	 */
//
// component.json is the source of truth. Source-file headers are
// validation hints for version, status, deps, and readability.
// UpsertObserver is the post-upsert seam other domains hook into. The
// indexer calls Observe after a successful repo.Upsert with the parsed
// header fields, so cross-domain consumers (currently req 10's deps
// recorder) can re-sync without parsing files themselves. nil observer
// means "no hook"; production wires deps.Service via a small adapter
// in main.go.
type UpsertObserver interface {
	Observe(ctx context.Context, c Component, headerFields map[string]string) error
}

type Indexer struct {
	repo     Repository
	root     string
	fs       fs.FS // injected for tests; nil means use os.DirFS(root)
	observer UpsertObserver
}

// SetUpsertObserver installs the post-upsert seam. Designed to be
// called once at boot before any Run call; not concurrency-safe with
// in-flight Runs.
func (idx *Indexer) SetUpsertObserver(o UpsertObserver) { idx.observer = o }

// NewIndexer constructs an Indexer rooted at root. The root is the
// absolute path on disk; consumers resolve it via api-core/storage
// before calling. fsys may be nil — production passes nil and the
// indexer wraps os.DirFS(root); tests pass an in-memory fs.FS so they
// don't touch disk.
func NewIndexer(repo Repository, root string, fsys fs.FS) *Indexer {
	if fsys == nil && root != "" {
		fsys = os.DirFS(root)
	}
	return &Indexer{repo: repo, root: root, fs: fsys}
}

// IndexResult summarizes one Run.
type IndexResult struct {
	Scanned    int
	Indexed    int
	Skipped    int
	Deleted    int
	Errors     []error
	LibraryIDs []string // upserted IDs in walk order — useful for tests
}

// Run walks the root, upserts every manifest with valid version folders,
// and returns a result. Malformed manifests are recorded in Errors but
// do not stop the walk — a single broken component should not hide an
// otherwise healthy run.
func (idx *Indexer) Run(ctx context.Context) (IndexResult, error) {
	var result IndexResult
	if idx.fs == nil {
		return result, fmt.Errorf("indexer has no filesystem configured")
	}

	walkErr := fs.WalkDir(idx.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "component.json" {
			return nil
		}
		result.Scanned++
		in, fields, perr := idx.buildManifestInput(path)
		if perr != nil {
			result.Errors = append(result.Errors, perr)
			return nil
		}
		comp, err := idx.repo.UpsertManifest(ctx, in)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("upsert %s: %w", path, err))
			return nil
		}
		if idx.observer != nil {
			if oerr := idx.observer.Observe(ctx, comp, fields); oerr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("observe %s: %w", path, oerr))
				// continue — observer failure must not hide the upsert.
			}
		}
		result.Indexed++
		result.LibraryIDs = append(result.LibraryIDs, in.Manifest.LibraryID)
		return nil
	})
	if walkErr != nil {
		return result, fmt.Errorf("walk %s: %w", idx.root, walkErr)
	}

	deleted, derr := idx.repo.DeleteMissing(ctx, result.LibraryIDs)
	if derr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("delete missing: %w", derr))
	}
	result.Deleted = deleted

	return result, nil
}

type manifestFile struct {
	LibraryID          string   `json:"libraryId"`
	DisplayName        string   `json:"displayName"`
	Description        string   `json:"description"`
	Tags               []string `json:"tags"`
	Latest             string   `json:"latest"`
	Draft              string   `json:"draft"`
	DeprecatedVersions []string `json:"deprecatedVersions"`
}

func (idx *Indexer) buildManifestInput(path string) (IndexManifestInput, map[string]string, error) {
	raw, err := fs.ReadFile(idx.fs, path)
	if err != nil {
		return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", path, err)
	}
	var mf manifestFile
	if err := json.Unmarshal(raw, &mf); err != nil {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "component.json", Reason: err.Error()}
	}
	slug := filepath.Base(filepath.Dir(path))
	manifest := ComponentManifest{
		LibraryID:          strings.TrimSpace(mf.LibraryID),
		Slug:               slug,
		DisplayName:        strings.TrimSpace(mf.DisplayName),
		Description:        strings.TrimSpace(mf.Description),
		ManifestPath:       filepath.ToSlash(path),
		LatestVersion:      strings.TrimSpace(mf.Latest),
		DraftVersion:       strings.TrimSpace(mf.Draft),
		DeprecatedVersions: append([]string(nil), mf.DeprecatedVersions...),
		Tags:               append([]string(nil), mf.Tags...),
	}
	if manifest.LibraryID == "" {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	if manifest.DisplayName == "" {
		manifest.DisplayName = slug
	}
	if manifest.LatestVersion == "" {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "required"}
	}
	versionRoot := filepath.ToSlash(filepath.Join(filepath.Dir(path), "versions"))
	entries, err := fs.ReadDir(idx.fs, versionRoot)
	if err != nil {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "versions", Reason: err.Error()}
	}
	deprecated := map[string]bool{}
	for _, v := range manifest.DeprecatedVersions {
		deprecated[strings.TrimSpace(v)] = true
	}
	var versions []ComponentVersion
	var latestFound bool
	var draftFound bool
	fieldsForDeps := map[string]string{"libraryId": manifest.LibraryID}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := strings.TrimSpace(entry.Name())
		versionPath := filepath.ToSlash(filepath.Join(versionRoot, version))
		files, err := fs.ReadDir(idx.fs, versionPath)
		if err != nil {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: err.Error()}
		}
		var tsx []fs.DirEntry
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".tsx") {
				tsx = append(tsx, f)
			}
		}
		if len(tsx) != 1 {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: "expected exactly one .tsx file"}
		}
		sourcePath := filepath.ToSlash(filepath.Join(versionPath, tsx[0].Name()))
		src, err := fs.ReadFile(idx.fs, sourcePath)
		if err != nil {
			return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", sourcePath, err)
		}
		headers := map[string]string{}
		if header, ok := extractHeaderBlock(string(src)); ok {
			headers, err = parseHeader(sourcePath, header)
			if err != nil {
				return IndexManifestInput{}, nil, err
			}
			if hv := strings.TrimSpace(headers["version"]); hv != "" && hv != version {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: sourcePath, Field: "version", Reason: "does not match version folder"}
			}
			if hid := strings.TrimSpace(headers["libraryId"]); hid != "" && hid != manifest.LibraryID {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: sourcePath, Field: "libraryId", Reason: "does not match manifest"}
			}
		}
		status := VersionStatusReleased
		if strings.Contains(version, "-") {
			status = VersionStatusDraft
		}
		if deprecated[version] {
			status = VersionStatusDeprecated
		}
		if version == manifest.LatestVersion {
			latestFound = true
			if status != VersionStatusReleased {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "must point to a released non-deprecated version"}
			}
			for k, v := range headers {
				fieldsForDeps[k] = v
			}
		}
		if version == manifest.DraftVersion {
			draftFound = true
			if status != VersionStatusDraft {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "draft", Reason: "must point to a draft/pre-release version"}
			}
		}
		versions = append(versions, ComponentVersion{
			LibraryID:     manifest.LibraryID,
			Version:       version,
			Status:        status,
			SourcePath:    sourcePath,
			Content:       string(src),
			ContentSHA256: digestBytes(src),
		})
	}
	if !latestFound {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "version folder not found"}
	}
	if manifest.DraftVersion != "" && !draftFound {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "draft", Reason: "version folder not found"}
	}
	return IndexManifestInput{Manifest: manifest, Versions: versions}, fieldsForDeps, nil
}

// headerBlockRe captures the first /** … */ comment block in a file.
// (?s) flag makes . match newlines.
var headerBlockRe = regexp.MustCompile(`(?s)/\*\*(.*?)\*/`)

// headerFieldRe captures @field-name value pairs on each header line.
// The leading `*` and surrounding whitespace are stripped by the
// caller before this matches.
var headerFieldRe = regexp.MustCompile(`^@([A-Za-z][A-Za-z0-9_-]*)\s+(.*)$`)

// extractHeaderBlock returns the inner text of the first JSDoc-style
// header block, or ok=false if no such block exists. Only blocks that
// contain `@libraryId` are treated as library headers; otherwise the
// indexer would claim every JSDoc-commented file.
func extractHeaderBlock(src string) (string, bool) {
	matches := headerBlockRe.FindStringSubmatch(src)
	if len(matches) < 2 {
		return "", false
	}
	body := matches[1]
	if !strings.Contains(body, "@libraryId") {
		return "", false
	}
	return body, true
}

// parseHeader extracts field/value pairs from a header block. Multi-
// line values are folded onto the @field line until the next @field
// or end-of-block (so JSDoc continuation lines are tolerated).
func parseHeader(path, body string) (map[string]string, error) {
	out := map[string]string{}
	var currentField string
	var currentValue strings.Builder

	flush := func() {
		if currentField == "" {
			return
		}
		out[currentField] = strings.TrimSpace(currentValue.String())
		currentField = ""
		currentValue.Reset()
	}

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := headerFieldRe.FindStringSubmatch(line); m != nil {
			flush()
			currentField = m[1]
			currentValue.WriteString(m[2])
			continue
		}
		if currentField != "" {
			currentValue.WriteByte(' ')
			currentValue.WriteString(line)
		}
	}
	flush()

	if _, ok := out["libraryId"]; !ok {
		return nil, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	return out, nil
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
