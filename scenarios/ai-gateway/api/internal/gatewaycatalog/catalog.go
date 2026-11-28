// Package gatewaycatalog owns the operator-facing profile vocabulary and the
// generated capability reference. The profile catalog is data; the TypeScript
// constant and the Markdown reference are projections of that data, kept in
// sync by cmd/gen-gateway-catalog and its -check mode.
package gatewaycatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// SchemaVersion is the current profile-catalog document version.
const SchemaVersion = 1

var (
	profileKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	profileI18nKey    = regexp.MustCompile(`^profiles\.[a-zA-Z][a-zA-Z0-9]*$`)
)

// Profile is one operator-selectable routing profile.
type Profile struct {
	Proto          string `json:"proto"`
	Key            string `json:"key"`
	LabelKey       string `json:"label_key"`
	DescriptionKey string `json:"description_key"`
	Order          int    `json:"order"`
}

// ProfileCatalog is the on-disk profile vocabulary document.
type ProfileCatalog struct {
	SchemaVersion int       `json:"schema_version"`
	Profiles      []Profile `json:"profiles"`
}

// LoadProfiles reads and validates the profile catalog at path.
func LoadProfiles(path string) (*ProfileCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var catalog ProfileCatalog
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("parse profile catalog %s: %w", path, err)
	}
	if err = catalog.Validate(); err != nil {
		return nil, fmt.Errorf("validate profile catalog %s: %w", path, err)
	}
	return &catalog, nil
}

// Sorted returns the catalog rows in declared order. Load already validates
// uniqueness; sorting here makes every renderer deterministic regardless of
// the order an editor happened to write the JSON in.
func (c *ProfileCatalog) Sorted() []Profile {
	out := append([]Profile(nil), c.Profiles...)
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

// Validate enforces the profile catalog's contract:
//
//   - every proto Profile value except PROFILE_UNSPECIFIED is declared exactly once
//   - no proto name outside the generated enum is declared
//   - keys, i18n key paths, and order values are unique
//   - order values are positive
func (c *ProfileCatalog) Validate() error {
	if c == nil {
		return fmt.Errorf("catalog is nil")
	}
	if c.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version = %d, want %d", c.SchemaVersion, SchemaVersion)
	}
	if len(c.Profiles) == 0 {
		return fmt.Errorf("profiles must not be empty")
	}
	state := newProfileValidationState()
	for _, row := range c.Profiles {
		if err := state.validateRow(row); err != nil {
			return err
		}
	}
	return state.validateComplete()
}

type profileValidationState struct {
	expected        map[string]struct{}
	seenProto       map[string]struct{}
	seenKey         map[string]struct{}
	seenLabel       map[string]struct{}
	seenDescription map[string]struct{}
	seenOrder       map[int]string
}

func newProfileValidationState() profileValidationState {
	state := profileValidationState{
		expected: map[string]struct{}{}, seenProto: map[string]struct{}{},
		seenKey: map[string]struct{}{}, seenLabel: map[string]struct{}{},
		seenDescription: map[string]struct{}{}, seenOrder: map[int]string{},
	}
	for _, name := range sharedv1.Profile_name {
		if name != sharedv1.Profile_PROFILE_UNSPECIFIED.String() {
			state.expected[name] = struct{}{}
		}
	}
	return state
}

func (s *profileValidationState) validateRow(row Profile) error {
	if err := s.validateProto(row); err != nil {
		return err
	}
	if err := s.validateKey(row); err != nil {
		return err
	}
	if err := s.validateI18nKey(row.LabelKey, "label_key", row.Key, s.seenLabel); err != nil {
		return err
	}
	if err := s.validateI18nKey(row.DescriptionKey, "description_key", row.Key, s.seenDescription); err != nil {
		return err
	}
	if row.Order <= 0 {
		return fmt.Errorf("profile %q: order must be positive", row.Key)
	}
	if other, duplicate := s.seenOrder[row.Order]; duplicate {
		return fmt.Errorf("profiles %q and %q share order %d", other, row.Key, row.Order)
	}
	s.seenOrder[row.Order] = row.Key
	return nil
}

func (s *profileValidationState) validateProto(row Profile) error {
	if _, ok := s.expected[row.Proto]; !ok {
		if row.Proto == sharedv1.Profile_PROFILE_UNSPECIFIED.String() {
			return fmt.Errorf("profile %q: PROFILE_UNSPECIFIED is not an operator-selectable profile", row.Key)
		}
		return fmt.Errorf("profile %q: unknown proto enum name %q", row.Key, row.Proto)
	}
	if _, duplicate := s.seenProto[row.Proto]; duplicate {
		return fmt.Errorf("proto enum %q is declared more than once", row.Proto)
	}
	s.seenProto[row.Proto] = struct{}{}
	return nil
}

func (s *profileValidationState) validateKey(row Profile) error {
	if !profileKeyPattern.MatchString(row.Key) {
		return fmt.Errorf("profile %q: key must be kebab-case", row.Key)
	}
	if _, duplicate := s.seenKey[row.Key]; duplicate {
		return fmt.Errorf("profile key %q is declared more than once", row.Key)
	}
	s.seenKey[row.Key] = struct{}{}
	return nil
}

func (s *profileValidationState) validateI18nKey(key, field, profile string, seen map[string]struct{}) error {
	if !profileI18nKey.MatchString(key) {
		return fmt.Errorf("profile %q: %s %q must look like profiles.camelCase", profile, field, key)
	}
	if _, duplicate := seen[key]; duplicate {
		return fmt.Errorf("profile %q: %s %q is declared more than once", profile, field, key)
	}
	seen[key] = struct{}{}
	return nil
}

func (s *profileValidationState) validateComplete() error {
	var missing []string
	for name := range s.expected {
		if _, ok := s.seenProto[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) != 0 {
		sort.Strings(missing)
		return fmt.Errorf("proto enum values missing from the profile catalog: %s", strings.Join(missing, ", "))
	}
	return nil
}

// TSEnumMember maps PROFILE_LOCAL_ONLY to LOCAL_ONLY, the generated
// TypeScript enum member name.
func TSEnumMember(protoName string) (string, error) {
	const prefix = "PROFILE_"
	if !strings.HasPrefix(protoName, prefix) {
		return "", fmt.Errorf("%q is not a profile enum name", protoName)
	}
	member := strings.TrimPrefix(protoName, prefix)
	if member == "" || member == "UNSPECIFIED" {
		return "", fmt.Errorf("%q has no TypeScript enum member", protoName)
	}
	return member, nil
}

// RenderProfilesTS renders the deterministic TypeScript profile catalog.
func RenderProfilesTS(c *ProfileCatalog) ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	out.WriteString("//\n")
	out.WriteString("// Source : config/profile-catalog.json\n")
	out.WriteString("// Codegen: GOWORK=off go run ./cmd/gen-gateway-catalog (from api/)\n")
	out.WriteString("// Verify : GOWORK=off go run ./cmd/gen-gateway-catalog -check (from api/)\n")
	out.WriteString("import { Profile } from \"@vrooli/proto-types/ai-gateway/v1/shared/gateway_pb\";\n")
	out.WriteString("import { strings } from \"./strings\";\n\n")
	out.WriteString("export const PROFILE_CATALOG = [\n")
	for _, row := range c.Sorted() {
		member, err := TSEnumMember(row.Proto)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, "  { value: Profile.%s, key: %q, labelKey: strings.%s, descriptionKey: strings.%s },\n",
			member, row.Key, row.LabelKey, row.DescriptionKey)
	}
	out.WriteString("] as const;\n")
	return out.Bytes(), nil
}
