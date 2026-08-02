package securestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// keyringFixture builds a file in GNOME Keyring's textual format. The shape —
// blank line between sections, attributes in their own [entry:attributeN]
// sections, no trailing blank inside an entry — is copied from a real
// login.keyring, because a parser that only handles a shape this package
// invented proves nothing about the files it will actually be pointed at.
func keyringFixture(entries ...string) string {
	header := "[keyring]\ndisplay-name=Login\nctime=0\nmtime=1752960121\nlock-on-idle=false\nlock-after=false\n"
	return header + strings.Join(entries, "") + ""
}

func keyringEntry(section, label, secret, service, key string) string {
	return "\n[" + section + "]\n" +
		"item-type=0\n" +
		"display-name=" + label + "\n" +
		"secret=" + secret + "\n" +
		"mtime=1785434783\n" +
		"ctime=1785434783\n" +
		"\n[" + section + ":attribute0]\n" +
		"name=key\ntype=string\nvalue=" + key + "\n" +
		"\n[" + section + ":attribute1]\n" +
		"name=service\ntype=string\nvalue=" + service + "\n"
}

func writeKeyring(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "login.keyring")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestInspectHealthyKeyringReportsLoadable(t *testing.T) {
	path := writeKeyring(t, keyringFixture(
		keyringEntry("1", "GNOME Remote Desktop RDP credentials", "single-line-secret", "org.gnome.RemoteDesktop", "rdp"),
		keyringEntry("116", "Vrooli managed resource", "postgres-password", "vrooli.credentials.v1", "vrooli/postgres:password"),
	))

	report, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatalf("InspectKeyringFile: %v", err)
	}
	if !report.Loadable {
		t.Fatalf("healthy keyring reported unloadable: %+v", report.Defects)
	}
	if len(report.Defects) != 0 {
		t.Fatalf("healthy keyring reported %d defects", len(report.Defects))
	}
}

func TestInspectFindsMultiLineVrooliSecret(t *testing.T) {
	path := writeKeyring(t, keyringFixture(
		keyringEntry("1", "GNOME Remote Desktop RDP credentials", "rdp-secret", "org.gnome.RemoteDesktop", "rdp"),
		keyringEntry("10453", "Vrooli managed resource", multiLinePEM, "vrooli.credentials.v1", "vrooli/release-authority:rsa-pkcs8-v1"),
	))

	report, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatalf("InspectKeyringFile: %v", err)
	}
	if report.Loadable {
		t.Fatal("keyring with a multi-line value reported loadable")
	}
	if len(report.Defects) != 1 {
		t.Fatalf("got %d defects, want 1", len(report.Defects))
	}
	defect := report.Defects[0]
	if defect.Section != "10453" || defect.Field != "secret" {
		t.Fatalf("defect identifies the wrong field: %+v", defect)
	}
	if !defect.Repairable {
		t.Fatalf("Vrooli-owned secret reported unrepairable: %s", defect.Reason)
	}
	if defect.Service != "vrooli.credentials.v1" || defect.Key != "vrooli/release-authority:rsa-pkcs8-v1" {
		t.Fatalf("defect lost its attributes: %+v", defect)
	}
	if defect.LineCount < 4 {
		t.Fatalf("LineCount = %d, want the PEM's real span", defect.LineCount)
	}
}

// TestDefectNeverCarriesTheValue is a privacy guarantee, not a formatting
// preference: these reports reach logs and incident records, and a multi-line
// value in a keyring is a private key often enough to assume it always is.
func TestDefectNeverCarriesTheValue(t *testing.T) {
	path := writeKeyring(t, keyringFixture(
		keyringEntry("10453", "Vrooli managed resource", multiLinePEM, "vrooli.credentials.v1", "k"),
	))
	report, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatalf("InspectKeyringFile: %v", err)
	}
	body := multiLinePEM[strings.Index(multiLinePEM, "\n")+1 : strings.Index(multiLinePEM, "\n")+40]
	for _, defect := range report.Defects {
		rendered := defect.Section + defect.Field + defect.Label + defect.Service + defect.Key + defect.Reason
		if strings.Contains(rendered, body) {
			t.Fatal("defect report carried secret material")
		}
	}
}

func TestRepairRestoresLoadabilityAndPreservesValues(t *testing.T) {
	rdpSecret := "the-rdp-password"
	original := keyringFixture(
		keyringEntry("1", "GNOME Remote Desktop RDP credentials", rdpSecret, "org.gnome.RemoteDesktop", "rdp"),
		keyringEntry("10453", "Vrooli managed resource", multiLinePEM, "vrooli.credentials.v1", "vrooli/release-authority:rsa-pkcs8-v1"),
		keyringEntry("116", "Vrooli managed resource", "postgres-password", "vrooli.credentials.v1", "vrooli/postgres:password"),
	)
	path := writeKeyring(t, original)

	report, err := RepairKeyringFile(path)
	if err != nil {
		t.Fatalf("RepairKeyringFile: %v", err)
	}
	if report.Repaired != 1 {
		t.Fatalf("Repaired = %d, want 1", report.Repaired)
	}
	if !report.Loadable {
		t.Fatal("repair did not restore loadability")
	}

	after, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatalf("re-inspect: %v", err)
	}
	if !after.Loadable || len(after.Defects) != 0 {
		t.Fatalf("repaired file still defective: %+v", after.Defects)
	}

	// The unrelated entries must be untouched, byte for byte. A repair that
	// quietly reformats a neighbouring secret is the same class of bug as the
	// one being repaired.
	repaired, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read repaired: %v", err)
	}
	if !strings.Contains(string(repaired), "secret="+rdpSecret+"\n") {
		t.Fatal("repair altered the RDP entry it does not own")
	}
	if !strings.Contains(string(repaired), "secret=postgres-password\n") {
		t.Fatal("repair altered an unrelated single-line Vrooli entry")
	}

	// The repaired value must decode back to exactly the PEM that was stored.
	fields, _, err := parseKeyringFile(string(repaired))
	if err != nil {
		t.Fatalf("parse repaired: %v", err)
	}
	var found bool
	for _, field := range fields {
		if field.section != "10453" || field.name != "secret" {
			continue
		}
		found = true
		decoded, err := decodeValue(field.value)
		if err != nil {
			t.Fatalf("decode repaired value: %v", err)
		}
		if decoded != multiLinePEM {
			t.Fatal("repaired value does not decode back to the original PEM")
		}
	}
	if !found {
		t.Fatal("repaired file lost the entry")
	}
}

func TestRepairBacksUpBeforeWriting(t *testing.T) {
	original := keyringFixture(
		keyringEntry("10453", "Vrooli managed resource", multiLinePEM, "vrooli.credentials.v1", "k"),
	)
	path := writeKeyring(t, original)

	report, err := RepairKeyringFile(path)
	if err != nil {
		t.Fatalf("RepairKeyringFile: %v", err)
	}
	if report.BackupPath == "" {
		t.Fatal("repair wrote without recording a backup")
	}
	backup, err := os.ReadFile(report.BackupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backup) != original {
		t.Fatal("backup does not hold the pre-repair bytes")
	}
	info, err := os.Stat(report.BackupPath)
	if err != nil {
		t.Fatalf("stat backup: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup mode = %v, want 0600 — it holds the same secrets", info.Mode().Perm())
	}
}

// TestRepairDeclinesForeignEntries is the boundary that keeps a repair from
// becoming a second incident. Vrooli knows how Vrooli reads a value back;
// guessing that for another application is not a repair, it is a rewrite.
func TestRepairDeclinesForeignEntries(t *testing.T) {
	original := keyringFixture(
		keyringEntry("42", "Some other app", "line-one\nline-two", "com.example.other", "k"),
	)
	path := writeKeyring(t, original)

	report, err := RepairKeyringFile(path)
	if err != nil {
		t.Fatalf("RepairKeyringFile: %v", err)
	}
	if report.Repaired != 0 {
		t.Fatalf("Repaired = %d, want 0 for a foreign entry", report.Repaired)
	}
	if report.Loadable {
		t.Fatal("file with an unrepaired defect reported loadable")
	}
	if len(report.Defects) != 1 || report.Defects[0].Repairable {
		t.Fatalf("foreign defect misclassified: %+v", report.Defects)
	}
	if report.Defects[0].Reason == "" {
		t.Fatal("unrepairable defect gave the operator no reason")
	}

	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(current) != original {
		t.Fatal("repair modified a file it declined to repair")
	}
}

func TestRepairIsIdempotent(t *testing.T) {
	path := writeKeyring(t, keyringFixture(
		keyringEntry("10453", "Vrooli managed resource", multiLinePEM, "vrooli.credentials.v1", "k"),
	))

	if _, err := RepairKeyringFile(path); err != nil {
		t.Fatalf("first repair: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	report, err := RepairKeyringFile(path)
	if err != nil {
		t.Fatalf("second repair: %v", err)
	}
	if report.Repaired != 0 {
		t.Fatalf("second repair changed %d entries, want 0", report.Repaired)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("second repair modified an already-repaired file")
	}
}

// TestEncryptedKeyringIsNotADefect keeps password-protected hosts — the
// majority — from being told their keyring is broken because it is binary.
func TestEncryptedKeyringIsNotADefect(t *testing.T) {
	path := writeKeyring(t, "GnomeKeyring\n\r\x00\n\x00\x01binary payload")

	report, err := InspectKeyringFile(path)
	if err != nil {
		t.Fatalf("InspectKeyringFile: %v", err)
	}
	if !report.Loadable || len(report.Defects) != 0 {
		t.Fatalf("encrypted keyring reported defective: %+v", report)
	}
}

// TestParserAccountsForEveryByte is the precondition that lets this package
// write to a file whose format nobody documents. If the parse cannot reproduce
// the input, the repair refuses rather than guessing.
func TestParserAccountsForEveryByte(t *testing.T) {
	contents := keyringFixture(
		keyringEntry("1", "One", "a", "org.gnome.RemoteDesktop", "rdp"),
		keyringEntry("10453", "Vrooli managed resource", multiLinePEM, "vrooli.credentials.v1", "k"),
		keyringEntry("116", "Vrooli managed resource", "trailing", "vrooli.credentials.v1", "k2"),
	)
	fields, lines, err := parseKeyringFile(contents)
	if err != nil {
		t.Fatalf("parseKeyringFile: %v", err)
	}
	if len(fields) == 0 {
		t.Fatal("parser found no fields")
	}
	if rebuilt := renderKeyringFile(lines, nil); rebuilt != contents {
		t.Fatal("render without replacements did not reproduce the input")
	}
}

// TestSweepRemovesOnlyAbandonedTemporaries guards the dangerous half of the
// cleanup. A temporary that is still being written is the keyring; removing it
// would destroy the file the operator is trying to save.
func TestSweepRemovesOnlyAbandonedTemporaries(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	write := func(name string, age time.Duration) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		stamp := now.Add(-age)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
		return path
	}

	oldTemp := write("login.keyring.temp-1009390427", 48*time.Hour)
	freshTemp := write("login.keyring.temp-42", time.Minute)
	realKeyring := write("login.keyring", 48*time.Hour)
	backup := write("login.keyring.corrupt-backup", 48*time.Hour)
	unrelated := write("notes.txt", 48*time.Hour)

	removed, err := SweepAbandonedTemporaries(dir, now)
	if err != nil {
		t.Fatalf("SweepAbandonedTemporaries: %v", err)
	}
	if len(removed) != 1 || removed[0] != "login.keyring.temp-1009390427" {
		t.Fatalf("removed %v, want only the abandoned temporary", removed)
	}
	if _, err := os.Stat(oldTemp); !os.IsNotExist(err) {
		t.Fatal("abandoned temporary survived the sweep")
	}
	for _, keep := range []string{freshTemp, realKeyring, backup, unrelated} {
		if _, err := os.Stat(keep); err != nil {
			t.Fatalf("sweep removed %s, which it must never touch", filepath.Base(keep))
		}
	}
}

func TestSweepIsSafeOnACleanDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "login.keyring"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	removed, err := SweepAbandonedTemporaries(dir, time.Now())
	if err != nil {
		t.Fatalf("SweepAbandonedTemporaries: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("removed %v from a clean directory", removed)
	}
}
