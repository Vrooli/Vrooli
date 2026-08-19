// Package pathhygiene keeps the canonical Vrooli CLI directory on the
// operator's interactive PATH exactly once, and first.
//
// # Why this exists
//
// Vrooli installs its CLI to ~/.vrooli/bin (see the vrooli_launcher
// safeguard for why it lives in user-writable space). For interactive
// shells that directory has to be on PATH, which historically meant every
// scenario's CLI installer appending its own line to ~/.bashrc:
//
//	# Added by <scenario> CLI installer
//	export PATH="$PATH:/home/<user>/.vrooli/bin"
//
// An append with no guard and no marker can only accumulate. On the
// development host this produced 105 such lines from eight different
// installers, and because ~/.bashrc is re-sourced by nested shells the
// live PATH reached 236 entries of which 17 were unique — 211 copies of
// ~/.vrooli/bin. The installers that wrote them have all since been
// retired, but nothing prevented the shape from coming back.
//
// # Why duplicates are not merely untidy
//
// PATH lookup is a linear scan that stats each directory in turn, so
// duplicates tax every process launch. Measured on the affected host with
// exec.LookPath: a hit went 4.59µs -> 8.05µs, and a *miss* went 20µs ->
// 259µs, a 12.9x penalty. Misses dominate capability probing, where the
// host requirement layer deliberately looks for tools that are absent.
//
// # Why presence is not enough — the block moves the entry to the front
//
// The obvious guard ("add it if it is missing") is wrong here, and would
// have introduced a regression. ~/.profile sources ~/.bashrc *before*
// making its own PATH edits, and both files prepend $GOPATH/bin. With a
// presence-only guard the ~/.bashrc block runs first, ~/.profile then
// prepends $GOPATH/bin ahead of it, and the guard in ~/.profile declines
// to act because the entry is technically present. A stale
// `go install ./cmd/vrooli` copy in ~/go/bin then wins every bare `vrooli`
// invocation — the same class of failure as a service unit pinned to a
// build output. The managed block therefore strips every existing
// occurrence and prepends, which makes the outcome independent of what
// other startup files do afterwards.
//
// # Scope: this safeguard fixes only what it owns
//
// Apply rewrites the region between its own markers and removes legacy
// unguarded Vrooli PATH lines. It deliberately does NOT delete duplicate
// non-Vrooli PATH entries or competing vrooli binaries found elsewhere on
// PATH — those belong to other tools and to the operator. Inspect reports
// them as notes so they are visible and actionable.
package pathhygiene

import (
	"regexp"
	"strings"
)

const (
	// BeginMarker and EndMarker delimit the managed region. Their presence
	// is what makes a rewrite a replacement instead of another append.
	BeginMarker = "# >>> vrooli managed path >>>"
	EndMarker   = "# <<< vrooli managed path <<<"

	// CanonicalBinSuffix is the home-relative canonical CLI directory.
	CanonicalBinSuffix = ".vrooli/bin"
)

// ManagedBlock is the exact text written between the markers.
//
// It is pure POSIX parameter expansion with no subprocesses: this runs on
// every shell startup, so a tr/grep/paste pipeline would put three forks in
// the interactive path. The while loop handles repeated occurrences left
// behind by other tools.
const ManagedBlock = BeginMarker + `
# Managed by the vrooli path_hygiene safeguard -- do not edit between the
# markers; "vrooli setup" rewrites this block in place.
#
# Ensures ~/.vrooli/bin appears exactly once and FIRST. Presence alone is
# not enough: ~/.profile sources ~/.bashrc before making its own PATH
# edits, so an "add if missing" guard would leave $GOPATH/bin ahead of the
# canonical CLI and a stale "go install" copy would win. Stripping then
# prepending makes the result independent of what other startup files do
# afterwards.
vrooli_bin="$HOME/` + CanonicalBinSuffix + `"
if [ -d "$vrooli_bin" ]; then
    vrooli_p=":$PATH:"
    while :; do
        case "$vrooli_p" in
            *":$vrooli_bin:"*)
                vrooli_p="${vrooli_p%%":$vrooli_bin:"*}:${vrooli_p#*":$vrooli_bin:"}" ;;
            *) break ;;
        esac
    done
    vrooli_p="${vrooli_p#:}"
    vrooli_p="${vrooli_p%:}"
    PATH="$vrooli_bin${vrooli_p:+:$vrooli_p}"
    export PATH
    unset vrooli_p
fi
unset vrooli_bin
` + EndMarker

// legacyLine matches an unguarded PATH assignment referencing the
// canonical directory, in any of the forms the retired installers wrote:
// appended or prepended, $HOME-relative or absolute, with or without
// `export`.
var legacyLine = regexp.MustCompile(`^\s*(export\s+)?PATH=.*\.vrooli/bin.*$`)

// installerCaption matches the comment an installer wrote directly above
// its PATH line ("# Vrooli CLI tools", "# Added by scenario-to-android",
// "# Elo Swipe CLI", ...). Requiring one of these shapes keeps the rewrite
// from eating an unrelated comment that happens to sit above a PATH line.
var installerCaption = regexp.MustCompile(`(?i)^\s*#\s*(added by |.*\bcli\b|vrooli )`)

// Findings describes what a rewrite of one file changed.
type Findings struct {
	LegacyLines    int  // unguarded PATH lines removed
	CaptionLines   int  // installer captions removed with them
	ReplacedBlock  bool // an existing managed block was replaced
	InsertedBlock  bool // no managed block existed; one was appended
	AlreadyCurrent bool // the file already had exactly the wanted content
}

// Changed reports whether the rewrite altered the file at all.
func (f Findings) Changed() bool { return !f.AlreadyCurrent }

// blockSentinel marks where an existing managed block sat, so its position
// survives the removal pass. Tracking a raw index would not: deleting a
// legacy line above the block shifts every index below it.
const blockSentinel = "\x00vrooli-managed-block\x00"

// Rewrite returns the managed form of a shell startup file: every legacy
// unguarded Vrooli PATH line removed, and exactly one current managed
// block. An existing block is replaced where it sits, preserving any
// position the operator chose; otherwise the block is appended, which for
// a startup file means it runs after that file's other PATH edits.
func Rewrite(content string) (string, Findings) {
	var findings Findings
	lines := strings.Split(content, "\n")

	// Pass 1: swap the existing managed block for a sentinel.
	kept := exciseBlock(lines, &findings)

	// Pass 2: drop legacy lines and the captions that introduce them.
	cleaned := make([]string, 0, len(kept))
	for i, line := range kept {
		if legacyLine.MatchString(line) {
			findings.LegacyLines++
			continue
		}
		if installerCaption.MatchString(line) && nextNonBlankIsLegacy(kept, i) {
			findings.CaptionLines++
			continue
		}
		cleaned = append(cleaned, line)
	}
	cleaned = collapseBlankRuns(cleaned)

	// Pass 3: place exactly one current block.
	out := insertBlock(cleaned, &findings)

	result := strings.Join(out, "\n")
	findings.AlreadyCurrent = result == content
	return result, findings
}

// exciseBlock replaces the managed region with a single sentinel line and
// returns the result. An unterminated begin marker is treated as running to
// end of file rather than being left behind half-managed.
func exciseBlock(lines []string, findings *Findings) []string {
	begin, end := -1, -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case BeginMarker:
			if begin < 0 {
				begin = i
			}
		case EndMarker:
			if begin >= 0 && end < 0 {
				end = i
			}
		}
	}
	if begin < 0 {
		return lines
	}
	if end < 0 {
		end = len(lines) - 1
	}
	findings.ReplacedBlock = true
	kept := append([]string{}, lines[:begin]...)
	kept = append(kept, blockSentinel)
	return append(kept, lines[end+1:]...)
}

func nextNonBlankIsLegacy(lines []string, i int) bool {
	for j := i + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "" {
			continue
		}
		return legacyLine.MatchString(lines[j])
	}
	return false
}

func collapseBlankRuns(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" && len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func insertBlock(lines []string, findings *Findings) []string {
	block := strings.Split(ManagedBlock, "\n")
	for i, line := range lines {
		if line == blockSentinel {
			out := append([]string{}, lines[:i]...)
			out = append(out, block...)
			return append(out, lines[i+1:]...)
		}
	}
	// No prior block: append after the file's other PATH edits, keeping a
	// trailing newline.
	findings.InsertedBlock = true
	out := append([]string{}, lines...)
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	out = append(out, "")
	out = append(out, block...)
	return append(out, "")
}
