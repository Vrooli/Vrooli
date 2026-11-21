// Package opencodeversion is the single source of truth for the behavioural
// differences between OpenCode 1.x and 2.x.
//
// OpenCode 1 and 2 share a command name, configuration locations, and much of
// their stream contract, but they differ at a few narrow boundaries: how a run
// is pinned to a directory, how a model variant is expressed, where sessions are
// stored, whether a local model catalog exists, and which plugin API is loaded.
// Callers should detect the installed major once and ask this package for the
// matching capabilities instead of scattering version checks through business
// logic.
package opencodeversion

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// DirectoryStyle describes how a run is pinned to a working directory.
type DirectoryStyle string

const (
	// DirectoryFlag means the CLI accepts `--dir <path>` on `run`.
	DirectoryFlag DirectoryStyle = "flag"
	// DirectoryProcess means the CLI resolves the project from the process
	// working directory (the runner already sets cmd.Dir) and has no --dir flag.
	DirectoryProcess DirectoryStyle = "process-cwd"
)

// VariantStyle describes how a model reasoning variant is expressed.
type VariantStyle string

const (
	// VariantFlag means `--variant <name>` is a separate flag.
	VariantFlag VariantStyle = "flag"
	// VariantModelSuffix means the variant is appended to the model id as
	// `provider/model#variant`.
	VariantModelSuffix VariantStyle = "model-suffix"
)

// CatalogProbe identifies how local model validation can read a catalog.
type CatalogProbe string

const (
	// CatalogModelsJSON is the OpenCode 1 cache at
	// <user-cache>/opencode/models.json.
	CatalogModelsJSON CatalogProbe = "models-json"
	// CatalogNone means the installed major has no stable, local catalog file.
	// ProbeModel must degrade to "cannot confirm" rather than guessing.
	CatalogNone CatalogProbe = "none"
)

// HookSurface identifies the native plugin API the installed major loads.
type HookSurface string

const (
	// HookSurfaceV1 is the `tool.execute.before` plugin API.
	HookSurfaceV1 HookSurface = "v1-plugin"
	// HookSurfaceV2 is the `Plugin.define` / `ctx.permission.hook("evaluate")`
	// API.
	HookSurfaceV2 HookSurface = "v2-plugin"
)

// Capabilities is the narrow set of version-dependent behaviours the runner,
// resource CLI, and policy layer need. It deliberately does not attempt to model
// every upstream feature.
type Capabilities struct {
	Major         int
	Directory     DirectoryStyle
	Variant       VariantStyle
	SessionTable  string
	Catalog       CatalogProbe
	PrivateServer bool
	HookSurface   HookSurface
}

// FromMajor returns the capability profile for a known major version. Unknown
// or older versions are treated as 1.x, which is the conservative command
// contract: it is the shape every 1.x release has shipped.
func FromMajor(major int) Capabilities {
	if major >= 2 {
		return Capabilities{
			Major:         major,
			Directory:     DirectoryProcess,
			Variant:       VariantModelSuffix,
			SessionTable:  "session_v2",
			Catalog:       CatalogNone,
			PrivateServer: true,
			HookSurface:   HookSurfaceV2,
		}
	}
	return Capabilities{
		Major:        1,
		Directory:    DirectoryFlag,
		Variant:      VariantFlag,
		SessionTable: "session",
		Catalog:      CatalogModelsJSON,
		HookSurface:  HookSurfaceV1,
	}
}

// IsV2 reports whether this profile targets OpenCode 2 or newer.
func (c Capabilities) IsV2() bool { return c.Major >= 2 }

var majorPattern = regexp.MustCompile(`(?i)(?:^|[^0-9])v?(\d+)\.\d+`)

// ParseMajor extracts the major version from a version string such as
// "1.19.5", "v2.3.2", or "opencode v2.3.2". It returns false when no major
// version is present.
func ParseMajor(raw string) (int, bool) {
	match := majorPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) != 2 {
		return 0, false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil || major <= 0 {
		return 0, false
	}
	return major, true
}

// Detect runs `<binary> --version` and returns the capability profile for the
// installed major. The caller owns the timeout.
func Detect(ctx context.Context, binary string) (Capabilities, error) {
	if strings.TrimSpace(binary) == "" {
		return Capabilities{}, fmt.Errorf("opencode version: empty binary path")
	}
	out, err := exec.CommandContext(ctx, binary, "--version").CombinedOutput()
	if err != nil {
		return Capabilities{}, fmt.Errorf("opencode version: %w: %s", err, strings.TrimSpace(string(out)))
	}
	major, ok := ParseMajor(string(out))
	if !ok {
		return Capabilities{}, fmt.Errorf("opencode version: could not parse %q", strings.TrimSpace(string(out)))
	}
	return FromMajor(major), nil
}
