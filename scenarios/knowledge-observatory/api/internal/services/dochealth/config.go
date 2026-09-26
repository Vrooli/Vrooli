package dochealth

import (
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

// staticConfig holds knowledge-observatory's server-side defaults for the
// DocHealth validator suite. Per-call overrides arrive through
// DocHealthOptions (populated from DocHealthRequest fields in the handler).
type staticConfig struct {
	mermaidStrict bool

	linkConcurrency int
	linkTimeout     time.Duration
	linkIgnore      []string

	pathAllow []string

	codeExtensions []string
	referencesSkip []string

	manifestRel              string
	requireAllDocsRegistered bool

	scanExcludeDirs  []string
	scanExcludeGlobs []string
}

func defaultStaticConfig() staticConfig {
	rel, _ := repocontract.ScenarioDocsManifestRel("")
	return staticConfig{
		mermaidStrict:   true,
		linkConcurrency: 6,
		linkTimeout:     5 * time.Second,
		codeExtensions:  []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py", ".rs", ".java", ".kt"},
		manifestRel:     rel,
	}
}

// effective merges the static defaults with per-call options. Each option
// pointer that is non-nil overrides its corresponding default.
type effective struct {
	staticConfig

	strictExternal    bool
	skipExternal      bool
	requireRegistered bool
}

func (c staticConfig) withOptions(opts DocHealthOptions) effective {
	out := effective{staticConfig: c, requireRegistered: c.requireAllDocsRegistered}
	if opts.StrictExternalLinks != nil {
		out.strictExternal = *opts.StrictExternalLinks
	}
	if opts.SkipExternalLinks != nil {
		out.skipExternal = *opts.SkipExternalLinks
	}
	if opts.RequireAllDocsRegistered != nil {
		out.requireRegistered = *opts.RequireAllDocsRegistered
	}
	return out
}
