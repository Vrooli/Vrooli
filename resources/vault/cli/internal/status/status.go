package status

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

type Handlers struct {
	Stdout io.Writer
	Stderr io.Writer
	Run    func(name string, args ...string) ([]byte, error)
}

type Report struct {
	Container       string `json:"container"`
	Endpoint        string `json:"endpoint,omitempty"`
	Initialized     bool   `json:"initialized"`
	Sealed          bool   `json:"sealed"`
	StorageType     string `json:"storage_type"`
	Version         string `json:"version,omitempty"`
	PersistenceSafe bool   `json:"persistence_safe"`
	Mode            string `json:"mode"`
	Runtime         string `json:"runtime"`
}

type vaultStatus struct {
	Initialized bool   `json:"initialized"`
	Sealed      bool   `json:"sealed"`
	StorageType string `json:"storage_type"`
	Version     string `json:"version"`
}

func Default() *Handlers {
	return &Handlers{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Run:    runCommand,
	}
}

func Command(h *Handlers) cliapp.CommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.CommandGroup{
		Title: "Vault",
		Commands: []cliapp.Command{{
			Name:        "status",
			Description: "Show live Vault status without printing secrets",
			Usage:       "resource-vault status [--json]",
			Run:         h.Status,
		}},
	}
}

func (h *Handlers) Status(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOut := fs.Bool("json", false, "Print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	report, err := h.report()
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(h.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	fmt.Fprintf(h.Stdout, "Initialized: %v\n", report.Initialized)
	fmt.Fprintf(h.Stdout, "Sealed: %v\n", report.Sealed)
	fmt.Fprintf(h.Stdout, "Storage: %s\n", report.StorageType)
	fmt.Fprintf(h.Stdout, "Mode: %s\n", report.Mode)
	fmt.Fprintf(h.Stdout, "Persistence safe: %v\n", report.PersistenceSafe)
	if report.Version != "" {
		fmt.Fprintf(h.Stdout, "Version: %s\n", report.Version)
	}
	return nil
}

func (h *Handlers) report() (Report, error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("VROOLI_MANAGED_PROVIDER")), "remote-vrooli") {
		return Report{}, fmt.Errorf("remote-vrooli provider must use the scenario API; direct Vault status access is forbidden")
	}
	run := h.Run
	if run == nil {
		run = runCommand
	}
	endpoint := strings.TrimSpace(os.Getenv("VAULT_ADDR"))
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8200"
	}
	runtime := "managed-service"
	out, err := run(vaultBinary(), "status", "-format=json")
	if err != nil {
		return Report{}, fmt.Errorf("Vault status through managed endpoint %q: %w", endpoint, err)
	}
	var parsed vaultStatus
	if err := json.Unmarshal(out, &parsed); err != nil {
		return Report{}, fmt.Errorf("parse vault status: %w", err)
	}
	mode := "persistent-local"
	persistenceSafe := parsed.StorageType != "" && parsed.StorageType != "inmem"
	if !persistenceSafe {
		mode = "ephemeral-dev"
	}
	return Report{
		Container:       "managed-service",
		Endpoint:        endpoint,
		Initialized:     parsed.Initialized,
		Sealed:          parsed.Sealed,
		StorageType:     parsed.StorageType,
		Version:         parsed.Version,
		PersistenceSafe: persistenceSafe,
		Mode:            mode,
		Runtime:         runtime,
	}, nil
}

func runCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok && len(out) > 0 {
			return out, nil
		}
		return nil, err
	}
	return out, nil
}

func vaultBinary() string {
	if binary := strings.TrimSpace(os.Getenv("VROOLI_VAULT_BINARY")); binary != "" {
		return binary
	}
	if binary := strings.TrimSpace(os.Getenv("VROOLI_MANAGED_SERVICE_ARTIFACT")); binary != "" {
		return binary
	}
	return "vault"
}
