package envkit

import (
	"runtime"
	"strconv"
	"strings"
)

// Toolchain environment variables. The floor names them once so a site that
// needs to read them has a symbol, not a string.
const (
	// GoFlagsKey carries Go's build flags; the floor appends "-p=<width>".
	GoFlagsKey = "GOFLAGS"
	// GoMaxProcsKey bounds Go runtime threads; esbuild (vite's bundler) is a
	// Go program and honors it as well.
	GoMaxProcsKey = "GOMAXPROCS"
	// PnpmChildConcurrencyKey bounds pnpm's concurrent install children.
	// pnpm reads every setting from "npm_config_<setting>".
	PnpmChildConcurrencyKey = "npm_config_child_concurrency"
	// PnpmWorkspaceConcurrencyKey bounds pnpm's concurrent workspace projects.
	PnpmWorkspaceConcurrencyKey = "npm_config_workspace_concurrency"
	// BuildWidthKey is the operator override for the build width. It is the
	// same variable internal/tuning resolves for BuildWidth; a shared package
	// that cannot import the control plane's lever reads it from the
	// environment it is composing instead.
	BuildWidthKey = "VROOLI_TUNING_BUILD_WIDTH"
	// goWidthFlag is the Go build-parallelism flag the floor composes.
	goWidthFlag = "-p"
	// buildWidthCeiling caps the default width regardless of host size; the
	// 2026-09-02 storm was three sessions each building at NumCPU width.
	buildWidthCeiling = 4
	// buildWidthCPUDivisor derives the default width from the host's cores.
	buildWidthCPUDivisor = 4
)

// DefaultBuildWidth is the one derivation of the build width from the host:
// min(4, max(1, NumCPU/4)). internal/tuning uses it as the compiled default
// of the BuildWidth lever; no call site derives a width from NumCPU itself.
func DefaultBuildWidth() int {
	return min(buildWidthCeiling, max(1, runtime.NumCPU()/buildWidthCPUDivisor))
}

// BuildWidthFrom reads the operator override from an environment slice and
// falls back to DefaultBuildWidth when it is absent, malformed or not
// positive.
func BuildWidthFrom(env Env) int {
	return BuildWidthFromWithPlatform(env, DefaultPlatform())
}

// BuildWidthFromWithPlatform is BuildWidthFrom with an injected platform
// policy for case folding.
func BuildWidthFromWithPlatform(env Env, platform Platform) int {
	want := fold(BuildWidthKey, platform)
	for _, entry := range env {
		key, value, ok := split(entry)
		if !ok || fold(key, platform) != want {
			continue
		}
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return DefaultBuildWidth()
}

// ToolchainOptions sizes the toolchain floor.
type ToolchainOptions struct {
	// Width is the number of concurrent compile or link processes a build
	// may run. The control plane passes tuning.BuildWidth(); a value of 0 or
	// less resolves through BuildWidthFrom on the environment being composed,
	// so a shared package outside the control plane gets the same lever.
	Width int
	// GoFlags are extra GOFLAGS tokens a site needs ("-mod=mod"). Each is
	// appended after the inherited tokens when it is not already present,
	// so a later token overrides an inherited one the way Go parses flags,
	// and no site ever assigns GOFLAGS outright.
	GoFlags []string
}

// Toolchain applies the build-width floor to an environment. It composes:
// an inherited GOFLAGS keeps every token and gains "-p=<width>" only when no
// -p token is present; GOMAXPROCS and the pnpm concurrency settings are set
// only when absent. The overlay never replaces an inherited value.
func Toolchain(env Env, opts ToolchainOptions) Env {
	return ToolchainWithPlatform(env, opts, DefaultPlatform())
}

// ToolchainWithPlatform is Toolchain with an injected platform policy so the
// Windows case folding can be exercised on any host.
func ToolchainWithPlatform(env Env, opts ToolchainOptions, platform Platform) Env {
	if opts.Width <= 0 {
		opts.Width = BuildWidthFromWithPlatform(env, platform)
	}
	width := strconv.Itoa(opts.Width)
	out := make(Env, 0, len(env)+4)
	// An inherited value is kept as long as it is non-empty; an empty value
	// ("GOMAXPROCS=") is no value at all to the toolchain and is filled.
	seen := map[string]bool{}
	for _, entry := range env {
		key, value, ok := split(entry)
		if !ok {
			out = append(out, entry)
			continue
		}
		folded := fold(key, platform)
		switch {
		case folded == fold(GoFlagsKey, platform) && !seen[folded]:
			out = append(out, key+"="+composeGoFlags(value, width, opts.GoFlags))
			seen[folded] = true
		case value == "" && !seen[folded]:
			// Dropped: the default appended below supplies the value.
		default:
			out = append(out, entry)
			seen[folded] = true
		}
	}
	defaults := Env{
		GoFlagsKey + "=" + composeGoFlags("", width, opts.GoFlags),
		GoMaxProcsKey + "=" + strconv.Itoa(opts.Width*2),
		PnpmChildConcurrencyKey + "=" + width,
		PnpmWorkspaceConcurrencyKey + "=" + width,
	}
	for _, entry := range defaults {
		key, _, _ := split(entry)
		if !seen[fold(key, platform)] {
			out = append(out, entry)
		}
	}
	return out
}

// composeGoFlags appends the width token and the site's extra tokens to an
// inherited GOFLAGS value and keeps every existing token byte-identical. An
// explicit -p wins; an extra token already present is not repeated.
func composeGoFlags(inherited, width string, extra []string) string {
	tokens := strings.Fields(inherited)
	hasWidth := false
	for _, token := range tokens {
		if token == goWidthFlag || strings.HasPrefix(token, goWidthFlag+"=") {
			hasWidth = true
		}
	}
	composed := strings.TrimSpace(inherited)
	if !hasWidth {
		composed = joinToken(composed, goWidthFlag+"="+width)
	}
	for _, token := range extra {
		token = strings.TrimSpace(token)
		if token == "" || containsToken(tokens, token) {
			continue
		}
		composed = joinToken(composed, token)
		tokens = append(tokens, token)
	}
	return composed
}

func joinToken(value, token string) string {
	if value == "" {
		return token
	}
	return value + " " + token
}

func containsToken(tokens []string, token string) bool {
	for _, candidate := range tokens {
		if candidate == token {
			return true
		}
	}
	return false
}
