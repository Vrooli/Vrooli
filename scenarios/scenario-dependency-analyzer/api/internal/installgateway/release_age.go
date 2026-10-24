package installgateway

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ensureReleaseAge installs the repository cooldown at the governed mutation
// boundary, retaining stricter policies and every unrelated workspace setting.
func ensureReleaseAge(root string) error {
	path := filepath.Join(root, "pnpm-workspace.yaml")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		raw = []byte("packages:\n  - .\n")
	} else if err != nil {
		return fmt.Errorf("read pnpm workspace policy: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse pnpm workspace policy: %w", err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("pnpm workspace policy must be a mapping")
	}
	mapping := doc.Content[0]
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != "minimumReleaseAge" {
			continue
		}
		var minutes int
		if err := mapping.Content[i+1].Decode(&minutes); err != nil {
			return fmt.Errorf("invalid minimumReleaseAge: %w", err)
		}
		if minutes >= 10080 {
			return nil
		}
		mapping.Content[i+1].SetString("10080")
		mapping.Content[i+1].Tag = "!!int"
		return writeReleaseAge(path, &doc)
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "minimumReleaseAge"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "10080"})
	return writeReleaseAge(path, &doc)
}

func writeReleaseAge(path string, doc *yaml.Node) error {
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
