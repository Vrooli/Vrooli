package components

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	LibraryID          string   `json:"libraryId"`
	DisplayName        string   `json:"displayName"`
	Description        string   `json:"description"`
	Tags               []string `json:"tags"`
	Slot               string   `json:"slot,omitempty"`
	Latest             string   `json:"latest"`
	Draft              string   `json:"draft,omitempty"`
	DeprecatedVersions []string `json:"deprecatedVersions"`
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
	fileName, err := normalizeTSXFileName(firstNonEmpty(in.FileName, slug+".tsx"))
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
		Latest:             version,
		DeprecatedVersions: []string{},
	}
	if err := writeJSONFile(manifestAbs, mf); err != nil {
		return "", "", err
	}
	source := strings.TrimSpace(in.InitialSource)
	if source == "" {
		source = defaultComponentSource(libraryID, displayName, mf.Description, version, mf.Tags)
	} else {
		source = ensureHeaderFields(source, libraryID, displayName, mf.Description, version, mf.Tags)
	}
	if err := os.MkdirAll(filepath.Dir(sourceAbs), 0o755); err != nil {
		return "", "", fmt.Errorf("create component version dir: %w", err)
	}
	if err := os.WriteFile(sourceAbs, []byte(source), 0o600); err != nil {
		return "", "", fmt.Errorf("write component source %q: %w", sourcePath, err)
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
	fileName, err := normalizeTSXFileName(firstNonEmpty(in.FileName, filepath.Base(c.SourcePath), normalizeSlug(c.DisplayName)+".tsx"))
	if err != nil {
		return "", err
	}
	sourcePath := filepath.ToSlash(filepath.Join("components", c.Slug, "versions", version, fileName))
	sourceAbs, err := s.resolveCreatable(sourcePath)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(sourceAbs); err == nil {
		return "", ErrInvalidHeader{SourcePath: sourcePath, Field: "version", Reason: "version source already exists"}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat component source %q: %w", sourcePath, err)
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		from := firstNonEmpty(in.FromVersion, c.DraftVersion, c.LatestVersion, c.Version)
		fromPath := filepath.ToSlash(filepath.Join("components", c.Slug, "versions", from, fileName))
		if raw, err := os.ReadFile(filepath.Join(s.root, fromPath)); err == nil {
			source = string(raw)
		} else if raw, err := os.ReadFile(filepath.Join(s.root, c.SourcePath)); err == nil {
			source = string(raw)
		} else {
			source = defaultComponentSource(c.LibraryID, c.DisplayName, c.Description, version, c.Tags)
		}
	}
	source = setHeaderVersion(ensureHeaderFields(source, c.LibraryID, c.DisplayName, c.Description, version, c.Tags), version)
	if err := os.MkdirAll(filepath.Dir(sourceAbs), 0o755); err != nil {
		return "", fmt.Errorf("create component version dir: %w", err)
	}
	if err := os.WriteFile(sourceAbs, []byte(source), 0o600); err != nil {
		return "", fmt.Errorf("write component source %q: %w", sourcePath, err)
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
	}
	if err := s.UpdateManifest(context.Background(), c, update); err != nil {
		return "", err
	}
	return sourcePath, nil
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
	current := sourceManifestFile{
		LibraryID:          c.LibraryID,
		DisplayName:        firstNonEmpty(c.DisplayName, c.Slug),
		Description:        c.Description,
		Tags:               cleanTags(c.Tags),
		Latest:             c.LatestVersion,
		Draft:              c.DraftVersion,
		DeprecatedVersions: []string{},
	}
	if raw, err := os.ReadFile(abs); err == nil {
		_ = json.Unmarshal(raw, &current)
	}
	current.DisplayName = firstNonEmpty(strings.TrimSpace(in.DisplayName), current.DisplayName)
	current.Description = strings.TrimSpace(in.Description)
	if in.Tags != nil {
		current.Tags = cleanTags(in.Tags)
	}
	current.Latest = firstNonEmpty(strings.TrimSpace(in.LatestVersion), current.Latest)
	current.Draft = strings.TrimSpace(in.DraftVersion)
	if in.DeprecatedVersions != nil {
		current.DeprecatedVersions = cleanTags(in.DeprecatedVersions)
	}
	if current.Latest == "" {
		return ErrInvalidHeader{SourcePath: manifestPath, Field: "latest", Reason: "required"}
	}
	return writeJSONFile(abs, current)
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
	slugInvalidRe   = regexp.MustCompile(`[^A-Za-z0-9_-]+`)
	versionTokenRe  = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`)
	headerVersionRe = regexp.MustCompile(`(?m)^(\s*\*\s*@version\s+).*$`)
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
	if _, ok := extractHeaderBlock(source); ok {
		return setHeaderVersion(source, version)
	}
	return strings.TrimRight(defaultComponentHeader(libraryID, displayName, description, version, tags), "\n") + "\n" + strings.TrimLeft(source, "\n")
}

func defaultComponentHeader(libraryID, displayName, description, version string, tags []string) string {
	return fmt.Sprintf(`/**
 * @libraryId %s
 * @displayName %s
 * @description %s
 * @version %s
 * @tags %s
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
`, libraryID, displayName, description, version, tagsJSON(tags))
}

func setHeaderVersion(source, version string) string {
	if headerVersionRe.MatchString(source) {
		return headerVersionRe.ReplaceAllString(source, "${1}"+version)
	}
	return source
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
