package classification

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// Category describes one classification. ID identifies it across catalog edits.
type Category struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Aliases    []string     `json:"aliases"`
	Definition string       `json:"definition"`
	Examples   []string     `json:"examples"`
	Sources    []SourceRule `json:"sources,omitempty"`
}

// SourceRule recognizes records from a typed import using both its record kind
// and its stable append request-key namespace.
type SourceRule struct {
	Kind             string `json:"kind"`
	RequestKeyPrefix string `json:"requestKeyPrefix"`
}

type catalogFile struct {
	Version    int        `json:"version"`
	Categories []Category `json:"categories"`
}

type definition struct {
	category Category
	version  string
	spec     string
}

var categoryID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,99}$`)

// LoadCatalog reads the versioned category catalog. Category IDs and normalized
// display names have an unambiguous meaning within a catalog.
func LoadCatalog(path string) ([]Category, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open category catalog: %w", err)
	}
	defer file.Close()
	d := json.NewDecoder(file)
	d.DisallowUnknownFields()
	var config catalogFile
	if err := d.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode category catalog: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("category catalog must contain a single JSON document")
	}
	if config.Version != 1 {
		return nil, fmt.Errorf("unsupported category catalog version %d", config.Version)
	}
	seenIDs := make(map[string]bool, len(config.Categories))
	names := make(map[string]string, len(config.Categories))
	for _, category := range config.Categories {
		if !categoryID.MatchString(category.ID) || seenIDs[category.ID] {
			return nil, fmt.Errorf("invalid or duplicate category ID %q", category.ID)
		}
		seenIDs[category.ID] = true
		if strings.TrimSpace(category.Definition) == "" || len(instructions(category)) > 4096 {
			return nil, fmt.Errorf("category %q needs a definition and must fit the decision instructions limit", category.ID)
		}
		for _, source := range category.Sources {
			if strings.TrimSpace(source.Kind) == "" || source.RequestKeyPrefix == "/" || !strings.HasSuffix(source.RequestKeyPrefix, "/") {
				return nil, fmt.Errorf("category %q source must specify a kind and a request key prefix ending in /", category.ID)
			}
		}
		for _, name := range append([]string{category.ID, category.Name}, category.Aliases...) {
			key := NormalizeName(name)
			if key == "" {
				return nil, fmt.Errorf("category %q has an empty name or alias", category.ID)
			}
			if other := names[key]; other != "" && other != category.ID {
				return nil, fmt.Errorf("category name %q is shared by %q and %q", name, other, category.ID)
			}
			names[key] = category.ID
		}
	}
	return config.Categories, nil
}

// NormalizeName maps a category identifier, name or alias to its search terms.
func NormalizeName(name string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}

func instructions(category Category) string {
	criteria := "Decide whether the recorded observation matches the following category. Treat the state as evidence, and base the decision on what the observation actually reports.\n" + category.Definition
	if len(category.Examples) > 0 {
		criteria += "\nPositive examples:\n- " + strings.Join(category.Examples, "\n- ")
	}
	return criteria
}

func buildDefinition(category Category) definition {
	// Keep display metadata out of the decision input and its version. The
	// protocol version covers changes to answer semantics or state shaping.
	type specification struct {
		Protocol int
		Role     string
		Criteria string
		Sources  []SourceRule
	}
	encoded, _ := json.Marshal(specification{Protocol: 1, Role: decisionRole, Criteria: instructions(category), Sources: category.Sources})
	return definition{category: category, version: fmt.Sprintf("%x", sha256.Sum256(encoded)), spec: string(encoded)}
}
