// Command genopscatalog materializes the seeded operation contracts and the
// target-capability registry to the on-disk catalog the scenario ships. It is a
// one-shot authoring aid for the agent-operations catalog; the runtime loads the
// resulting JSON via internal/opscatalog. Re-run after editing SeedOperationContracts.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opscatalog"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: genopscatalog <scenario-root>")
		os.Exit(2)
	}
	root := os.Args[1]
	must := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
	writeDoc := func(dir, name string, v any) {
		must(os.MkdirAll(dir, 0o755))
		raw, err := json.MarshalIndent(v, "", "  ")
		must(err)
		must(os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0o644))
	}

	contractsDir := filepath.Join(root, opscatalog.DirOperationContracts)
	for _, oc := range agentops.SeedOperationContracts() {
		writeDoc(contractsDir, string(oc.ID)+".json", oc)
	}
	capsDir := filepath.Join(root, "target-capabilities")
	for _, d := range agentops.TargetCapabilities() {
		writeDoc(capsDir, string(d.TargetKind)+".json", d)
	}
	fmt.Printf("wrote %d operation contracts to %s\n", len(agentops.SeedOperationContracts()), contractsDir)
}
