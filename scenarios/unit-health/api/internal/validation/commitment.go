package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"unit-health/internal/adapters"
)

// CoverageCommitmentRequest selects which coverage-commitment edits Unit Health
// may plan for one scenario.
type CoverageCommitmentRequest struct {
	// Ratchet raises existing coverage.committed floors to the measured
	// coverage they have earned (see RatchetedCommitment).
	Ratchet bool
	// NativeGates maps a required role's match path (for example "ui") to the
	// native runner thresholds found in that workspace. A class without a
	// commitment adopts one at its measured baseline, so the native gate can
	// be deleted without losing regression protection.
	NativeGates map[string]map[string]float64
}

// CoverageCommitmentChange is one policy class whose commitment changes.
type CoverageCommitmentChange struct {
	PolicyClass string
	Workspace   string
	Artifact    string
	Measured    map[string]float64
	From        map[string]float64
	To          map[string]float64
}

// CoverageCommitmentPlan is Unit Health's proposed edit to the scenario's
// `.vrooli/testing.json`, plus which workspaces' native gates are owned by a
// commitment once it is applied.
type CoverageCommitmentPlan struct {
	TestingPath string
	Before      string
	After       string
	Changes     []CoverageCommitmentChange
	// GateOwned lists role match paths whose class has a commitment after the
	// plan; only their native thresholds may be removed.
	GateOwned map[string]bool
	Messages  []string
}

// PlanCoverageCommitments computes coverage.committed edits from the coverage
// artifacts already on disk. It never lowers a floor and never commits a
// metric it cannot measure; a workspace without a readable artifact keeps its
// current gate and gets a message instead.
func PlanCoverageCommitments(scenarioRoot string, req CoverageCommitmentRequest) (CoverageCommitmentPlan, error) {
	plan := CoverageCommitmentPlan{TestingPath: filepath.Join(scenarioRoot, ".vrooli", "testing.json"), GateOwned: map[string]bool{}}
	profile, _, ok, findings := loadUnitPolicyProfile("", scenarioRoot, "")
	if !ok {
		if len(findings) > 0 {
			return plan, fmt.Errorf("unit.policy_profile unreadable: %s", findings[0].Observed)
		}
		return plan, nil
	}
	raw, err := os.ReadFile(plan.TestingPath)
	if err != nil {
		return plan, err
	}
	plan.Before, plan.After = string(raw), string(raw)

	proposed := map[string]map[string]float64{}
	roles := append([]unitPolicyRequiredRole(nil), profile.RequiredRoles...)
	sort.SliceStable(roles, func(i, j int) bool { return roles[i].Match.Path < roles[j].Match.Path })
	for _, role := range roles {
		class, exists := profile.PolicyClasses[role.PolicyClass]
		if !exists || role.Match.Path == "" {
			continue
		}
		native, hasNative := req.NativeGates[role.Match.Path]
		committed := len(class.Coverage.Committed) > 0
		if committed {
			plan.GateOwned[role.Match.Path] = true
		}
		// A class adopts a commitment when a native gate moves into the
		// policy, or when the ratchet runs for a class gated by default.
		adoptDefault := !committed && !hasNative && req.Ratchet && commitmentExpected(class)
		adopt := (!committed && hasNative) || adoptDefault
		if !adopt && !(committed && req.Ratchet) {
			continue
		}
		if adoptDefault {
			native = map[string]float64{}
			for _, metric := range adapters.CoverageMetricNames {
				native[metric] = 0
			}
		}
		wsRoot := filepath.Join(scenarioRoot, filepath.FromSlash(role.Match.Path))
		artifacts := adapters.DefaultCoverageArtifacts(normalizeLanguage(class.Language, wsRoot))
		report, ok := adapters.ReadCoverage(wsRoot, artifacts)
		if !ok {
			plan.Messages = append(plan.Messages, fmt.Sprintf("%s: no coverage artifact under %s; run the coverage command first. The current gate is unchanged.", role.Match.Path, wsRoot))
			continue
		}
		change := CoverageCommitmentChange{
			PolicyClass: role.PolicyClass,
			Workspace:   role.Match.Path,
			Artifact:    describeArtifact(wsRoot, artifacts),
			Measured:    map[string]float64{},
			From:        class.Coverage.Committed,
			To:          map[string]float64{},
		}
		target := class.Coverage.MinimumPercent
		if adopt {
			// Adopt the native gate's metrics at the measured baseline, capped
			// at the policy target (or a stricter native value).
			for metric, nativeValue := range native {
				total, ok := report.Totals[metric]
				if !ok || !knownCoverageMetric(metric) {
					continue
				}
				change.Measured[metric] = round2(total.Percent())
				change.To[metric] = RatchetedCommitment(0, total.Percent(), math.Max(target, nativeValue))
			}
			if len(change.To) == 0 {
				plan.Messages = append(plan.Messages, fmt.Sprintf("%s: the coverage artifact reports none of the native threshold metrics; the native gate is kept.", role.Match.Path))
				continue
			}
		} else {
			for metric, floor := range class.Coverage.Committed {
				change.To[metric] = floor
				if total, ok := report.Totals[metric]; ok {
					change.Measured[metric] = round2(total.Percent())
					change.To[metric] = RatchetedCommitment(floor, total.Percent(), target)
				}
			}
			if sameCommitment(change.From, change.To) {
				continue
			}
		}
		if prior, seen := proposed[role.PolicyClass]; seen {
			// One class serving several workspaces commits to the weakest
			// measurement so no workspace starts red.
			for metric, value := range prior {
				if current, ok := change.To[metric]; !ok || value < current {
					change.To[metric] = value
				}
			}
		}
		proposed[role.PolicyClass] = change.To
		plan.GateOwned[role.Match.Path] = true
		plan.Changes = append(plan.Changes, change)
	}
	classes := make([]string, 0, len(proposed))
	for class := range proposed {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	for _, class := range classes {
		next, err := SetCoverageCommitment(plan.After, class, proposed[class])
		if err != nil {
			return plan, err
		}
		plan.After = next
	}
	return plan, nil
}

func sameCommitment(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for metric, value := range a {
		if other, ok := b[metric]; !ok || other != value {
			return false
		}
	}
	return true
}

func describeArtifact(root string, artifacts []adapters.Artifact) string {
	for _, artifact := range artifacts {
		path := artifact.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if info, err := os.Stat(path); err == nil {
			return fmt.Sprintf("%s (written %s)", path, info.ModTime().UTC().Format(time.RFC3339))
		}
	}
	return root
}

// FormatCommitment renders a commitment in the canonical single-line JSON
// form used in testing.json, metrics in reporting order.
func FormatCommitment(committed map[string]float64) string {
	parts := make([]string, 0, len(committed))
	for _, metric := range committedMetricOrder(committed) {
		parts = append(parts, strconv.Quote(metric)+": "+strconv.FormatFloat(committed[metric], 'f', -1, 64))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// SetCoverageCommitment writes `coverage.committed` for one policy class into
// testing.json text, editing only that member so the rest of the file keeps
// its formatting.
func SetCoverageCommitment(doc, policyClass string, committed map[string]float64) (string, error) {
	src := []byte(doc)
	if !json.Valid(src) {
		return doc, fmt.Errorf("testing.json is not valid JSON")
	}
	path := []string{"unit", "policy_profile", "policy_classes", policyClass, "coverage"}
	open, closeAt := skipJSONSpace(src, 0), len(src)-1
	for _, key := range path {
		member, ok := findJSONMember(src, open, key)
		if !ok {
			return doc, fmt.Errorf("testing.json has no %s", strings.Join(path, "."))
		}
		if src[member.valueStart] != '{' {
			return doc, fmt.Errorf("testing.json %s is not an object", key)
		}
		open, closeAt = member.valueStart, member.valueEnd-1
	}
	value := FormatCommitment(committed)
	if member, ok := findJSONMember(src, open, "committed"); ok {
		return string(src[:member.valueStart]) + value + string(src[member.valueEnd:]), nil
	}
	members := jsonMembers(src, open)
	if len(members) == 0 {
		return string(src[:open+1]) + `"committed": ` + value + string(src[closeAt:]), nil
	}
	last := members[len(members)-1]
	separator := ", "
	if bytes.IndexByte(src[open:members[0].keyStart], '\n') >= 0 {
		lineStart := bytes.LastIndexByte(src[:members[0].keyStart], '\n') + 1
		separator = ",\n" + string(src[lineStart:members[0].keyStart])
	}
	return string(src[:last.valueEnd]) + separator + `"committed": ` + value + string(src[last.valueEnd:]), nil
}

type jsonMember struct {
	key        string
	keyStart   int
	valueStart int
	valueEnd   int
}

func findJSONMember(src []byte, open int, key string) (jsonMember, bool) {
	for _, member := range jsonMembers(src, open) {
		if member.key == key {
			return member, true
		}
	}
	return jsonMember{}, false
}

// jsonMembers lists the members of the (already validated) JSON object whose
// opening brace is at open, with byte offsets into src.
func jsonMembers(src []byte, open int) []jsonMember {
	var members []jsonMember
	i := skipJSONSpace(src, open+1)
	for i < len(src) && src[i] != '}' {
		keyEnd := jsonValueEnd(src, i)
		var key string
		_ = json.Unmarshal(src[i:keyEnd], &key)
		valueStart := skipJSONSpace(src, skipJSONSpace(src, keyEnd)+1)
		valueEnd := jsonValueEnd(src, valueStart)
		members = append(members, jsonMember{key: key, keyStart: i, valueStart: valueStart, valueEnd: valueEnd})
		i = skipJSONSpace(src, valueEnd)
		if i < len(src) && src[i] == ',' {
			i = skipJSONSpace(src, i+1)
		}
	}
	return members
}

// jsonValueEnd returns the offset just past the JSON value starting at i.
func jsonValueEnd(src []byte, i int) int {
	switch src[i] {
	case '"':
		for j := i + 1; j < len(src); j++ {
			if src[j] == '\\' {
				j++
			} else if src[j] == '"' {
				return j + 1
			}
		}
		return len(src)
	case '{', '[':
		depth := 0
		for j := i; j < len(src); j++ {
			switch src[j] {
			case '"':
				j = jsonValueEnd(src, j) - 1
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return j + 1
				}
			}
		}
		return len(src)
	default:
		j := i
		for j < len(src) && !strings.ContainsRune(",}] \t\r\n", rune(src[j])) {
			j++
		}
		return j
	}
}

func skipJSONSpace(src []byte, i int) int {
	for i < len(src) && strings.ContainsRune(" \t\r\n", rune(src[i])) {
		i++
	}
	return i
}
