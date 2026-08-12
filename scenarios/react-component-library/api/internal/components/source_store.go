package components

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const defaultInitialVersion = "0.1.0"

type SourceStore interface {
	InitializeComponent(ctx context.Context, in InitializeComponentInput) (manifestPath string, sourcePath string, error error)
	CreateVersion(ctx context.Context, c Component, in CreateComponentVersionInput) (sourcePath string, error error)
	UpdateManifest(ctx context.Context, c Component, in UpdateComponentManifestInput) error
	Root() string
}

type sourceManifestFile struct {
	LibraryID          string                 `json:"libraryId"`
	DisplayName        string                 `json:"displayName"`
	Description        string                 `json:"description"`
	Tags               []string               `json:"tags"`
	Slot               string                 `json:"slot,omitempty"`
	Category           string                 `json:"category,omitempty"`
	Entry              string                 `json:"entry,omitempty"`
	Latest             string                 `json:"latest"`
	Draft              string                 `json:"draft,omitempty"`
	DeprecatedVersions []string               `json:"deprecatedVersions"`
	DesignStyles       []sourceDesignAffinity `json:"designStyles"`
}

type sourceDesignAffinity struct {
	StyleID  string `json:"styleId"`
	Affinity string `json:"affinity"`
	Reason   string `json:"reason,omitempty"`
}

func (s *FSContentStore) Root() string { return s.root }

func (s *FSContentStore) InitializeComponent(_ context.Context, in InitializeComponentInput) (string, string, error) {
	slug := normalizeSlug(firstNonEmpty(in.Slug, in.DisplayName, in.LibraryID))
	if slug == "" {
		return "", "", ErrInvalidHeader{SourcePath: "component.json", Field: "slug", Reason: "required"}
	}
	libraryID := strings.TrimSpace(in.LibraryID)
	if libraryID == "" {
		libraryID = "react-component-library:" + slug
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = slug
	}
	version := strings.TrimSpace(in.InitialVersion)
	if version == "" {
		version = defaultInitialVersion
	}
	if err := validateVersionToken(version); err != nil {
		return "", "", err
	}
	if strings.Contains(version, "-") {
		return "", "", ErrInvalidHeader{SourcePath: "component.json", Field: "initialVersion", Reason: "initial component version must be a released semver-like version"}
	}
	files, fileName, err := normalizeVersionFiles(in.InitialFiles, firstNonEmpty(in.FileName, slug+".tsx"), in.InitialSource)
	if err != nil {
		return "", "", err
	}
	manifestPath := filepath.ToSlash(filepath.Join("components", slug, "component.json"))
	sourcePath := filepath.ToSlash(filepath.Join("components", slug, "versions", version, fileName))
	manifestAbs, err := s.resolveCreatable(manifestPath)
	if err != nil {
		return "", "", err
	}
	sourceAbs, err := s.resolveCreatable(sourcePath)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(manifestAbs); err == nil {
		return "", "", ErrComponentAlreadyExists{Slug: slug}
	} else if !os.IsNotExist(err) {
		return "", "", fmt.Errorf("stat component manifest %q: %w", manifestPath, err)
	}
	mf := sourceManifestFile{
		LibraryID:          libraryID,
		DisplayName:        displayName,
		Description:        strings.TrimSpace(in.Description),
		Tags:               cleanTags(in.Tags),
		Slot:               strings.TrimSpace(in.Slot),
		Category:           strings.TrimSpace(in.Category),
		Entry:              fileName,
		Latest:             version,
		DeprecatedVersions: []string{},
		DesignStyles: []sourceDesignAffinity{{
			StyleID: "vrooli-default", Affinity: "native", Reason: "token-native operational-console baseline",
		}},
	}
	if err := writeJSONFile(manifestAbs, mf); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(sourceAbs), 0o755); err != nil {
		return "", "", fmt.Errorf("create component version dir: %w", err)
	}
	for _, file := range files {
		body := strings.TrimSpace(file.Content)
		if file.IsEntry {
			if body == "" {
				body = defaultComponentSource(libraryID, displayName, mf.Description, version, mf.Tags)
			} else {
				body = ensureHeaderFields(body, libraryID, displayName, mf.Description, version, mf.Tags)
			}
		}
		path := filepath.Join(filepath.Dir(sourceAbs), file.Path)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return "", "", fmt.Errorf("write component source %q: %w", filepath.ToSlash(filepath.Join(filepath.Dir(sourcePath), file.Path)), err)
		}
	}
	if in.ScaffoldExamples {
		if err := scaffoldStoryFile(filepath.Dir(sourceAbs)); err != nil {
			return "", "", err
		}
	}
	return manifestPath, sourcePath, nil
}

func (s *FSContentStore) CreateVersion(_ context.Context, c Component, in CreateComponentVersionInput) (string, error) {
	version := strings.TrimSpace(in.Version)
	if err := validateVersionToken(version); err != nil {
		return "", err
	}
	intent := in.Intent
	if intent == "" {
		if strings.Contains(version, "-") {
			intent = VersionIntentDraft
		} else {
			intent = VersionIntentRelease
		}
	}
	if intent == VersionIntentRelease && strings.Contains(version, "-") {
		return "", ErrInvalidHeader{SourcePath: c.ManifestPath, Field: "version", Reason: "release versions must not be prerelease values"}
	}
	if intent == VersionIntentDraft && !strings.Contains(version, "-") {
		return "", ErrInvalidHeader{SourcePath: c.ManifestPath, Field: "version", Reason: "draft versions must be prerelease values"}
	}
	files, fileName, err := normalizeVersionFiles(in.Files, firstNonEmpty(in.FileName, filepath.Base(c.SourcePath), normalizeSlug(c.DisplayName)+".tsx"), in.Source)
	if err != nil {
		return "", err
	}
	assetRoot := componentAssetRoot(c)
	sourcePath := filepath.ToSlash(filepath.Join(assetRoot, c.Slug, "versions", version, fileName))
	sourceAbs, err := s.resolveCreatable(sourcePath)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(sourceAbs); err == nil {
		return "", ErrInvalidHeader{SourcePath: sourcePath, Field: "version", Reason: "version source already exists"}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat component source %q: %w", sourcePath, err)
	}
	if len(in.Files) == 0 && strings.TrimSpace(in.Source) == "" {
		from := firstNonEmpty(in.FromVersion, c.DraftVersion, c.LatestVersion, c.Version)
		fromDir := filepath.Join(s.root, assetRoot, c.Slug, "versions", from)
		if entries, err := os.ReadDir(fromDir); err == nil {
			files = files[:0]
			for _, entry := range entries {
				if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".ts") && !strings.HasSuffix(entry.Name(), ".tsx") && entry.Name() != "experience-contract.json") {
					continue
				}
				raw, err := os.ReadFile(filepath.Join(fromDir, entry.Name()))
				if err != nil {
					return "", fmt.Errorf("read source to copy %q: %w", entry.Name(), err)
				}
				files = append(files, ComponentVersionFile{Path: entry.Name(), Content: string(raw), IsEntry: entry.Name() == fileName})
			}
			if len(files) > 0 {
				sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
			}
		}
		if len(files) == 0 || !hasEntryFile(files) {
			files, _, err = normalizeVersionFiles(nil, fileName, defaultComponentSource(c.LibraryID, c.DisplayName, c.Description, version, c.Tags))
			if err != nil {
				return "", err
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(sourceAbs), 0o755); err != nil {
		return "", fmt.Errorf("create component version dir: %w", err)
	}
	for _, file := range files {
		body := strings.TrimSpace(file.Content)
		if file.IsEntry {
			body = ensureHeaderFields(body, c.LibraryID, c.DisplayName, c.Description, version, c.Tags)
		}
		path := filepath.Join(filepath.Dir(sourceAbs), file.Path)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return "", fmt.Errorf("write component source %q: %w", filepath.ToSlash(filepath.Join(filepath.Dir(sourcePath), file.Path)), err)
		}
	}
	if in.ParityReport != nil {
		if err := writeParityReport(filepath.Join(filepath.Dir(sourceAbs), "parity.json"), *in.ParityReport); err != nil {
			return "", err
		}
	}
	if strings.TrimSpace(in.ExperienceContract) != "" {
		if err := os.WriteFile(filepath.Join(filepath.Dir(sourceAbs), "experience-contract.json"), []byte(in.ExperienceContract), 0o600); err != nil {
			return "", fmt.Errorf("write version experience contract: %w", err)
		}
	}
	if in.ScaffoldExamples {
		if err := scaffoldStoryFile(filepath.Dir(sourceAbs)); err != nil {
			return "", err
		}
	}
	update := UpdateComponentManifestInput{
		ComponentID:        c.ID,
		DisplayName:        c.DisplayName,
		Description:        c.Description,
		Tags:               c.Tags,
		LatestVersion:      c.LatestVersion,
		DraftVersion:       c.DraftVersion,
		DeprecatedVersions: nil,
	}
	if intent == VersionIntentDraft {
		update.DraftVersion = version
	} else {
		update.LatestVersion = version
		if in.FromVersion != "" && in.FromVersion == c.DraftVersion {
			update.DraftVersion = ""
		}
	}
	if err := s.UpdateManifest(context.Background(), c, update); err != nil {
		return "", err
	}
	return sourcePath, nil
}

// componentAssetRoot keeps source authoring aligned with the manifest that
// was indexed. New components still default to components/, while existing
// primitives (and any future renderable root) retain their real location.
func componentAssetRoot(c Component) string {
	manifest := filepath.ToSlash(strings.TrimSpace(c.ManifestPath))
	if manifest != "" {
		root := filepath.ToSlash(filepath.Dir(filepath.Dir(manifest)))
		if root != "." && root != "" {
			return root
		}
	}
	return "components"
}

const scaffoldStoryJSON = `{
  "schemaVersion": 1,
  "kind": "component",
  "args": { "fields": [] },
  "environment": { "fixtures": [] },
  "stories": [{ "id": "default", "name": "Default", "args": {}, "expect": [] }]
}
`

func scaffoldStoryFile(versionDir string) error {
	path := filepath.Join(versionDir, "story.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat story scaffold %q: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(scaffoldStoryJSON), 0o600); err != nil {
		return fmt.Errorf("write story scaffold %q: %w", path, err)
	}
	return nil
}

func writeParityReport(path string, report IngestParityReport) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ingest parity report: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write ingest parity report %q: %w", path, err)
	}
	return nil
}

func hasEntryFile(files []ComponentVersionFile) bool {
	for _, file := range files {
		if file.IsEntry {
			return true
		}
	}
	return false
}

func normalizeVersionFiles(files []ComponentVersionFile, fallbackName, fallbackSource string) ([]ComponentVersionFile, string, error) {
	if len(files) == 0 {
		name, err := normalizeTSXFileName(fallbackName)
		if err != nil {
			return nil, "", err
		}
		return []ComponentVersionFile{{Path: name, Content: fallbackSource, IsEntry: true}}, name, nil
	}
	normalized := make([]ComponentVersionFile, 0, len(files))
	seen := map[string]bool{}
	entry := ""
	for _, file := range files {
		path := strings.TrimSpace(file.Path)
		if filepath.Base(path) != path || (!strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx")) {
			return nil, "", ErrInvalidHeader{SourcePath: path, Field: "files", Reason: "files must be .ts or .tsx basenames"}
		}
		if seen[path] {
			return nil, "", ErrInvalidHeader{SourcePath: path, Field: "files", Reason: "duplicate file"}
		}
		seen[path] = true
		if file.IsEntry {
			if entry != "" {
				return nil, "", ErrInvalidHeader{SourcePath: path, Field: "files", Reason: "exactly one entry file is required"}
			}
			if !strings.HasSuffix(path, ".tsx") {
				return nil, "", ErrInvalidHeader{SourcePath: path, Field: "files", Reason: "entry file must be .tsx"}
			}
			entry = path
		}
		normalized = append(normalized, ComponentVersionFile{Path: path, Content: file.Content, IsEntry: file.IsEntry})
	}
	if entry == "" {
		return nil, "", ErrInvalidHeader{SourcePath: "files", Field: "files", Reason: "exactly one entry file is required"}
	}
	return normalized, entry, nil
}

func (s *FSContentStore) UpdateManifest(_ context.Context, c Component, in UpdateComponentManifestInput) error {
	manifestPath := c.ManifestPath
	if manifestPath == "" {
		manifestPath = filepath.ToSlash(filepath.Join("components", c.Slug, "component.json"))
	}
	abs, err := s.resolveCreatable(manifestPath)
	if err != nil {
		return err
	}

	// Read-modify-write on the raw JSON object. UpdateManifest owns only a small
	// set of fields (identity, tags, version pointers); every other key on disk —
	// slot, category, entry, fileSlots, designStyles, and any author-added or
	// future field — is preserved verbatim in its original position. A prior
	// struct round-trip silently dropped unknown fields, so designStyles
	// affinities had to be hand-restored after every version cut (bug c71f56c0).
	raw, _ := os.ReadFile(abs)
	manifest, order, err := decodeOrderedObject(raw)
	if err != nil {
		return ErrInvalidHeader{SourcePath: manifestPath, Field: "component.json", Reason: fmt.Sprintf("not a JSON object: %v", err)}
	}

	setManifestField(manifest, &order, "libraryId", c.LibraryID)
	displayName := firstNonEmpty(strings.TrimSpace(in.DisplayName), stringField(manifest, "displayName"), c.DisplayName, c.Slug)
	setManifestField(manifest, &order, "displayName", displayName)
	setManifestField(manifest, &order, "description", strings.TrimSpace(in.Description))
	if in.Tags != nil {
		setManifestField(manifest, &order, "tags", cleanTags(in.Tags))
	} else if _, ok := manifest["tags"]; !ok {
		setManifestField(manifest, &order, "tags", cleanTags(c.Tags))
	}
	latest := firstNonEmpty(strings.TrimSpace(in.LatestVersion), stringField(manifest, "latest"), c.LatestVersion)
	if latest == "" {
		return ErrInvalidHeader{SourcePath: manifestPath, Field: "latest", Reason: "required"}
	}
	setManifestField(manifest, &order, "latest", latest)
	setManifestField(manifest, &order, "draft", strings.TrimSpace(in.DraftVersion))
	if in.DeprecatedVersions != nil {
		setManifestField(manifest, &order, "deprecatedVersions", cleanTags(in.DeprecatedVersions))
	} else if _, ok := manifest["deprecatedVersions"]; !ok {
		setManifestField(manifest, &order, "deprecatedVersions", []string{})
	}
	if _, ok := manifest["designStyles"]; !ok {
		setManifestField(manifest, &order, "designStyles", []sourceDesignAffinity{{
			StyleID: "vrooli-default", Affinity: "native", Reason: "token-native operational-console baseline",
		}})
	}

	encoded, err := marshalOrderedObject(manifest, order)
	if err != nil {
		return fmt.Errorf("encode component manifest: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("create component manifest dir: %w", err)
	}
	if err := os.WriteFile(abs, encoded, 0o600); err != nil {
		return fmt.Errorf("write component manifest: %w", err)
	}
	return nil
}

// decodeOrderedObject parses a JSON object into its key→value map plus the key
// order as they appear on disk, so a rewrite can preserve field ordering. Empty
// input yields an empty object, not an error (a fresh manifest is seeded by the
// caller).
func decodeOrderedObject(data []byte) (map[string]json.RawMessage, []string, error) {
	manifest := map[string]json.RawMessage{}
	var order []string
	if len(bytes.TrimSpace(data)) == 0 {
		return manifest, order, nil
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, nil, fmt.Errorf("expected a JSON object")
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, nil, fmt.Errorf("expected string object key")
		}
		order = append(order, key)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, nil, err
		}
	}
	return manifest, order, nil
}

// setManifestField writes value into manifest under key, appending key to order
// when it is new so existing fields keep their on-disk position.
func setManifestField(manifest map[string]json.RawMessage, order *[]string, key string, value any) {
	if _, ok := manifest[key]; !ok {
		*order = append(*order, key)
	}
	encoded, _ := json.Marshal(value)
	manifest[key] = encoded
}

// stringField returns the string value stored under key, or "" when the key is
// absent or not a JSON string.
func stringField(manifest map[string]json.RawMessage, key string) string {
	raw, ok := manifest[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// marshalOrderedObject re-emits the object in the given key order using the same
// two-space indentation json.MarshalIndent produces, so preserved fields render
// identically to the ones this package writes. Keys present in the map but not in
// order (defensive) are appended deterministically.
func marshalOrderedObject(manifest map[string]json.RawMessage, order []string) ([]byte, error) {
	seen := make(map[string]bool, len(order))
	emitted := make([]string, 0, len(manifest))
	for _, key := range order {
		if seen[key] {
			continue
		}
		if _, ok := manifest[key]; ok {
			seen[key] = true
			emitted = append(emitted, key)
		}
	}
	extra := make([]string, 0)
	for key := range manifest {
		if !seen[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	emitted = append(emitted, extra...)

	var buf bytes.Buffer
	buf.WriteString("{\n")
	for i, key := range emitted {
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		var value bytes.Buffer
		if err := json.Indent(&value, manifest[key], "  ", "  "); err != nil {
			return nil, err
		}
		buf.WriteString("  ")
		buf.Write(keyJSON)
		buf.WriteString(": ")
		buf.Write(value.Bytes())
		if i < len(emitted)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString("}\n")
	return buf.Bytes(), nil
}

func (s *FSContentStore) resolveCreatable(sourcePath string) (string, error) {
	if sourcePath == "" || filepath.IsAbs(sourcePath) {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	cleaned := filepath.Clean(sourcePath)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	abs := filepath.Join(s.root, cleaned)
	rel, err := filepath.Rel(s.root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape{SourcePath: sourcePath, Root: s.root}
	}
	return abs, nil
}

func writeJSONFile(abs string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode component manifest: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("create component manifest dir: %w", err)
	}
	if err := os.WriteFile(abs, raw, 0o600); err != nil {
		return fmt.Errorf("write component manifest: %w", err)
	}
	return nil
}

var (
	slugInvalidRe  = regexp.MustCompile(`[^A-Za-z0-9_-]+`)
	versionTokenRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`)
)

func normalizeSlug(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, ":", "-")
	raw = slugInvalidRe.ReplaceAllString(raw, "-")
	return strings.Trim(raw, "-_")
}

func normalizeTSXFileName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", ErrInvalidHeader{SourcePath: "component source", Field: "fileName", Reason: "required"}
	}
	if filepath.Base(name) != name || strings.Contains(name, "..") || filepath.IsAbs(name) {
		return "", ErrInvalidHeader{SourcePath: name, Field: "fileName", Reason: "must be a single TSX file name"}
	}
	if !strings.HasSuffix(name, ".tsx") {
		name += ".tsx"
	}
	return name, nil
}

func validateVersionToken(version string) error {
	if strings.TrimSpace(version) == "" {
		return ErrInvalidHeader{SourcePath: "component source", Field: "version", Reason: "required"}
	}
	if !versionTokenRe.MatchString(version) {
		return ErrInvalidHeader{SourcePath: "component source", Field: "version", Reason: "must be semver-like, e.g. 0.1.0 or 0.2.0-beta.1"}
	}
	return nil
}

func cleanTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func defaultComponentSource(libraryID, displayName, description, version string, tags []string) string {
	return fmt.Sprintf(`/**
 * @libraryId %s
 * @displayName %s
 * @description %s
 * @version %s
 * @tags %s
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";

export default function %s() {
  return (
    <section className="rcl-component">
      <h1>%s</h1>
    </section>
  );
}
`, libraryID, displayName, description, version, tagsJSON(tags), exportedName(displayName), displayName)
}

func ensureHeaderFields(source, libraryID, displayName, description, version string, tags []string) string {
	deps := metadataHeaderFieldValue(source, "deps")
	// A harvested source can contain both a prior catalog header and its source
	// header. Normalize the whole metadata set to one canonical leading block;
	// otherwise a valid @deps line in the later block is invisible to indexing.
	body := componentMetadataHeaderRe.ReplaceAllStringFunc(source, func(block string) string {
		if strings.Contains(block, "@libraryId") {
			return ""
		}
		return block
	})
	return strings.TrimRight(componentHeader(libraryID, displayName, description, version, tags, deps), "\n") + "\n" + strings.TrimLeft(body, "\n")
}

func componentHeader(libraryID, displayName, description, version string, tags []string, deps string) string {
	depsLine := ""
	if deps = strings.TrimSpace(deps); deps != "" {
		depsLine = " * @deps " + deps + "\n"
	}
	return fmt.Sprintf(`/**
 * @libraryId %s
 * @displayName %s
 * @description %s
 * @version %s
 * @tags %s
%s * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
`, libraryID, displayName, description, version, tagsJSON(tags), depsLine)
}

func headerFieldValue(source, field string) string {
	header, ok := extractHeaderBlock(source)
	if !ok {
		return ""
	}
	re := regexp.MustCompile(`(?m)^\s*\*\s*@` + regexp.QuoteMeta(field) + `\s+(.+?)\s*$`)
	match := re.FindStringSubmatch(header)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

var componentMetadataHeaderRe = regexp.MustCompile(`(?s)/\*\*.*?\*/`)

func metadataHeaderFieldValue(source, field string) string {
	for _, block := range componentMetadataHeaderRe.FindAllString(source, -1) {
		if !strings.Contains(block, "@libraryId") {
			continue
		}
		if value := headerFieldValue(block, field); value != "" {
			return value
		}
	}
	return ""
}

func tagsJSON(tags []string) string {
	raw, _ := json.Marshal(cleanTags(tags))
	return string(raw)
}

func exportedName(raw string) string {
	name := slugInvalidRe.ReplaceAllString(strings.TrimSpace(raw), " ")
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "Component"
	}
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}
