package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ImportResult struct {
	Created, Revised int
	Errors           []string
}

// ImportCanon reads but never writes the marketing catalogue. A hash of the
// source path plus normalized JSON is the idempotency key: whitespace and map
// ordering do not create duplicates, and no positional watermark is needed.
func (s *Studio) ImportCanon(root, actorID string, now time.Time) ImportResult {
	result := ImportResult{}
	for _, kind := range []IdentityKind{Character, Scene, Product} {
		dir := filepath.Join(root, string(kind)+"s")
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				result.Errors = append(result.Errors, err.Error())
			}
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasPrefix(entry.Name(), "_") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
				continue
			}
			identity, hash, err := identityFromCanon(kind, path, raw)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
				continue
			}
			if s.ImportHashes[path] == hash {
				continue
			}
			if existing, ok := s.Identities[identity.ID]; ok {
				identity.ID = existing.ID
				_, err = s.Revise(identity, actorID, Agent, now)
				if err == nil {
					result.Revised++
				}
			} else {
				err = s.Author(identity, actorID, Agent, now)
				if err == nil {
					result.Created++
				}
			}
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
				continue
			}
			s.ImportHashes[path] = hash
		}
	}
	sort.Strings(result.Errors)
	return result
}

func identityFromCanon(kind IdentityKind, path string, raw []byte) (Identity, string, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return Identity{}, "", err
	}
	canonical, err := json.Marshal(data)
	if err != nil {
		return Identity{}, "", err
	}
	hashBytes := sha256.Sum256(append([]byte(path+"\n"), canonical...))
	hash := hex.EncodeToString(hashBytes[:])
	slug, _ := data["slug"].(string)
	name, _ := data["display_name"].(string)
	if slug == "" || name == "" || strings.Contains(strings.ToUpper(slug+name), "REPLACE") {
		return Identity{}, "", fmt.Errorf("template placeholder is not an authored canon item")
	}
	traits := map[string]string{}
	switch kind {
	case Product:
		traits["form"] = text(data, "product_kind")
		traits["finish"] = nestedText(data, "brand_element_placement_rules", "palette_lock")
	case Scene:
		traits["environment"] = nestedText(data, "environment", "description")
		traits["lighting"] = nestedText(data, "lighting", "quality")
	case Character:
		traits["face"] = nestedText(data, "identity_block", "face_structure")
		traits["build"] = nestedText(data, "identity_block", "body_type")
	}
	return Identity{ID: slug, Name: name, Kind: kind, Traits: traits, CredentialClaims: ""}, hash, nil
}
func text(data map[string]any, key string) string { value, _ := data[key].(string); return value }
func nestedText(data map[string]any, key, child string) string {
	nested, _ := data[key].(map[string]any)
	return text(nested, child)
}
