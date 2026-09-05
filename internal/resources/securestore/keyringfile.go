package securestore

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/config"
)

const (
	secureStoreFieldValue  = "value"
	secureStoreFieldSecret = "secret"
)

// GNOME Keyring stores a passwordless keyring in GKeyFile's textual format,
// where every value occupies exactly one line. A value carrying a real newline
// — a PEM private key is the usual way this happens — makes the file
// unparseable, and GNOME Keyring's response is to reject the entire keyring
// rather than the offending entry. Every secret in the file goes dark at the
// next login, including ones written years earlier by unrelated applications.
//
// values.go stops Vrooli from creating that state again. This file cleans up a
// host that is already in it, because the operator otherwise has no way back:
// the daemon will not load the file, so no Secret Service API can reach the
// entry, and the only tool left is a text editor and a format nobody documents.
//
// The repair is deliberately narrow. It rewrites entries Vrooli owns and
// reports everything else, because collapsing a foreign application's
// multi-line value means guessing how that application will read it back.

// keyringSectionPattern matches a section header: [keyring], [10453], or
// [10453:attribute0].
var keyringSectionPattern = regexp.MustCompile(`^\[([^\]]+)\]$`)

// keyringFieldPattern matches the start of a field. A line that does not match
// it and is not a section header is a continuation of the field above — which
// is exactly the malformation this file exists to find.
var keyringFieldPattern = regexp.MustCompile(`^([A-Za-z0-9_.:-]+)=(.*)$`)

// vrooliServicePrefix identifies an entry this package wrote. Ownership decides
// repairability: Vrooli knows how Vrooli reads a value back, and guessing that
// for another application is how a repair turns into a second incident.
const vrooliServicePrefix = "vrooli."

// KeyringDefect is one malformed entry.
//
// It never carries the value. A defect report is written to logs and incident
// records, and a multi-line value in a keyring is a private key often enough
// that the safe assumption is always.
type KeyringDefect struct {
	// Section is the keyring's own identifier for the entry, e.g. "10453".
	Section string `json:"section"`
	// Field is the malformed field, e.g. "secret".
	Field string `json:"field"`
	// Label is the entry's display-name, which is operator-facing metadata
	// rather than secret material.
	Label string `json:"label,omitempty"`
	// Service and Key are the entry's Secret Service attributes when present.
	Service string `json:"service,omitempty"`
	Key     string `json:"key,omitempty"`
	// LineCount is how many lines the value spans. One is never a defect.
	LineCount int `json:"lineCount"`
	// Repairable reports whether this package may rewrite the entry.
	Repairable bool `json:"repairable"`
	// Reason explains an unrepairable defect to the operator.
	Reason string `json:"reason,omitempty"`
}

// abandonedTempAge is how old a keyring temporary must be before this package
// will remove it. GNOME Keyring's temporaries exist for the milliseconds
// between a write and its rename, so anything older than this was abandoned by
// a write that never completed — but the margin is wide because deleting the
// temporary of a write that is still in flight would destroy the keyring the
// operator is trying to save.
const abandonedTempAge = time.Hour

// keyringTempPattern matches the temporaries GNOME Keyring leaves behind, e.g.
// login.keyring.temp-1009390427.
var keyringTempPattern = regexp.MustCompile(`\.keyring\.temp-[0-9]+$`)

// SweepAbandonedTemporaries removes keyring temporaries a failed write left
// behind, and reports what it removed.
//
// On the host that motivated this package there were 214 of them, all holding
// the same bytes, from a window when writes were failing repeatedly. They are
// harmless to GNOME Keyring, which only reads *.keyring — but they are each a
// full copy of every secret in the keyring, sitting in a directory the operator
// believes holds one file, and they make the real fault hard to see.
func SweepAbandonedTemporaries(dir string, now time.Time) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read keyring directory: %w", err)
	}
	var removed []string
	for _, entry := range entries {
		if entry.IsDir() || !keyringTempPattern.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) < abandonedTempAge {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := os.Remove(path); err != nil {
			return removed, fmt.Errorf("remove abandoned keyring temporary %s: %w", entry.Name(), err)
		}
		removed = append(removed, entry.Name())
	}
	return removed, nil
}

// KeyringReport is the result of inspecting or repairing a keyring file.
type KeyringReport struct {
	Path string `json:"path"`
	// Format is the on-disk representation: "plaintext" for the GKeyFile text
	// form this package can parse, "encrypted" for the binary form it cannot,
	// and "unknown" for anything else. It exists because Loadable is only
	// meaningful for one of them.
	Format string `json:"format,omitempty"`
	// Assessed reports whether Loadable is an actual verdict. It is false for an
	// encrypted keyring, whose bytes are opaque to this package.
	//
	// This field exists because the absence of it was a live defect: an
	// encrypted keyring returned Loadable=true by falling through to the zero
	// path, so a host whose daemon was wedged read as healthy. A diagnostic that
	// cannot see a fault must say so, never report the fault's absence.
	Assessed bool `json:"assessed"`
	// Loadable reports whether GNOME Keyring can parse this file. It is false
	// whenever any defect is present, which is the whole failure: one bad entry
	// takes the file down, not just itself. Read it only when Assessed is true.
	Loadable bool            `json:"loadable"`
	Defects  []KeyringDefect `json:"defects,omitempty"`
	// Repaired counts entries this call rewrote. Always zero for an inspection.
	Repaired int `json:"repaired"`
	// BackupPath is the copy taken before any write.
	BackupPath string `json:"backupPath,omitempty"`
	// StaleDaemon is true when the file on disk is newer than the running GNOME
	// keyring daemon, which may still hold the pre-repair state.
	StaleDaemon       bool   `json:"staleDaemon"`
	StaleDaemonDetail string `json:"staleDaemonDetail,omitempty"`
	// StaleDaemonCheck says whether the daemon comparison ran. It is "checked"
	// when a daemon start time was readable and "not-run" when this host could
	// not provide one. The stale fields stay empty in the latter case.
	StaleDaemonCheck string          `json:"staleDaemonCheck,omitempty"`
	Backups          []KeyringBackup `json:"backups,omitempty"`
	Verdict          string          `json:"verdict,omitempty"`
	VerdictReason    string          `json:"verdict_reason,omitempty"`
}

type KeyringBackup struct {
	Path       string `json:"path"`
	ModifiedAt string `json:"modified_at"`
	AgeSeconds int64  `json:"age_seconds"`
}

// keyringField is one parsed field and the exact line span it occupies, so a
// repair can replace a span and leave every other byte of the file alone.
type keyringField struct {
	section string
	name    string
	value   string
	start   int
	end     int
}

func (field keyringField) multiline() bool { return field.end > field.start }

// Keyring on-disk representations. GNOME Keyring writes the textual GKeyFile
// form only for a passwordless keyring; a passphrase-protected one is an opaque
// binary blob whose magic is "GnomeKeyring".
const (
	keyringFormatPlaintext = "plaintext"
	keyringFormatEncrypted = "encrypted"
	keyringFormatUnknown   = "unknown"

	keyringPlaintextPrefix = "[keyring]"
	keyringEncryptedMagic  = "GnomeKeyring"
)

// keyringFormat classifies a keyring file by its leading bytes. It never guesses
// from the filename: two files in the same directory routinely differ.
func keyringFormat(contents []byte) string {
	switch {
	case strings.HasPrefix(string(contents), keyringPlaintextPrefix):
		return keyringFormatPlaintext
	case strings.HasPrefix(string(contents), keyringEncryptedMagic):
		return keyringFormatEncrypted
	default:
		return keyringFormatUnknown
	}
}

// DefaultKeyringDir reports where GNOME Keyring keeps its keyring files.
func DefaultKeyringDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); dir != "" {
		return filepath.Join(dir, "keyrings"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "keyrings"), nil
}

// IsPasswordlessKeyring reports whether a keyring file uses the plaintext
// GKeyFile representation. This is the intentional high-risk state in which
// GNOME Keyring can load the login collection without a master passphrase.
// Encrypted keyrings are opaque here and return false without being treated as
// malformed.
func IsPasswordlessKeyring(path string) (bool, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read keyring file: %w", err)
	}
	return strings.HasPrefix(string(contents), "[keyring]"), nil
}

// parseKeyringFile splits a keyring file into fields with their line spans.
//
// It returns an error when re-joining the parsed spans does not reproduce the
// input byte for byte. That check is the reason this parser can be trusted to
// write: the format is undocumented, so rather than claim the parse is correct,
// the code proves it against the file in front of it and refuses to touch a
// file it does not fully account for.
func parseKeyringFile(contents string) ([]keyringField, []string, error) {
	lines := strings.Split(contents, "\n")
	var fields []keyringField
	section := ""

	for index := 0; index < len(lines); {
		line := lines[index]
		if match := keyringSectionPattern.FindStringSubmatch(line); match != nil {
			section = match[1]
			index++
			continue
		}
		match := keyringFieldPattern.FindStringSubmatch(line)
		if match == nil {
			// A blank line between sections, or a stray line the scanner did
			// not attribute to a field. The reconstruction check below is what
			// catches the second case.
			index++
			continue
		}

		start := index
		valueLines := []string{match[2]}
		next := index + 1
		for next < len(lines) &&
			!keyringSectionPattern.MatchString(lines[next]) &&
			!keyringFieldPattern.MatchString(lines[next]) {
			valueLines = append(valueLines, lines[next])
			next++
		}
		end := next - 1

		// The blank line before a section header is the format's separator, not
		// part of the value above it. The same is true of the empty string that
		// splitting a newline-terminated file leaves at the end.
		terminatesBlock := next >= len(lines) || keyringSectionPattern.MatchString(lines[next])
		if terminatesBlock && len(valueLines) > 1 && valueLines[len(valueLines)-1] == "" {
			valueLines = valueLines[:len(valueLines)-1]
			end--
		}

		fields = append(fields, keyringField{
			section: section,
			name:    match[1],
			value:   strings.Join(valueLines, "\n"),
			start:   start,
			end:     end,
		})
		index = next
	}

	if rebuilt := renderKeyringFile(lines, nil); rebuilt != contents {
		return nil, nil, fmt.Errorf("keyring parser did not account for every byte of the file; refusing to modify it")
	}
	return fields, lines, nil
}

// renderKeyringFile rebuilds the file, substituting replacement lines for the
// spans given. Spans not named in replacements are copied verbatim, so an
// untouched entry is byte-identical by construction rather than by care.
func renderKeyringFile(lines []string, replacements map[int]keyringReplacement) string {
	var out []string
	for index := 0; index < len(lines); index++ {
		replacement, ok := replacements[index]
		if !ok {
			out = append(out, lines[index])
			continue
		}
		out = append(out, replacement.line)
		index = replacement.end
	}
	return strings.Join(out, "\n")
}

type keyringReplacement struct {
	line string
	end  int
}

// keyringAttributes maps each entry to its Secret Service attributes, which
// live in their own [<entry>:attributeN] sections rather than on the entry.
func keyringAttributes(fields []keyringField) map[string]map[string]string {
	type pair struct{ name, value string }
	raw := map[string]*pair{}
	for _, field := range fields {
		if !strings.Contains(field.section, ":attribute") {
			continue
		}
		entry := raw[field.section]
		if entry == nil {
			entry = &pair{}
			raw[field.section] = entry
		}
		switch field.name {
		case "name":
			entry.name = field.value
		case secureStoreFieldValue:
			entry.value = field.value
		}
	}

	attributes := map[string]map[string]string{}
	for section, entry := range raw {
		base, _, _ := strings.Cut(section, ":")
		if attributes[base] == nil {
			attributes[base] = map[string]string{}
		}
		if entry.name != "" {
			attributes[base][entry.name] = entry.value
		}
	}
	return attributes
}

// keyringLabels maps each entry to its display-name.
func keyringLabels(fields []keyringField) map[string]string {
	labels := map[string]string{}
	for _, field := range fields {
		if field.name == "display-name" && !strings.Contains(field.section, ":attribute") {
			labels[field.section] = field.value
		}
	}
	return labels
}

// InspectKeyringFile reports whether GNOME Keyring can load a keyring file and
// which entries stop it. It never writes.
func InspectKeyringFile(path string) (KeyringReport, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return KeyringReport{Path: path}, fmt.Errorf("read keyring file: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return KeyringReport{Path: path}, fmt.Errorf("stat keyring file: %w", err)
	}
	report := KeyringReport{Path: path, Format: keyringFormat(contents)}
	addStaleDaemonReport(&report, info)
	if backups, backupErr := listKeyringBackups(filepath.Dir(path), path, time.Now()); backupErr == nil {
		report.Backups = backups
	}
	if report.Format != keyringFormatPlaintext {
		// An encrypted keyring is binary and is not this file's problem. Saying
		// so beats reporting a false defect on every password-protected host —
		// but it must not report a clean bill of health either, so Assessed
		// stays false and Loadable carries no verdict.
		return report, nil
	}
	report.Assessed = true
	report.Loadable = true

	fields, _, err := parseKeyringFile(string(contents))
	if err != nil {
		return KeyringReport{Path: path, Format: report.Format}, err
	}
	attributes := keyringAttributes(fields)
	labels := keyringLabels(fields)

	for _, field := range fields {
		if !field.multiline() {
			continue
		}
		report.Loadable = false
		report.Defects = append(report.Defects, describeKeyringDefect(field, attributes, labels))
	}
	return report, nil
}

func listKeyringBackups(dir, livePath string, now time.Time) ([]KeyringBackup, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var backups []KeyringBackup
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(livePath) || !strings.Contains(entry.Name(), "keyring") || !strings.Contains(entry.Name(), "backup") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		age := now.Sub(info.ModTime())
		if age < 0 {
			age = 0
		}
		backups = append(backups, KeyringBackup{Path: filepath.Join(dir, entry.Name()), ModifiedAt: info.ModTime().UTC().Format(time.RFC3339), AgeSeconds: int64(age / time.Second)})
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].Path < backups[j].Path })
	return backups, nil
}

// RetireKeyringBackup removes one explicitly named backup after an operator
// has decided it is no longer needed. It never accepts a directory, a glob, or
// a path that does not look like a keyring backup; callers must name the exact
// file so a cleanup cannot broaden into the whole keyring directory.
func RetireKeyringBackup(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("keyring backup path is required")
	}
	base := filepath.Base(path)
	if !strings.Contains(base, "keyring") || !strings.Contains(base, "backup") {
		return fmt.Errorf("refusing to retire non-keyring-backup path %q", path)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect keyring backup %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to retire non-regular keyring backup %q", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("retire keyring backup %q: %w", path, err)
	}
	return nil
}

func describeKeyringDefect(field keyringField, attributes map[string]map[string]string, labels map[string]string) KeyringDefect {
	entry := attributes[field.section]
	defect := KeyringDefect{
		Section:   field.section,
		Field:     field.name,
		Label:     labels[field.section],
		Service:   entry["service"],
		Key:       entry["key"],
		LineCount: field.end - field.start + 1,
	}
	switch {
	case !strings.HasPrefix(defect.Service, vrooliServicePrefix):
		defect.Reason = "entry was not written by Vrooli; collapsing it would guess how its owner reads the value back"
	case field.name != secureStoreFieldSecret:
		defect.Reason = "only a secret field is rewritten; other fields are metadata this repair does not own"
	default:
		defect.Repairable = true
	}
	return defect
}

// RepairKeyringFile rewrites the Vrooli-owned entries that stop GNOME Keyring
// from loading a keyring file, and reports the rest.
//
// The rewritten form is exactly what a current Vrooli writes — the encoding
// from values.go — so the file ends up in the state it would have been in had
// the guard existed, rather than in a third state that only a repair produces.
func RepairKeyringFile(path string) (KeyringReport, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return KeyringReport{Path: path}, fmt.Errorf("read keyring file: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return KeyringReport{Path: path}, fmt.Errorf("stat keyring file: %w", err)
	}

	report := KeyringReport{Path: path, Format: keyringFormat(contents)}
	addStaleDaemonReport(&report, info)
	if report.Format != keyringFormatPlaintext {
		return report, nil
	}
	report.Assessed = true
	report.Loadable = true

	fields, lines, err := parseKeyringFile(string(contents))
	if err != nil {
		return KeyringReport{Path: path, Format: report.Format}, err
	}
	attributes := keyringAttributes(fields)
	labels := keyringLabels(fields)

	replacements := map[int]keyringReplacement{}
	for _, field := range fields {
		if !field.multiline() {
			continue
		}
		report.Loadable = false
		defect := describeKeyringDefect(field, attributes, labels)
		report.Defects = append(report.Defects, defect)
		if !defect.Repairable {
			continue
		}
		replacements[field.start] = keyringReplacement{
			line: field.name + "=" + encodeValue(field.value),
			end:  field.end,
		}
	}
	if len(replacements) == 0 {
		return report, nil
	}

	repaired := renderKeyringFile(lines, replacements)

	// Verify before writing, not after. A repair that has to be undone by hand
	// on a file the daemon already refuses to load is worse than no repair.
	verifyFields, _, err := parseKeyringFile(repaired)
	if err != nil {
		return report, fmt.Errorf("repaired keyring did not re-parse; nothing was written: %w", err)
	}
	for _, field := range verifyFields {
		if field.multiline() {
			return report, fmt.Errorf("repaired keyring still holds a multi-line value in [%s]; nothing was written", field.section)
		}
	}
	if err := verifyKeyringValuesPreserved(fields, verifyFields); err != nil {
		return report, fmt.Errorf("%w; nothing was written", err)
	}

	backup, err := backupKeyringFile(path, contents, info.Mode().Perm())
	if err != nil {
		return report, err
	}
	report.BackupPath = backup

	if err := writeFileAtomic(path, []byte(repaired), info.Mode().Perm()); err != nil {
		return report, fmt.Errorf("write repaired keyring: %w", err)
	}
	report.Repaired = len(replacements)
	report.Loadable = true
	for index := range report.Defects {
		if report.Defects[index].Repairable {
			report.Defects[index].Reason = "repaired"
		}
	}
	return report, nil
}

// verifyKeyringValuesPreserved proves the repair changed encoding and nothing
// else: same fields, same order, and every value either untouched or the exact
// encoding of what it was.
func verifyKeyringValuesPreserved(before, after []keyringField) error {
	if len(before) != len(after) {
		return fmt.Errorf("repair changed the number of fields from %d to %d", len(before), len(after))
	}
	for index := range before {
		original, repaired := before[index], after[index]
		if original.section != repaired.section || original.name != repaired.name {
			return fmt.Errorf("repair reordered fields at position %d", index)
		}
		if original.value == repaired.value {
			continue
		}
		decoded, err := decodeValue(repaired.value)
		if err != nil {
			return fmt.Errorf("repaired value in [%s] does not decode: %w", original.section, err)
		}
		if decoded != original.value {
			return fmt.Errorf("repaired value in [%s] does not decode back to the original", original.section)
		}
	}
	return nil
}

// backupKeyringFile keeps the pre-repair bytes next to the original. It never
// overwrites an existing backup: on a host being repaired twice, the first copy
// is the one taken before anything touched the file.
func backupKeyringFile(path string, contents []byte, mode os.FileMode) (string, error) {
	return writeKeyringBackup(path+".corrupt-backup", contents, mode)
}

// BackupKeyringFileAs copies a keyring file to `<path><suffix>` with the same
// permissions, never overwriting an earlier backup. It is the one backup writer
// for every caller that must preserve a keyring before changing it, so
// listKeyringBackups can recognise every backup by construction rather than by
// guessing at suffixes.
func BackupKeyringFileAs(path, suffix string) (string, error) {
	if !strings.Contains(suffix, "backup") {
		return "", fmt.Errorf("keyring backup suffix %q must contain \"backup\" so it is listed as one", suffix)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read keyring: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat keyring: %w", err)
	}
	return writeKeyringBackup(path+suffix, contents, info.Mode().Perm())
}

func writeKeyringBackup(backup string, contents []byte, mode os.FileMode) (string, error) {
	for suffix := 0; ; suffix++ {
		candidate := backup
		if suffix > 0 {
			candidate = fmt.Sprintf("%s.%d", backup, suffix)
		}
		file, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create keyring backup: %w", err)
		}
		defer func() { _ = file.Close() }()
		if _, err := file.Write(contents); err != nil {
			return "", fmt.Errorf("write keyring backup: %w", err)
		}
		return candidate, file.Sync()
	}
}

// writeFileAtomic replaces a file through a temporary in the same directory, so
// a crash mid-write leaves the original rather than a truncated keyring.
func writeFileAtomic(path string, contents []byte, mode os.FileMode) error {
	if err := config.WriteOwnedFileAtomic(path, contents, mode); err != nil {
		return fmt.Errorf("write keyring file: %w", err)
	}
	return nil
}
