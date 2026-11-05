package manifest

import (
	"regexp"
	"strings"
)

// Evaluate is the pure decision boundary for manifest-based diff
// classification. Given a manifest and the diff produced by a skill
// run, it returns a Verdict describing whether the diff matches the
// manifest's expectations or violates them (and how).
//
// The function performs no I/O and has no time/env dependency. Every
// edge case is exhaustively table-testable in evaluator_test.go.
//
// Algorithm:
//   - For every DiffFile:
//   - If WildcardAllowed is false, the path must match at least one
//     entry in AllowedPaths. Otherwise → unexpected path mutation.
//   - For each ContentRule whose PathGlob matches the diff path:
//     every MustContain substring must appear in DiffFile.Content,
//     and every MustNotContain substring must be absent.
//   - If any DiffFile triggers a violation, Verdict.Kind =
//     VerdictUnexpectedMutation; otherwise VerdictPass.
//   - ConvergenceTarget is not checked here — it is a property of the
//     re-run loop in the validation_run worker, not the per-diff
//     verdict.
func Evaluate(m Manifest, diff []DiffFile) Verdict {
	v := Verdict{Kind: VerdictPass}
	if len(diff) == 0 {
		return v
	}
	for _, f := range diff {
		if !m.WildcardAllowed && !matchesAnyGlob(f.Path, m.AllowedPaths) {
			v.Kind = VerdictUnexpectedMutation
			v.Violations = append(v.Violations, Violation{
				Path:   f.Path,
				Reason: "path not allowed by manifest",
			})
			continue
		}
		for _, rule := range m.ContentRules {
			if !globMatch(rule.PathGlob, f.Path) {
				continue
			}
			for _, needle := range rule.MustContain {
				if needle == "" {
					continue
				}
				if !strings.Contains(f.Content, needle) {
					v.Kind = VerdictUnexpectedMutation
					v.Violations = append(v.Violations, Violation{
						Path:   f.Path,
						Reason: "content missing required substring: " + needle,
					})
				}
			}
			for _, forbidden := range rule.MustNotContain {
				if forbidden == "" {
					continue
				}
				if strings.Contains(f.Content, forbidden) {
					v.Kind = VerdictUnexpectedMutation
					v.Violations = append(v.Violations, Violation{
						Path:   f.Path,
						Reason: "content contains forbidden substring: " + forbidden,
					})
				}
			}
		}
	}
	return v
}

func matchesAnyGlob(p string, globs []string) bool {
	for _, g := range globs {
		if globMatch(g, p) {
			return true
		}
	}
	return false
}

// globMatch matches a path against a glob pattern. Semantics:
//
//	**        — matches any sequence (including `/`)
//	*         — matches any sequence of non-`/` chars
//	?         — matches a single non-`/` char
//	other     — literal match (regex-meta chars are escaped)
//
// Implemented by translating the glob to a regex anchored on both
// ends. Patterns are compiled per-call; if hot-path performance ever
// matters, cache by pattern at Service init time.
func globMatch(pattern, p string) bool {
	if pattern == "" {
		return false
	}
	re, err := compileGlob(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(p)
}

func compileGlob(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	i := 0
	for i < len(pattern) {
		c := pattern[i]
		// Recognize `**/` (matches zero or more directory segments) and
		// the trailing/standalone `**` (matches anything including /).
		if c == '*' && i+1 < len(pattern) && pattern[i+1] == '*' {
			// /**/ — collapse the preceding slash too so zero dirs is legal.
			if i+2 < len(pattern) && pattern[i+2] == '/' {
				// strip the trailing slash from output (we just wrote it
				// for the previous char) — but easier: emit a regex that
				// matches "/" plus zero+ dirs OR just nothing extra.
				// Implement by writing `(?:[^/]*/)*` and skipping the
				// trailing `/`.
				b.WriteString("(?:[^/]*/)*")
				i += 3
				continue
			}
			// Trailing or standalone **
			b.WriteString(".*")
			i += 2
			continue
		}
		switch c {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
		i++
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
