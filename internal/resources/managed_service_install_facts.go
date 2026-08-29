package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/binaryfetch"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

const (
	managedServiceInstallFactsParameterA = 12
)

// installFactsFile is the sidecar written beside a staged artifact. It records
// the host facts the artifact was chosen under.
//
// Without it, "the host changed" and "the bytes are corrupt" are the same
// observation: both surface as a checksum mismatch, and the operator is told
// neither the cause nor the cure. That is exactly what turned reranker into an
// unavailable resource with no remediation and took search-hub's ranking leg
// down with it.
const installFactsFile = ".vrooli-install-facts.json"

// InstallFacts is the recorded provenance of one staged artifact.
type InstallFacts struct {
	Resource   string            `json:"resource"`
	RecordedAt time.Time         `json:"recorded_at"`
	Facts      map[string]string `json:"facts"`
	// ArtifactSHA256 is the digest of the target that was selected. Comparing
	// it against the digest the resolver selects today is what makes fact drift
	// computable.
	ArtifactSHA256 string `json:"artifact_sha256"`
	Layout         string `json:"layout,omitempty"`
	Version        string `json:"version,omitempty"`
}

// ErrFactDrift is returned when the host facts have changed enough that the
// resolver now selects a different artifact than the one on disk.
var ErrFactDrift = errors.New("needs_reacquire")

// FactDriftError names both fact sets, both targets, and the command that
// repairs it.
type FactDriftError struct {
	Resource       string
	InstalledFacts map[string]string
	CurrentFacts   map[string]string
	InstalledSHA   string
	ResolvedSHA    string
}

func (e *FactDriftError) Error() string {
	changed := changedFacts(e.InstalledFacts, e.CurrentFacts)
	detail := "no fact difference was recorded"
	if len(changed) > 0 {
		detail = strings.Join(changed, "; ")
	}
	return fmt.Sprintf(
		"resource %s was installed under different host facts (%s), so the resolver now selects artifact %s instead of the staged %s; the bytes on disk are not corrupt, the host changed. Re-acquire with `vrooli resource install %s --reacquire`",
		e.Resource, detail, shortDigest(e.ResolvedSHA), shortDigest(e.InstalledSHA), e.Resource)
}

// Unwrap lets errors.Is(err, ErrFactDrift) succeed.
func (e *FactDriftError) Unwrap() error { return ErrFactDrift }

// Remediation is the exact command that repairs this state.
func (e *FactDriftError) Remediation() string {
	return "vrooli resource install " + e.Resource + " --reacquire"
}

// changedFacts renders the difference between two fact sets in a stable order,
// naming only what actually moved.
func changedFacts(installed, current map[string]string) []string {
	keys := map[string]struct{}{}
	for key := range installed {
		keys[key] = struct{}{}
	}
	for key := range current {
		keys[key] = struct{}{}
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	slices.Sort(ordered)

	var changes []string
	for _, key := range ordered {
		before, hadBefore := installed[key]
		after, hasAfter := current[key]
		switch {
		case hadBefore && hasAfter && before != after:
			changes = append(changes, fmt.Sprintf("%s was %q and is now %q", key, before, after))
		case hadBefore && !hasAfter:
			changes = append(changes, fmt.Sprintf("%s was %q and is now absent", key, before))
		case !hadBefore && hasAfter:
			changes = append(changes, fmt.Sprintf("%s was absent and is now %q", key, after))
		}
	}
	return changes
}

func shortDigest(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return "(none)"
	}
	if len(digest) <= managedServiceInstallFactsParameterA {
		return digest
	}
	return digest[:12]
}

// installFactsPath is the sidecar location for a staged artifact. It sits
// beside the artifact so removing the artifact directory removes the record
// with it, and a stale record can never outlive the bytes it describes.
func installFactsPath(artifactPath string) string {
	return filepath.Join(filepath.Dir(artifactPath), installFactsFile)
}

// writeInstallFacts records the facts an artifact was staged under. A failure
// to write is not fatal: the artifact is already staged and correct, and the
// worst outcome is that a later drift reports as a checksum mismatch, which is
// the behaviour that existed before this record.
func writeInstallFacts(artifactPath, resource string, facts binaryfetch.Facts, target binaryfetch.AcquisitionTarget, artifact resourcedeployment.ServiceArtifact, now time.Time) error {
	record := InstallFacts{
		Resource:       resource,
		RecordedAt:     now.UTC(),
		Facts:          map[string]string(facts),
		ArtifactSHA256: strings.TrimSpace(target.ArtifactSHA256),
		Layout:         strings.TrimSpace(target.Layout),
		Version:        strings.TrimSpace(artifact.Version),
	}
	if record.ArtifactSHA256 == "" {
		record.ArtifactSHA256 = strings.TrimSpace(artifact.SHA256)
	}
	if record.Layout == "" {
		record.Layout = strings.TrimSpace(artifact.Layout)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	path := installFactsPath(artifactPath)
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermDir); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), tuning.PermFile)
}

// readInstallFacts reads the sidecar. ok is false with a nil error when no
// record exists, which is the state of every artifact staged before this
// record existed.
func readInstallFacts(artifactPath string) (InstallFacts, bool, error) {
	data, err := os.ReadFile(installFactsPath(artifactPath))
	if err != nil {
		if os.IsNotExist(err) {
			return InstallFacts{}, false, nil
		}
		return InstallFacts{}, false, err
	}
	var record InstallFacts
	if err := json.Unmarshal(data, &record); err != nil {
		return InstallFacts{}, false, fmt.Errorf("parse install facts for %s: %w", artifactPath, err)
	}
	return record, true, nil
}

// checkFactDrift compares the target the resolver selects now against the one
// recorded at install. It returns nil when there is no record to compare
// against: absence of evidence is not drift.
func checkFactDrift(artifactPath, resource string, current binaryfetch.Facts, resolved binaryfetch.AcquisitionTarget) error {
	record, ok, err := readInstallFacts(artifactPath)
	if err != nil || !ok {
		return err
	}
	resolvedSHA := strings.TrimSpace(resolved.ArtifactSHA256)
	if resolvedSHA == "" || record.ArtifactSHA256 == "" || resolvedSHA == record.ArtifactSHA256 {
		return nil
	}
	return &FactDriftError{
		Resource:       resource,
		InstalledFacts: record.Facts,
		CurrentFacts:   map[string]string(current),
		InstalledSHA:   record.ArtifactSHA256,
		ResolvedSHA:    resolvedSHA,
	}
}

// StatusCodeNeedsReacquire marks an artifact whose host facts moved. It is
// deliberately distinct from an unavailable artifact: the bytes are fine, the
// host changed, and there is a command that fixes it.
const StatusCodeNeedsReacquire = resourcecontrol.StatusCodeNeedsReacquire

// DiscardStagedArtifact removes the staged artifact and its install-facts
// record for one resource, so the next install re-resolves the target under the
// host's current facts and downloads it again.
//
// It refuses to delete anything outside the per-user artifact store: the
// staged-artifact path is derived from the manifest, and a manifest that points
// at the source checkout is not something an install command may remove.
func (c *Controller) DiscardStagedArtifact(name string, progress io.Writer) error {
	manifest, err := c.ResourceManifest(name)
	if err != nil {
		return err
	}
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return fmt.Errorf("resource %s does not acquire an artifact, so there is nothing to re-acquire", name)
	}
	path, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		return err
	}
	root, err := managedServiceArtifactStoreRoot(c.Home)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(root)+string(filepath.Separator)) {
		return fmt.Errorf("refusing to discard %s: it is not inside the per-user artifact store %s", path, root)
	}
	if progress != nil {
		fmt.Fprintf(progress, "discarding staged artifact for %s: %s\n", name, path)
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("discard staged %s artifact: %w", name, err)
	}
	if err := os.Remove(installFactsPath(path)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("discard %s install facts: %w", name, err)
	}
	return nil
}
