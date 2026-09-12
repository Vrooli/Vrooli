package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactRef names an artifact logically. It deliberately carries no path:
// the resolver remains the only authority for class-root placement.
type ArtifactRef struct {
	Owner    string
	Domain   string
	Class    Class
	Segments []string
}

// ArtifactPath resolves a logical artifact reference to an absolute path.
// Callers name an owner, domain, and logical segments; this method delegates
// every location decision to Resolver.Path.
func (r *Resolver) ArtifactPath(opts Options, ref ArtifactRef) (string, error) {
	owner := cleanScenarioID(ref.Owner)
	if !isValidScenarioID(owner) {
		return "", &Error{Kind: ErrInvalidInput, Message: "invalid artifact owner", Details: ref.Owner}
	}
	domain, err := cleanDomain(ref.Domain)
	if err != nil {
		return "", err
	}

	scenarioID := cleanScenarioID(opts.ScenarioID)
	if scenarioID == "" {
		scenarioID = owner
	} else if scenarioID != owner && !strings.HasPrefix(scenarioID, owner+"_") {
		return "", &Error{
			Kind:    ErrInvalidInput,
			Message: "artifact owner does not match scenario namespace",
			Details: fmt.Sprintf("owner=%s scenario=%s", owner, scenarioID),
		}
	}
	opts.ScenarioID = scenarioID

	parts := make([]string, 0, len(ref.Segments)+1)
	parts = append(parts, domain)
	for i, segment := range ref.Segments {
		if err := validateArtifactSegment(segment); err != nil {
			return "", &Error{
				Kind:    ErrInvalidInput,
				Message: "invalid artifact segment",
				Details: fmt.Sprintf("segment %d: %s", i, err),
			}
		}
		parts = append(parts, segment)
	}

	return r.Path(opts, ref.Class, filepath.Join(parts...))
}

// EnsureArtifactDir resolves and creates the directory named by ref.
func (r *Resolver) EnsureArtifactDir(opts Options, ref ArtifactRef, perm os.FileMode) (string, error) {
	path, err := r.ArtifactPath(opts, ref)
	if err != nil {
		return "", err
	}
	if err := EnsureDirectory(path, perm); err != nil {
		return "", err
	}
	return path, nil
}

func validateArtifactSegment(segment string) error {
	if segment == "" {
		return fmt.Errorf("must not be empty")
	}
	if strings.HasPrefix(segment, ".") {
		return fmt.Errorf("must not start with a dot")
	}
	if strings.Contains(segment, "..") {
		return fmt.Errorf("must not contain '..'")
	}
	if strings.ContainsAny(segment, `/\`) {
		return fmt.Errorf("must not contain a path separator")
	}
	return nil
}
