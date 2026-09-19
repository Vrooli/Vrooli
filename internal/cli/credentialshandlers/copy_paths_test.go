package credentialshandlers

import (
	"strings"
	"testing"
)

// The encrypted credential-store copy was fully implemented in the service
// layer (StoreCopy, StoreCopyConfigure, StoreCopyScheduled) but reachable from
// nothing: `credentials store` bound eight subcommands and none of them was
// `copy`, so the command PROBLEMS.md told an operator to run did not exist and
// no host's store was ever backed up (2026-09-15). These paths are pinned so a
// future edit cannot silently orphan the feature again.
func TestStoreCopyVerbsAreRegistered(t *testing.T) {
	registered := make(map[string]bool)
	for _, path := range RegisteredCommandPaths() {
		registered[path] = true
	}
	for _, want := range []string{
		"credentials store copy",
		"credentials store copy-configure",
		"credentials store copy-scheduled",
	} {
		if !registered[want] {
			t.Fatalf("%q is not a registered command path; the copy feature is unreachable again", want)
		}
	}
}

// Every registered path must be dispatchable, so the routing table and the
// dispatch switch cannot drift apart.
func TestEveryRegisteredStoreVerbIsNamedInTheRoutingTable(t *testing.T) {
	names := make(map[string]bool)
	for _, name := range credentialGroupNames["store"] {
		names[name] = true
	}
	for _, want := range []string{"copy", "copy-configure", "copy-scheduled"} {
		if !names[want] {
			t.Fatalf("store verb %q missing from credentialGroupNames", want)
		}
	}
	for _, path := range RegisteredCommandPaths() {
		if !strings.HasPrefix(path, "credentials store ") {
			continue
		}
		verb := strings.TrimPrefix(path, "credentials store ")
		if !names[verb] {
			t.Fatalf("registered path %q has no entry in the routing table", path)
		}
	}
}
