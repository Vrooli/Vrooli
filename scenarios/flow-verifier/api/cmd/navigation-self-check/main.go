// Command navigation-self-check runs the navigation kind's full
// pipeline (load → validate → verify → reconcile → codegen) against a
// scenario's own navigation.json. It avoids the API service so CI can
// gate on navigation drift without bringing up the full stack.
//
// Usage:
//
//	go run ./cmd/navigation-self-check \
//	  --contract ../ui/flow/navigation.json \
//	  --scenario ..                              \
//	  --emit                                      // write routes.generated.ts
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation"
	"flow-verifier/internal/flows/kinds/navigation/codegen"
	"flow-verifier/internal/flows/kinds/navigation/reconcile"
)

func main() {
	contractPath := flag.String("contract", "ui/flow/navigation.json", "path to navigation.json")
	scenarioRoot := flag.String("scenario", ".", "scenario root containing ui/src")
	emit := flag.Bool("emit", false, "write ui/src/routes.generated.ts")
	flag.Parse()

	if err := run(*contractPath, *scenarioRoot, *emit); err != nil {
		fmt.Fprintf(os.Stderr, "navigation-self-check: %v\n", err)
		os.Exit(1)
	}
}

func run(contractPath, scenarioRoot string, emit bool) error {
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		return fmt.Errorf("read contract: %w", err)
	}

	k, ok := kind.Get(navigation.Name)
	if !ok {
		return fmt.Errorf("navigation kind not registered")
	}

	spec, err := k.Load(raw, contractPath)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	fmt.Printf("validate: ok (%d routes)\n", len(spec.(*navigation.Spec).Graph().Contract.Routes))

	vres, err := k.Verify(context.Background(), spec)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	for _, f := range vres.Findings {
		status := "PASS"
		if !f.Passed {
			status = "FAIL"
		}
		fmt.Printf("  %s  %s — %s\n", status, f.ID, f.Message)
	}
	if !vres.Passed {
		return fmt.Errorf("verify: one or more invariants failed")
	}
	fmt.Printf("verify: ok (%d findings)\n", len(vres.Findings))

	rres, err := reconcile.Run(spec.(*navigation.Spec).Graph(), scenarioRoot)
	if err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}
	for _, f := range rres.Findings {
		loc := ""
		if f.SourceFile != "" {
			loc = fmt.Sprintf(" (%s:%d)", f.SourceFile, f.SourceLine)
		}
		fmt.Printf("  %s  %s — %s%s\n", f.Severity, f.ID, f.Message, loc)
	}
	if !rres.Passed {
		return fmt.Errorf("reconcile: errors against %s (scanned %d files)", scenarioRoot, rres.FilesScanned)
	}
	fmt.Printf("reconcile: ok (%d files scanned, %d findings)\n", rres.FilesScanned, len(rres.Findings))

	if emit {
		arts, err := k.Codegen(spec, kind.LanguageTypeScript)
		if err != nil {
			return fmt.Errorf("codegen: %w", err)
		}
		body, ok := arts.Files[codegen.DefaultTypeScriptPath]
		if !ok {
			return fmt.Errorf("codegen: missing artifact %s", codegen.DefaultTypeScriptPath)
		}
		out := filepath.Join(scenarioRoot, codegen.DefaultTypeScriptPath)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		if err := os.WriteFile(out, body, 0o644); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		fmt.Printf("codegen: wrote %s (%d bytes)\n", out, len(body))
	}

	return nil
}
