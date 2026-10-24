package stagedmessages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

type Atom struct {
	Start    int
	Old, New string
}

// EditAtoms is bounded line alignment against a retained common base. Large
// ambiguous regions remain one conservative fragment; they are not guessed.
func EditAtoms(before, after []byte) []Atom {
	a, b := splitLines(before), splitLines(after)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	endA, endB := len(a), len(b)
	for endA > prefix && endB > prefix && a[endA-1] == b[endB-1] {
		endA--
		endB--
	}
	a, b = a[prefix:endA], b[prefix:endB]
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	if len(a)*len(b) > 1_000_000 {
		return []Atom{{prefix, strings.Join(a, ""), strings.Join(b, "")}}
	}
	width := len(b) + 1
	dp := make([]int, (len(a)+1)*width)
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i*width+j] = 1 + dp[(i+1)*width+j+1]
			} else {
				dp[i*width+j] = max(dp[(i+1)*width+j], dp[i*width+j+1])
			}
		}
	}
	var out []Atom
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		if i < len(a) && j < len(b) && a[i] == b[j] {
			i++
			j++
			continue
		}
		startI, startJ := i, j
		for i < len(a) || j < len(b) {
			if i < len(a) && j < len(b) && a[i] == b[j] {
				break
			}
			if i < len(a) && (j == len(b) || dp[(i+1)*width+j] >= dp[i*width+j+1]) {
				i++
			} else {
				j++
			}
		}
		if i-startI == j-startJ && i > startI {
			for k := 0; k < i-startI; k++ {
				out = append(out, Atom{prefix + startI + k, a[startI+k], b[startJ+k]})
			}
		} else {
			out = append(out, splitUnequal(a[startI:i], b[startJ:j], prefix+startI)...)
		}
	}
	return out
}

func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	s := strings.SplitAfter(string(b), "\n")
	if s[len(s)-1] == "" {
		s = s[:len(s)-1]
	}
	return s
}

func renderPatch(ctx context.Context, path string, before, after []byte) (string, error) {
	if bytes.Equal(before, after) {
		return "", nil
	}
	dir, e := os.MkdirTemp("", "gct-staged-diff-")
	if e != nil {
		return "", fmt.Errorf("patch temporary input unavailable")
	}
	defer os.RemoveAll(dir)
	old, newPath := filepath.Join(dir, "before"), filepath.Join(dir, "after")
	if os.WriteFile(old, before, 0o600) != nil || os.WriteFile(newPath, after, 0o600) != nil {
		return "", fmt.Errorf("patch temporary input unavailable")
	}
	cmd := exec.CommandContext(ctx, "git", "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--no-color", "--", old, newPath)
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return "", fmt.Errorf("patch rendering failed: %w", err)
		}
	}
	s := strings.ReplaceAll(string(out), old, path)
	return strings.ReplaceAll(s, newPath, path), nil
}

type RetainedFile struct {
	Path, BaseBlobID, BlobID, OldMode, NewMode string
	Before, After                              []byte
	Captured                                   bool
	BytesAvailable                             bool
}
type RetainedMessage struct {
	ID, Revision, Title, Body, Lifecycle string
	Files                                []RetainedFile
}
type MessageReader interface {
	Messages(context.Context, string, *v1.StagedMessageSnapshot) ([]RetainedMessage, []*v1.Omission, error)
}

// ExactRetainedFile compares both sides and modes of a retained capture. It
// supports omitted/binary details without pretending those bytes were inspected.
func ExactRetainedFile(sf *v1.StagedMessageFile, rf RetainedFile) bool {
	return sf != nil && rf.Captured && rf.Path == sf.Path && rf.BaseBlobID == sf.OldBlobId &&
		rf.BlobID == sf.NewBlobId && normalizedMode(rf.OldMode) == sf.OldMode && normalizedMode(rf.NewMode) == sf.NewMode
}

func exactFileAtom(f *v1.StagedMessageFile) Atom {
	return Atom{-1, "exact retained file", f.NewBlobId + ":" + f.NewMode}
}

// matchRetained returns relevance separately from eligibility for reusing text.
// A whole retained message is reusable only if all its edits survive staging.
func matchRetained(c Captured, m RetainedMessage) (*v1.StagedMessageContext, map[string]map[Atom]bool, bool) {
	context := &v1.StagedMessageContext{Id: m.ID, Kind: "proposal", Reference: m.ID, Revision: m.Revision, Title: m.Title, Body: m.Body, Status: "no_content_match"}
	covered := map[string]map[Atom]bool{}
	all := len(m.Files) > 0
	files := map[string]*v1.StagedMessageFile{}
	for _, f := range c.Snapshot.Files {
		files[f.Path] = f
	}
	for _, rf := range m.Files {
		sf, ok := files[rf.Path]
		if !ok || !rf.Captured || rf.BaseBlobID != sf.OldBlobId || normalizedMode(rf.OldMode) != sf.OldMode {
			all = false
			continue
		}
		if ExactRetainedFile(sf, rf) {
			covered[rf.Path] = map[Atom]bool{exactFileAtom(sf): true}
			continue
		}
		if sf.OmissionReason != "" || !rf.BytesAvailable && rf.Before == nil && rf.After == nil {
			all = false
			continue
		}
		fb, ok := c.Bytes[rf.Path]
		if !ok {
			all = false
			continue
		}
		stage := EditAtoms(fb.Before, fb.After)
		evidence := EditAtoms(rf.Before, rf.After)
		staged := map[Atom]bool{}
		for _, a := range stage {
			staged[a] = true
		}
		for _, a := range evidence {
			if staged[a] {
				if covered[rf.Path] == nil {
					covered[rf.Path] = map[Atom]bool{}
				}
				covered[rf.Path][a] = true
			} else {
				all = false
			}
		}
		if rf.NewMode != sf.NewMode {
			all = false
		}
		// Exact equality also covers empty additions/deletions and mode-only edits.
		if len(evidence) == 0 && rf.BlobID == sf.NewBlobId && rf.NewMode == sf.NewMode {
			if covered[rf.Path] == nil {
				covered[rf.Path] = map[Atom]bool{}
			}
			covered[rf.Path][Atom{-1, "mode", rf.NewMode}] = true
		}
	}
	for p, atoms := range covered {
		if len(atoms) > 0 {
			context.MatchedPaths = append(context.MatchedPaths, p)
			if atoms[exactFileAtom(files[p])] {
				context.MatchedEdits += max(1, files[p].EditCount)
			} else {
				context.MatchedEdits += int32(len(atoms))
			}
		}
	}
	sort.Strings(context.MatchedPaths)
	context.FullMatch = all && context.MatchedEdits > 0
	if context.MatchedEdits > 0 {
		context.Status = "partial_content_match"
		if context.FullMatch {
			context.Status = "retained_edits_included"
		}
	} else {
		context.Detail = "No verified retained edit corresponds to this staged selection"
	}
	context.Detail = strings.TrimSpace(context.Detail + " Retained source lifecycle: " + m.Lifecycle)
	context.Unknowns = []string{"Content correspondence does not establish authorship"}
	return context, covered, context.FullMatch && strings.TrimSpace(m.Title) != "" && len(m.Title) <= 220 && len(m.Body) <= 32000
}

func coverage(c Captured, covered map[string]map[Atom]bool) *v1.StagedMessageCoverage {
	cv := &v1.StagedMessageCoverage{}
	for _, f := range c.Snapshot.Files {
		n := f.EditCount
		if n == 0 {
			n = 1
		}
		cv.TotalEdits += n
		count := int32(len(covered[f.Path]))
		exact := covered[f.Path][exactFileAtom(f)]
		if exact {
			count = n
		}
		if count > n {
			count = n
		}
		cv.MatchedEdits += count
		if count == n && (f.OmissionReason == "" || exact) {
			cv.CoveredPaths = append(cv.CoveredPaths, f.Path)
		} else {
			cv.UncoveredPaths = append(cv.UncoveredPaths, f.Path)
		}
		if f.OmissionReason != "" {
			cv.Omissions = append(cv.Omissions, &v1.Omission{Path: f.Path, Reason: f.OmissionReason})
		}
	}
	cv.Partial = len(cv.UncoveredPaths) > 0
	return cv
}

func suggestion(c Captured, m RetainedMessage, covered map[string]map[Atom]bool) *v1.StagedMessageSuggestion {
	cv := coverage(c, covered)
	title := m.Title
	body := m.Body
	if cv.Partial {
		title += " and related updates"
		body = strings.TrimSpace(body) + "\n\nAdditional staged edits are not described by this retained message."
	}
	return &v1.StagedMessageSuggestion{Id: m.ID, Title: title, Body: strings.TrimSpace(body), Coverage: cv, SourceIds: []string{m.ID}, Unknowns: []string{"Retained text is context, not an attribution claim"}}
}

func combineSuggestions(c Captured, ms []RetainedMessage, covered map[string]map[Atom]bool) *v1.StagedMessageSuggestion {
	var titles, body, ids []string
	for _, m := range ms {
		titles = append(titles, m.Title)
		body = append(body, "- "+m.Title)
		ids = append(ids, m.ID)
	}
	title := strings.Join(titles, "; ")
	if len(title) > 120 {
		title = fmt.Sprintf("Combine %d staged updates", len(ms))
	}
	cv := coverage(c, covered)
	if cv.Partial {
		title += " and related updates"
	}
	return &v1.StagedMessageSuggestion{Id: "combined", Title: title, Body: strings.Join(body, "\n"), Coverage: cv, SourceIds: ids, Unknowns: []string{"Combined retained titles; review wording against selected changes"}}
}

// Unique unchanged leading identifiers anchor unequal replacement regions.
// Added nearby lines must not hide an exact retained replacement. Repeated or
// absent anchors stay coarse rather than inventing a correspondence.
func splitUnequal(a, b []string, start int) []Atom {
	key := func(s string) string {
		s = strings.TrimSpace(s)
		i := 0
		for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_') {
			i++
		}
		return s[:i]
	}
	ac, bc := map[string]int{}, map[string]int{}
	bi := map[string]int{}
	for _, line := range a {
		ac[key(line)]++
	}
	for j, line := range b {
		k := key(line)
		bc[k]++
		bi[k] = j
	}
	var out []Atom
	ai, bj := 0, 0
	emit := func(x, y int) {
		if x > ai || y > bj {
			out = append(out, Atom{start + ai, strings.Join(a[ai:x], ""), strings.Join(b[bj:y], "")})
		}
	}
	for i, line := range a {
		k := key(line)
		j, ok := bi[k]
		if k == "" || !ok || ac[k] != 1 || bc[k] != 1 || j < bj {
			continue
		}
		emit(i, j)
		out = append(out, Atom{start + i, line, b[j]})
		ai, bj = i+1, j+1
	}
	emit(len(a), len(b))
	return out
}

func normalizedMode(mode string) string {
	if mode == "" {
		return "000000"
	}
	return mode
}
