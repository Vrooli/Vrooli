package commandref

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	measures "github.com/vrooli/measures-go"

	"cli-health/internal/aisearch"
)

// Issue codes emitted only under the DOCS policy.
const (
	issuePlaceholderNameMismatch = "placeholder_name_mismatch"
	issueEnumPlaceholderMismatch = "enum_placeholder_mismatch"
	issueInvalidLiteralValue     = "invalid_literal_value"
)

// ParamSchemaReader resolves the proto-derived request param schema for a
// manifest binding. Mirrors manifestvalidation.MeasureSchemaReader — the seam
// is optional: a nil reader skips typed literal checks entirely, degrading the
// same way measure validation does when the descriptor image is absent.
type ParamSchemaReader interface {
	RequestParams(service, method string) ([]measures.ParamSchema, error)
}

// isDocsPolicy reports whether the request policy selects the DOCS validation
// branch. The Connect handler passes the proto enum name; bare "docs" is
// accepted for direct callers.
func isDocsPolicy(policy string) bool {
	p := strings.ToLower(strings.TrimSpace(policy))
	return p == "docs" || p == "command_reference_policy_docs"
}

// docsSlot is one resolved argument slot from the DOCS-mode arg walk: the
// supplied token plus the manifest slot it landed in.
type docsSlot struct {
	token    string
	slotName string       // positional name, or flag name for flag values
	flag     *cliapp.Flag // nil for positionals
}

// validateDocsArgs is the DOCS-policy counterpart of cliapp.ValidateArgs:
// structural checks mirror the runtime parser, but placeholder tokens
// (quoted "<...>" survives tokenization as <...>) are matched against manifest
// slots instead of rejected, enum alternations are compared to the effective
// vocabulary, and literal values are checked against descriptor-derived
// constraints where they exist.
func (s Service) validateDocsArgs(res *Result, match aisearch.CommandRecord, argTokens []string) {
	schema := *match.Args
	slots, issues := walkDocsArgs(schema, argTokens)
	params := s.requestParams(match.Binding)
	for _, slot := range slots {
		issues = append(issues, checkDocsSlot(slot, params)...)
	}

	res.Issues = append(res.Issues, issues...)
	for _, issue := range issues {
		if issue.Severity == "error" {
			res.Verdict = VerdictInvalid
			res.Guidance = append(res.Guidance, "Fix flags, positional arguments, and example values to match the command manifest and its proto constraints.")
			return
		}
	}
	res.Verdict = VerdictValid
	res.Level = LevelArgumentShapeValidated
	res.Guidance = append(res.Guidance, "Command path and argument shape validated from manifest metadata (DOCS policy: placeholders matched against manifest slots).")
}

// walkDocsArgs mirrors the runtime parser's token walk (flags, inline values,
// --, positional assignment, required checks) but collects (token, slot) pairs
// instead of building a RunContext. Structural violations surface as
// invalid_arguments issues with parser-equivalent messages.
func walkDocsArgs(schema cliapp.ArgSchema, args []string) ([]docsSlot, []Issue) {
	var slots []docsSlot
	var issues []Issue
	structural := func(format string, a ...any) {
		issues = append(issues, Issue{Code: "invalid_arguments", Message: fmt.Sprintf(format, a...), Severity: "error"})
	}

	seen := make(map[string]bool)
	var positionals []string
	endOfFlags := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if endOfFlags {
			positionals = append(positionals, arg)
			continue
		}
		if arg == "--" {
			endOfFlags = true
			continue
		}
		if arg == "--help" || arg == "-h" || arg == "--json" {
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}

		name := arg
		inlineValue := ""
		hasInline := false
		if eq := strings.Index(arg, "="); eq != -1 {
			name, inlineValue, hasInline = arg[:eq], arg[eq+1:], true
		}
		canonical := strings.TrimLeft(name, "-")
		flag, ok := findFlag(schema, canonical)
		if !ok {
			structural("unknown option: %s", name)
			continue
		}
		seen[flag.Name] = true
		if flag.Bool {
			if hasInline {
				structural("--%s does not accept a value", flag.Name)
			}
			continue
		}
		value := inlineValue
		if !hasInline {
			if i+1 >= len(args) {
				structural("missing value for --%s", flag.Name)
				continue
			}
			i++
			value = args[i]
		}
		slots = append(slots, docsSlot{token: value, slotName: flag.Name, flag: flag})
	}

	for i := range schema.Flags {
		f := &schema.Flags[i]
		if f.Required && !seen[f.Name] {
			structural("missing required flag --%s", f.Name)
		}
	}

	posSlots, posIssues := assignDocsPositionals(schema.Positionals, positionals)
	return append(slots, posSlots...), append(issues, posIssues...)
}

func findFlag(schema cliapp.ArgSchema, name string) (*cliapp.Flag, bool) {
	for i := range schema.Flags {
		f := &schema.Flags[i]
		if f.Name == name {
			return f, true
		}
		for _, alias := range f.Aliases {
			if alias == name {
				return f, true
			}
		}
	}
	return nil, false
}

func assignDocsPositionals(specs []cliapp.Positional, values []string) ([]docsSlot, []Issue) {
	var issues []Issue
	if len(specs) == 0 {
		if len(values) > 0 {
			issues = append(issues, Issue{Code: "invalid_arguments", Message: "unexpected positional arguments: " + strings.Join(values, " "), Severity: "error"})
		}
		return nil, issues
	}

	minRequired := 0
	for _, p := range specs {
		if p.Required {
			minRequired++
		}
	}
	if len(values) < minRequired {
		issues = append(issues, Issue{Code: "invalid_arguments", Message: fmt.Sprintf("missing required positional <%s>", specs[len(values)].Name), Severity: "error"})
	}

	var slots []docsSlot
	last := specs[len(specs)-1]
	if last.Repeated {
		fixed := specs[:len(specs)-1]
		for i, p := range fixed {
			if i >= len(values) {
				break
			}
			slots = append(slots, docsSlot{token: values[i], slotName: p.Name})
		}
		if len(values) > len(fixed) {
			for _, v := range values[len(fixed):] {
				slots = append(slots, docsSlot{token: v, slotName: last.Name})
			}
		}
		return slots, issues
	}

	if len(values) > len(specs) {
		issues = append(issues, Issue{Code: "invalid_arguments", Message: "unexpected positional arguments: " + strings.Join(values[len(specs):], " "), Severity: "error"})
		values = values[:len(specs)]
	}
	for i, v := range values {
		slots = append(slots, docsSlot{token: v, slotName: specs[i].Name})
	}
	return slots, issues
}

// requestParams resolves the proto param schema for a "Service.Method"
// binding key. Any failure (nil reader, malformed key, absent descriptor
// image, unknown method) degrades to nil so typed checks are skipped —
// mirroring how measure validation degrades when the image is unavailable.
func (s Service) requestParams(binding string) []measures.ParamSchema {
	if s.Schemas == nil || binding == "" {
		return nil
	}
	dot := strings.LastIndex(binding, ".")
	if dot <= 0 || dot == len(binding)-1 {
		return nil
	}
	params, err := s.Schemas.RequestParams(binding[:dot], binding[dot+1:])
	if err != nil {
		return nil
	}
	return params
}

// checkDocsSlot validates one (token, slot) pair: placeholder tokens are
// matched against the slot, literal tokens against the manifest vocabulary and
// descriptor-derived constraints.
func checkDocsSlot(slot docsSlot, params []measures.ParamSchema) []Issue {
	groups := placeholderGroups(slot.token)
	if len(groups) == 0 {
		return checkDocsLiteral(slot, params)
	}
	// Composite tokens (e.g. "<c1>,<c2>") carry placeholders inside a literal
	// scaffold; there is no single slot semantic to check, so they pass as
	// placeholder-bearing.
	if len(groups) != 1 || slot.token != "<"+groups[0]+">" {
		return nil
	}
	inner := groups[0]
	if strings.Contains(inner, "|") {
		return checkEnumPlaceholder(slot, inner, params)
	}
	return checkNamedPlaceholder(slot, inner)
}

// placeholderGroups extracts the inner text of every balanced, non-nested
// <...> group in a token. Unbalanced or nested brackets yield no groups, so
// the token falls through to literal checking.
func placeholderGroups(token string) []string {
	var groups []string
	depth := 0
	start := -1
	for i, r := range token {
		switch r {
		case '<':
			if depth > 0 {
				return nil // nested — not a placeholder token
			}
			depth = 1
			start = i + 1
		case '>':
			if depth == 0 {
				return nil // unbalanced
			}
			depth = 0
			if i > start {
				groups = append(groups, token[start:i])
			}
		}
	}
	if depth != 0 {
		return nil
	}
	return groups
}

// checkEnumPlaceholder compares an <a|b|c> alternation against the slot's
// effective vocabulary: manifest Values union proto enum values. When no
// vocabulary metadata exists there is nothing to drift from, so it passes.
func checkEnumPlaceholder(slot docsSlot, inner string, params []measures.ParamSchema) []Issue {
	vocab := effectiveVocabulary(slot, params)
	if len(vocab) == 0 {
		return nil
	}
	alts := strings.Split(inner, "|")
	altSet := make(map[string]bool, len(alts))
	for _, a := range alts {
		altSet[strings.TrimSpace(a)] = true
	}
	vocabSet := make(map[string]bool, len(vocab))
	for _, v := range vocab {
		vocabSet[v] = true
	}
	match := len(altSet) == len(vocabSet)
	if match {
		for v := range vocabSet {
			if !altSet[v] {
				match = false
				break
			}
		}
	}
	if match {
		return nil
	}
	sorted := append([]string(nil), vocab...)
	sort.Strings(sorted)
	return []Issue{{
		Code:     issueEnumPlaceholderMismatch,
		Message:  fmt.Sprintf("placeholder <%s> for %s does not match the declared vocabulary; expected <%s>", inner, slotLabel(slot), strings.Join(sorted, "|")),
		Severity: "error",
	}}
}

// checkNamedPlaceholder matches a <name> placeholder against the slot it
// occupies (positional name, flag name, aliases, or bound proto field).
// Mismatches are warnings: the shape is fine, the naming has drifted.
func checkNamedPlaceholder(slot docsSlot, inner string) []Issue {
	accepted := []string{slot.slotName}
	if slot.flag != nil {
		accepted = append(accepted, slot.flag.Aliases...)
		if slot.flag.Bind.Field != "" {
			accepted = append(accepted, slot.flag.Bind.Field)
		}
	}
	name := normalizeSlotName(inner)
	for _, candidate := range accepted {
		if name == normalizeSlotName(candidate) {
			return nil
		}
	}
	return []Issue{{
		Code:     issuePlaceholderNameMismatch,
		Message:  fmt.Sprintf("placeholder <%s> does not match the %s it fills", inner, slotLabel(slot)),
		Severity: "warning",
	}}
}

func normalizeSlotName(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "-", "_")
}

func slotLabel(slot docsSlot) string {
	if slot.flag != nil {
		return "--" + slot.flag.Name + " value"
	}
	return "positional <" + slot.slotName + ">"
}

// effectiveVocabulary is the union of the manifest flag's declared Values and
// the bound proto field's enum value names. Proto names whose conventional CLI
// short form (suffix after the shared prefix, e.g. "plan" for
// COMMAND_REFERENCE_POLICY_PLAN) is already declared in the manifest are not
// duplicated, and *_UNSPECIFIED zero values are never part of the vocabulary.
func effectiveVocabulary(slot docsSlot, params []measures.ParamSchema) []string {
	var vocab []string
	seen := make(map[string]bool)
	if slot.flag != nil {
		for _, v := range slot.flag.Values {
			if !seen[normalizeEnumToken(v)] {
				seen[normalizeEnumToken(v)] = true
				vocab = append(vocab, v)
			}
		}
	}
	if param := paramForSlot(slot, params); param != nil {
		prefix := enumSharedPrefix(param.EnumValues)
		for _, name := range param.EnumValues {
			if strings.HasSuffix(name, "_UNSPECIFIED") {
				continue
			}
			short := normalizeEnumToken(strings.TrimPrefix(name, prefix))
			if seen[normalizeEnumToken(name)] || seen[short] {
				continue
			}
			seen[normalizeEnumToken(name)] = true
			vocab = append(vocab, name)
		}
	}
	return vocab
}

// enumSharedPrefix returns the '_'-terminated prefix all enum value names
// share ("COMMAND_REFERENCE_POLICY_"), or "" when there is none.
func enumSharedPrefix(names []string) string {
	if len(names) < 2 {
		return ""
	}
	prefix := names[0]
	for _, n := range names[1:] {
		for !strings.HasPrefix(n, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	if i := strings.LastIndex(prefix, "_"); i >= 0 {
		return prefix[:i+1]
	}
	return ""
}

func normalizeEnumToken(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "-", "_"))
}

// enumAccepts reports whether a literal names a proto enum member — exactly,
// or via the conventional CLI short form (the name's suffix after the shared
// prefix, compared case-insensitively with '-' folded to '_', e.g. "plan" for
// COMMAND_REFERENCE_POLICY_PLAN or "on-miss" for
// COMMAND_REFERENCE_REFRESH_POLICY_ON_MISS).
func enumAccepts(names []string, v string) bool {
	if containsString(names, v) {
		return true
	}
	prefix := enumSharedPrefix(names)
	nv := normalizeEnumToken(v)
	for _, n := range names {
		if normalizeEnumToken(strings.TrimPrefix(n, prefix)) == nv {
			return true
		}
	}
	return false
}

// paramForSlot resolves the proto request field backing a slot: an explicit
// bind field wins, otherwise the slot name is matched against field names the
// same way protodispatch's scalar fallback does.
func paramForSlot(slot docsSlot, params []measures.ParamSchema) *measures.ParamSchema {
	if len(params) == 0 {
		return nil
	}
	target := normalizeSlotName(slot.slotName)
	if slot.flag != nil && slot.flag.Bind.Field != "" {
		target = normalizeSlotName(slot.flag.Bind.Field)
	}
	for i := range params {
		if normalizeSlotName(params[i].Name) == target {
			return &params[i]
		}
	}
	return nil
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// checkDocsLiteral validates a literal example value against the manifest
// vocabulary and any descriptor-derived constraints. Slots with no constraint
// metadata pass silently.
func checkDocsLiteral(slot docsSlot, params []measures.ParamSchema) []Issue {
	invalid := func(format string, a ...any) []Issue {
		return []Issue{{Code: issueInvalidLiteralValue, Message: fmt.Sprintf(format, a...), Severity: "error"}}
	}

	if slot.flag != nil && len(slot.flag.Values) > 0 && !flagAcceptsValue(*slot.flag, slot.token) {
		sorted := append([]string(nil), slot.flag.Values...)
		sort.Strings(sorted)
		return invalid("value %q for %s is not in the declared vocabulary (%s)", slot.token, slotLabel(slot), strings.Join(sorted, ", "))
	}

	param := paramForSlot(slot, params)
	if param == nil {
		return nil
	}
	switch param.Type {
	case "int32", "int64", "sint32", "sint64", "sfixed32", "sfixed64":
		n, err := strconv.ParseInt(slot.token, 10, 64)
		if err != nil {
			return invalid("value %q for %s is not a valid %s", slot.token, slotLabel(slot), param.Type)
		}
		return checkIntBounds(slot, param, n)
	case "uint32", "uint64", "fixed32", "fixed64":
		n, err := strconv.ParseUint(slot.token, 10, 64)
		if err != nil {
			return invalid("value %q for %s is not a valid %s", slot.token, slotLabel(slot), param.Type)
		}
		return checkIntBounds(slot, param, int64(n))
	case "float", "double":
		if _, err := strconv.ParseFloat(slot.token, 64); err != nil {
			return invalid("value %q for %s is not a valid %s", slot.token, slotLabel(slot), param.Type)
		}
	case "bool":
		if _, err := strconv.ParseBool(slot.token); err != nil {
			return invalid("value %q for %s is not a valid bool", slot.token, slotLabel(slot))
		}
	case "enum":
		if len(param.EnumValues) > 0 && !enumAccepts(param.EnumValues, slot.token) &&
			(slot.flag == nil || !flagAcceptsValue(*slot.flag, slot.token)) {
			sorted := append([]string(nil), param.EnumValues...)
			sort.Strings(sorted)
			return invalid("value %q for %s is not a member of the proto enum (%s)", slot.token, slotLabel(slot), strings.Join(sorted, ", "))
		}
	case "string":
		runes := uint64(len([]rune(slot.token)))
		if param.MinLen != nil && runes < *param.MinLen {
			return invalid("value %q for %s is shorter than the minimum length %d", slot.token, slotLabel(slot), *param.MinLen)
		}
		if param.MaxLen != nil && runes > *param.MaxLen {
			return invalid("value %q for %s exceeds the maximum length %d", slot.token, slotLabel(slot), *param.MaxLen)
		}
		if param.Format == "uuid" && !uuidRe.MatchString(slot.token) {
			return invalid("value %q for %s is not a valid uuid", slot.token, slotLabel(slot))
		}
	}
	return nil
}

func checkIntBounds(slot docsSlot, param *measures.ParamSchema, n int64) []Issue {
	if param.Min != nil && n < *param.Min {
		return []Issue{{Code: issueInvalidLiteralValue, Message: fmt.Sprintf("value %d for %s is below the minimum %d", n, slotLabel(slot), *param.Min), Severity: "error"}}
	}
	if param.Max != nil && n > *param.Max {
		return []Issue{{Code: issueInvalidLiteralValue, Message: fmt.Sprintf("value %d for %s is above the maximum %d", n, slotLabel(slot), *param.Max), Severity: "error"}}
	}
	return nil
}

func containsString(vals []string, v string) bool {
	for _, s := range vals {
		if s == v {
			return true
		}
	}
	return false
}

// flagAcceptsValue reports vocabulary membership: a declared value or a
// declared synonym.
func flagAcceptsValue(f cliapp.Flag, v string) bool {
	for _, allowed := range f.Values {
		if v == allowed {
			return true
		}
	}
	_, ok := f.ValueAliases[v]
	return ok
}
