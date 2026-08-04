package vroolicli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
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

// writeRecoveryStatus reports whether this host's credentials exist anywhere
// other than this host.
//
// It is here because the absence of a backup is silent by nature: a host that
// has never exported a bundle looks exactly like one that has, right up to the
// moment the difference is permanent and unfixable. `doctor` is the command an
// operator already runs when something is wrong, which makes it the one place
// the gap will actually be seen.
//
// It reports staleness as well as absence. A bundle taken before half these
// credentials existed is arguably worse than none, because it invites an
// operator to believe they are covered.
func writeRecoveryStatus(ctx *CommandContext, entries []credentialEntry) {
	stateDir, err := recoveryStateDir()
	if err != nil {
		return
	}
	receipt, found, err := credentialauthority.ReadRecoveryReceipt(stateDir)
	if err != nil || !found {
		fmt.Fprintf(ctx.Stdout, "\nRecovery\n  No bundle has ever been exported on this host. Every configured credential\n"+
			"  exists in exactly one place. Create one with:\n"+
			"    printf '%%s' \"$PASSPHRASE\" | vrooli credentials recovery export --all --output <path>\n")
		return
	}

	uncovered := []string{}
	for _, entry := range entries {
		if !entry.Configured {
			continue
		}
		identity, parseErr := credentialauthority.ParseIdentity(entry.LogicalID)
		if parseErr != nil {
			continue
		}
		if !receipt.Covers(identity, entry.Field) {
			uncovered = append(uncovered, entry.LogicalID+":"+entry.Field)
		}
	}

	// Deduplicate before counting, not just before printing: several resources
	// share one credential, so a raw count describes declarations rather than
	// credentials and disagrees with the list right beneath it.
	uncovered = dedupeStrings(uncovered)

	fmt.Fprintf(ctx.Stdout, "\nRecovery\n  Last bundle: %s (%d credential(s))\n    %s\n",
		receipt.ExportedAt.Local().Format("2006-01-02 15:04"), len(receipt.Entries), receipt.Path)
	if len(uncovered) == 0 {
		fmt.Fprintf(ctx.Stdout, "  Every configured credential on this host is in that bundle.\n")
		return
	}
	fmt.Fprintf(ctx.Stdout, "  STALE — %d configured credential(s) are not in it:\n", len(uncovered))
	for _, missing := range uncovered {
		fmt.Fprintf(ctx.Stdout, "    %s\n", missing)
	}
	fmt.Fprintf(ctx.Stdout, "  Re-export to cover them.\n")
}

// dedupeStrings keeps a shared credential from being listed once per resource
// that declares it.
func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// credentialLabelFor names a credential in operator-facing output.
//
// A descriptor bound to a process environment is best known by its variable,
// which is what an operator sees in a config file. One resolved directly by
// Vrooli-authored code has no variable, and printing an empty string there left
// rows reading "tunnel-manager → " with nothing after the arrow — the credential
// was named nowhere at all.
func credentialLabelFor(entry credentialEntry) string {
	if env := strings.TrimSpace(entry.Env); env != "" {
		return env
	}
	return entry.LogicalID + ":" + entry.Field
}

// credentialsDoctor explains this host's credential backend and then every
// declared credential on it. It is the one command an operator runs when a
// resource says its credential could not be read.
func credentialsDoctor(ctx *CommandContext, args []string) error {
	fs := flag.NewFlagSet("credentials doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := "text"
	checkWrites := false
	fs.StringVar(&format, "format", "text", "output format: text or json")
	// Opt-in because the probe writes to the operator's real credential store.
	// A diagnostic that mutates what it is diagnosing is the wrong default, and
	// on a keyring already in trouble the write is what raises an unlock prompt
	// nobody can answer — making `doctor` hang for the full Secret Service
	// timeout while explaining that something is hanging.
	fs.BoolVar(&checkWrites, "check-writes", false, "additionally prove a credential can be stored, by writing and removing a throwaway value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("credentials doctor accepts no positional arguments")
	}
	format = strings.TrimSpace(format)
	if format != "text" && format != "json" {
		return fmt.Errorf("credentials doctor format must be text or json")
	}

	diagnosis := securestore.Diagnose()
	if checkWrites {
		diagnosis = securestore.DiagnoseWritable()
	}
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
	if diagnosis.NativeStorageStrength != "" {
		caveat := diagnosis.NativeStorageCaveat
		if caveat != "" {
			caveat = " — " + caveat
		}
		fmt.Fprintf(ctx.Stdout, "  Storage:   %s%s\n", diagnosis.NativeStorageStrength, caveat)
	}
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
	fmt.Fprintf(ctx.Stdout, "  Writable:  %s\n", diagnosis.WriteCondition)
	if diagnosis.Explanation != "" {
		fmt.Fprintf(ctx.Stdout, "  Why:       %s\n", diagnosis.Explanation)
	}
	if diagnosis.SessionRepair != "" {
		fmt.Fprintf(ctx.Stdout, "  Repaired:  %s\n", diagnosis.SessionRepair)
	}
	if diagnosis.Fix != "" {
		fmt.Fprintf(ctx.Stdout, "  Fix:       %s\n", diagnosis.Fix)
	}
	if diagnosis.WriteExplanation != "" {
		fmt.Fprintf(ctx.Stdout, "  Write why: %s\n", diagnosis.WriteExplanation)
	}
	if diagnosis.WriteFix != "" && diagnosis.WriteCondition != "available" {
		fmt.Fprintf(ctx.Stdout, "  Write fix: %s\n", diagnosis.WriteFix)
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
	writeRecoveryStatus(ctx, entries)

	fmt.Fprintf(ctx.Stdout, "\nUnresolved (%d) — a scenario still starts; these resources stay degraded until fixed:\n", unresolved)
	for _, entry := range entries {
		if entry.Configured {
			continue
		}
		fmt.Fprintf(ctx.Stdout, "  %s → %s\n      %s\n", entry.Resource, credentialLabelFor(entry), entry.Remediation)
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
