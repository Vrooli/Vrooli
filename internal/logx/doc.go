// Package logx defines the project-level logging contract for the Vrooli Go
// orchestrator.
//
// The package has three responsibilities:
//   - bootstrap process-wide slog configuration for CLI and API entrypoints
//   - provide stable structured field keys used across internal packages
//   - render errors into a machine-readable shape with consistent categories
//
// Environment contract:
//   - VROOLI_LOG_LEVEL: "", info, debug, warn, warning, error, err
//   - VROOLI_LOG_FORMAT: "", text, json
//
// Operational guidance:
//   - Use InstallAndReport at process bootstrap so configuration warnings are
//     emitted consistently.
//   - Use WithSubsystem to scope child loggers without changing the top-level
//     component identity.
//   - Use Error or ErrorArgs when emitting failures so category and diagnostic
//     fields remain consistent across the codebase.
package logx
