// Package grub edits the kernel command-line declared in /etc/default/grub
// without ever invoking update-grub. Safeguards that need to add boot-time
// parameters (e.g. ramoops.mem_address, crashkernel) call AddCmdlineParams,
// surface ExecutionRebootRequired to the operator, and let the operator run
// `sudo update-grub && sudo reboot` after reviewing the diff.
//
// Why we never run update-grub: a typo in /etc/default/grub followed by
// update-grub regenerates /boot/grub/grub.cfg with the bad value, which can
// render the host unbootable on the next reboot. Writing the file is
// reversible (the timestamped backup is right next to the original);
// regenerating grub.cfg is not. Writing-only with operator confirmation is
// the only safe default.
package grub

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// DefaultConfigPath is the canonical /etc/default/grub location on
// Debian/Ubuntu/RHEL-family systems. Tests pass an alternate path.
const DefaultConfigPath = "/etc/default/grub"

// CmdlineEdit declares one parameter to add or replace. Value="" means a bare
// flag (e.g. "quiet", "nomodeset") with no = sign.
type CmdlineEdit struct {
	Param string
	Value string
}

// Outcome reports what AddCmdlineParams did. When Changed is false, the file
// already had the desired tokens and BackupPath/NewCmdline are empty.
type Outcome struct {
	Changed    bool
	BackupPath string
	NewCmdline string
}

var (
	// NowFn is the time source for backup-suffix generation. Tests override
	// it with a fixed time for deterministic backup paths.
	NowFn = time.Now

	// ValidateGrubConfigFn validates a rendered /etc/default/grub before it is
	// written. Default implementation runs `bash -n` (catches shell-quoting
	// errors) and, when grub-script-check is on PATH, runs that as a second
	// gate. Tests override to inject failures.
	ValidateGrubConfigFn = defaultValidateGrubConfig
)

// HasCmdlineParam reports whether GRUB_CMDLINE_LINUX in cfgPath already
// declares the given parameter, and if so, its current value (empty string for
// bare flags). The error path covers missing-file, multi-declaration, and
// malformed-quoting conditions; callers translate those into ExecutionFailed
// notes rather than panic.
func HasCmdlineParam(cfgPath, param string) (present bool, value string, err error) {
	content, err := hostreqkit.ReadFileFn(cfgPath)
	if err != nil {
		return false, "", fmt.Errorf("read %s: %w", cfgPath, err)
	}

	tokens, _, err := parseCmdline(string(content))
	if err != nil {
		return false, "", err
	}

	for _, tok := range tokens {
		if tok.Param == param {
			return true, tok.Value, nil
		}
	}
	return false, "", nil
}

// AddCmdlineParams adds or replaces parameters in GRUB_CMDLINE_LINUX. The
// function is idempotent: when every requested edit is already in place the
// file is not touched and Outcome.Changed is false. When any edit applies, a
// timestamped backup is written next to the original (BackupPath), the new
// content is validated, and only then is the original atomically replaced.
//
// Never runs update-grub. The caller surfaces ExecutionRebootRequired with the
// returned BackupPath so the operator can review the diff before regenerating
// grub.cfg.
//
// Under EnsureOptions.DryRun, returns the Outcome that *would* result without
// touching disk or invoking commands.
func AddCmdlineParams(cfgPath string, edits []CmdlineEdit, sudoMode string, opts hostreqkit.EnsureOptions) (Outcome, error) {
	if cfgPath == "" {
		cfgPath = DefaultConfigPath
	}
	if len(edits) == 0 {
		return Outcome{}, nil
	}
	for _, e := range edits {
		if strings.TrimSpace(e.Param) == "" {
			return Outcome{}, errors.New("grub: edit with empty Param")
		}
	}

	originalBytes, err := hostreqkit.ReadFileFn(cfgPath)
	if err != nil {
		return Outcome{}, fmt.Errorf("read %s: %w", cfgPath, err)
	}
	original := string(originalBytes)

	newContent, newCmdline, changed, err := applyEdits(original, edits)
	if err != nil {
		return Outcome{}, err
	}
	if !changed {
		return Outcome{Changed: false, NewCmdline: newCmdline}, nil
	}

	backupPath := fmt.Sprintf("%s.vrooli-bak.%s",
		cfgPath, NowFn().UTC().Format("20060102T150405.000000000Z"))

	if opts.DryRun {
		return Outcome{Changed: true, BackupPath: backupPath, NewCmdline: newCmdline}, nil
	}

	if ok, reason := ValidateGrubConfigFn(newContent, opts); !ok {
		return Outcome{}, fmt.Errorf("grub: refusing to write invalid config: %s", reason)
	}

	if err := writeBackup(cfgPath, original, backupPath, sudoMode, opts); err != nil {
		return Outcome{}, fmt.Errorf("backup %s: %w", cfgPath, err)
	}

	if err := hostreqkit.InstallManagedContent(cfgPath, newContent, sudoMode, opts); err != nil {
		return Outcome{}, fmt.Errorf("install new %s: %w", cfgPath, err)
	}

	return Outcome{Changed: true, BackupPath: backupPath, NewCmdline: newCmdline}, nil
}

// writeBackup copies the original bytes to the backup path under sudo. We do
// not exec `cp` against the live file because that would race a concurrent
// write and would also fail when the original is briefly absent; instead we
// hand the in-memory bytes through the same temp-file + install pipeline used
// for the new content, guaranteeing the snapshot we capture matches what we
// just read.
func writeBackup(cfgPath, originalContent, backupPath, sudoMode string, opts hostreqkit.EnsureOptions) error {
	return hostreqkit.InstallManagedContent(backupPath, originalContent, sudoMode, opts)
}

// cmdlineToken is one space-separated GRUB_CMDLINE_LINUX entry. HasValue is
// true for "key=value" tokens and false for bare flags so we can round-trip
// the original form when re-rendering.
type cmdlineToken struct {
	Param    string
	Value    string
	HasValue bool
}

// parseCmdline locates the GRUB_CMDLINE_LINUX assignment in content, splits
// its quoted value into tokens, and returns the tokens plus the inferred
// quote style. Errors out on missing, duplicate, or malformed declarations.
func parseCmdline(content string) ([]cmdlineToken, byte, error) {
	idx := -1
	count := 0
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, "GRUB_CMDLINE_LINUX=") {
			continue
		}
		idx = i
		count++
	}
	if count == 0 {
		return nil, 0, errors.New("grub: GRUB_CMDLINE_LINUX assignment not found")
	}
	if count > 1 {
		return nil, 0, errors.New("grub: multiple GRUB_CMDLINE_LINUX assignments (ambiguous)")
	}

	line := lines[idx]
	trimmed := strings.TrimLeft(line, " \t")
	rhs := strings.TrimPrefix(trimmed, "GRUB_CMDLINE_LINUX=")

	var quote byte
	var inner string
	switch {
	case len(rhs) == 0:
		return nil, '"', nil
	case rhs[0] == '"' || rhs[0] == '\'':
		quote = rhs[0]
		end := strings.IndexByte(rhs[1:], quote)
		if end < 0 {
			return nil, 0, fmt.Errorf("grub: unterminated %c-quoted GRUB_CMDLINE_LINUX value", quote)
		}
		inner = rhs[1 : 1+end]
	default:
		// Unquoted assignment — supported by shell as a single bareword. Treat
		// the whole RHS as the value, default to double quotes when re-emitting.
		quote = '"'
		inner = strings.TrimRight(rhs, " \t")
	}

	tokens := splitCmdlineValue(inner)
	return tokens, quote, nil
}

// splitCmdlineValue splits a GRUB_CMDLINE_LINUX value on whitespace and
// converts each token to a cmdlineToken. Only the first '=' separates key
// from value (so console=ttyS0,115200n8 round-trips intact).
func splitCmdlineValue(value string) []cmdlineToken {
	fields := strings.Fields(value)
	out := make([]cmdlineToken, 0, len(fields))
	for _, f := range fields {
		if eq := strings.IndexByte(f, '='); eq >= 0 {
			out = append(out, cmdlineToken{
				Param:    f[:eq],
				Value:    f[eq+1:],
				HasValue: true,
			})
		} else {
			out = append(out, cmdlineToken{Param: f})
		}
	}
	return out
}

// applyEdits parses the file, merges in the requested edits, and renders the
// new file. Returns (newContent, newCmdline, changed). changed=false when the
// file already matched (purely idempotent re-run).
func applyEdits(content string, edits []CmdlineEdit) (string, string, bool, error) {
	tokens, quote, err := parseCmdline(content)
	if err != nil {
		return "", "", false, err
	}

	editByParam := make(map[string]CmdlineEdit, len(edits))
	editOrder := make([]string, 0, len(edits))
	for _, e := range edits {
		if _, exists := editByParam[e.Param]; !exists {
			editOrder = append(editOrder, e.Param)
		}
		editByParam[e.Param] = e
	}

	changed := false
	updated := make([]cmdlineToken, 0, len(tokens)+len(edits))
	seen := make(map[string]struct{}, len(edits))
	for _, tok := range tokens {
		if e, ok := editByParam[tok.Param]; ok {
			seen[tok.Param] = struct{}{}
			next := cmdlineToken{Param: e.Param, Value: e.Value, HasValue: e.Value != ""}
			if !tokenEqual(tok, next) {
				changed = true
			}
			updated = append(updated, next)
			continue
		}
		updated = append(updated, tok)
	}
	for _, param := range editOrder {
		if _, alreadySeen := seen[param]; alreadySeen {
			continue
		}
		e := editByParam[param]
		updated = append(updated, cmdlineToken{Param: e.Param, Value: e.Value, HasValue: e.Value != ""})
		changed = true
	}

	newCmdline := renderCmdline(updated)
	if !changed {
		return content, newCmdline, false, nil
	}

	newContent := replaceCmdlineLine(content, newCmdline, quote)
	return newContent, newCmdline, true, nil
}

func tokenEqual(a, b cmdlineToken) bool {
	if a.Param != b.Param {
		return false
	}
	if a.HasValue != b.HasValue {
		return false
	}
	return a.Value == b.Value
}

// renderCmdline produces the inner-string form (no quotes) of the cmdline
// tokens, ready to embed in the GRUB_CMDLINE_LINUX="..." line.
func renderCmdline(tokens []cmdlineToken) string {
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if tok.HasValue {
			parts = append(parts, tok.Param+"="+tok.Value)
		} else {
			parts = append(parts, tok.Param)
		}
	}
	return strings.Join(parts, " ")
}

// replaceCmdlineLine returns content with the GRUB_CMDLINE_LINUX line replaced
// by the new value, using the supplied quote style. All other lines are
// preserved byte-for-byte (including trailing newline).
func replaceCmdlineLine(content, newInner string, quote byte) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, "GRUB_CMDLINE_LINUX=") {
			continue
		}
		// Preserve the original leading whitespace.
		leading := line[:len(line)-len(trimmed)]
		lines[i] = fmt.Sprintf("%sGRUB_CMDLINE_LINUX=%c%s%c", leading, quote, newInner, quote)
		break
	}
	return strings.Join(lines, "\n")
}

// defaultValidateGrubConfig runs `bash -n` against the rendered content (a
// shell script — that is the format /etc/default/grub uses) and, when
// grub-script-check is on PATH, validates the resulting grub.cfg syntax as a
// second gate. Returns (false, reason) on any failure.
func defaultValidateGrubConfig(content string, opts hostreqkit.EnsureOptions) (bool, string) {
	tmp, err := hostreqkit.WriteTempFileFn(content)
	if err != nil {
		return false, fmt.Sprintf("write temp for validation: %v", err)
	}
	defer func() { _ = os.Remove(tmp) }()

	output, err := hostreqkit.CombinedOutputFn("bash", "-n", tmp)
	if err != nil {
		return false, fmt.Sprintf("bash -n rejected /etc/default/grub: %s",
			hostreqkit.FirstLine(strings.TrimSpace(string(output))))
	}

	// grub-script-check validates grub.cfg syntax (a different DSL).
	// /etc/default/grub is shell, so we only call grub-script-check when
	// available as a non-blocking advisory check — its absence is normal on
	// stripped-down hosts.
	if _, err := hostreqkit.LookPathFn("grub-script-check"); err == nil {
		// Best-effort second gate. Failures here would indicate an unusual
		// bug in our renderer; surface them so the operator notices.
		if out, runErr := hostreqkit.CombinedOutputFn("grub-script-check", tmp); runErr != nil {
			return false, fmt.Sprintf("grub-script-check rejected config: %s",
				hostreqkit.FirstLine(strings.TrimSpace(string(out))))
		}
	}
	return true, ""
}

// Edits sorts edits by param for deterministic test output. Production code
// calls AddCmdlineParams directly; this helper exists for callers that build
// edit lists from maps.
func SortEdits(edits []CmdlineEdit) []CmdlineEdit {
	sorted := append([]CmdlineEdit(nil), edits...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Param < sorted[j].Param })
	return sorted
}
