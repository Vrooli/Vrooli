package dispatch

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/cliresolve"

	"github.com/vrooli/api-core/scopecatalog"
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

var defaultManifestCache struct {
	sync.Once
	entries []string
	outputs map[string][]ArtifactOutput
	err     error
}

// BuildManifest resolves the checked-in dispatch artifact against the shared
// CLI scope catalog. It is intentionally called lazily rather than during
// package initialization: a malformed scenario manifest must produce a
// typed degraded service, not a process-start panic. The result is cached
// because the checked-in manifest and catalog are process configuration.
func BuildManifest() ([]string, map[string][]ArtifactOutput, error) {
	defaultManifestCache.Do(func() {
		defaultManifestCache.entries, defaultManifestCache.outputs, defaultManifestCache.err = buildManifest()
	})
	return append([]string(nil), defaultManifestCache.entries...), cloneOutputs(defaultManifestCache.outputs), defaultManifestCache.err
}

func buildManifest() ([]string, map[string][]ArtifactOutput, error) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return nil, nil, fmt.Errorf("locate repository for dispatch scope catalog: %w", err)
	}
	catalog, err := scopecatalog.BuildResilient(root)
	if err != nil {
		return nil, nil, err
	}

	var artifact manifestArtifact
	if err := json.Unmarshal(dispatchManifestJSON, &artifact); err != nil {
		return nil, nil, fmt.Errorf("decode dispatch manifest: %w", err)
	}
	if err := validateDefaultGrants(catalog, artifact.DefaultGrants); err != nil {
		return nil, nil, err
	}
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
		verb := scope.Verb()
		effect := string(scope.Effect)
		required := "vrooli-bridge:" + effect
		if !catalog.HasScope(required) {
			return nil, nil, fmt.Errorf("catalog command %q requires missing bridge effect scope %q", scope.Command, required)
		}
		// A node must hold both the namespace-and-effect grant and Bridge's
		// transport-level effect capability. The namespace grant follows the
		// derived catalog, so adding a command needs no per-node verb edit while
		// still granting nothing in an unconceded scenario or effect.
		result = append(result, encodeManifestEntry(verb, []string{scope.Value, required}))
	}
	result = append(result, bindings...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, manifestOutputs(artifact), nil
}

func cloneOutputs(input map[string][]ArtifactOutput) map[string][]ArtifactOutput {
	if input == nil {
		return nil
	}
	output := make(map[string][]ArtifactOutput, len(input))
	for verb, entries := range input {
		output[verb] = append([]ArtifactOutput(nil), entries...)
	}
	return output
}

func validateDefaultGrants(catalog scopecatalog.Catalog, grants []string) error {
	for _, grant := range grants {
		grant = strings.TrimSpace(grant)
		if grant == "" {
			return fmt.Errorf("dispatch manifest contains an empty default grant")
		}
		if grant != string(scopecatalog.EffectRead) && grant != string(scopecatalog.EffectWrite) && grant != string(scopecatalog.EffectDestructive) {
			return fmt.Errorf("dispatch manifest contains invalid default grant %q", grant)
		}
		required := "vrooli-bridge:" + grant
		if !catalog.HasScope(required) {
			return fmt.Errorf("dispatch default grant %q requires missing derived scope %q", grant, required)
		}
	}
	return nil
}

func manifestOutputs(artifact manifestArtifact) map[string][]ArtifactOutput {
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
		if err := cliresolve.ValidateArgvToken(tok); err != nil {
			return ErrUnsafeToken{Token: tok, Reason: err.Error()}
		}
	}

	if !inManifest(j.Verb, manifest) {
		return ErrVerbNotInManifest{Verb: j.Verb}
	}
	if allowed, missing := anyScopeMatches(scopes, j.Verb, manifest); !allowed {
		return ErrVerbOutOfScope{Verb: j.Verb, RequiredScope: missing}
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

// anyScopeMatches reports whether every catalog-derived requirement for one
// matching entry is held. It returns the first missing concrete scope so a
// refusal tells the operator exactly what authority is absent.
func anyScopeMatches(scopes []string, verb string, manifest []string) (bool, string) {
	missing := ""
	for _, entry := range manifest {
		if manifestEntryVerb(entry) != verb {
			continue
		}
		required := manifestEntryScopes(entry)
		if len(required) == 0 {
			continue
		}
		entryAllowed := true
		for _, requirement := range required {
			if !scopecatalog.Resolve(scopes, requirement) {
				entryAllowed = false
				if missing == "" {
					missing = requirement
				}
			}
		}
		if entryAllowed {
			return true, ""
		}
	}
	return false, missing
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
