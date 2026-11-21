// Package uxcapture owns the durable, versioned metadata contract for EX9 UI
// captures. Browser orchestration remains owned by BAS capture services.
package uxcapture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	uxcapturepb "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxcapture"
)

const SchemaVersion = "ux-capture.v1"

type SourceKind string

const (
	SourceScenario SourceKind = "scenario"
	SourceURL      SourceKind = "url"
	SourceFile     SourceKind = "file"
)

type LensName string

const (
	LensPixels     LensName = "L0"
	LensFrozenPage LensName = "L1"
	LensOutline    LensName = "L2"
	LensStyle      LensName = "L3"
	LensIdentity   LensName = "L4"
	LensNetwork    LensName = "L5"
	LensStates     LensName = "L6"
)

type LensStatus string

// ArtifactKind is the stable EX9 payload vocabulary. Each kind has a hard
// persistence limit and every declaration binds bytes, media type and digest.
type ArtifactKind string

const (
	ArtifactPixels         ArtifactKind = "pixels"
	ArtifactFullPagePixels ArtifactKind = "full-page-pixels"
	ArtifactRegionPixels   ArtifactKind = "region-pixels"
	ArtifactFrozenPage     ArtifactKind = "frozen-page"
	ArtifactAccessibility  ArtifactKind = "accessibility-snapshot"
	ArtifactConsole        ArtifactKind = "console-summary"
	ArtifactPerformance    ArtifactKind = "performance-metrics"
	ArtifactRegionStyle    ArtifactKind = "region-style-sample"
)

var artifactLimits = map[string]int64{
	string(ArtifactPixels):         20 * 1024 * 1024,
	string(ArtifactFullPagePixels): 20 * 1024 * 1024,
	string(ArtifactRegionPixels):   5 * 1024 * 1024,
	string(ArtifactFrozenPage):     5 * 1024 * 1024,
	string(ArtifactAccessibility):  40 * 1024,
	string(ArtifactConsole):        1024 * 1024,
	string(ArtifactPerformance):    1024 * 1024,
	string(ArtifactRegionStyle):    8 * 1024,
	"outline":                      1024 * 1024, "style-census": 1024 * 1024, "redacted-network": 1024 * 1024,
	"component-identity": 1024 * 1024, "strict-token-diagnostic": 1024 * 1024,
}

const (
	LensPresent LensStatus = "present"
	LensPartial LensStatus = "partial"
	LensAbsent  LensStatus = "absent"
)

// Bundle is the manifest persisted beside the files produced by one BAS
// browser session. Unavailable lenses are represented explicitly so readers
// can distinguish unsupported surfaces from failed capture.
type Bundle struct {
	SchemaVersion string            `json:"schema_version"`
	ID            string            `json:"id"`
	CapturedAt    string            `json:"captured_at"`
	Source        Source            `json:"source"`
	Session       Session           `json:"session"`
	Viewport      Viewport          `json:"viewport"`
	Theme         string            `json:"theme,omitempty"`
	Lenses        map[LensName]Lens `json:"lenses"`
	Artifacts     []Artifact        `json:"artifacts,omitempty"`
	Privacy       Privacy           `json:"privacy"`
	NextCommands  []string          `json:"next_commands,omitempty"`
}

type Source struct {
	Kind     SourceKind `json:"kind"`
	Scenario string     `json:"scenario,omitempty"`
	Route    string     `json:"route,omitempty"`
	URL      string     `json:"url,omitempty"`
	Filename string     `json:"filename,omitempty"`
	BuildID  string     `json:"build_id,omitempty"`
	Revision string     `json:"revision,omitempty"`
}

type Session struct {
	ID       string `json:"id"`
	Browser  string `json:"browser,omitempty"`
	Identity string `json:"identity,omitempty"`
}

type Viewport struct {
	Width       int32   `json:"width"`
	Height      int32   `json:"height"`
	DeviceScale float64 `json:"device_scale_factor,omitempty"`
}

type Lens struct {
	Status LensStatus `json:"status"`
	Reason string     `json:"reason,omitempty"`
	Detail string     `json:"detail,omitempty"`
}

type Artifact struct {
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	MediaType string `json:"media_type,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type ArtifactPayload struct {
	Path    string
	Content []byte
}

type Privacy struct {
	RedactionStatus string   `json:"redaction_status"`
	RedactionReason string   `json:"redaction_reason,omitempty"`
	RedactedClasses []string `json:"redacted_classes,omitempty"`
	External        bool     `json:"external,omitempty"`
	Ephemeral       bool     `json:"ephemeral,omitempty"`
}

var lensOrder = []LensName{LensPixels, LensFrozenPage, LensOutline, LensStyle, LensIdentity, LensNetwork, LensStates}

// Validate rejects incomplete or unsafe manifests before they are persisted.
func (b Bundle) Validate() error {
	// The generated proto is the canonical cross-language bundle schema. Keep
	// this ergonomic Go manifest representation checked against that descriptor
	// before applying the stricter filesystem and privacy invariants below.
	wire, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("encode bundle for canonical schema validation: %w", err)
	}
	var canonical uxcapturepb.Bundle
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(wire, &canonical); err != nil {
		return fmt.Errorf("bundle does not match the canonical proto schema: %w", err)
	}
	if b.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %q", SchemaVersion)
	}
	if strings.TrimSpace(b.ID) == "" {
		return errors.New("id is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, b.CapturedAt); err != nil {
		return fmt.Errorf("captured_at must be RFC3339: %w", err)
	}
	if b.Session.ID == "" {
		return errors.New("session.id is required")
	}
	if b.Viewport.Width <= 0 || b.Viewport.Height <= 0 {
		return errors.New("viewport width and height must be positive")
	}
	switch b.Source.Kind {
	case SourceScenario:
		if strings.TrimSpace(b.Source.Scenario) == "" {
			return errors.New("source.scenario is required for scenario captures")
		}
	case SourceURL:
		if strings.TrimSpace(b.Source.URL) == "" {
			return errors.New("source.url is required for URL captures")
		}
		parsed, err := url.Parse(b.Source.URL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("source.url must be an absolute HTTP(S) URL")
		}
		if parsed.User != nil {
			return errors.New("source.url must not contain credentials")
		}
	case SourceFile:
		if strings.TrimSpace(b.Source.Filename) == "" {
			return errors.New("source.filename is required for file captures")
		}
	default:
		return fmt.Errorf("unsupported source kind %q", b.Source.Kind)
	}
	if b.Privacy.RedactionStatus != "complete" && b.Privacy.RedactionStatus != "partial" {
		return errors.New("redaction status must be complete or partial")
	}
	if b.Privacy.RedactionStatus == "partial" && strings.TrimSpace(b.Privacy.RedactionReason) == "" {
		return errors.New("partial redaction requires a reason")
	}
	if b.Privacy.External && !b.Privacy.Ephemeral {
		return errors.New("external captures must be marked ephemeral")
	}
	for _, name := range lensOrder {
		lens, ok := b.Lenses[name]
		if !ok {
			return fmt.Errorf("lens %s must be explicitly present, partial, or absent", name)
		}
		switch lens.Status {
		case LensPresent:
		case LensPartial, LensAbsent:
			if strings.TrimSpace(lens.Reason) == "" {
				return fmt.Errorf("lens %s requires a reason when %s", name, lens.Status)
			}
		default:
			return fmt.Errorf("lens %s has invalid status %q", name, lens.Status)
		}
	}
	artifactPaths := make(map[string]struct{}, len(b.Artifacts))
	for _, artifact := range b.Artifacts {
		clean := filepath.Clean(artifact.Path)
		if artifact.Kind == "" || strings.ContainsRune(artifact.Path, '\\') || clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("artifact path must be a safe relative path: %q", artifact.Path)
		}
		if artifact.SizeBytes < 0 {
			return fmt.Errorf("artifact %q has negative size", artifact.Path)
		}
		if limit, known := artifactLimits[artifact.Kind]; known && artifact.SizeBytes > limit {
			return fmt.Errorf("artifact %q exceeds the %d-byte %s limit", artifact.Path, limit, artifact.Kind)
		}
		if len(artifact.SHA256) != 64 {
			return fmt.Errorf("artifact %q must include a SHA-256 digest", artifact.Path)
		}
		if _, err := hex.DecodeString(artifact.SHA256); err != nil {
			return fmt.Errorf("artifact %q has invalid SHA-256 digest", artifact.Path)
		}
		key := filepath.Clean(filepath.FromSlash(artifact.Path))
		if _, duplicate := artifactPaths[key]; duplicate {
			return fmt.Errorf("artifact path %q is duplicated", artifact.Path)
		}
		artifactPaths[key] = struct{}{}
	}
	return nil
}

// WriteBundle writes the manifest and README using temp files followed by
// rename. The caller owns the private destination directory and its cleanup.
func WriteBundle(dir string, bundle Bundle) error {
	return WriteBundleWithArtifacts(dir, bundle, nil)
}

// WriteBundleWithArtifacts publishes payloads, manifest, and README as one
// private directory rename. Every declared artifact must match one payload.
func WriteBundleWithArtifacts(dir string, bundle Bundle, payloads []ArtifactPayload) error {
	if err := bundle.Validate(); err != nil {
		return err
	}
	declared := make(map[string]Artifact, len(bundle.Artifacts))
	for _, artifact := range bundle.Artifacts {
		declared[filepath.Clean(filepath.FromSlash(artifact.Path))] = artifact
	}
	if len(declared) != len(payloads) {
		return errors.New("every manifest artifact must have exactly one payload")
	}
	seen := make(map[string]struct{}, len(payloads))
	for _, payload := range payloads {
		key := filepath.Clean(filepath.FromSlash(payload.Path))
		artifact, ok := declared[key]
		if !ok {
			return fmt.Errorf("artifact payload %q is not declared in manifest", payload.Path)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("artifact payload %q is duplicated", payload.Path)
		}
		seen[key] = struct{}{}
		if int64(len(payload.Content)) != artifact.SizeBytes || Digest(payload.Content) != artifact.SHA256 {
			return fmt.Errorf("artifact payload %q does not match declared size and digest", payload.Path)
		}
	}
	dir = filepath.Clean(dir)
	if dir == "." || dir == string(filepath.Separator) {
		return errors.New("bundle destination must name a dedicated directory")
	}
	if _, err := os.Lstat(dir); err == nil {
		return fmt.Errorf("bundle destination already exists: %s", dir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect bundle destination: %w", err)
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create bundle parent directory: %w", err)
	}
	tmpDir, err := os.MkdirTemp(parent, ".ux-capture-bundle-*")
	if err != nil {
		return fmt.Errorf("create temporary bundle directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := os.Chmod(tmpDir, 0o700); err != nil {
		return fmt.Errorf("set private bundle directory mode: %w", err)
	}
	for _, payload := range payloads {
		target := filepath.Join(tmpDir, filepath.FromSlash(payload.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("create artifact directory for %q: %w", payload.Path, err)
		}
		if err := os.WriteFile(target, payload.Content, 0o600); err != nil {
			return fmt.Errorf("write artifact %q: %w", payload.Path, err)
		}
	}
	manifest, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), append(manifest, '\n'), 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(bundle.Readme()), 0o600); err != nil {
		return fmt.Errorf("write README: %w", err)
	}
	if err := os.Rename(tmpDir, dir); err != nil {
		return fmt.Errorf("publish bundle directory: %w", err)
	}
	return nil
}

func (b Bundle) Readme() string {
	var out strings.Builder
	fmt.Fprintf(&out, "# UX capture %s\n\n", b.ID)
	fmt.Fprintf(&out, "- Schema: `%s`\n- Captured: `%s`\n- Session: `%s`\n", b.SchemaVersion, b.CapturedAt, b.Session.ID)
	fmt.Fprintf(&out, "- Source: %s\n- Viewport: %d × %d", b.Source.Kind, b.Viewport.Width, b.Viewport.Height)
	if b.Theme != "" {
		fmt.Fprintf(&out, " (%s)", b.Theme)
	}
	out.WriteString("\n\n## Fidelity\n\n| Lens | Status | Reason |\n|---|---|---|\n")
	for _, name := range lensOrder {
		lens := b.Lenses[name]
		fmt.Fprintf(&out, "| %s | %s | %s |\n", name, lens.Status, emptyDash(lens.Reason))
	}
	out.WriteString("\n## Artifacts\n\n")
	if len(b.Artifacts) == 0 {
		out.WriteString("No files were emitted.\n")
	} else {
		artifacts := append([]Artifact(nil), b.Artifacts...)
		sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
		for _, artifact := range artifacts {
			fmt.Fprintf(&out, "- %s: `%s` — %d bytes\n", emptyDash(artifact.Kind), strings.ReplaceAll(artifact.Path, "`", "\\`"), artifact.SizeBytes)
		}
	}
	out.WriteString("\n## Privacy\n\n")
	fmt.Fprintf(&out, "Redaction status: **%s**.\n", b.Privacy.RedactionStatus)
	if b.Privacy.RedactionReason != "" {
		fmt.Fprintf(&out, "Redaction note: %s\n", b.Privacy.RedactionReason)
	}
	if b.Privacy.External {
		out.WriteString("This capture came from a public external page. Review it locally and remove the bundle after use; do not commit it.\n")
	}
	out.WriteString("\n## Next steps\n\n")
	if len(b.NextCommands) == 0 {
		out.WriteString("Inspect `manifest.json`, then choose the next action from the available lenses and their stated limits.\n")
	} else {
		for _, command := range b.NextCommands {
			fmt.Fprintf(&out, "- `%s`\n", command)
		}
	}
	return out.String()
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return strings.ReplaceAll(value, "|", "\\|")
}

// Digest returns the content-addressed identity used by the manifest for a
// stored artifact.
func Digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
