package ensure

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const modelProvenanceFilename = "ollama-install-provenance.json"

// modelInstallProvenance records a verified install's origin. It is evidence
// for inventory and later restoration checks; it does not authorize retention
// or deletion.
type modelInstallProvenance struct {
	Name           string    `json:"name"`
	Digest         string    `json:"digest"`
	Registry       string    `json:"registry"`
	Repository     string    `json:"repository"`
	Tag            string    `json:"tag"`
	ManifestDigest string    `json:"manifest_digest"`
	VerifiedAt     time.Time `json:"verified_at"`
}

type modelProvenanceDocument struct {
	SchemaVersion int                      `json:"schema_version"`
	Models        []modelInstallProvenance `json:"models"`
}

// recordModelInstallProvenance writes a durable receipt for the exact
// manifest and installed tag verified by ensure. The managed Ollama lifecycle
// lock serializes ensure writers; atomic rename protects readers from partial
// JSON if the process exits during the update.
func recordModelInstallProvenance(modelRoot string, pull *PinnedPull, model ModelInfo, verifiedAt time.Time) error {
	if pull == nil || pull.repository == "" || pull.tag == "" || pull.digest == "" {
		return errors.New("pinned pull is unavailable for provenance receipt")
	}
	if strings.TrimSpace(model.Name) == "" || model.Size <= 0 || model.Digest == "" || model.Digest != pull.ManifestDigest() {
		return errors.New("installed model identity does not match the verified pinned manifest")
	}
	if verifiedAt.IsZero() {
		return errors.New("provenance verification time is unavailable")
	}
	ref := strings.TrimPrefix(pull.repository, "library/") + ":" + pull.tag
	if model.Name != ref && model.Name != withLatestTag(ref) {
		return fmt.Errorf("installed model name %q does not match pinned repository tag %q", model.Name, ref)
	}

	resourceDataRoot := filepath.Dir(modelRoot)
	if !filepath.IsAbs(resourceDataRoot) {
		return errors.New("model provenance root is not absolute")
	}
	rootInfo, err := os.Lstat(resourceDataRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("model provenance root is unavailable, linked, or not a directory")
	}
	path := filepath.Join(resourceDataRoot, modelProvenanceFilename)
	doc := modelProvenanceDocument{SchemaVersion: 1, Models: []modelInstallProvenance{}}
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("model provenance path is not a regular file")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read model provenance: %w", err)
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("decode model provenance: %w", err)
		}
		if doc.SchemaVersion != 1 {
			return fmt.Errorf("unsupported model provenance schema version %d", doc.SchemaVersion)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect model provenance: %w", err)
	}

	entry := modelInstallProvenance{
		Name: model.Name, Digest: model.Digest, Registry: RegistryHost,
		Repository: pull.repository, Tag: pull.tag,
		ManifestDigest: pull.ManifestDigest(), VerifiedAt: verifiedAt.UTC(),
	}
	updated := false
	for i := range doc.Models {
		if doc.Models[i].Name == model.Name {
			doc.Models[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		doc.Models = append(doc.Models, entry)
	}
	sort.Slice(doc.Models, func(i, j int) bool { return doc.Models[i].Name < doc.Models[j].Name })

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode model provenance: %w", err)
	}
	tmp, err := os.CreateTemp(resourceDataRoot, ".ollama-provenance-*.tmp")
	if err != nil {
		return fmt.Errorf("stage model provenance: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect staged model provenance: %w", err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write model provenance: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync model provenance: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close model provenance: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish model provenance: %w", err)
	}
	if err := syncProvenanceDirectory(resourceDataRoot); err != nil {
		return fmt.Errorf("sync model provenance directory: %w", err)
	}
	return nil
}

// syncProvenanceDirectory makes the rename durable on filesystems that honor
// directory fsync. Provenance is recorded before the pull lease is completed,
// so unsupported durability semantics fail the acquisition closed.
func syncProvenanceDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("provenance durability path is not a directory")
	}
	return dir.Sync()
}
