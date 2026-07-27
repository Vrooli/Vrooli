package hygiene

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Compiled executables must never be tracked. They are build output: large,
// unmergeable, platform-specific, and instantly stale. Every instance found in
// this repo arrived the same way -- a build left a binary beside its source and
// a broad `git add` swept it in -- and together they accumulated ~164 MB of
// permanent history across ten months before anyone noticed.
//
// The per-scenario/per-resource .gitignore convention is the prevention; this
// check is the backstop, because that convention is maintained by hand and a
// single missed entry is invisible until the binary is already committed (BAS
// ignored bundle/runtime/linux-x64/runtime but missed its sibling runtimectl,
// which is exactly how that one landed).

// executableMagics are the leading bytes of the executable formats this repo
// could plausibly produce: ELF (Linux), Mach-O 64-bit both endiannesses and the
// universal/fat wrapper (macOS), and MZ (Windows PE).
var executableMagics = [][]byte{
	{0x7f, 'E', 'L', 'F'},
	{0xfe, 0xed, 0xfa, 0xce},
	{0xfe, 0xed, 0xfa, 0xcf},
	{0xce, 0xfa, 0xed, 0xfe},
	{0xcf, 0xfa, 0xed, 0xfe},
	{0xca, 0xfe, 0xba, 0xbe},
	{'M', 'Z'},
}

// binaryCheckSkipExts are extensions that cannot be a stray build output. This
// is a speed filter over ~41k tracked files, not a correctness boundary: a
// compiled binary normally has no extension or a platform one (.exe, .so), none
// of which appear here.
var binaryCheckSkipExts = map[string]struct{}{
	".go": {}, ".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {}, ".mjs": {}, ".cjs": {},
	".json": {}, ".yaml": {}, ".yml": {}, ".toml": {}, ".md": {}, ".txt": {},
	".sh": {}, ".bash": {}, ".py": {}, ".sql": {}, ".proto": {}, ".html": {},
	".css": {}, ".scss": {}, ".svg": {}, ".png": {}, ".jpg": {}, ".jpeg": {},
	".gif": {}, ".webp": {}, ".ico": {}, ".mp3": {}, ".mp4": {}, ".wav": {},
	".glb": {}, ".gltf": {}, ".woff": {}, ".woff2": {}, ".ttf": {}, ".pdf": {},
	".lock": {}, ".sum": {}, ".mod": {}, ".env": {}, ".gitignore": {},
	".jsonl": {}, ".ndjson": {}, ".csv": {}, ".xml": {}, ".pb": {}, ".pyi": {},
}

// checkTrackedBinaries fails when git tracks a compiled executable.
func (s Service) checkTrackedBinaries(report *Report) {
	root := report.Root
	tracked, err := trackedFiles(root)
	if err != nil {
		// Not a git checkout (or git unavailable): nothing to assert. This is a
		// repository-state check, not a filesystem one, so skipping is correct.
		report.addCheck("tracked_binaries", true, SeverityInfo, "skipped: "+err.Error())
		return
	}

	var offenders []string
	for _, rel := range tracked {
		if _, skip := binaryCheckSkipExts[strings.ToLower(filepath.Ext(rel))]; skip {
			continue
		}
		if isExecutableFile(filepath.Join(root, rel)) {
			offenders = append(offenders, rel)
		}
	}
	sort.Strings(offenders)

	if len(offenders) == 0 {
		report.addCheck("tracked_binaries", true, SeverityInfo, "no compiled binaries tracked")
		return
	}

	report.addCheck("tracked_binaries", false, SeverityError,
		pluralizeBinaries(len(offenders))+" tracked in git")
	report.addFinding(Finding{
		Severity:   SeverityError,
		Code:       "tracked_compiled_binary",
		Locations:  offenders,
		Message:    pluralizeBinaries(len(offenders)) + " tracked in git; build output must never be committed",
		Why:        "Compiled executables are large, unmergeable, platform-specific, and stale the moment their source changes. Once committed they stay in history forever, so the cost is permanent even after deletion.",
		Fixability: FixabilityGuided,
		NextActions: []Action{{
			Code:       "untrack_compiled_binary",
			Message:    "Remove each binary from the index and add its path to the owning scenario/resource .gitignore.",
			Command:    "git rm --cached <path> && echo '<path-relative-to-owner>' >> <owner>/.gitignore",
			Fixability: FixabilityGuided,
		}},
	})
}

func pluralizeBinaries(n int) string {
	if n == 1 {
		return "1 compiled binary"
	}
	return strconv.Itoa(n) + " compiled binaries"
}

// trackedFiles lists repo-relative paths that git tracks.
func trackedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "-C", root, "ls-files", "-z")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, rel := range strings.Split(string(out), "\x00") {
		if rel != "" {
			files = append(files, rel)
		}
	}
	return files, nil
}

// isExecutableFile reports whether path begins with a known executable magic.
// A deleted or unreadable file is not an offender: this check must never fail
// the repo over a path git tracks but the working tree does not have.
//
// Symlinks are skipped deliberately. What git stores for a symlink is the few
// bytes of its target path, not the target's content, so following one would
// report an 8 MB binary that is not actually in history and hand the operator a
// `git rm --cached` remediation aimed at the wrong object. (A tracked symlink
// pointing into ignored build output is its own, separate defect.)
func isExecutableFile(path string) bool {
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	var header [4]byte
	n, err := f.Read(header[:])
	if err != nil || n < 2 {
		return false
	}
	for _, magic := range executableMagics {
		if n >= len(magic) && bytes.Equal(header[:len(magic)], magic) {
			return true
		}
	}
	return false
}
