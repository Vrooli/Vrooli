package vroolicli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/resources/catalog"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

// credentialEntry is one declared credential and what the host currently knows
// about it. It deliberately has no value field: this whole surface exists to be
// printed, and a value must never reach an output stream.
type credentialEntry struct {
	Resource    string `json:"resource"`
	Env         string `json:"env"`
	LogicalID   string `json:"logical_id"`
	Field       string `json:"field"`
	Label       string `json:"label,omitempty"`
	Required    bool   `json:"required"`
	Configured  bool   `json:"configured"`
	State       string `json:"state"`
	Remediation string `json:"remediation,omitempty"`
}

// collectCredentialEntries reads every resource manifest and reports the state
// of every credential it declares. A resource whose manifest cannot be loaded
// is skipped rather than fatal: a diagnostic that refuses to run because one
// manifest is broken is a diagnostic that fails when it is needed most.
func collectCredentialEntries(root string) ([]credentialEntry, error) {
	// Outside a repository there are no manifests to read. That is not an
	// error for a diagnostic — the provider half of the answer is still the
	// half the operator came for.
	if strings.TrimSpace(root) == "" {
		return nil, nil
	}
	names, err := catalog.New(root).ManifestNames()
	if err != nil {
		return nil, fmt.Errorf("discover resource manifests: %w", err)
	}
	sort.Strings(names)

	entries := []credentialEntry{}
	for _, name := range names {
		resourceManifest, err := manifestpkg.Load(manifestpkg.DefaultPath(root, name))
		if err != nil {
			continue
		}
		descriptors := resourceManifest.Credentials.All()
		if len(descriptors) == 0 {
			continue
		}
		gaps, err := resourceenv.ResolveCredentialGaps(resourceManifest)
		if err != nil {
			continue
		}
		gapByEnv := make(map[string]resourceenv.MissingCredential, len(gaps.Missing))
		for _, gap := range gaps.Missing {
			gapByEnv[gap.Env] = gap
		}
		for _, descriptor := range descriptors {
			envName := strings.TrimSpace(descriptor.Env)
			field := strings.TrimSpace(descriptor.Field)
			if field == "" {
				field = "value"
			}
			entry := credentialEntry{
				Resource:   resourceManifest.Name,
				Env:        envName,
				LogicalID:  strings.TrimSpace(descriptor.LogicalID),
				Field:      field,
				Label:      strings.TrimSpace(descriptor.Label),
				Required:   descriptor.Required,
				Configured: true,
				State:      "configured",
			}
			if gap, missing := gapByEnv[envName]; missing {
				entry.Configured = false
				entry.State = string(gap.Reason)
				entry.Remediation = gap.Remediation
			}
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// credentialsDoctor explains this host's credential backend and then every
// declared credential on it. It is the one command an operator runs when a
// resource says its credential could not be read.
func credentialsDoctor(ctx *CommandContext, args []string) error {
	format, err := outputFormatFlag("credentials doctor", args)
	if err != nil {
		return err
	}
	diagnosis := securestore.Diagnose()
	entries, err := collectCredentialEntries(ctx.Root)
	if err != nil {
		return err
	}

	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(struct {
			Provider    securestore.Diagnosis `json:"provider"`
			Credentials []credentialEntry     `json:"credentials"`
		}{Provider: diagnosis, Credentials: entries})
	}

	fmt.Fprintf(ctx.Stdout, "Credential provider\n")
	fmt.Fprintf(ctx.Stdout, "  Platform:  %s\n", diagnosis.Platform)
	fmt.Fprintf(ctx.Stdout, "  Backend:   %s\n", diagnosis.Backend)
	fmt.Fprintf(ctx.Stdout, "  Adapter:   %s\n", diagnosis.Adapter)
	// The key wrap is reported on every host that has one, because the wraps
	// are not equally strong and an operator who does not know which is active
	// cannot judge what protects their values.
	if diagnosis.KeyWrap != "" {
		fmt.Fprintf(ctx.Stdout, "  Key wrap:  %s (%s)%s\n", diagnosis.KeyWrap, diagnosis.KeyStore, keyStoreCaveat(diagnosis.KeyStore))
	} else if diagnosis.Backend == "encrypted-file" {
		fmt.Fprintf(ctx.Stdout, "  Key wrap:  none open — the store is locked or not initialized\n")
	}
	fmt.Fprintf(ctx.Stdout, "  Condition: %s\n", diagnosis.Condition)
	if diagnosis.Explanation != "" {
		fmt.Fprintf(ctx.Stdout, "  Why:       %s\n", diagnosis.Explanation)
	}
	if diagnosis.SessionRepair != "" {
		fmt.Fprintf(ctx.Stdout, "  Repaired:  %s\n", diagnosis.SessionRepair)
	}
	if diagnosis.Fix != "" {
		fmt.Fprintf(ctx.Stdout, "  Fix:       %s\n", diagnosis.Fix)
	}

	if len(entries) == 0 {
		if strings.TrimSpace(ctx.Root) == "" {
			fmt.Fprintf(ctx.Stdout, "\nRun from a Vrooli repository to also list every declared credential.\n")
			return nil
		}
		fmt.Fprintf(ctx.Stdout, "\nNo resource declares a credential.\n")
		return nil
	}

	fmt.Fprintf(ctx.Stdout, "\nDeclared credentials (%d)\n", len(entries))
	writeCredentialTable(ctx.Stdout, entries)

	unresolved := 0
	for _, entry := range entries {
		if !entry.Configured {
			unresolved++
		}
	}
	if unresolved == 0 {
		fmt.Fprintf(ctx.Stdout, "\nEvery declared credential resolves on this host.\n")
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "\nUnresolved (%d) — a scenario still starts; these resources stay degraded until fixed:\n", unresolved)
	for _, entry := range entries {
		if entry.Configured {
			continue
		}
		fmt.Fprintf(ctx.Stdout, "  %s → %s\n      %s\n", entry.Resource, entry.Env, entry.Remediation)
	}
	return nil
}

// credentialsList prints declarations and state for every resource. It never
// prints a value, so it is safe in a shared terminal or a pasted bug report.
func credentialsList(ctx *CommandContext, args []string) error {
	format, err := outputFormatFlag("credentials list", args)
	if err != nil {
		return err
	}
	entries, err := collectCredentialEntries(ctx.Root)
	if err != nil {
		return err
	}
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(entries)
	}
	if len(entries) == 0 {
		if strings.TrimSpace(ctx.Root) == "" {
			return fmt.Errorf("credentials list needs a Vrooli repository root to read resource manifests")
		}
		fmt.Fprintln(ctx.Stdout, "No resource declares a credential.")
		return nil
	}
	writeCredentialTable(ctx.Stdout, entries)
	return nil
}

func writeCredentialTable(out io.Writer, entries []credentialEntry) {
	widths := []int{len("RESOURCE"), len("VARIABLE"), len("IDENTITY"), len("FIELD"), len("REQUIRED")}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		required := "no"
		if entry.Required {
			required = "yes"
		}
		row := []string{entry.Resource, entry.Env, entry.LogicalID, entry.Field, required, entry.State}
		for index := range widths {
			if len(row[index]) > widths[index] {
				widths[index] = len(row[index])
			}
		}
		rows = append(rows, row)
	}
	header := []string{"RESOURCE", "VARIABLE", "IDENTITY", "FIELD", "REQUIRED", "STATE"}
	writeCredentialRow(out, header, widths)
	for _, row := range rows {
		writeCredentialRow(out, row, widths)
	}
}

func writeCredentialRow(out io.Writer, row []string, widths []int) {
	parts := make([]string, 0, len(row))
	for index, cell := range row {
		if index < len(widths) {
			parts = append(parts, fmt.Sprintf("%-*s", widths[index], cell))
		} else {
			parts = append(parts, cell)
		}
	}
	fmt.Fprintln(out, "  "+strings.TrimRight(strings.Join(parts, "  "), " "))
}

// outputFormatFlag parses the single --format flag shared by the read-only
// credential commands.
func outputFormatFlag(name string, args []string) (string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := "text"
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if len(fs.Args()) != 0 {
		return "", fmt.Errorf("%s accepts no positional arguments", name)
	}
	format = strings.TrimSpace(format)
	if format != "text" && format != "json" {
		return "", fmt.Errorf("%s format must be text or json", name)
	}
	return format, nil
}
