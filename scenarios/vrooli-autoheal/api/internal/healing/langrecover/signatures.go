package langrecover

import "strings"

// GoSignature classifies a Go build-failure log into a healable category.
type GoSignature int

const (
	// GoSignatureNone means the log does not match any healable pattern.
	GoSignatureNone GoSignature = iota
	// GoSignatureMissingSum matches "missing go.sum entry for module providing
	// package X". `go mod download` is sufficient — it adds the missing
	// content hashes without rewriting the dependency graph.
	GoSignatureMissingSum
	// GoSignatureMissingModule matches "no required module provides package X"
	// or "inconsistent vendoring". `go mod tidy` is required because the
	// dependency graph itself needs to change.
	GoSignatureMissingModule
)

// DetectGoSignature scans failure output for a known healable Go pattern.
// Returns GoSignatureNone if nothing matches.
func DetectGoSignature(output string) GoSignature {
	if output == "" {
		return GoSignatureNone
	}
	lower := strings.ToLower(output)
	if strings.Contains(lower, "no required module provides package") ||
		strings.Contains(lower, "inconsistent vendoring") ||
		strings.Contains(lower, "updates to go.mod needed") {
		// "updates to go.mod needed" is emitted by `go build` when go.mod is
		// out of sync with imports — only `go mod tidy` resolves it, since
		// the dependency graph itself needs updating.
		return GoSignatureMissingModule
	}
	if strings.Contains(lower, "missing go.sum entry") {
		return GoSignatureMissingSum
	}
	return GoSignatureNone
}

// PnpmSignature classifies a pnpm install-failure log into a healable category.
type PnpmSignature int

const (
	// PnpmSignatureNone means the log does not match any healable pattern.
	PnpmSignatureNone PnpmSignature = iota
	// PnpmSignatureOutdatedLockfile matches "ERR_PNPM_OUTDATED_LOCKFILE".
	// Recovered by re-installing with --no-frozen-lockfile.
	PnpmSignatureOutdatedLockfile
	// PnpmSignatureLinkingFailed matches "ERR_PNPM_LINKING_FAILED" or a
	// missing/corrupt node_modules tree (ENOENT referring to node_modules).
	// Recovered by removing node_modules and reinstalling.
	PnpmSignatureLinkingFailed
)

// DetectPnpmSignature scans failure output for a known healable pnpm pattern.
func DetectPnpmSignature(output string) PnpmSignature {
	if output == "" {
		return PnpmSignatureNone
	}
	lower := strings.ToLower(output)
	if strings.Contains(lower, "err_pnpm_outdated_lockfile") {
		return PnpmSignatureOutdatedLockfile
	}
	if strings.Contains(lower, "err_pnpm_linking_failed") {
		return PnpmSignatureLinkingFailed
	}
	if strings.Contains(lower, "enoent") && strings.Contains(lower, "node_modules") {
		return PnpmSignatureLinkingFailed
	}
	return PnpmSignatureNone
}
