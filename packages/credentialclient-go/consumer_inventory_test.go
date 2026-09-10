package credentialclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConsumerFixture(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, ".vrooli", "service.json"), `{"service":{"name":"fixture"}}`)
	write(filepath.Join(root, "scenarios", "consumer", ".vrooli", "service.json"), `{
  "service":{"name":"consumer"},
  "credentials":{"descriptors":[
    {"logical_id":"vrooli/fixture","field":"token","required":true},
    {"logical_id":"vrooli/fixture","field":"optional","required":false},
    {"logical_id":"vrooli/fixture","field":"env-token","env":"FIXTURE_TOKEN","required":true}
  ]}
}`)
	write(filepath.Join(root, "scenarios", "consumer", "api", "main.go"), source)
	return root
}

func TestDiscoverConsumerInventoryBindsAuthorityAndEnvironmentUses(t *testing.T) {
	root := writeConsumerFixture(t, `package main

import "os"

func run() {
  _ = resolve("vrooli/fixture", "token")
  _ = os.Getenv("FIXTURE_TOKEN")
}

func resolve(identity, field string) string { return identity + field }
`)
	// The scanner only attributes literal calls. Use the exact supported form
	// in a second source line so this test verifies the contract, not a helper
	// function's implementation.
	path := filepath.Join(root, "scenarios", "consumer", "api", "binding.go")
	if err := os.WriteFile(path, []byte(`package main
func bind() { _ = client.Resolve(ctx, "vrooli/fixture", "token") }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	inventory, err := DiscoverConsumerInventory(root, Scope{Scenarios: []string{"consumer"}})
	if err != nil {
		t.Fatal(err)
	}
	if inventory.DeclarationSiteCount != 3 || inventory.DistinctAddressCount != 3 {
		t.Fatalf("inventory counts = (%d, %d), want (3, 3)", inventory.DeclarationSiteCount, inventory.DistinctAddressCount)
	}
	var token ConsumerInventoryRow
	for _, row := range inventory.Rows {
		if row.Field == "token" {
			token = row
		}
	}
	if len(token.Consumers) != 1 || token.Consumers[0].Kind != ConsumerAuthority {
		t.Fatalf("token consumers = %#v, want one authority binding", token.Consumers)
	}
	if !strings.HasPrefix(token.SourceRef, "/") || !strings.Contains(token.SourceRef, "service.json") {
		t.Fatalf("source ref = %q, want an absolute manifest path", token.SourceRef)
	}
	var env ConsumerInventoryRow
	for _, row := range inventory.Rows {
		if row.Field == "env-token" {
			env = row
		}
	}
	if len(env.Consumers) != 1 || env.Consumers[0].Kind != ConsumerEnvironment {
		t.Fatalf("env consumers = %#v, want one environment binding", env.Consumers)
	}
	if len(inventory.Gaps) != 0 {
		t.Fatalf("gaps = %#v, want the optional unbound declaration omitted from blockers", inventory.Gaps)
	}
}

func TestDiscoverConsumerInventoryRejectsUndeclaredAndRequiredUnboundInputs(t *testing.T) {
	root := writeConsumerFixture(t, `package main
func bind() {
  _ = client.Resolve(ctx, "vrooli/missing", "token")
}
`)
	inventory, err := DiscoverConsumerInventory(root, Scope{Scenarios: []string{"consumer"}})
	if err != nil {
		t.Fatal(err)
	}
	if inventory.UnresolvedConsumerCount != 1 {
		t.Fatalf("unresolved consumer count = %d, want 1", inventory.UnresolvedConsumerCount)
	}
	var reasons []string
	for _, gap := range inventory.Gaps {
		reasons = append(reasons, gap.Reason)
	}
	joined := strings.Join(reasons, "\n")
	if !strings.Contains(joined, "no matching declaration") {
		t.Fatalf("gaps = %#v, want undeclared consumer failure", inventory.Gaps)
	}
	if !strings.Contains(joined, "required declaration has no observed runtime consumer") {
		t.Fatalf("gaps = %#v, want required unbound failure", inventory.Gaps)
	}
}

func TestDiscoverConsumerInventoryHonorsTierClosure(t *testing.T) {
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "scenarios", "tiered", ".vrooli", "service.json"), `{
  "service":{"name":"tiered"},
  "credentials":{"descriptors":[
    {"logical_id":"vrooli/tiered","field":"local","required":true,"tiers":["tier-1-local"]},
    {"logical_id":"vrooli/tiered","field":"remote","required":true,"tiers":["tier-4-saas"]}
  ]}
}`)
	write(filepath.Join(root, "scenarios", "tiered", "api", "main.go"), `package main
func bind() { _ = client.Resolve(ctx, "vrooli/tiered", "local") }
`)
	inventory, err := DiscoverConsumerInventory(root, Scope{Scenarios: []string{"tiered"}, Tier: "tier-1-local"})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Rows) != 1 || inventory.Rows[0].Field != "local" {
		t.Fatalf("tier inventory = %#v, want only local input", inventory.Rows)
	}
	if len(inventory.Gaps) != 0 {
		t.Fatalf("tier inventory gaps = %#v, want none", inventory.Gaps)
	}
}

func TestDiscoverConsumerInventoryUsesExplicitDynamicRegistrations(t *testing.T) {
	root := t.TempDir()
	serviceDir := filepath.Join(root, "scenarios", "dynamic", ".vrooli")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "service.json"), []byte(`{
  "service":{"name":"dynamic"},
  "credentials":{
    "descriptors":[
      {"logical_id":"vrooli/dynamic","field":"session","required":true},
      {"logical_id":"vrooli/dynamic","field":"forward-declared","required":true}
    ],
    "consumers":[
      {"logical_id":"vrooli/dynamic","field":"session","kind":"delegated","consumer":"account session broker","source_ref":"api/session.go:17","required":true},
      {"logical_id":"vrooli/dynamic","field":"forward-declared","kind":"dynamic","consumer":"late-bound provider","source_ref":"api/provider.go:18","required":true},
      {"logical_id":"vrooli/undeclared","field":"token","kind":"external","consumer":"provider callback","source_ref":"api/provider.go:29","required":true}
    ]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "dynamic", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	inventory, err := DiscoverConsumerInventory(root, Scope{Scenarios: []string{"dynamic"}})
	if err != nil {
		t.Fatal(err)
	}
	var bound, forward, missing bool
	for _, row := range inventory.Rows {
		address := row.LogicalID + ":" + row.Field
		if address == "vrooli/dynamic:session" {
			bound = row.Disposition == "bound" && len(row.Consumers) == 1 && row.Consumers[0].Consumer == "account session broker"
		}
		if address == "vrooli/dynamic:forward-declared" {
			forward = row.Disposition == "bound" && len(row.Consumers) == 1 && row.Consumers[0].Consumer == "late-bound provider"
		}
		if address == "vrooli/undeclared:token" {
			missing = !row.Declared && row.Disposition == "declaration-missing"
		}
	}
	if !bound || !forward || !missing {
		t.Fatalf("rows = %#v, want bound and declaration-missing registrations", inventory.Rows)
	}
	if len(inventory.Gaps) != 1 || inventory.Gaps[0].Consumer != "provider callback" {
		t.Fatalf("gaps = %#v, want one unresolved explicit consumer", inventory.Gaps)
	}
}

func TestDiscoverConsumerInventoryBindsAddressPatternRegistrations(t *testing.T) {
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "scenarios", "backup", ".vrooli", "service.json"), `{
  "service":{"name":"backup"},
  "credentials":{"descriptors":[
    {"logical_id":"vrooli/backup/source-a","field":"passphrase","required":true},
    {"logical_id":"vrooli/backup/source-b","field":"passphrase","required":true}
  ],"consumers":[
    {"address_pattern":"vrooli/backup/{name}:passphrase","kind":"backup","consumer":"kopia restore","source_ref":"api/restore.go:22","required":true}
  ]}
}`)
	inventory, err := DiscoverConsumerInventory(root, Scope{Scenarios: []string{"backup"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Gaps) != 0 {
		t.Fatalf("pattern inventory gaps = %#v, want none", inventory.Gaps)
	}
	for _, row := range inventory.Rows {
		if row.Disposition != "bound" || len(row.Consumers) != 1 || row.Consumers[0].Kind != ConsumerBackup {
			t.Fatalf("row = %#v, want one bound backup consumer", row)
		}
	}
}
