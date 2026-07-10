package operatingmode

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrArtifactNotFound = errors.New("operating mode artifact not found")

type ArtifactSnapshot struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Content     string `json:"content,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
}

func (s *Store) ArtifactPath(initiativeName string, mode Mode, relPath string) (string, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return "", err
	}
	return s.ArtifactPathForDefinition(initiativeName, def, relPath)
}

func (s *Store) ArtifactPathForDefinition(scopeID string, def Definition, relPath string) (string, error) {
	clean, err := cleanModeRelativePath(def, relPath)
	if err != nil {
		return "", err
	}
	root, err := s.scopeDir(scopeID, def)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(clean)), nil
}

func (s *Store) WriteArtifact(initiativeName string, mode Mode, relPath string, content []byte) (string, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return "", err
	}
	return s.WriteArtifactForDefinition(initiativeName, def, relPath, content)
}

func (s *Store) WriteArtifactForDefinition(scopeID string, def Definition, relPath string, content []byte) (string, error) {
	path, err := s.ArtifactPathForDefinition(scopeID, def, relPath)
	if err != nil {
		return "", err
	}
	clean, err := cleanModeRelativePath(def, relPath)
	if err != nil {
		return "", err
	}
	if err := writeFileAtomic(path, content); err != nil {
		return "", fmt.Errorf("write artifact: %w", err)
	}
	return clean, nil
}

func (s *Store) ReadArtifact(initiativeName string, mode Mode, relPath string) (ArtifactSnapshot, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	return s.ReadArtifactForDefinition(initiativeName, def, relPath)
}

func (s *Store) ReadArtifactForDefinition(scopeID string, def Definition, relPath string) (ArtifactSnapshot, error) {
	clean, err := cleanModeRelativePath(def, relPath)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	root, err := s.scopeDir(scopeID, def)
	if err != nil {
		return ArtifactSnapshot{}, err
	}
	path := filepath.Join(root, filepath.FromSlash(clean))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ArtifactSnapshot{}, ErrArtifactNotFound
		}
		return ArtifactSnapshot{}, fmt.Errorf("read artifact: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return ArtifactSnapshot{}, fmt.Errorf("stat artifact: %w", err)
	}
	decl := artifactDeclaration(def, clean)
	snapshot := ArtifactSnapshot{
		Path:        clean,
		ContentType: decl.ContentType,
		Required:    decl.Required,
		Content:     string(data),
		SizeBytes:   info.Size(),
	}
	if !info.ModTime().IsZero() {
		snapshot.UpdatedAt = info.ModTime().UTC().Format(timeFormatRFC3339)
	}
	return snapshot, nil
}

func (s *Store) ListDeclaredArtifacts(initiativeName string, mode Mode) ([]ArtifactSnapshot, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return nil, err
	}
	return s.ListDeclaredArtifactsForDefinition(initiativeName, def)
}

func (s *Store) ListDeclaredArtifactsForDefinition(scopeID string, def Definition) ([]ArtifactSnapshot, error) {
	seen := map[string]ArtifactDefinition{}
	for _, phase := range def.PhaseGraph.Phases {
		for _, artifact := range phase.OutputArtifacts {
			if strings.TrimSpace(artifact.Path) == "" {
				continue
			}
			seen[filepath.ToSlash(filepath.Clean(artifact.Path))] = artifact
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sortStrings(paths)

	out := make([]ArtifactSnapshot, 0, len(paths))
	for _, path := range paths {
		snapshot, err := s.ReadArtifactForDefinition(scopeID, def, path)
		if errors.Is(err, ErrArtifactNotFound) {
			artifact := seen[path]
			out = append(out, ArtifactSnapshot{
				Path:        path,
				ContentType: artifact.ContentType,
				Required:    artifact.Required,
			})
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, snapshot)
	}
	return out, nil
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

func cleanModeRelativePath(def Definition, relPath string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(relPath)))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	if strings.HasPrefix(clean, "../") || clean == ".." || filepath.IsAbs(clean) {
		return "", fmt.Errorf("artifact path must be relative to initiative")
	}
	root := filepath.ToSlash(filepath.Clean(def.Artifact.Root))
	if root == "." || root == "" {
		return "", fmt.Errorf("mode %q has no artifact root", def.Mode)
	}
	if clean != root && !strings.HasPrefix(clean, root+"/") {
		return "", fmt.Errorf("artifact path %q is outside mode root %q", relPath, root)
	}
	return clean, nil
}

func artifactDeclaration(def Definition, path string) ArtifactDefinition {
	for _, phase := range def.PhaseGraph.Phases {
		for _, artifact := range phase.OutputArtifacts {
			if filepath.ToSlash(filepath.Clean(artifact.Path)) == path {
				return artifact
			}
		}
	}
	return ArtifactDefinition{Path: path}
}

func writeFileAtomic(path string, content []byte) error {
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(parentDir, "tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if rmErr := os.Remove(tmpName); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.Debug("operatingmode: remove temp artifact failed", "err", rmErr, "path", tmpName)
		}
	}()
	if _, err := tmp.Write(content); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			slog.Debug("operatingmode: close temp artifact failed", "err", closeErr)
		}
		return err
	}
	if err := tmp.Sync(); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			slog.Debug("operatingmode: close temp artifact failed", "err", closeErr)
		}
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func sortStrings(values []string) {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
}
