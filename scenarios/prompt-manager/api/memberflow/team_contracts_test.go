package memberflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"prompt-manager/teamcontract"
)

// stubContract is a tiny test helper that builds a *teamcontract.OperatingContract
// holding only the decision-context ids the registry tests care about.
// Constructing the real type keeps the registry's HasDecisionContext /
// TeamsForDecisionContext accessors honest — if their lookup field ever
// renames, these tests will fail to compile.
func stubContract(contextIDs ...string) *teamcontract.OperatingContract {
	dc := make(map[string]teamcontract.DecisionContext, len(contextIDs))
	for _, id := range contextIDs {
		dc[id] = teamcontract.DecisionContext{Description: id}
	}
	return &teamcontract.OperatingContract{
		SchemaVersion:   teamcontract.SchemaVersion,
		DecisionContext: dc,
	}
}

// writeTeamFile is a small fixture helper: lays out a teams/<id>/team.json
// under root and returns the populated root. The minimal payload is enough
// to exercise LoadAllTeamContracts; the loader only reads `id` and
// `operatingContract.decisionContexts`.
func writeTeamFile(t *testing.T, root, id string, body string) string {
	t.Helper()
	dir := filepath.Join(root, "teams", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", dir, err)
	}
	path := filepath.Join(dir, "team.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
	return path
}

func TestLoadAllTeamContracts_EmptyStoreDir(t *testing.T) {
	reg, err := LoadAllTeamContracts("")
	if err != nil {
		t.Fatalf("expected no error on empty storeDir; got %v", err)
	}
	if len(reg) != 0 {
		t.Errorf("expected empty registry on empty storeDir; got %d entries", len(reg))
	}
}

func TestLoadAllTeamContracts_MissingTeamsDir(t *testing.T) {
	root := t.TempDir() // no teams/ subdir
	reg, err := LoadAllTeamContracts(root)
	if err != nil {
		t.Fatalf("missing teams/ should not error; got %v", err)
	}
	if len(reg) != 0 {
		t.Errorf("missing teams/ should yield empty registry; got %d entries", len(reg))
	}
}

func TestLoadAllTeamContracts_LoadsAndIndexesByID(t *testing.T) {
	root := t.TempDir()

	// Two valid teams with distinct decision contexts.
	writeTeamFile(t, root, "alpha", `{
		"id": "alpha",
		"operatingContract": {
			"schemaVersion": 1,
			"decisionContexts": {
				"alpha-launch": {"description": "launch"}
			}
		}
	}`)
	writeTeamFile(t, root, "beta", `{
		"id": "beta",
		"operatingContract": {
			"schemaVersion": 1,
			"decisionContexts": {
				"beta-promote": {"description": "promote"}
			}
		}
	}`)

	reg, err := LoadAllTeamContracts(root)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}

	if got, want := len(reg), 2; got != want {
		t.Fatalf("registry size: got %d, want %d", got, want)
	}
	if reg["alpha"] == nil || reg["alpha"].Contract == nil {
		t.Fatalf("alpha not indexed correctly: %#v", reg["alpha"])
	}
	if _, ok := reg["alpha"].Contract.DecisionContext["alpha-launch"]; !ok {
		t.Errorf("alpha contract missing alpha-launch decision context")
	}
	if reg["beta"].SourcePath == "" {
		t.Errorf("beta SourcePath was not populated")
	}
	if got, want := reg.IDs(), []string{"alpha", "beta"}; !reflect.DeepEqual(got, want) {
		t.Errorf("IDs() = %v, want %v", got, want)
	}
}

func TestLoadAllTeamContracts_LoadsTopicCatalog(t *testing.T) {
	root := t.TempDir()
	writeTeamFile(t, root, "alpha", `{
		"id": "alpha",
		"topicCatalog": [
			{
				"prefix": "audience-scan/*",
				"status": "live",
				"purpose": "Audience evidence."
			},
			{
				"prefix": "publish-performance/*",
				"qualifier": "future",
				"status": "target",
				"purpose": "Performance evidence."
			}
		],
		"operatingContract": {"schemaVersion": 1, "decisionContexts": {}}
	}`)

	reg, err := LoadAllTeamContracts(root)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}
	got := reg["alpha"].TopicCatalog
	if len(got) != 2 {
		t.Fatalf("TopicCatalog length = %d, want 2", len(got))
	}
	if got[0].Prefix != "audience-scan/*" || got[0].Purpose != "Audience evidence." {
		t.Fatalf("unexpected first topic catalog entry: %+v", got[0])
	}
	if got[1].Qualifier != "future" {
		t.Fatalf("future qualifier not preserved: %+v", got[1])
	}
}

func TestLoadAllTeamContracts_RejectsInvalidTopicCatalog(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing prefix",
			body: `{"id":"alpha","topicCatalog":[{"status":"live","purpose":"x"}]}`,
		},
		{
			name: "malformed prefix",
			body: `{"id":"alpha","topicCatalog":[{"prefix":"bad * prefix","status":"live","purpose":"x"}]}`,
		},
		{
			name: "unknown status",
			body: `{"id":"alpha","topicCatalog":[{"prefix":"x/*","status":"maybe","purpose":"x"}]}`,
		},
		{
			name: "missing current purpose",
			body: `{"id":"alpha","topicCatalog":[{"prefix":"x/*","status":"live"}]}`,
		},
		{
			name: "duplicate",
			body: `{"id":"alpha","topicCatalog":[{"prefix":"x/*","status":"live","purpose":"x"},{"prefix":"x/*","status":"live","purpose":"y"}]}`,
		},
		{
			name: "wrong qualifier",
			body: `{"id":"alpha","topicCatalog":[{"prefix":"x/*","qualifier":"future","status":"live","purpose":"x"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTeamFile(t, root, "alpha", tt.body)
			if _, err := LoadAllTeamContracts(root); err == nil {
				t.Fatalf("expected invalid topicCatalog to fail")
			}
		})
	}
}

func TestLoadAllTeamContracts_PrefersFileIDOverDirName(t *testing.T) {
	// The file's own id field is authoritative, not the directory name.
	// This matches the convention elsewhere in prompt-manager (taxonomy
	// loader resolves by id, not by basename).
	root := t.TempDir()
	writeTeamFile(t, root, "wrong-dir-name", `{
		"id": "actual-team-id",
		"operatingContract": {"schemaVersion": 1, "decisionContexts": {}}
	}`)

	reg, err := LoadAllTeamContracts(root)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}
	if _, ok := reg["actual-team-id"]; !ok {
		t.Errorf("registry should index by file id; got keys %v", reg.IDs())
	}
	if _, ok := reg["wrong-dir-name"]; ok {
		t.Errorf("registry should not index by directory name")
	}
}

func TestLoadAllTeamContracts_SkipsTeamDirsWithoutTeamJSON(t *testing.T) {
	// A team directory mid-creation may not yet have team.json. The
	// loader should silently skip rather than fail the whole walk.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "scaffold-only"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTeamFile(t, root, "fully-formed", `{
		"id": "fully-formed",
		"operatingContract": {"schemaVersion": 1, "decisionContexts": {"x": {"description": "x"}}}
	}`)

	reg, err := LoadAllTeamContracts(root)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}
	if _, ok := reg["fully-formed"]; !ok {
		t.Errorf("loader skipped well-formed team")
	}
	if _, ok := reg["scaffold-only"]; ok {
		t.Errorf("loader should skip directories without team.json")
	}
}

func TestLoadAllTeamContracts_SkipsHiddenTeamDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", ".hidden"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTeamFile(t, root, ".hidden", `{
		"id": "hidden",
		"operatingContract": {"schemaVersion": 1, "decisionContexts": {}}
	}`)

	reg, err := LoadAllTeamContracts(root)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}
	if _, ok := reg["hidden"]; ok {
		t.Errorf("loader should skip dot-prefixed team directories")
	}
}

func TestLoadAllTeamContracts_RejectsMissingID(t *testing.T) {
	root := t.TempDir()
	writeTeamFile(t, root, "anon", `{
		"operatingContract": {"schemaVersion": 1, "decisionContexts": {}}
	}`)

	_, err := LoadAllTeamContracts(root)
	if err == nil {
		t.Fatalf("expected error for team.json missing id")
	}
}

func TestLoadAllTeamContracts_RejectsMalformedJSON(t *testing.T) {
	root := t.TempDir()
	writeTeamFile(t, root, "broken", `{ this is not json `)
	_, err := LoadAllTeamContracts(root)
	if err == nil {
		t.Fatalf("expected error for malformed team.json")
	}
}

func TestLoadAllTeamContracts_RejectsDuplicateID(t *testing.T) {
	root := t.TempDir()
	writeTeamFile(t, root, "dir-one", `{
		"id": "shared-id",
		"operatingContract": {"schemaVersion": 1, "decisionContexts": {}}
	}`)
	writeTeamFile(t, root, "dir-two", `{
		"id": "shared-id",
		"operatingContract": {"schemaVersion": 1, "decisionContexts": {}}
	}`)
	_, err := LoadAllTeamContracts(root)
	if err == nil {
		t.Fatalf("expected error for duplicate team id across files")
	}
}

func TestLoadAllTeamContracts_AllowsNilOperatingContract(t *testing.T) {
	// Mid-scaffold teams may not yet declare an operating contract. The
	// registry should index them with Contract=nil rather than reject;
	// downstream methods (HasDecisionContext etc.) treat nil contracts
	// as "no contexts declared," which is correct.
	root := t.TempDir()
	writeTeamFile(t, root, "scaffold", `{"id": "scaffold"}`)

	reg, err := LoadAllTeamContracts(root)
	if err != nil {
		t.Fatalf("nil operatingContract should not fail loader; got %v", err)
	}
	if reg["scaffold"] == nil {
		t.Fatalf("scaffold not indexed")
	}
	if reg["scaffold"].Contract != nil {
		t.Errorf("expected Contract=nil for scaffold; got %#v", reg["scaffold"].Contract)
	}
	if reg.HasDecisionContext("anything") {
		t.Errorf("nil contract should never resolve a decision context")
	}
}

func TestRegistry_HasDecisionContext(t *testing.T) {
	reg := TeamContractRegistry{
		"alpha": {TeamID: "alpha", Contract: stubContract("alpha-launch")},
		"beta":  {TeamID: "beta", Contract: stubContract("beta-promote")},
	}
	if !reg.HasDecisionContext("alpha-launch") {
		t.Errorf("alpha-launch should resolve")
	}
	if !reg.HasDecisionContext("beta-promote") {
		t.Errorf("beta-promote should resolve")
	}
	if reg.HasDecisionContext("ghost") {
		t.Errorf("ghost id should not resolve")
	}
	if reg.HasDecisionContext("") {
		t.Errorf("empty id should never resolve")
	}
	if reg.HasDecisionContext("   ") {
		t.Errorf("whitespace-only id should never resolve")
	}
}

func TestRegistry_TeamsForDecisionContext(t *testing.T) {
	// A decision-context id might (in theory) appear under multiple teams.
	// The accessor returns lex-sorted teams so diagnostic detail strings
	// have a stable shape.
	reg := TeamContractRegistry{
		"beta":  {TeamID: "beta", Contract: stubContract("shared-context")},
		"alpha": {TeamID: "alpha", Contract: stubContract("shared-context")},
	}
	got := reg.TeamsForDecisionContext("shared-context")
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TeamsForDecisionContext = %v, want %v", got, want)
	}

	if reg.TeamsForDecisionContext("ghost") != nil {
		t.Errorf("ghost id should yield nil slice")
	}
	if reg.TeamsForDecisionContext("") != nil {
		t.Errorf("empty id should yield nil slice")
	}
}

func TestRegistry_NilEntriesAndContracts(t *testing.T) {
	reg := TeamContractRegistry{
		"nil-entry":    nil,
		"nil-contract": {TeamID: "nil-contract", Contract: nil},
	}
	if reg.HasDecisionContext("anything") {
		t.Errorf("nil entries / contracts should never resolve")
	}
	if reg.TeamsForDecisionContext("anything") != nil {
		t.Errorf("nil entries / contracts should yield nil teams slice")
	}
}

// TestTeamRoleMemberDrift proves roles.json is checked against the contract in
// both directions, and that a team without roles.json is skipped rather than
// flagged.
func TestTeamRoleMemberDrift(t *testing.T) {
	storeDir := t.TempDir()
	writeRolesFixtureTeam(t, storeDir, "alpha", []string{"researcher", "publisher"}, []string{"researcher", "ghost"})
	writeRolesFixtureTeam(t, storeDir, "beta", []string{"scanner"}, nil)

	registry, err := LoadAllTeamContracts(storeDir)
	if err != nil {
		t.Fatalf("load team contracts: %v", err)
	}

	findings := ruleTeamRoleMemberDrift(ValidationOptions{TeamContracts: registry})
	byMember := map[string]string{}
	for _, f := range findings {
		if f.Rule != "team_role_member_drift" {
			t.Errorf("unexpected rule %q", f.Rule)
		}
		if f.Severity != SeverityError {
			t.Errorf("%s: severity = %q, want error", f.Member.Member, f.Severity)
		}
		byMember[f.Member.Team+"/"+f.Member.Member] = f.Detail
	}

	if _, ok := byMember["alpha/ghost"]; !ok {
		t.Errorf("role %q has no matching member and should be flagged; got %v", "ghost", byMember)
	}
	if _, ok := byMember["alpha/publisher"]; !ok {
		t.Errorf("member %q has no role entry and should be flagged; got %v", "publisher", byMember)
	}
	if _, ok := byMember["alpha/researcher"]; ok {
		t.Errorf("member %q matches a role and should not be flagged", "researcher")
	}
	if _, ok := byMember["beta/scanner"]; ok {
		t.Errorf("team beta declares no roles.json and must be skipped, not flagged")
	}
	if len(findings) != 2 {
		t.Fatalf("expected exactly 2 findings; got %d: %v", len(findings), byMember)
	}
}

// TestBundledTeamRolesMatchContractMembers is the live-roster guard: the rule
// above only helps if the shipped teams satisfy it.
func TestBundledTeamRolesMatchContractMembers(t *testing.T) {
	registry, err := LoadAllTeamContracts(requirePromptManagerStoreDir(t))
	if err != nil {
		t.Fatalf("load bundled team contracts: %v", err)
	}
	if len(registry) == 0 {
		t.Fatalf("expected bundled teams")
	}
	for _, f := range ruleTeamRoleMemberDrift(ValidationOptions{TeamContracts: registry}) {
		t.Errorf("%s: %s", f.Member.Team, f.Detail)
	}
}

func writeRolesFixtureTeam(t *testing.T, storeDir, teamID string, memberIDs, roleIDs []string) {
	t.Helper()
	teamDir := filepath.Join(storeDir, "teams", teamID)
	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", teamDir, err)
	}

	members := map[string]any{}
	for _, id := range memberIDs {
		members[id] = map[string]any{"lane": "test lane"}
	}
	teamJSON, err := json.Marshal(map[string]any{
		"id":                teamID,
		"operatingContract": map[string]any{"members": members},
	})
	if err != nil {
		t.Fatalf("marshal team.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, "team.json"), teamJSON, 0o644); err != nil {
		t.Fatalf("write team.json: %v", err)
	}

	if roleIDs == nil {
		return
	}
	roles := make([]map[string]any, 0, len(roleIDs))
	for _, id := range roleIDs {
		roles = append(roles, map[string]any{"id": id, "name": id, "description": "test role"})
	}
	rolesJSON, err := json.Marshal(map[string]any{"kind": "team-roles", "teamId": teamID, "roles": roles})
	if err != nil {
		t.Fatalf("marshal roles.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, "roles.json"), rolesJSON, 0o644); err != nil {
		t.Fatalf("write roles.json: %v", err)
	}
}
