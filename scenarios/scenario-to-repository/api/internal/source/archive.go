package source

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Assemble(root, output string, closure Closure, recipe Recipe) (Artifact, error) {
	if !recipe.Deterministic || recipe.ArchiveFormat != "tar_gzip" {
		return Artifact{}, fmt.Errorf("only deterministic tar_gzip recipes are supported")
	}
	if len(closure.Unresolved) > 0 {
		return Artifact{}, fmt.Errorf("closure has unresolved obligations: %s", strings.Join(closure.Unresolved, ", "))
	}
	policy, err := CheckPublishability(root, closure)
	if err != nil {
		return Artifact{}, err
	}
	if !policy.Allowed {
		return Artifact{}, fmt.Errorf("publishability policy refused export: %s", strings.Join(policy.Violations, "; "))
	}
	entries := make([]ManifestEntry, 0, len(closure.Files))
	for _, file := range closure.Files {
		entries = append(entries, ManifestEntry{Path: file.ExportPath, SHA256: file.SHA256, Mode: file.Mode, SourcePath: file.SourcePath, SizeBytes: file.SizeBytes})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	manifestBytes, err := json.Marshal(struct {
		SchemaVersion int             `json:"schemaVersion"`
		Files         []ManifestEntry `json:"files"`
	}{1, entries})
	if err != nil {
		return Artifact{}, err
	}
	mh := sha256.Sum256(manifestBytes)
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return Artifact{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(output), ".source-export-*")
	if err != nil {
		return Artifact{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	gw := gzip.NewWriter(tmp)
	gw.Header.ModTime = time.Unix(0, 0)
	gw.Header.OS = 255
	tw := tar.NewWriter(gw)
	for _, entry := range entries {
		if err := validateArchivePath(entry.Path); err != nil {
			return Artifact{}, err
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.SourcePath)))
		if err != nil {
			return Artifact{}, err
		}
		h := &tar.Header{Name: entry.Path, Mode: int64(entry.Mode), Size: int64(len(data)), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(h); err != nil {
			return Artifact{}, err
		}
		if _, err := tw.Write(data); err != nil {
			return Artifact{}, err
		}
	}
	manifestHeader := &tar.Header{Name: "SOURCE-MANIFEST.json", Mode: 0o644, Size: int64(len(manifestBytes)), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(manifestHeader); err != nil {
		return Artifact{}, err
	}
	if _, err := tw.Write(manifestBytes); err != nil {
		return Artifact{}, err
	}
	if err := tw.Close(); err != nil {
		return Artifact{}, err
	}
	if err := gw.Close(); err != nil {
		return Artifact{}, err
	}
	if err := tmp.Close(); err != nil {
		return Artifact{}, err
	}
	data, err := os.ReadFile(tmpName)
	if err != nil {
		return Artifact{}, err
	}
	ah := sha256.Sum256(data)
	if err := os.Rename(tmpName, output); err != nil {
		return Artifact{}, err
	}
	artifactID := "artifact-" + hex.EncodeToString(ah[:8])
	return Artifact{ArtifactID: artifactID, SourceDigest: closure.SourceDigest, RecipeDigest: recipeDigest(recipe), ClosureDigest: closure.ClosureDigest, Files: entries, ManifestDigest: "sha256:" + hex.EncodeToString(mh[:]), ArchiveDigest: "sha256:" + hex.EncodeToString(ah[:]), ArchivePath: output, Status: "assembled"}, nil
}

func recipeDigest(recipe Recipe) string {
	data, _ := json.Marshal(recipe)
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func validateArchivePath(path string) error {
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe archive path %q", path)
	}
	return nil
}

func VerifyArchive(artifact Artifact) (Verification, error) {
	v := Verification{VerificationID: "verification-" + artifact.ArtifactID, ArtifactID: artifact.ArtifactID, VerifiedAt: time.Now().UTC()}
	data, err := os.ReadFile(artifact.ArchivePath)
	if err != nil {
		v.Status = "unavailable"
		v.Failures = []string{err.Error()}
		return v, nil
	}
	h := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(h[:])
	if digest != artifact.ArchiveDigest {
		v.Status = "failed"
		v.Failures = []string{"archive digest mismatch"}
		return v, nil
	}
	if err := validateArchiveBytes(data, artifact); err != nil {
		v.Status = "failed"
		v.Failures = []string{err.Error()}
		return v, nil
	}
	if artifact.ManifestDigest == "" {
		v.Status = "failed"
		v.Failures = []string{"manifest digest missing"}
		return v, nil
	}
	v.Status = "passed"
	v.GatesPassed = []string{"integrity", "manifest"}
	receipt, _ := json.Marshal(v)
	rh := sha256.Sum256(receipt)
	v.ReceiptDigest = "sha256:" + hex.EncodeToString(rh[:])
	return v, nil
}

func validateArchiveBytes(data []byte, artifact Artifact) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	seen := map[string]struct{}{}
	lower := map[string]string{}
	manifestBytes := []byte(nil)
	total := int64(0)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid tar archive: %w", err)
		}
		if err := validateArchivePath(header.Name); err != nil {
			return err
		}
		if _, ok := seen[header.Name]; ok {
			return fmt.Errorf("duplicate archive entry: %s", header.Name)
		}
		seen[header.Name] = struct{}{}
		folded := strings.ToLower(header.Name)
		if prior, ok := lower[folded]; ok && prior != header.Name {
			return fmt.Errorf("case-colliding archive paths: %s and %s", prior, header.Name)
		}
		lower[folded] = header.Name
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("unsupported archive entry type: %s", header.Name)
		}
		total += header.Size
		if total > 512<<20 || header.Size > 512<<20 {
			return fmt.Errorf("archive uncompressed size exceeds scan budget")
		}
		content, err := io.ReadAll(io.LimitReader(tr, 512<<20+1))
		if err != nil {
			return err
		}
		if int64(len(content)) != header.Size {
			return fmt.Errorf("archive entry size mismatch: %s", header.Name)
		}
		if header.Name == "SOURCE-MANIFEST.json" {
			manifestBytes = content
		}
	}
	if len(manifestBytes) == 0 {
		return fmt.Errorf("source manifest is missing")
	}
	manifestDigest := sha256.Sum256(manifestBytes)
	if got := "sha256:" + hex.EncodeToString(manifestDigest[:]); got != artifact.ManifestDigest {
		return fmt.Errorf("manifest digest mismatch")
	}
	var manifest struct {
		SchemaVersion int             `json:"schemaVersion"`
		Files         []ManifestEntry `json:"files"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("invalid source manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Files) != len(artifact.Files) {
		return fmt.Errorf("source manifest does not match artifact")
	}
	for i := range manifest.Files {
		if manifest.Files[i] != artifact.Files[i] {
			return fmt.Errorf("source manifest entry mismatch: %s", manifest.Files[i].Path)
		}
	}
	return nil
}
