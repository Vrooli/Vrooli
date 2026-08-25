package vroolicli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/credentialspec"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

type recoveryStatus struct {
	ReceiptExists bool       `json:"receipt_exists"`
	ExportedAt    *time.Time `json:"exported_at"`
	EntryCount    int        `json:"entry_count"`
	Uncovered     []string   `json:"uncovered"`
	// RequiredAbsent is deliberately separate from Uncovered. Uncovered means
	// a value exists locally but is not in the latest receipt; the operator
	// action is to export again. RequiredAbsent means the authority has no
	// value at all; the operator action is to provision one.
	RequiredAbsent []string `json:"required_absent"`
	// RequiredAbsentDetails carries the descriptor's explanation without
	// changing the stable address-only list consumed by existing callers.
	RequiredAbsentDetails []recoveryGapDetail `json:"required_absent_details"`
	Path                  string              `json:"path,omitempty"`
	// RootCopy remains present as null when no receipt exists so consumers can
	// distinguish an absent copy from an omitted field in older output.
	RootCopy                 *securestore.CopyStatus `json:"root_copy"`
	RootCopyIssues           []string                `json:"root_copy_issues,omitempty"`
	Basis                    string                  `json:"basis"`
	ManagedInstancesIncluded bool                    `json:"managed_instances_included"`
}

type recoveryGapDetail struct {
	Address     string `json:"address"`
	Description string `json:"description,omitempty"`
}

func distinctCredentialCount(entries []credentialEntry) int {
	addresses := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		addresses[entry.LogicalID+":"+entry.Field] = struct{}{}
	}
	return len(addresses)
}

// credentialEntry is one declared credential and what the host currently knows
// about it. It deliberately has no value field: this whole surface exists to be
// printed, and a value must never reach an output stream.
type credentialEntry struct {
	Resource     string `json:"resource"`
	Env          string `json:"env"`
	LogicalID    string `json:"logical_id"`
	Field        string `json:"field"`
	Label        string `json:"label,omitempty"`
	Description  string `json:"description,omitempty"`
	Provisioning string `json:"provisioning,omitempty"`
	DerivedFrom  string `json:"derived_from,omitempty"`
	Required     bool   `json:"required"`
	Configured   bool   `json:"configured"`
	State        string `json:"state"`
	Remediation  string `json:"remediation,omitempty"`
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
	status := computeRecoveryStatus(entries)
	if !status.ReceiptExists {
		fmt.Fprintf(ctx.Stdout, "\nRecovery\n  No bundle has ever been exported on this host. Every configured credential\n"+
			"  exists in exactly one place. Create one with:\n"+
			"    vrooli credentials recovery export --all --output <path>\n"+
			"  Enter the recovery passphrase at vrooli's secure prompt.\n")
		writeRequiredAbsent(ctx.Stdout, status)
		writeRootCopyStatus(ctx.Stdout, status)
		return
	}
	fmt.Fprintf(ctx.Stdout, "\nRecovery\n  Last bundle: %s (%d credential(s))\n    %s\n",
		status.ExportedAt.Local().Format("2006-01-02 15:04"), status.EntryCount, status.Path)
	if len(status.Uncovered) == 0 {
		fmt.Fprintf(ctx.Stdout, "  Every configured credential on this host is in that bundle.\n")
	} else {
		fmt.Fprintf(ctx.Stdout, "  STALE — %d configured credential(s) are not in it:\n", len(status.Uncovered))
		for _, missing := range status.Uncovered {
			fmt.Fprintf(ctx.Stdout, "    %s\n", missing)
		}
		fmt.Fprintf(ctx.Stdout, "  Re-export to cover them.\n")
	}
	writeRequiredAbsent(ctx.Stdout, status)
	writeRootCopyStatus(ctx.Stdout, status)
}

func writeRootCopyStatus(out io.Writer, status recoveryStatus) {
	if len(status.RootCopyIssues) == 0 {
		return
	}
	fmt.Fprintln(out, "  ROOT-COPY — credential-store escrow is not current:")
	for _, issue := range status.RootCopyIssues {
		fmt.Fprintf(out, "    %s\n", issue)
	}
}

func computeRecoveryStatus(entries []credentialEntry) recoveryStatus {
	status := recoveryStatus{Uncovered: []string{}, RequiredAbsent: []string{}, RequiredAbsentDetails: []recoveryGapDetail{}, Basis: "distinct_addresses", ManagedInstancesIncluded: true}
	status.RootCopy, status.RootCopyIssues = inspectCredentialStoreCopy()
	configured, requiredAbsent := classifyRecoveryEntries(entries)
	for _, entry := range requiredAbsent {
		appendRequiredAbsent(&status, entry)
	}
	stateDir, err := recoveryStateDir()
	if err != nil {
		status.Uncovered = dedupeStrings(configured)
		return status
	}
	receipt, found, err := credentialauthority.ReadRecoveryReceipt(stateDir)
	if err != nil || !found {
		status.Uncovered = dedupeStrings(configured)
		return status
	}
	exportedAt := receipt.ExportedAt
	status.ReceiptExists = true
	status.ExportedAt = &exportedAt
	status.EntryCount = len(receipt.Entries)
	status.Path = receipt.Path
	for _, entry := range entries {
		if !entry.Configured {
			continue
		}
		identity, parseErr := credentialauthority.ParseIdentity(entry.LogicalID)
		if parseErr != nil || !receipt.Covers(identity, entry.Field) {
			status.Uncovered = append(status.Uncovered, entry.LogicalID+":"+entry.Field)
		}
	}
	status.Uncovered = dedupeStrings(status.Uncovered)
	status.RequiredAbsent = dedupeStrings(status.RequiredAbsent)
	status.RequiredAbsentDetails = dedupeRecoveryDetails(status.RequiredAbsentDetails)
	return status
}

func inspectCredentialStoreCopy() (*securestore.CopyStatus, []string) {
	receiptPath, err := config.VrooliPath(repocontract.HomeKeyState, "credential-store-copy.json")
	if err != nil {
		return nil, []string{"cannot resolve credential-store copy receipt path"}
	}
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []string{"no encrypted credential-store copy receipt exists"}
		}
		return nil, []string{"encrypted credential-store copy receipt is unreadable"}
	}
	var copyStatus securestore.CopyStatus
	if err := json.Unmarshal(data, &copyStatus); err != nil || copyStatus.CopiedAt.IsZero() {
		return nil, []string{"encrypted credential-store copy receipt is invalid"}
	}
	issues := []string{}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(copyStatus.Path)), "s3://") {
		// An object-store receipt is the durable acknowledgement returned by
		// the PUT. The CLI intentionally does not issue a second GET during
		// doctor because that would needlessly materialize object-store
		// credentials on every health check.
	} else {
		if _, statErr := os.Stat(copyStatus.Path); statErr != nil {
			if os.IsNotExist(statErr) {
				issues = append(issues, "encrypted credential-store copy is missing from its receipt location")
			} else {
				issues = append(issues, "encrypted credential-store copy cannot be inspected")
			}
		}
	}
	storeStatus, err := securestore.DescribeStore()
	if err != nil || !storeStatus.Initialized {
		return &copyStatus, issues
	}
	if info, statErr := os.Stat(storeStatus.Path); statErr == nil && info.ModTime().After(copyStatus.CopiedAt) {
		issues = append(issues, "encrypted credential-store copy predates the newest credential-store write")
	}
	if generation, generationErr := securestore.StoreGeneration(storeStatus.Path); generationErr == nil && generation != copyStatus.Generation {
		issues = append(issues, "encrypted credential-store copy uses an older passphrase generation")
	}
	return &copyStatus, issues
}

func classifyRecoveryEntries(entries []credentialEntry) ([]string, []credentialEntry) {
	configured := make([]string, 0, len(entries))
	requiredAbsent := make([]credentialEntry, 0)
	for _, entry := range entries {
		if entry.Configured {
			configured = append(configured, entry.LogicalID+":"+entry.Field)
			continue
		}
		if requiredCredentialAbsent(entry) {
			requiredAbsent = append(requiredAbsent, entry)
		}
	}
	return configured, requiredAbsent
}

func requiredCredentialAbsent(entry credentialEntry) bool {
	// An unavailable provider cannot prove absence. Only the normal
	// unconfigured state is a genuine required-but-absent signal.
	return entry.Required && credentialspec.Descriptor{Provisioning: entry.Provisioning}.OperatorSupplied() && !entry.Configured && entry.State == "unconfigured"
}

func appendRequiredAbsent(status *recoveryStatus, entry credentialEntry) {
	address := entry.LogicalID + ":" + entry.Field
	status.RequiredAbsent = append(status.RequiredAbsent, address)
	status.RequiredAbsentDetails = append(status.RequiredAbsentDetails, recoveryGapDetail{
		Address: address, Description: strings.TrimSpace(entry.Description),
	})
}

func dedupeRecoveryDetails(values []recoveryGapDetail) []recoveryGapDetail {
	seen := map[string]bool{}
	out := make([]recoveryGapDetail, 0, len(values))
	for _, value := range values {
		if seen[value.Address] {
			continue
		}
		seen[value.Address] = true
		out = append(out, value)
	}
	return out
}

func writeRequiredAbsent(out io.Writer, status recoveryStatus) {
	if len(status.RequiredAbsent) == 0 {
		return
	}
	fmt.Fprintf(out, "  REQUIRED-BUT-ABSENT — %d required credential(s) have no value:\n", len(status.RequiredAbsent))
	for _, detail := range status.RequiredAbsentDetails {
		if detail.Description != "" {
			fmt.Fprintf(out, "    %s — %s\n", detail.Address, detail.Description)
			continue
		}
		fmt.Fprintf(out, "    %s\n", detail.Address)
	}
	fmt.Fprintln(out, "  Provision them before treating this host as recoverable.")
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
			Provider                 securestore.Diagnosis `json:"provider"`
			Credentials              []credentialEntry     `json:"credentials"`
			CredentialCount          int                   `json:"credential_count"`
			DeclarationSiteCount     int                   `json:"declaration_site_count"`
			InventoryBasis           string                `json:"inventory_basis"`
			ManagedInstancesIncluded bool                  `json:"managed_instances_included"`
			Recovery                 recoveryStatus        `json:"recovery"`
		}{Provider: diagnosis, Credentials: entries, CredentialCount: distinctCredentialCount(entries), DeclarationSiteCount: len(entries), InventoryBasis: "distinct_addresses", ManagedInstancesIncluded: true, Recovery: computeRecoveryStatus(entries)})
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
	if diagnosis.NativeWrap != "" {
		fmt.Fprintf(ctx.Stdout, "  Native wrap: %s\n", diagnosis.NativeWrap)
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

	fmt.Fprintf(ctx.Stdout, "\nCredential addresses (%d; basis=distinct_addresses; managed_instances_included=true)\n", distinctCredentialCount(entries))
	fmt.Fprintf(ctx.Stdout, "Declaration sites: %d (basis=declaration_sites)\n", len(entries))
	writeCredentialTable(ctx.Stdout, entries)

	unresolved := 0
	for _, entry := range entries {
		if !entry.Configured {
			unresolved++
		}
	}
	writeRecoveryStatus(ctx, entries)
	if unresolved == 0 {
		fmt.Fprintf(ctx.Stdout, "\nEvery declared credential resolves on this host.\n")
		return nil
	}

	fmt.Fprintf(ctx.Stdout, "\nUnresolved (%d) — a scenario still starts; these resources stay degraded until fixed:\n", unresolved)
	for _, entry := range entries {
		if entry.Configured {
			continue
		}
		fmt.Fprintf(ctx.Stdout, "  %s → %s\n      %s\n", entry.Resource, credentialLabelFor(entry), entry.Remediation)
	}
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
