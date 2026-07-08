package cliapp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Static primitive-evidence artifact.
//
// This is the production channel behind verified L4 command-architecture
// maturity. A scenario's declared architecture (architecture.primitive in
// cli/manifest.json) is forgeable text; the primitive a handler was ACTUALLY
// built from is observed evidence only a cli-core primitive builder can produce
// (the evidence-bearing fields are unexported — see primitive.go / app.go). This
// file exports that observed evidence to a committed, versioned JSON artifact
// that CLI Health reads STATICALLY — it never has to execute the scenario's
// commands to learn what primitive each was built from (plan decision D1).
//
// The artifact is GENERATED, not handwritten, and lives at a scenario-local
// generated path (EvidenceArtifactRelPath) rather than beside the handwritten CLI
// code (plan decision D2). It carries explicit do-not-edit / source attribution
// so a reader — human or tool — can tell at a glance that it is generated and how
// to regenerate it.
//
// The exporter reads assembled command metadata only. Assembling a scenario's
// command tree (each domain's Register -> LoadFromManifestPrimitives) wires
// handler closures but never invokes them, so evidence generation has no command
// side effects. A generator wires a core-less command tree, calls
// BuildPrimitiveEvidence, and writes the artifact; see the exporter tests and
// scenarios/cli-health/cli for the reference generator.

// EvidenceSchemaID identifies the static primitive-evidence artifact format.
// CLI Health rejects an artifact whose schema it does not recognize.
const EvidenceSchemaID = "cli-primitive-evidence/v1"

// EvidenceGeneratorVersion is the version of this exporter. It is recorded in
// every artifact so a consumer can tell which generator produced it; bump it
// when the exporter changes what it records.
const EvidenceGeneratorVersion = "v1.0.0"

// EvidenceArtifactRelPath is the canonical scenario-relative location of a
// scenario's committed static primitive-evidence artifact. It lives under a
// generated/ directory in .vrooli — NOT beside the handwritten cli/ code — so it
// visibly reads as a generated contract rather than source (plan decision D2).
// Resolve an absolute path with EvidenceArtifactPath.
const EvidenceArtifactRelPath = ".vrooli/generated/cli-primitive-evidence.json"

// EvidenceArtifactFilename is the deprecated basename of the pre-migration
// artifact location (scenarios/<name>/cli/primitive-evidence.json, beside
// cli/manifest.json). The CLI Health reader still falls back to it so scenarios
// mid-migration validate, but new evidence must be written to
// EvidenceArtifactRelPath. Prefer EvidenceArtifactPath for new code.
const EvidenceArtifactFilename = "primitive-evidence.json"

// EvidenceDoNotEditNotice is the do-not-edit banner recorded in every generated
// artifact. It is a constant so the artifact stays byte-stable for golden diffs.
const EvidenceDoNotEditNotice = "GENERATED FILE — do not edit by hand. Regenerate it from the scenario's CLI evidence test (UPDATE_CLI_EVIDENCE=1), which rebuilds this artifact from cli/manifest.json and the cli-core primitive handlers."

// EvidenceSourceManifest is the scenario-relative path of the manifest the
// artifact is generated from, recorded for provenance. Constant for golden
// stability.
const EvidenceSourceManifest = "cli/manifest.json"

// EvidenceArtifactPath resolves the canonical primitive-evidence artifact path
// under a scenario root (the directory that contains cli/ and .vrooli/).
func EvidenceArtifactPath(scenarioRoot string) string {
	return filepath.Join(scenarioRoot, filepath.FromSlash(EvidenceArtifactRelPath))
}

// PrimitiveEvidenceArtifact is the committed, static record of the cli-core
// primitive each of a scenario's commands was built from. CLI Health reads it
// as the unforgeable side of the verified-primitive contract.
type PrimitiveEvidenceArtifact struct {
	// Schema is EvidenceSchemaID. A consumer that does not recognize it must
	// refuse to derive maturity from the artifact rather than trust unknown data.
	Schema string `json:"schema"`
	// DoNotEdit is EvidenceDoNotEditNotice: a self-describing banner declaring the
	// artifact generated and how to regenerate it. Attribution only; consumers
	// never key behavior on it.
	DoNotEdit string `json:"do_not_edit"`
	// SourceManifest is EvidenceSourceManifest: the scenario-relative manifest the
	// artifact was generated from (provenance).
	SourceManifest string `json:"source_manifest"`
	// Scenario is the scenario id the artifact was generated for.
	Scenario string `json:"scenario"`
	// Generator is the EvidenceGeneratorVersion that produced the artifact.
	Generator string `json:"generator"`
	// ManifestHash is the SHA-256 (hex) of the manifest bytes the artifact was
	// generated against. CLI Health re-hashes the on-disk manifest and treats a
	// mismatch as a stale artifact — the evidence describes a manifest that no
	// longer matches the source, so it cannot award verified maturity.
	ManifestHash string `json:"manifest_hash,omitempty"`
	// Commands is the per-command evidence, sorted by Path for stable diffs.
	Commands []CommandPrimitiveEvidence `json:"commands"`
}

// CommandPrimitiveEvidence records, for one command, both what the manifest
// DECLARES (declared_primitive / declared_exception) and what cli-core OBSERVED
// when the handler was built (observed_primitive). Threading both onto one entry
// lets a reader detect declared-only debt and declaration/implementation
// contradictions without re-parsing the manifest.
type CommandPrimitiveEvidence struct {
	// Path is the normalized command path ("group command", or "command" for a
	// top-level command). It is the key CLI Health matches against the manifest.
	Path string `json:"path"`
	// Group is the source subcommand group (empty for a top-level command).
	Group string `json:"group,omitempty"`
	// Command is the leaf command name.
	Command string `json:"command"`
	// Binding is the "Service.Method" connect-rpc binding key, when known.
	Binding string `json:"binding,omitempty"`
	// DeclaredPrimitive is the manifest architecture.primitive, if any.
	DeclaredPrimitive string `json:"declared_primitive,omitempty"`
	// DeclaredException is the manifest architecture.exception class, if any.
	DeclaredException string `json:"declared_exception,omitempty"`
	// ObservedPrimitive is the cli-core primitive class the handler was built
	// from (Command.PrimitiveEvidence()). Empty means no primitive evidence.
	ObservedPrimitive string `json:"observed_primitive,omitempty"`
}

// EvidenceExportInput carries the assembled command metadata the exporter reads.
// Groups and TopLevel must be the result of the scenario's own registration
// (which stamps the command's unexported evidence via a cli-core builder); the exporter never builds handlers or
// runs commands itself.
type EvidenceExportInput struct {
	// Scenario is the scenario id recorded in the artifact.
	Scenario string
	// ManifestRaw is the cli/manifest.json bytes the command tree was assembled
	// from. Used for the ManifestHash and to enrich per-command binding keys.
	// Optional: omit to skip hashing/binding enrichment.
	ManifestRaw []byte
	// Groups are the assembled subcommand groups (evidence already attached).
	Groups []SubcommandGroup
	// TopLevel are commands registered outside any group (e.g. a top-level
	// durable-run exception command wired via Command.WithPrimitive).
	TopLevel []Command
}

// BuildPrimitiveEvidence exports a PrimitiveEvidenceArtifact from assembled
// command metadata WITHOUT invoking any handler. It reads Command.Architecture
// (declared) and the command's observed primitive evidence (Command.PrimitiveEvidence(), stamped by the primitive
// builders attached by construction.
//
// It fails when a command's declared primitive/exception contradicts the
// observed primitive: an artifact must never record a contradiction as if it
// were evidence. (LoadFromManifestPrimitives already rejects contradictions at
// assembly time; this is the independent guarantee for top-level WithPrimitive
// commands and a defense-in-depth check for the manifest path.)
func BuildPrimitiveEvidence(in EvidenceExportInput) (PrimitiveEvidenceArtifact, error) {
	artifact := PrimitiveEvidenceArtifact{
		Schema:         EvidenceSchemaID,
		DoNotEdit:      EvidenceDoNotEditNotice,
		SourceManifest: EvidenceSourceManifest,
		Scenario:       strings.TrimSpace(in.Scenario),
		Generator:      EvidenceGeneratorVersion,
	}
	if len(in.ManifestRaw) > 0 {
		sum := sha256.Sum256(in.ManifestRaw)
		artifact.ManifestHash = hex.EncodeToString(sum[:])
	}

	bindings := manifestBindingIndex(in.ManifestRaw)

	var entries []CommandPrimitiveEvidence
	for _, g := range in.Groups {
		for _, c := range g.Subcommands {
			entry, err := commandEvidence(g.Name, c, bindings)
			if err != nil {
				return PrimitiveEvidenceArtifact{}, err
			}
			entries = append(entries, entry)
		}
	}
	for _, c := range in.TopLevel {
		entry, err := commandEvidence("", c, bindings)
		if err != nil {
			return PrimitiveEvidenceArtifact{}, err
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	artifact.Commands = entries
	return artifact, nil
}

// commandEvidence builds one CommandPrimitiveEvidence entry and fails on a
// declared-vs-observed contradiction.
func commandEvidence(group string, c Command, bindings map[string]string) (CommandPrimitiveEvidence, error) {
	path := normalizeEvidencePath(strings.TrimSpace(group + " " + c.Name))
	declared := c.Architecture
	observed := c.primitiveEvidence

	// Contradiction guards, mirroring the manifest classifier so the artifact and
	// cli-health agree on what "contradiction" means.
	if declared.Primitive != "" && ClassifyPrimitiveEvidence(declared.Primitive, observed) == EvidenceContradiction {
		return CommandPrimitiveEvidence{}, fmt.Errorf("command %q declares architecture.primitive %q but its handler was built with the %q primitive", path, declared.Primitive, observed)
	}
	if declared.Exception != "" && observed != "" && observed.SatisfiesException() != declared.Exception {
		return CommandPrimitiveEvidence{}, fmt.Errorf("command %q declares a %s exception but its handler was built with the %q primitive (which satisfies %q)", path, declared.Exception, observed, observed.SatisfiesException())
	}

	return CommandPrimitiveEvidence{
		Path:              path,
		Group:             group,
		Command:           c.Name,
		Binding:           bindings[path],
		DeclaredPrimitive: string(declared.Primitive),
		DeclaredException: string(declared.Exception),
		ObservedPrimitive: string(observed),
	}, nil
}

// manifestBindingIndex parses the manifest permissively and maps normalized
// command path -> "Service.Method" binding key. Best-effort: a manifest that
// does not parse yields an empty index (binding enrichment is informational).
func manifestBindingIndex(raw []byte) map[string]string {
	out := map[string]string{}
	if len(raw) == 0 {
		return out
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return out
	}
	for _, g := range m.Groups {
		for _, c := range g.Commands {
			path := normalizeEvidencePath(strings.TrimSpace(g.Name + " " + c.Name))
			out[path] = c.Binding.BindingKey()
		}
	}
	return out
}

// normalizeEvidencePath collapses internal whitespace so evidence paths match
// the normalization cli-health uses (see normalizeCommandPath there).
func normalizeEvidencePath(path string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(path)), " ")
}

// ObservedPrimitives projects the artifact to a command-path -> observed
// primitive map — the shape CLI Health consumes as unforgeable evidence.
func (a PrimitiveEvidenceArtifact) ObservedPrimitives() map[string]PrimitiveClass {
	out := make(map[string]PrimitiveClass, len(a.Commands))
	for _, c := range a.Commands {
		if c.ObservedPrimitive == "" {
			continue
		}
		out[c.Path] = PrimitiveClass(c.ObservedPrimitive)
	}
	return out
}

// ParsePrimitiveEvidence decodes artifact bytes and checks the schema id. A
// wrong/absent schema is an error so a consumer never derives maturity from an
// unrecognized artifact format.
func ParsePrimitiveEvidence(raw []byte) (PrimitiveEvidenceArtifact, error) {
	var a PrimitiveEvidenceArtifact
	if err := json.Unmarshal(raw, &a); err != nil {
		return PrimitiveEvidenceArtifact{}, fmt.Errorf("parse primitive evidence artifact: %w", err)
	}
	if a.Schema != EvidenceSchemaID {
		return PrimitiveEvidenceArtifact{}, fmt.Errorf("primitive evidence artifact schema %q is not %q", a.Schema, EvidenceSchemaID)
	}
	return a, nil
}

// MarshalPrimitiveEvidence renders the artifact as indented JSON with a trailing
// newline (stable, diff-friendly, matches how the generator writes it).
func MarshalPrimitiveEvidence(a PrimitiveEvidenceArtifact) ([]byte, error) {
	body, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal primitive evidence artifact: %w", err)
	}
	return append(body, '\n'), nil
}

// WritePrimitiveEvidence writes the artifact to path (creating parent dirs). Use
// it from a scenario's evidence generator (a build/test helper).
func WritePrimitiveEvidence(path string, a PrimitiveEvidenceArtifact) error {
	body, err := MarshalPrimitiveEvidence(a)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create evidence dir %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write primitive evidence artifact %q: %w", path, err)
	}
	return nil
}
