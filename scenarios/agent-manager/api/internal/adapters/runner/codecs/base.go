// Package codecs — base.go holds baseCodec, the embeddable struct every
// concrete codec composes for the surface that is genuinely identical across
// runners: binary resolution + availability, static identity (runner type,
// binary description, tag env key, continuation-tag prefix, status labels),
// the available-gated ProbeModel default, and the transcript-line delegate.
//
// Before this seam each codec file re-implemented these byte-for-byte
// (NewXxx LookPath, NewXxxForTest fake, Available, ProbeModel, ContinueTag,
// Labels, Type, ParseTranscriptLine). Concrete codecs now embed baseCodec and
// contain ONLY their genuinely-unique surface: BuildArgs/BuildContinueArgs,
// the stream decoder, classification, metrics, and cost.
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md (Codec / baseCodec).
package codecs

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// baseCodec is embedded by every concrete codec. The binary-resolution
// fields (binaryPath/available/message/installHint) are populated by
// [resolveBinary]; the identity fields are set by each codec's *base()
// helper so the real and *ForTest constructors share one definition.
type baseCodec struct {
	binaryPath  string
	available   bool
	message     string
	installHint string

	runnerType     domain.RunnerType
	binaryDesc     string
	tagEnvKey      string
	continuePrefix string
	labels         Labels

	// newParser builds a fresh stateful transcript parser. Each codec's
	// constructor wires this to its own NewTranscriptParser so the shared
	// [baseCodec.ParseTranscriptLine] can delegate without the base ever
	// referencing the embedding type.
	newParser func() runner.TranscriptParser
}

// Type satisfies [Codec].
func (b *baseCodec) Type() domain.RunnerType { return b.runnerType }

// ToolCapabilityMap satisfies [Codec] for codecs that have no native tool
// declarations. Concrete harness codecs override it with their vocabulary.
func (b *baseCodec) ToolCapabilityMap() map[string]string { return map[string]string{} }

// BinaryPath satisfies [Codec].
func (b *baseCodec) BinaryPath() string { return b.binaryPath }

// BinaryDescription satisfies [Codec].
func (b *baseCodec) BinaryDescription() string { return b.binaryDesc }

// TagEnvKey satisfies [Codec].
func (b *baseCodec) TagEnvKey() string { return b.tagEnvKey }

// Labels satisfies [Codec].
func (b *baseCodec) Labels() Labels { return b.labels }

// Available satisfies [Codec]. Reports the construction-time failure (with
// install hint) when the binary was never resolved, re-checks the resolved
// path on disk otherwise, and yields a uniform "<desc> is available" on
// success.
func (b *baseCodec) Available(_ context.Context) (bool, string) {
	if !b.available {
		msg := b.message
		if b.installHint != "" {
			msg += ". " + b.installHint
		}
		return false, msg
	}
	if _, err := os.Stat(b.binaryPath); os.IsNotExist(err) {
		msg := b.binaryDesc + " not found"
		if b.installHint != "" {
			msg += ". " + b.installHint
		}
		return false, msg
	}
	return true, b.binaryDesc + " is available"
}

// ProbeModel satisfies [Codec] with the available-gate-only default shared by
// the Anthropic-native and codex codecs: a deep model check would cost vendor
// quota, so the authoritative "model is gone" signal comes from runtime
// classification on the first real invocation. Codecs with a quota-free local
// validation (OpenCode) override this.
func (b *baseCodec) ProbeModel(ctx context.Context, _ string) error {
	if available, msg := b.Available(ctx); !available {
		return fmt.Errorf("%s unavailable: %s", b.runnerType, msg)
	}
	return nil
}

// ContinueTag satisfies [Codec]. Synthesised tag distinguishes continuation
// runs from initial runs of the same RunID for /proc-based reconciliation.
func (b *baseCodec) ContinueTag(req runner.ContinueRequest) string {
	return fmt.Sprintf("%s-continue-%s", b.continuePrefix, req.RunID.String()[:8])
}

// OnEarlyTerminate satisfies [Codec] with the common "drain to EOF" default.
// Only codecs with an in-stream terminator (OpenCode's terminal step_finish)
// override this.
func (b *baseCodec) OnEarlyTerminate(_ State, _ string) bool { return false }

// PostClassify satisfies [Codec] with the common no-op default. Codecs that
// rewrite the result after seeing the full stream (Claude rate-limit flip,
// OpenCode no-op detection) override this.
func (b *baseCodec) PostClassify(_ State, _ *runner.ExecuteResult) {}

// ParseTranscriptLine satisfies [Codec] for single-line transcript parsing by
// delegating to a fresh parser. Multi-line replay must use
// [Codec.NewTranscriptParser] directly so codec state is preserved across the
// stream.
func (b *baseCodec) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	return b.newParser().ParseTranscriptLine(runID, line)
}

// resolveBinary fills base's binary-resolution fields by looking cmd up on
// PATH. base must already carry the identity fields (binaryDesc, installHint,
// runnerType, …). It mirrors the "Available=false instead of error" contract
// every constructor relied on so the registry can register a stub when the
// binary is missing.
func resolveBinary(base baseCodec, cmd string) baseCodec {
	path, err := exec.LookPath(cmd)
	if err != nil {
		base.available = false
		base.message = base.binaryDesc + " not found in PATH"
		return base
	}
	base.binaryPath = path
	base.available = true
	base.message = base.binaryDesc + " available"
	return base
}

// testBase derives the *ForTest variant of an identity base: a fake binary
// path, Available=false, the given test message, and no install hint (so
// Available reports just the message). The identity fields (type/labels/tag)
// carry through unchanged so capability/labels/tag tests behave like the real
// codec.
func testBase(base baseCodec, fakePath, message string) baseCodec {
	base.binaryPath = fakePath
	base.available = false
	base.message = message
	base.installHint = ""
	return base
}

// standardBuildEnv builds the os.Environ()-shaped slice every codec's BuildEnv
// returns: the sanitized base env, the per-run tag entry, any codec-fixed
// extra vars, then the per-request overrides merged on top.
func standardBuildEnv(tagEnvKey, tag string, extras map[string]string, extraVars ...string) []string {
	env := runner.SanitizedBaseEnv()
	env = append(env, fmt.Sprintf("%s=%s", tagEnvKey, tag))
	env = append(env, extraVars...)
	return runner.AppendEnvMap(env, extras)
}
