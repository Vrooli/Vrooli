package dispatch

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/api-core/targetmodel"
	repocontract "github.com/vrooli/repo-contract-go"
)

// dispatch.manifest.json is the checked-in binding artifact for the currently
// supported typed dispatch verbs. It names each binding's governance effect;
// the concrete vrooli-bridge:<effect> scope is resolved from the catalog built
// from existing CLI governance metadata below. This keeps the typed dispatch
// vocabulary auditable without creating a second hand-maintained scope list.
//
//go:embed dispatch.manifest.json
var dispatchManifestJSON []byte

type manifestArtifact struct {
	// DefaultGrants documents the effect-level grants a newly trusted node may
	// receive. It is separate from the admitted vocabulary: all cataloged
	// commands are known to Allow, while a node still needs the matching effect
	// scope to execute one.
	DefaultGrants []string `json:"default_grants"`
	Outputs       []struct {
		Verb    string `json:"verb"`
		Entries []struct {
			Name       string `json:"name"`
			MediaType  string `json:"media_type"`
			OutputFlag string `json:"output_flag"`
			MaxBytes   int64  `json:"max_bytes"`
		} `json:"entries"`
	} `json:"outputs"`
}

// DefaultManifest is the constructor's immutable artifact input for the pure
// decision tests. It is resolved from the checked-in typed bindings plus the
// derived scope catalog, not maintained as a Go verb list or scope list.
var (
	DefaultManifest        = loadManifest()
	defaultManifestOutputs = loadManifestOutputs()
)

func loadManifest() []string {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		panic(fmt.Sprintf("locate repository for dispatch scope catalog: %v", err))
	}
	catalog, err := scopecatalog.Build(root)
	if err != nil {
		panic(fmt.Sprintf("build dispatch scope catalog: %v", err))
	}

	var artifact manifestArtifact
	if err := json.Unmarshal(dispatchManifestJSON, &artifact); err != nil {
		panic(fmt.Sprintf("decode dispatch manifest: %v", err))
	}
	validateDefaultGrants(catalog, artifact.DefaultGrants)
	bindings := make([]string, 0, len(catalog.Scopes))

	// Project CLI command names are the dispatch vocabulary. Their governance
	// effects come from the same catalog used by agent-manager and posture
	// validation, so adding a governed root command cannot silently leave the
	// node admission surface stale.
	result := make([]string, 0, len(catalog.Scopes))
	for _, scope := range catalog.Scopes {
		if !scope.RunEligible || strings.TrimSpace(scope.Command) == "" {
			continue
		}
		verb := strings.ReplaceAll(scope.Command, "/", " ")
		if scope.Scenario != scopecatalog.ProjectManifestIdentity {
			verb = scope.Scenario + " " + verb
		}
		effect := string(scope.Effect)
		required := "vrooli-bridge:" + effect
		if !catalog.HasScope(required) {
			panic(fmt.Sprintf("catalog command %q requires missing bridge effect scope %q", scope.Command, required))
		}
		// A node must hold both the effect grant and a verb-pattern grant.
		// Keeping the two requirements in the derived entry prevents adding a
		// new catalog command from widening every existing node that only has
		// the broad effect grant.
		result = append(result, encodeManifestEntry(verb, []string{required, verb}))
	}
	result = append(result, bindings...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func validateDefaultGrants(catalog scopecatalog.Catalog, grants []string) {
	for _, grant := range grants {
		grant = strings.TrimSpace(grant)
		if grant == "" {
			panic("dispatch manifest contains an empty default grant")
		}
		if grant != string(scopecatalog.EffectRead) && grant != string(scopecatalog.EffectWrite) && grant != string(scopecatalog.EffectDestructive) {
			panic(fmt.Sprintf("dispatch manifest contains invalid default grant %q", grant))
		}
		required := "vrooli-bridge:" + grant
		if !catalog.HasScope(required) {
			panic(fmt.Sprintf("dispatch default grant %q requires missing derived scope %q", grant, required))
		}
	}
}

func loadManifestOutputs() map[string][]ArtifactOutput {
	var artifact manifestArtifact
	if err := json.Unmarshal(dispatchManifestJSON, &artifact); err != nil {
		panic(fmt.Sprintf("decode dispatch manifest outputs: %v", err))
	}
	result := make(map[string][]ArtifactOutput)
	for _, output := range artifact.Outputs {
		verb := strings.TrimSpace(output.Verb)
		if verb == "" || len(output.Entries) == 0 {
			continue
		}
		for _, entry := range output.Entries {
			result[verb] = append(result[verb], ArtifactOutput{
				Name: strings.TrimSpace(entry.Name), MediaType: strings.TrimSpace(entry.MediaType),
				OutputFlag: strings.TrimSpace(entry.OutputFlag), MaxBytes: entry.MaxBytes,
			})
		}
	}
	return result
}

// shellMetachars are characters that would have meaning to a shell. A typed job
// never reaches a shell (the node-agent execs an argv, never `sh -c`), so any
// token containing one is rejected as defence in depth — it can only be an
// attempt to smuggle a shell construct through a typed field.
const shellMetachars = "|&;<>()$`\\\"'\n\r\t*?[]{}!#~"

// Allow validates the job against the manifest allowlist and the node's granted
// scopes. It is pure (no I/O) so the highest-stakes decision in the scenario is
// exhaustively table-testable. Returns nil when the verb is allowlisted and
// in-scope; otherwise a typed sentinel naming the precise reason.
//
// Order of checks (each a distinct, testable rejection reason):
//  1. verb non-empty (ErrInvalidJob)
//  2. no shell metacharacter in any token (ErrUnsafeToken)
//  3. verb is a recognised manifest verb (ErrVerbNotInManifest)
//  4. verb matches at least one node scope (ErrVerbOutOfScope)
func Allow(job Job, scopes, manifest []string) error {
	j := job.trimmed()
	if j.Verb == "" {
		return ErrInvalidJob{Field: "verb", Reason: "required"}
	}

	for _, tok := range append([]string{j.Verb, j.Scenario}, j.Args...) {
		if i := strings.IndexAny(tok, shellMetachars); i >= 0 {
			return ErrUnsafeToken{Token: tok, Reason: "contains a shell metacharacter"}
		}
	}

	if !inManifest(j.Verb, manifest) {
		return ErrVerbNotInManifest{Verb: j.Verb}
	}
	if !anyScopeMatches(scopes, j.Verb, manifest) {
		return ErrVerbOutOfScope{Verb: j.Verb}
	}
	return nil
}

// inManifest reports whether verb is exactly one of the manifest entries.
func inManifest(verb string, manifest []string) bool {
	for _, m := range manifest {
		if manifestEntryVerb(m) == verb {
			return true
		}
	}
	return false
}

// anyScopeMatches reports whether any granted scope covers verb.
func anyScopeMatches(scopes []string, verb string, manifest []string) bool {
	for _, entry := range manifest {
		if manifestEntryVerb(entry) != verb {
			continue
		}
		required := manifestEntryScopes(entry)
		if len(required) == 0 {
			// Keep hand-built unit-test manifests useful, while all production
			// entries emitted by loadManifest carry the paired requirements.
			return scopeMatchesAny(scopes, verb)
		}
		effectMatched, verbMatched := false, false
		for _, requirement := range required {
			if strings.HasPrefix(requirement, "vrooli-bridge:") {
				effectMatched = effectMatched || scopeMatchesAny(scopes, requirement)
			} else {
				verbMatched = verbMatched || scopeMatchesAny(scopes, requirement)
			}
		}
		if effectMatched && verbMatched {
			return true
		}
	}
	return false
}

func scopeMatchesAny(scopes []string, required string) bool {
	for _, s := range scopes {
		if scopeMatches(strings.TrimSpace(s), required) {
			return true
		}
	}
	return false
}

const manifestSeparator = "\x00"

func encodeManifestEntry(verb string, scopes []string) string {
	clean := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if value := strings.TrimSpace(scope); value != "" {
			clean = append(clean, value)
		}
	}
	return verb + manifestSeparator + strings.Join(clean, manifestSeparator)
}

func manifestEntryVerb(entry string) string {
	if index := strings.Index(entry, manifestSeparator); index >= 0 {
		return entry[:index]
	}
	return strings.TrimSpace(entry)
}

func manifestEntryScopes(entry string) []string {
	index := strings.Index(entry, manifestSeparator)
	if index < 0 || index+len(manifestSeparator) >= len(entry) {
		return nil
	}
	return strings.Split(entry[index+len(manifestSeparator):], manifestSeparator)
}

// scopeMatches implements the scope glob grammar: an exact match, a trailing-`*`
// prefix match (e.g. "scenario test*" covers "scenario test" and "scenario
// testall"), or the universal "*". The wildcard is only honoured as a trailing
// character — an interior `*` is treated literally, so a scope can never be
// tricked into matching across the namespace boundary.
func scopeMatches(scope, verb string) bool {
	return targetmodel.ScopeAllows([]string{scope}, verb)
}
