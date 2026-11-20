// gen-gateway-catalog projects the profile vocabulary and role/provider
// inventory into committed artifacts: the UI profile constant and the
// capability reference page. Run it from scenarios/ai-gateway/api.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"ai-gateway/internal/gatewaycatalog"
	"ai-gateway/internal/providers"
)

func main() {
	profilesPath := flag.String("profiles", "../config/profile-catalog.json", "profile catalog input")
	rolesPath := flag.String("roles", "../config/inference-role-catalog.json", "inference role catalog input")
	tsOut := flag.String("profiles-ts-out", "../ui/src/consts/profileCatalog.generated.ts", "generated TypeScript profile catalog")
	docsOut := flag.String("docs-out", "../docs/reference/capability-catalog.md", "generated capability reference")
	check := flag.Bool("check", false, "verify outputs without writing")
	flag.Parse()

	if err := run(*profilesPath, *rolesPath, *tsOut, *docsOut, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(profilesPath, rolesPath, tsOut, docsOut string, check bool) error {
	profiles, err := gatewaycatalog.LoadProfiles(profilesPath)
	if err != nil {
		return err
	}
	roles, err := gatewaycatalog.LoadRoles(rolesPath)
	if err != nil {
		return err
	}
	ts, err := gatewaycatalog.RenderProfilesTS(profiles)
	if err != nil {
		return err
	}
	docs, err := gatewaycatalog.RenderReference(profiles, roles, providers.DefaultProviderSpecs())
	if err != nil {
		return err
	}
	stale, err := writeArtifacts([]generatedArtifact{
		{path: tsOut, data: ts},
		{path: docsOut, data: docs},
	}, check)
	if err != nil {
		return err
	}
	if check {
		return reportStaleArtifacts(stale)
	}
	return nil
}

type generatedArtifact struct {
	path string
	data []byte
}

func writeArtifacts(artifacts []generatedArtifact, check bool) ([]string, error) {
	var stale []string
	for _, artifact := range artifacts {
		changed, err := writeOrCheck(artifact.path, artifact.data, check)
		if err != nil {
			return nil, err
		}
		if changed {
			stale = append(stale, artifact.path)
		}
	}
	return stale, nil
}

func reportStaleArtifacts(stale []string) error {
	if len(stale) > 0 {
		for _, path := range stale {
			fmt.Fprintf(os.Stderr, "stale generated artifact: %s\n", path)
		}
		return fmt.Errorf("gateway catalog stale; run `make gateway-catalog` from scenarios/ai-gateway")
	}
	return nil
}

// writeOrCheck reports whether the file content changed. In check mode it
// never writes; in write mode it only touches the file when content differs so
// regeneration is a no-op in a clean tree.
func writeOrCheck(path string, data []byte, check bool) (bool, error) {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, data) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if check {
		return true, nil
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err = os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}
