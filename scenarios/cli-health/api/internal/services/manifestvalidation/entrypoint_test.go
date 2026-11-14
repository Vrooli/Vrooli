package manifestvalidation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEntrypointFindings_CurrentScenarioAppMainPasses(t *testing.T) {
	src := []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
`)
	if got := analyzeMainEntrypoint("cli/main.go", src); len(got) != 0 {
		t.Fatalf("current scenario-app main should pass, got %+v", got)
	}
}

func TestEntrypointFindings_LifecycleGuardMainPasses(t *testing.T) {
	src := []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintln(os.Stderr, "run through lifecycle")
		os.Exit(1)
	}

	app, err := NewApp()
	if err != nil {
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
`)
	if got := analyzeMainEntrypoint("cli/main.go", src); len(got) != 0 {
		t.Fatalf("lifecycle guard plus app delegation should pass, got %+v", got)
	}
}

func TestEntrypointFindings_DirectServerSetupFails(t *testing.T) {
	src := []byte(`package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", handleUsers)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
`)
	got := analyzeMainEntrypoint("cli/main.go", src)
	if len(got) != 1 || got[0].Code != CodeCLIMainHeavy {
		t.Fatalf("expected %s finding, got %+v", CodeCLIMainHeavy, got)
	}
	if got[0].Severity != SeverityWarning {
		t.Fatalf("severity = %s, want warning", got[0].Severity)
	}
}

func TestEntrypointFindings_NonDelegatingBusyMainFails(t *testing.T) {
	src := []byte(`package main

import "fmt"

func main() {
	one()
	two()
	three()
	four()
	five()
	fmt.Println("done")
}
`)
	got := analyzeMainEntrypoint("cli/main.go", src)
	if len(got) != 1 || got[0].Code != CodeCLIMainHeavy {
		t.Fatalf("expected %s finding, got %+v", CodeCLIMainHeavy, got)
	}
}

func TestEntrypointFindings_ReadsMainNextToManifest(t *testing.T) {
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "manifest.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "main.go"), []byte(`package main

func main() {
	one()
	two()
	three()
	four()
	five()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := entrypointFindings(filepath.Join(cliDir, "manifest.json"))
	if len(got) != 1 || got[0].Code != CodeCLIMainHeavy {
		t.Fatalf("expected %s finding, got %+v", CodeCLIMainHeavy, got)
	}
}
