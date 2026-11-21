package dependencygovernance

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/mod/semver"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// exposureNotePrefix starts every note that names a locked version still
// inside an advisory. Callers (the advisory-lockfile-refresh program) read it,
// so the shape "exposed: name@version to GHSA-a, GHSA-b (fixed in x, y)" is a
// contract.
const exposureNotePrefix = "exposed: "

// advisoryFix is what Security Health knows about one advisory against one
// package: the stable versions that fix it, one per release line, and the
// versions that introduced it.
type advisoryFix struct {
	id string
	// known is false for evidence that carries no range at all; such an advisory
	// cannot be judged and is never reported as still applying.
	known      bool
	fixed      []string
	introduced []string
}

// advisoryFixes groups the evidence by package: one entry per advisory, with the
// fix and introduction versions merged across the observed versions it names.
func advisoryFixes(evidence []*governancev1.SecurityVulnerabilityEvidence) map[string][]advisoryFix {
	byPackage := map[string]map[string]*advisoryFix{}
	for _, item := range evidence {
		pkg, id := item.GetPackageName(), item.GetVulnerabilityId()
		if pkg == "" || id == "" {
			continue
		}
		if byPackage[pkg] == nil {
			byPackage[pkg] = map[string]*advisoryFix{}
		}
		fix := byPackage[pkg][id]
		if fix == nil {
			fix = &advisoryFix{id: id}
			byPackage[pkg][id] = fix
		}
		fix.known = fix.known || len(item.GetFixedRanges()) > 0 || len(item.GetAffectedRanges()) > 0
		for _, r := range item.GetFixedRanges() {
			fix.fixed = appendStable(fix.fixed, r.GetVersion(), r.GetFixed())
		}
		for _, r := range item.GetAffectedRanges() {
			fix.introduced = appendStable(fix.introduced, r.GetIntroduced())
			fix.fixed = appendStable(fix.fixed, r.GetFixed())
		}
	}
	out := make(map[string][]advisoryFix, len(byPackage))
	for pkg, advisories := range byPackage {
		for _, fix := range advisories {
			out[pkg] = append(out[pkg], *fix)
		}
		sort.Slice(out[pkg], func(i, j int) bool { return out[pkg][i].id < out[pkg][j].id })
	}
	return out
}

// appendStable adds each valid, non-prerelease, not-yet-listed version ("0" is
// OSV's "from the beginning", so it is not a version).
func appendStable(list []string, versions ...string) []string {
	for _, v := range versions {
		v = strings.TrimSpace(v)
		if v == "" || v == "0" || !semver.IsValid("v"+v) || semver.Prerelease("v"+v) != "" || containsString(list, v) {
			continue
		}
		list = append(list, v)
	}
	return list
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// fixedBy reports whether a locked version is outside the advisory: it is older
// than every introducing version, it meets the fix of its own release line, or
// it is on a newer line than any fix names (a fix that exists only on older
// lines says nothing about a newer one, so only a line above all of them is
// taken as fixed).
func (f advisoryFix) fixedBy(version string) bool {
	v := "v" + version
	if !semver.IsValid(v) {
		return true // not a version we can judge; never claim exposure we cannot show
	}
	if len(f.introduced) > 0 {
		oldest := f.introduced[0]
		for _, i := range f.introduced[1:] {
			if semver.Compare("v"+i, "v"+oldest) < 0 {
				oldest = i
			}
		}
		if semver.Compare(v, "v"+oldest) < 0 {
			return true
		}
	}
	if len(f.fixed) == 0 {
		return false
	}
	here := releaseLine(v)
	var highest [2]int
	for _, fixed := range f.fixed {
		fv := "v" + fixed
		line := releaseLine(fv)
		if line == here && semver.Compare(v, fv) >= 0 {
			return true
		}
		if lineAfter(line, highest) {
			highest = line
		}
	}
	return lineAfter(here, highest)
}

// releaseLine is what a caret range cannot cross: the major version, and for a
// 0.x version the minor too (^0.16.1 never admits 0.18).
func releaseLine(v string) [2]int {
	parts := strings.SplitN(strings.TrimPrefix(semver.Canonical(v), "v"), ".", 3)
	line := [2]int{majorNumber(parts[0]), 0}
	if line[0] == 0 && len(parts) > 1 {
		line[1] = majorNumber(parts[1])
	}
	return line
}

func lineAfter(a, b [2]int) bool { return a[0] > b[0] || a[0] == b[0] && a[1] > b[1] }

func majorNumber(major string) int {
	n := 0
	for _, c := range strings.TrimPrefix(major, "v") {
		n = n*10 + int(c-'0')
	}
	return n
}

// maxExposureIDs bounds the advisory ids one note lists.
const maxExposureIDs = 3

// exposureNotes names, per package and locked version, the advisories that
// still apply to it, so a caller learns which packages an update left behind and
// why without keeping a table of fixed versions: "fix on this line" means the
// release line the version is on has a fix the lock does not hold (a pin, the
// release-age cooldown or a stale resolution), "no fix on this line" means the
// fix needs another release line, so a manifest range change. locked maps
// package -> locked versions after the change.
func exposureNotes(locked map[string][]string, fixes map[string][]advisoryFix) []string {
	names := make([]string, 0, len(locked))
	for name := range locked {
		names = append(names, name)
	}
	sort.Strings(names)
	var notes []string
	for _, name := range names {
		for _, version := range locked[name] {
			version = strings.TrimSpace(version)
			if i := strings.IndexByte(version, '('); i >= 0 {
				version = version[:i]
			}
			var ids, fixedIn []string
			for _, fix := range fixes[name] {
				if !fix.known || fix.fixedBy(version) {
					continue
				}
				ids = append(ids, fix.id)
				for _, fixed := range fix.fixed {
					if !containsString(fixedIn, fixed) {
						fixedIn = append(fixedIn, fixed)
					}
				}
			}
			if len(ids) == 0 {
				continue
			}
			sort.Slice(fixedIn, func(i, j int) bool { return semver.Compare("v"+fixedIn[i], "v"+fixedIn[j]) < 0 })
			notes = append(notes, fmt.Sprintf("%s%s@%s to %s (%s)", exposureNotePrefix, name, version, listedIDs(ids), fixText(version, fixedIn)))
		}
	}
	return notes
}

func listedIDs(ids []string) string {
	if len(ids) <= maxExposureIDs {
		return strings.Join(ids, ", ")
	}
	return fmt.Sprintf("%s +%d more", strings.Join(ids[:maxExposureIDs], ", "), len(ids)-maxExposureIDs)
}

// fixText says where the fix is relative to the version's own release line.
func fixText(version string, fixedIn []string) string {
	if len(fixedIn) == 0 {
		return "no fixed version published"
	}
	line, own := releaseLine("v"+version), ""
	for _, fixed := range fixedIn { // ascending, so the last on the line is the newest
		if releaseLine("v"+fixed) == line {
			own = fixed
		}
	}
	if own != "" {
		return "fix on this line: " + own
	}
	return "no fix on this line; fixed in " + strings.Join(fixedIn, ", ")
}
