package commandref

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/api-core/markedrefs"
	"github.com/vrooli/cli-core/cliapp"

	"cli-health/internal/aisearch"
)

type (
	Verdict string
	Level   string
)

const (
	VerdictValid       Verdict = "valid"
	VerdictInvalid     Verdict = "invalid"
	VerdictPartial     Verdict = "partial"
	VerdictSkipped     Verdict = "skipped"
	VerdictUnknown     Verdict = "unknown"
	VerdictUnsupported Verdict = "unsupported"

	LevelParsed                 Level = "parsed"
	LevelOwnerIdentified        Level = "owner_identified"
	LevelCommandExists          Level = "command_exists"
	LevelArgumentShapeValidated Level = "argument_shape_validated"
	LevelSkippedByQualifier     Level = "skipped_by_qualifier"
	LevelUnsupportedSyntax      Level = "unsupported_syntax"
)

type Issue struct {
	Code     string
	Message  string
	Severity string
	// Fix, when non-empty, is the byte-exact replacement command text that
	// resolves this issue. Downstream fixers apply it verbatim.
	Fix string
}

type Suggestion struct {
	Command string
	Reason  string
}

type Result struct {
	CommandText      string
	Verdict          Verdict
	Level            Level
	CanonicalCommand string
	Owner            string
	Source           string
	Issues           []Issue
	Suggestions      []Suggestion
	Guidance         []string
}

type Request struct {
	CommandText   string
	Policy        string
	Qualifiers    []string
	RefreshPolicy string
}

type Service struct {
	Discovery aisearch.DiscoverySource
	// Schemas resolves proto-derived request param schemas for manifest
	// bindings (typed literal checks under the DOCS policy). Optional: nil
	// skips typed checks.
	Schemas ParamSchemaReader
}

type refreshDiscoverySource interface {
	RefreshOwner(ctx context.Context, owner string) ([]aisearch.CommandRecord, bool, error)
}

func (s Service) Validate(ctx context.Context, req Request) Result {
	commandText := strings.TrimSpace(req.CommandText)
	res := Result{
		CommandText: commandText,
		Verdict:     VerdictUnknown,
		Level:       LevelParsed,
	}
	if commandText == "" {
		res.Verdict = VerdictInvalid
		res.Issues = append(res.Issues, Issue{Code: "empty_command", Message: "command reference is empty", Severity: "error"})
		res.Guidance = append(res.Guidance, "Provide a Vrooli-owned command reference.")
		return res
	}
	if !requiresExistence(req.Qualifiers) {
		res.Verdict = VerdictSkipped
		res.Level = LevelSkippedByQualifier
		res.Guidance = append(res.Guidance, "Reference qualifier marks this command as non-current or literal, so current-command validation was skipped.")
		return res
	}
	var tokens []string
	var err error
	if isDocsPolicy(req.Policy) {
		var groups []unquotedGroup
		var fixed string
		tokens, groups, fixed, err = tokenizeCommandDocs(commandText)
		for _, g := range groups {
			res.Issues = append(res.Issues, Issue{
				Code:     "unquoted_placeholder",
				Message:  fmt.Sprintf("placeholder %s is unquoted; %s and %s are live shell operators when pasted verbatim — wrap the placeholder in double quotes", g.Text, "`<`", "`>`"),
				Severity: "warning",
				Fix:      fixed,
			})
		}
	} else {
		tokens, err = tokenizeCommand(commandText)
	}
	if err != nil {
		res.Verdict = VerdictUnsupported
		res.Level = LevelUnsupportedSyntax
		res.Issues = append(res.Issues, Issue{Code: "unsupported_shell_syntax", Message: err.Error(), Severity: "error"})
		res.Guidance = appendUnsupportedSyntaxGuidance(res.Guidance, err)
		return res
	}
	if len(tokens) == 0 {
		res.Verdict = VerdictInvalid
		res.Issues = append(res.Issues, Issue{Code: "empty_command", Message: "command reference is empty", Severity: "error"})
		return res
	}
	if s.Discovery == nil {
		res.Verdict = VerdictUnknown
		res.Issues = append(res.Issues, Issue{Code: "catalog_unavailable", Message: "CLI catalog is not configured", Severity: "warning"})
		return res
	}

	owner := tokens[0]
	res.Owner = owner
	records, known, err := s.ownerRecords(ctx, owner)
	if err != nil {
		res.Verdict = VerdictUnknown
		res.Level = LevelOwnerIdentified
		res.Issues = append(res.Issues, Issue{Code: "catalog_error", Message: err.Error(), Severity: "warning"})
		return res
	}
	if !known {
		res.Verdict = VerdictUnknown
		res.Issues = append(res.Issues, Issue{Code: "unknown_owner", Message: fmt.Sprintf("%q is not a known Vrooli-owned CLI root", owner), Severity: "warning"})
		res.Guidance = append(res.Guidance, "Use a Vrooli scenario CLI, the top-level vrooli CLI, or mark the reference as cli[external] if it is intentionally external.")
		return res
	}
	res.Level = LevelOwnerIdentified

	match, argTokens, ok := longestMatch(tokens, records)
	if !ok && shouldRefreshOnMiss(req.RefreshPolicy) {
		if refreshed, refreshedKnown, refreshErr := s.refreshOwner(ctx, owner); refreshErr != nil {
			res.Issues = append(res.Issues, Issue{Code: "catalog_refresh_failed", Message: refreshErr.Error(), Severity: "warning"})
		} else if refreshedKnown {
			records = refreshed
			match, argTokens, ok = longestMatch(tokens, records)
			if !ok {
				res.Issues = append(res.Issues, Issue{Code: "catalog_refresh_attempted", Message: "catalog was refreshed for the command owner, but the command path was still not found", Severity: "info"})
			}
		}
	}
	if !ok {
		res.Verdict = VerdictInvalid
		res.Issues = append(res.Issues, Issue{Code: "unknown_command", Message: "command path was not found in the CLI Health catalog", Severity: "error"})
		res.Suggestions = suggest(tokens, records, 3)
		res.Guidance = append(res.Guidance, "Fix the command to a current catalog command, or use cli[future]/cli[old] only when the reference is intentionally not current.")
		return res
	}
	res.CanonicalCommand = match.FullPath
	res.Source = match.Source
	res.Level = LevelCommandExists
	if match.Args == nil || match.Source != aisearch.SourceManifest {
		res.Verdict = VerdictPartial
		res.Issues = append(res.Issues, Issue{Code: "argument_schema_unavailable", Message: "command exists, but argument shape cannot be validated from reliable metadata", Severity: "info"})
		res.Guidance = append(res.Guidance, "CLI Health proved the command path exists, but not every flag or positional argument.")
		return res
	}
	if isDocsPolicy(req.Policy) {
		s.validateDocsArgs(&res, match, argTokens)
		return res
	}
	if err := cliapp.ValidateArgs(*match.Args, argTokens); err != nil {
		res.Verdict = VerdictInvalid
		res.Issues = append(res.Issues, Issue{Code: "invalid_arguments", Message: err.Error(), Severity: "error"})
		res.Guidance = append(res.Guidance, "Fix flags and positional arguments to match the command manifest.")
		return res
	}
	res.Verdict = VerdictValid
	res.Level = LevelArgumentShapeValidated
	res.Guidance = append(res.Guidance, "Command path and argument shape validated from manifest metadata.")
	return res
}

func appendUnsupportedSyntaxGuidance(guidance []string, err error) []string {
	var syntaxErr shellSyntaxError
	if errors.As(err, &syntaxErr) && syntaxErr.Kind == "redirection" {
		return append(guidance,
			"`<` and `>` are shell redirection operators in executable shell snippets. If this was a placeholder, wrap it in double quotes (e.g. \"<session>\") so it is shell-safe and machine-checkable; if the text is not meant to run, mark it as literal/text instead.",
		)
	}
	return append(guidance, "Use a single command reference without pipes, redirects, command substitution, or chained shell syntax.")
}

func shouldRefreshOnMiss(policy string) bool {
	policy = strings.ToLower(strings.TrimSpace(policy))
	return policy == "on_miss" || policy == "command_reference_refresh_policy_on_miss"
}

func requiresExistence(qualifiers []string) bool {
	return markedrefs.RequiresExistence(markedrefs.Reference{
		Marker:     markedrefs.MarkerCLI,
		Qualifiers: qualifiers,
	})
}

func (s Service) ownerRecords(ctx context.Context, owner string) ([]aisearch.CommandRecord, bool, error) {
	scenarios, err := s.Discovery.ListScenarios(ctx)
	if err != nil {
		return nil, false, err
	}
	for _, scenario := range scenarios {
		if scenario == owner {
			records, err := s.Discovery.Discover(ctx, owner)
			return records, true, err
		}
	}
	for _, cli := range s.Discovery.ListExternalCLIs() {
		if cli.Name == owner {
			records, err := s.Discovery.DiscoverExternal(ctx, cli)
			return records, true, err
		}
	}
	return nil, false, nil
}

func (s Service) refreshOwner(ctx context.Context, owner string) ([]aisearch.CommandRecord, bool, error) {
	refresher, ok := s.Discovery.(refreshDiscoverySource)
	if !ok {
		return nil, false, nil
	}
	return refresher.RefreshOwner(ctx, owner)
}

func longestMatch(tokens []string, records []aisearch.CommandRecord) (aisearch.CommandRecord, []string, bool) {
	var best aisearch.CommandRecord
	bestLen := -1
	for _, rec := range records {
		parts := strings.Fields(rec.FullPath)
		if len(parts) == 0 || len(parts) > len(tokens) || len(parts) <= bestLen {
			continue
		}
		match := true
		for i, part := range parts {
			if tokens[i] != part {
				match = false
				break
			}
		}
		if match {
			best = rec
			bestLen = len(parts)
		}
	}
	if bestLen < 0 {
		return aisearch.CommandRecord{}, nil, false
	}
	return best, tokens[bestLen:], true
}

func suggest(tokens []string, records []aisearch.CommandRecord, limit int) []Suggestion {
	if limit <= 0 {
		return nil
	}
	query := strings.ToLower(strings.Join(tokens, " "))
	type scored struct {
		rec   aisearch.CommandRecord
		score int
	}
	var scores []scored
	for _, rec := range records {
		path := strings.ToLower(rec.FullPath)
		score := commonPrefixTokens(query, path)*10 - abs(len(strings.Fields(path))-len(tokens))
		if strings.Contains(path, strings.ToLower(tokens[len(tokens)-1])) {
			score += 2
		}
		if score > 0 {
			scores = append(scores, scored{rec: rec, score: score})
		}
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		return scores[i].rec.FullPath < scores[j].rec.FullPath
	})
	out := make([]Suggestion, 0, limit)
	for _, s := range scores {
		out = append(out, Suggestion{Command: s.rec.FullPath, Reason: "closest catalog command"})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func commonPrefixTokens(a, b string) int {
	aa := strings.Fields(a)
	bb := strings.Fields(b)
	n := len(aa)
	if len(bb) < n {
		n = len(bb)
	}
	for i := 0; i < n; i++ {
		if aa[i] != bb[i] {
			return i
		}
	}
	return n
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
