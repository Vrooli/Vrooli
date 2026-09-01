package deployability

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	manifestvalidationNull = "null"
)

// ManifestDeclaration is the loader-neutral shape of one capability-bearing
// manifest. Path is the file the declaration was read from so a rejection can
// name it; the resolver package never globs for manifests itself.
type ManifestDeclaration struct {
	Path       string
	Name       string
	Capability string
	Role       string
	// Platforms is keyed by the raw OS token the manifest authored, so a
	// rejection can quote what was written rather than a normalized guess.
	Platforms map[string]PlatformDeclaration
	Pairs     []PlatformPairDeclaration
}

// ValidatePlatformPairEvidence enforces the stronger evidence rule for a
// cross-machine claim: a supported pair must point at an artifact that exists
// in the repository. Free-form prose is useful for build-verified cells but
// cannot qualify a real hardware pairing.
func ValidatePlatformPairEvidence(repoRoot, location, name string, pair PlatformPairDeclaration) error {
	status, err := ParsePlatformStatus(pair.Status)
	if err != nil {
		return fmt.Errorf("%s: capability manifest %q declares pair %s/%s %w", location, name, pair.Console, pair.Node, err)
	}
	if status != StatusSupported {
		return nil
	}
	evidence := strings.TrimSpace(pair.EvidenceRaw)
	if evidence != "" && evidence != manifestvalidationNull && evidence != `""` {
		var authoredPath string
		if json.Unmarshal([]byte(evidence), &authoredPath) == nil {
			evidence = strings.TrimSpace(authoredPath)
		}
	}
	if pair.Evidence != nil && strings.TrimSpace(pair.Evidence.ArtifactURI) != "" {
		evidence = strings.TrimSpace(pair.Evidence.ArtifactURI)
	}
	if evidence == "" || evidence == manifestvalidationNull || evidence == `""` {
		return fmt.Errorf("%s: capability manifest %q declares pair %s/%s as supported without evidence", location, name, pair.Console, pair.Node)
	}
	if filepath.IsAbs(evidence) {
		return fmt.Errorf("%s: capability manifest %q declares pair evidence with absolute path %q", location, name, evidence)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, evidence)); err != nil {
		return fmt.Errorf("%s: capability manifest %q declares supported pair %s/%s with missing evidence %q: %w", location, name, pair.Console, pair.Node, evidence, err)
	}
	return nil
}

// ValidateManifestDeclarations rejects any manifest that names a capability
// outside the vocabulary, carries an unknown capability role, or authors a
// platform status outside the platform status vocabulary. Every rejection
// names the offending file and token.
func ValidateManifestDeclarations(declarations []ManifestDeclaration, vocabulary []string) error {
	allowed := make(map[string]struct{}, len(vocabulary))
	for _, capability := range vocabulary {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			return fmt.Errorf("capability vocabulary contains an empty entry")
		}
		if _, duplicate := allowed[capability]; duplicate {
			return fmt.Errorf("capability vocabulary duplicates %q", capability)
		}
		allowed[capability] = struct{}{}
	}
	for _, item := range declarations {
		location := declarationLocation(item)
		capability := strings.TrimSpace(item.Capability)
		if capability == "" {
			return fmt.Errorf("%s: capability manifest %q has no capability", location, item.Name)
		}
		if _, ok := allowed[capability]; !ok {
			return fmt.Errorf("%s: capability manifest %q declares unknown capability %q", location, item.Name, capability)
		}
		switch strings.TrimSpace(item.Role) {
		case "primary", "peer", "control":
		default:
			return fmt.Errorf("%s: capability manifest %q has invalid capability_role %q", location, item.Name, item.Role)
		}
		for _, osName := range slices.Sorted(maps.Keys(item.Platforms)) {
			declaration := item.Platforms[osName]
			status, err := ParsePlatformStatus(declaration.Status)
			if err != nil {
				return fmt.Errorf("%s: capability manifest %q declares %s %w", location, item.Name, osName, err)
			}
			if status.Qualification().Rank() > QualificationBuildVerified.Rank() && !declaration.Evidence.Complete() {
				return fmt.Errorf("%s: capability manifest %q declares %s at status %s (qualification %s) without complete structured evidence", location, item.Name, osName, status, status.Qualification())
			}
			if err := validatePlatformDeclaration(location, item.Name, osName, status, declaration); err != nil {
				return err
			}
		}
		for _, pair := range item.Pairs {
			status, err := ParsePlatformStatus(pair.Status)
			if err != nil {
				return fmt.Errorf("%s: capability manifest %q declares pair %s/%s %w", location, item.Name, pair.Console, pair.Node, err)
			}
			if status != StatusUnsupported && status != StatusNotApplicable && strings.TrimSpace(pair.Mechanism) == "" {
				return fmt.Errorf("%s: capability manifest %q declares pair %s/%s at status %s without mechanism", location, item.Name, pair.Console, pair.Node, status)
			}
		}
	}
	return nil
}

func validatePlatformDeclaration(location, name, osName string, status PlatformStatus, declaration PlatformDeclaration) error {
	evidence := strings.TrimSpace(declaration.EvidenceRaw)
	switch status {
	case StatusNotApplicable:
		if strings.TrimSpace(declaration.Mechanism) != "" {
			return fmt.Errorf("%s: capability manifest %q declares %s as not_applicable with mechanism %q; closed cells cannot name an implementation mechanism", location, name, osName, declaration.Mechanism)
		}
		if strings.TrimSpace(declaration.ReviewBy) == "" {
			return fmt.Errorf("%s: capability manifest %q declares %s as not_applicable without review_by", location, name, osName)
		}
		if _, err := time.Parse("2006-01-02", declaration.ReviewBy); err != nil {
			return fmt.Errorf("%s: capability manifest %q declares %s with invalid review_by %q", location, name, osName, declaration.ReviewBy)
		}
		if evidence == "" || evidence == manifestvalidationNull || evidence == `""` {
			return fmt.Errorf("%s: capability manifest %q declares %s as not_applicable without platform evidence", location, name, osName)
		}
	case StatusNotImplemented:
		if strings.TrimSpace(declaration.Mechanism) == "" {
			return fmt.Errorf("%s: capability manifest %q declares %s as not_implemented without mechanism", location, name, osName)
		}
		if strings.EqualFold(strings.TrimSpace(declaration.Mechanism), "handler") {
			return fmt.Errorf("%s: capability manifest %q declares %s as not_implemented with generic handler mechanism", location, name, osName)
		}
		if strings.TrimSpace(declaration.Since) == "" {
			return fmt.Errorf("%s: capability manifest %q declares %s as not_implemented without since", location, name, osName)
		}
		if _, err := time.Parse("2006-01-02", declaration.Since); err != nil {
			return fmt.Errorf("%s: capability manifest %q declares %s with invalid since %q", location, name, osName, declaration.Since)
		}
		if evidence == "" || evidence == manifestvalidationNull || evidence == `""` {
			return fmt.Errorf("%s: capability manifest %q declares %s as not_implemented without gap evidence", location, name, osName)
		}
	}
	lower := strings.ToLower(evidence)
	for _, phrase := range []string{"no implementation for the", "not applicable on this host os", "no implementation", "not implemented by vrooli"} {
		if strings.Contains(lower, phrase) {
			return fmt.Errorf("%s: capability manifest %q declares %s with circular evidence phrase %q", location, name, osName, phrase)
		}
	}
	return nil
}

func declarationLocation(item ManifestDeclaration) string {
	if path := strings.TrimSpace(item.Path); path != "" {
		return path
	}
	return "<unknown manifest path>"
}
