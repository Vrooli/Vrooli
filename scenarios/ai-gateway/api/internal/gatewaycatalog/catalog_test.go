package gatewaycatalog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/requestvocab"
	"ai-gateway/internal/rolecatalog"
)

func scenarioRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(dir, "..", "..", ".."))
}

func loadCommitted(t *testing.T) (*ProfileCatalog, *rolecatalog.RoleCatalog, string) {
	t.Helper()
	root := scenarioRoot(t)
	profiles, err := LoadProfiles(filepath.Join(root, "config", "profile-catalog.json"))
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}
	roles, err := LoadRoles(filepath.Join(root, "config", "inference-role-catalog.json"))
	if err != nil {
		t.Fatalf("load roles: %v", err)
	}
	return profiles, roles, root
}

// The catalog is the operator vocabulary. It must cover every selectable proto
// Profile value on the current wire contract, and nothing else.
func TestProfileCatalogCoversProtoEnum(t *testing.T) {
	profiles, _, _ := loadCommitted(t)
	declared := map[string]bool{}
	for _, row := range profiles.Profiles {
		declared[row.Proto] = true
	}
	for _, name := range sharedv1.Profile_name {
		if name == sharedv1.Profile_PROFILE_UNSPECIFIED.String() {
			continue
		}
		if !declared[name] {
			t.Errorf("profile catalog is missing proto value %s", name)
		}
	}
	if len(declared) != len(sharedv1.Profile_name)-1 {
		t.Fatalf("catalog declares %d profiles, proto has %d selectable values", len(declared), len(sharedv1.Profile_name)-1)
	}
}

func TestProfileCatalogOrderIsStable(t *testing.T) {
	profiles, _, _ := loadCommitted(t)
	want := []string{
		"local-only",
		"local-first",
		"remote-only",
		"quality-first",
		"cheap-first",
		"privacy-sensitive",
	}
	sorted := profiles.Sorted()
	if len(sorted) != len(want) {
		t.Fatalf("got %d profiles, want %d", len(sorted), len(want))
	}
	for index, key := range want {
		if sorted[index].Key != key {
			t.Errorf("profile at order position %d = %q, want %q", index+1, sorted[index].Key, key)
		}
		if sorted[index].Order != index+1 {
			t.Errorf("profile %q order = %d, want %d", key, sorted[index].Order, index+1)
		}
	}
}

func TestValidateRejectsCatalogDrift(t *testing.T) {
	base := func() *ProfileCatalog {
		profiles, _, _ := loadCommitted(t)
		return profiles
	}

	t.Run("unknown proto", func(t *testing.T) {
		catalog := base()
		catalog.Profiles[0].Proto = "PROFILE_MADE_UP"
		if err := catalog.Validate(); err == nil {
			t.Fatal("expected unknown proto name to be rejected")
		}
	})

	t.Run("unspecified proto", func(t *testing.T) {
		catalog := base()
		catalog.Profiles[0].Proto = "PROFILE_UNSPECIFIED"
		if err := catalog.Validate(); err == nil {
			t.Fatal("expected PROFILE_UNSPECIFIED to be rejected")
		}
	})

	t.Run("duplicate proto", func(t *testing.T) {
		catalog := base()
		catalog.Profiles[1].Proto = catalog.Profiles[0].Proto
		if err := catalog.Validate(); err == nil {
			t.Fatal("expected duplicate proto declaration to be rejected")
		}
	})

	t.Run("missing profile", func(t *testing.T) {
		catalog := base()
		catalog.Profiles = catalog.Profiles[:len(catalog.Profiles)-1]
		if err := catalog.Validate(); err == nil {
			t.Fatal("expected missing proto value to be rejected")
		}
	})

	t.Run("duplicate key", func(t *testing.T) {
		catalog := base()
		catalog.Profiles[1].Key = catalog.Profiles[0].Key
		if err := catalog.Validate(); err == nil {
			t.Fatal("expected duplicate key to be rejected")
		}
	})

	t.Run("duplicate order", func(t *testing.T) {
		catalog := base()
		catalog.Profiles[1].Order = catalog.Profiles[0].Order
		if err := catalog.Validate(); err == nil {
			t.Fatal("expected duplicate order to be rejected")
		}
	})

	t.Run("bad i18n key", func(t *testing.T) {
		catalog := base()
		catalog.Profiles[0].LabelKey = "Profile.LocalOnly"
		if err := catalog.Validate(); err == nil {
			t.Fatal("expected non-profiles i18n key to be rejected")
		}
	})
}

func TestGeneratedProfilesTSMatchesCommitted(t *testing.T) {
	profiles, _, root := loadCommitted(t)
	generated, err := RenderProfilesTS(profiles)
	if err != nil {
		t.Fatalf("render TS: %v", err)
	}
	path := filepath.Join(root, "ui", "src", "consts", "profileCatalog.generated.ts")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Equal(committed, generated) {
		t.Fatalf("%s is stale; run `make gateway-catalog` from scenarios/ai-gateway", path)
	}
}

func TestGeneratedReferenceMatchesCommitted(t *testing.T) {
	profiles, roles, root := loadCommitted(t)
	generated, err := RenderReference(profiles, roles, providers.DefaultProviderSpecs())
	if err != nil {
		t.Fatalf("render reference: %v", err)
	}
	again, err := RenderReference(profiles, roles, providers.DefaultProviderSpecs())
	if err != nil {
		t.Fatalf("render reference second pass: %v", err)
	}
	if !bytes.Equal(generated, again) {
		t.Fatal("reference renderer is not deterministic")
	}
	path := filepath.Join(root, "docs", "reference", "capability-catalog.md")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Equal(committed, generated) {
		t.Fatalf("%s is stale; run `make gateway-catalog` from scenarios/ai-gateway", path)
	}
}

func TestReferenceTablesCoverEveryRoleAndProvider(t *testing.T) {
	profiles, roles, _ := loadCommitted(t)
	reference, err := RenderReference(profiles, roles, providers.DefaultProviderSpecs())
	if err != nil {
		t.Fatalf("render reference: %v", err)
	}
	doc := string(reference)
	sections := referenceSections(t, doc)
	assertProfileRows(t, sections["## Profiles"], profiles)
	assertPrivacyRows(t, sections["## Privacy Classes"])
	assertRequestKindRows(t, sections["## Request Kinds and Resource Capabilities"])
	assertRoleRows(t, sections["## Roles"], roles)
	assertProviderRows(t, sections["## Providers"])
	assertCandidateRows(t, sections["## Role to Candidate Matrix"], roles)
	assertStagedCandidateRows(t, doc, roles)
}

func referenceSections(t *testing.T, doc string) map[string]string {
	t.Helper()
	boundaries := []struct{ heading, next string }{
		{"## Profiles", "## Privacy Classes"},
		{"## Privacy Classes", "## Request Kinds and Resource Capabilities"},
		{"## Request Kinds and Resource Capabilities", "## Roles"},
		{"## Roles", "## Providers"},
		{"## Providers", "## Role to Candidate Matrix"},
		{"## Role to Candidate Matrix", "## Semantic Escalation Candidates (Caller Opt-In)"},
	}
	sections := make(map[string]string, len(boundaries))
	for _, boundary := range boundaries {
		start := strings.Index(doc, boundary.heading)
		if start < 0 {
			t.Fatalf("reference is missing heading %q", boundary.heading)
		}
		body := doc[start+len(boundary.heading):]
		end := strings.Index(body, boundary.next)
		if end < 0 {
			t.Fatalf("reference is missing heading %q", boundary.next)
		}
		sections[boundary.heading] = body[:end]
		if strings.Count(body[:end], "\n| `") == 0 {
			t.Errorf("%s table has no data rows", boundary.heading)
		}
	}
	return sections
}

func assertProfileRows(t *testing.T, section string, profiles *ProfileCatalog) {
	t.Helper()
	for _, profile := range profiles.Profiles {
		if !strings.Contains(section, "| `"+profile.Key+"` | `"+profile.Proto+"` |") {
			t.Errorf("reference profile table does not mention profile %q", profile.Key)
		}
	}
}

func assertPrivacyRows(t *testing.T, section string) {
	t.Helper()
	for value, proto := range sharedv1.PrivacyClass_name {
		if sharedv1.PrivacyClass(value) == sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
			continue
		}
		label := strings.ToLower(strings.TrimPrefix(proto, "PRIVACY_CLASS_"))
		if !strings.Contains(section, "| `"+label+"` | `"+proto+"` |") {
			t.Errorf("reference privacy table does not mention proto value %q", proto)
		}
	}
}

func assertRequestKindRows(t *testing.T, section string) {
	t.Helper()
	for _, definition := range requestvocab.RequestKindCatalog() {
		row := findReferenceRow(section, "| `"+definition.Kind.String()+"` |")
		if row == "" {
			t.Errorf("reference request-kind table does not mention %q", definition.Kind)
			continue
		}
		assertRowContains(t, row, definition.Kind.String(), definition.OutputKind)
		for _, modality := range modalityNames(definition.RequiredInputs) {
			assertRowContains(t, row, definition.Kind.String(), modality)
		}
		for _, capability := range definition.RequiredCapabilities {
			assertRowContains(t, row, definition.Kind.String(), capability)
		}
	}
}

func assertRoleRows(t *testing.T, section string, roles *rolecatalog.RoleCatalog) {
	t.Helper()
	for name := range roles.Roles {
		if !strings.Contains(section, "| `"+name+"` |") {
			t.Errorf("reference role table does not mention role %q", name)
		}
	}
}

func assertProviderRows(t *testing.T, section string) {
	t.Helper()
	for _, spec := range providers.DefaultProviderSpecs() {
		if !strings.Contains(section, "| `"+spec.ID+"` |") {
			t.Errorf("reference provider table does not mention provider %q", spec.ID)
		}
	}
}

func assertCandidateRows(t *testing.T, section string, roles *rolecatalog.RoleCatalog) {
	t.Helper()
	for name, role := range roles.Roles {
		if len(role.Candidates) > 0 && !strings.Contains(section, "| `"+name+"` |") {
			t.Errorf("reference matrix does not mention candidates for role %q", name)
		}
	}
}

func assertStagedCandidateRows(t *testing.T, doc string, roles *rolecatalog.RoleCatalog) {
	t.Helper()
	escalationSection := sectionBody(t, doc, "## Semantic Escalation Candidates (Caller Opt-In)", "## Qualification Candidates (Not Dispatchable)")
	qualificationSection := sectionBody(t, doc, "## Qualification Candidates (Not Dispatchable)", "")
	for name, role := range roles.Roles {
		assertEscalationRows(t, escalationSection, name, role.SemanticEscalationCandidates)
		assertQualificationRows(t, qualificationSection, name, role.QualificationCandidates)
	}
}

func assertEscalationRows(t *testing.T, section, role string, candidates []rolecatalog.Candidate) {
	t.Helper()
	for _, candidate := range candidates {
		row := "| `" + role + "` | `" + candidate.Provider + "` | `" + candidate.ResourceRole + "` |"
		if !strings.Contains(section, row) {
			t.Errorf("reference escalation table does not mention %s/%s", role, candidate.Provider)
		}
	}
}

func assertQualificationRows(t *testing.T, section, role string, candidates []rolecatalog.QualificationCandidate) {
	t.Helper()
	for _, qualified := range candidates {
		candidate := qualified.Candidate
		row := "| `" + role + "` | `" + candidate.Provider + "` | `" + candidate.ResourceRole + "` | " + qualified.Status + " |"
		if !strings.Contains(section, row) {
			t.Errorf("reference qualification table does not mention %s/%s status %s", role, candidate.Provider, qualified.Status)
		}
	}
}

func findReferenceRow(section, prefix string) string {
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func assertRowContains(t *testing.T, row, context, value string) {
	t.Helper()
	if !strings.Contains(row, "`"+value+"`") {
		t.Errorf("reference row for %q does not mention %q", context, value)
	}
}

func sectionBody(t *testing.T, doc, heading, next string) string {
	t.Helper()
	start := strings.Index(doc, heading)
	if start < 0 {
		t.Fatalf("reference is missing heading %q", heading)
	}
	body := doc[start+len(heading):]
	if next == "" {
		return body
	}
	end := strings.Index(body, next)
	if end < 0 {
		t.Fatalf("reference is missing heading %q", next)
	}
	return body[:end]
}
