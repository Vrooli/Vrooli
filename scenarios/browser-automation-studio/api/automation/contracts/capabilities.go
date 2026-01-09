package contracts

import (
	"errors"
	"fmt"
)

// EngineCapabilities advertises what an engine implementation supports so
// orchestration can select/validate engines before execution begins.
type EngineCapabilities struct {
	SchemaVersion         string `json:"schema_version"`
	Engine                string `json:"engine"`            // e.g., browserless, desktop
	Version               string `json:"version,omitempty"` // Implementation version/commit.
	RequiresDocker        bool   `json:"requires_docker"`
	RequiresXvfb          bool   `json:"requires_xvfb"`
	MaxConcurrentSessions int    `json:"max_concurrent_sessions"` // Hard cap enforced by engine.
	AllowsParallelTabs    bool   `json:"allows_parallel_tabs"`
	SupportsHAR           bool   `json:"supports_har"`
	SupportsVideo         bool   `json:"supports_video"`
	SupportsIframes       bool   `json:"supports_iframes"`
	SupportsFileUploads   bool   `json:"supports_file_uploads"`
	SupportsDownloads     bool   `json:"supports_downloads"`
	SupportsTracing       bool   `json:"supports_tracing"`
	MaxViewportWidth      int    `json:"max_viewport_width,omitempty"`  // 0 = unknown/unbounded.
	MaxViewportHeight     int    `json:"max_viewport_height,omitempty"` // 0 = unknown/unbounded.
	Notes                 string `json:"notes,omitempty"`
}

// CapabilityRequirement expresses the minimum feature set required by a
// compiled workflow (derived during preflight analysis).
type CapabilityRequirement struct {
	NeedsParallelTabs bool
	NeedsHAR          bool
	NeedsVideo        bool
	NeedsIframes      bool
	NeedsFileUploads  bool
	NeedsDownloads    bool
	NeedsTracing      bool
	MinViewportWidth  int
	MinViewportHeight int
}

// CapabilityGap captures missing or risky capability matches.
type CapabilityGap struct {
	Missing  []string // Hard requirements not satisfied.
	Warnings []string // Soft requirements that could not be validated.
}

// Satisfied reports whether all hard requirements are met.
func (g CapabilityGap) Satisfied() bool {
	return len(g.Missing) == 0
}

// Validate ensures the capability descriptor is internally consistent.
func (c EngineCapabilities) Validate() error {
	if c.SchemaVersion == "" {
		return errors.New("schema_version is required")
	}
	if c.SchemaVersion != CapabilitiesSchemaVersion {
		return fmt.Errorf("schema_version must be %s", CapabilitiesSchemaVersion)
	}
	if c.Engine == "" {
		return errors.New("engine is required")
	}
	if c.MaxConcurrentSessions <= 0 {
		return errors.New("max_concurrent_sessions must be positive")
	}
	if c.MaxViewportWidth < 0 {
		return errors.New("max_viewport_width cannot be negative")
	}
	if c.MaxViewportHeight < 0 {
		return errors.New("max_viewport_height cannot be negative")
	}
	return nil
}

// CheckCompatibility compares the engine descriptor against workflow
// requirements, returning any hard or soft gaps.
func (c EngineCapabilities) CheckCompatibility(req CapabilityRequirement) CapabilityGap {
	var gap CapabilityGap

	appendIf := func(flag bool, value string) {
		if flag {
			gap.Missing = append(gap.Missing, value)
		}
	}
	appendWarn := func(flag bool, value string) {
		if flag {
			gap.Warnings = append(gap.Warnings, value)
		}
	}

	appendIf(req.NeedsParallelTabs && !c.AllowsParallelTabs, "parallel_tabs")
	appendIf(req.NeedsHAR && !c.SupportsHAR, "har")
	appendIf(req.NeedsVideo && !c.SupportsVideo, "video")
	appendIf(req.NeedsIframes && !c.SupportsIframes, "iframes")
	appendIf(req.NeedsFileUploads && !c.SupportsFileUploads, "file_uploads")
	appendIf(req.NeedsDownloads && !c.SupportsDownloads, "downloads")
	appendIf(req.NeedsTracing && !c.SupportsTracing, "tracing")

	if req.MinViewportWidth > 0 {
		switch {
		case c.MaxViewportWidth == 0:
			appendWarn(true, fmt.Sprintf("viewport_width>=%d unknown", req.MinViewportWidth))
		case c.MaxViewportWidth < req.MinViewportWidth:
			appendIf(true, fmt.Sprintf("viewport_width>=%d", req.MinViewportWidth))
		}
	}

	if req.MinViewportHeight > 0 {
		switch {
		case c.MaxViewportHeight == 0:
			appendWarn(true, fmt.Sprintf("viewport_height>=%d unknown", req.MinViewportHeight))
		case c.MaxViewportHeight < req.MinViewportHeight:
			appendIf(true, fmt.Sprintf("viewport_height>=%d", req.MinViewportHeight))
		}
	}

	return gap
}
