# Doc Parse

`doc-parse` is the repo-owned portable document parsing resource. It provides
one parser contract for office documents and PDFs,
including PDF classification, text extraction, tables, geometry, and explicit
terminal states for malformed, scanned, and password-protected inputs.

## Intent

- Resource ID: `doc-parse`
- Category: `developer-tooling`
- Driver: `native-cli`
- Parser backend: Rust (`anydoc` and `pdf-inspector`), selected as one WASI
  module executed through Wazero; native artifacts remain the fallback
- Artifact maturity: checksum-verified WASI artifact with a cross-platform Go boundary

The Rust shim is the parser implementation. The Go CLI is the stable resource
boundary: it resolves and verifies the selected WASI artifact, executes it
through Wazero, exposes health and capabilities, and returns structured parse
results to scenario callers.

## Use Cases

- Parse DOCX, ODT, EPUB, RTF, CSV, XLSX, and PPTX inputs to normalized Markdown.
- Classify PDFs as text-based or scan-like and preserve positioned text when
  geometry is requested.
- Return deterministic terminal states for malformed and password-protected
  inputs so document-manager can route them without scraping logs.

## Architecture

`cli/main.go` remains bootstrap-only while the Go boundary owns operator-facing
behavior. The Rust shim is built separately and is not treated as an installed
third-party executable.

- `resource.json` is the declarative authority for install, invoke, freshness, portability, and exported environment contracts.
- `cli/` is the single binary entrypoint and build/install surface.
- `cli/internal/app` is the default home for command registration and CLI wiring.
- `cli/internal/domain` owns resource-specific Go logic.
- `cli/internal/discovery`, `cli/internal/install`, `cli/internal/version`, and `cli/internal/env` carry the shared native-resource concerns around runtime resolution and build/install behavior.

The intended escalation path is:

1. express behavior in `resource.json`
2. keep `cli/main.go` as bootstrap only
3. add operator-facing command wiring in `cli/internal/app`
4. implement real resource behavior in `cli/internal/<domain>` packages

Current implementation surfaces:

- `cli/internal/app`: command registration and CLI wiring
- `cli/internal/domain`: parser invocation and result normalization
- `cli/internal/discovery`: runtime and source-root resolution helpers
- `cli/internal/install`: binary rebuild/install helpers
- `cli/internal/version`: manifest/build metadata helpers
- `cli/internal/env`: environment/config helpers

See [`docs/OPERATIONS.md`](docs/OPERATIONS.md) for the build lane and
artifact-packaging contract. The native/WASI comparison is recorded in
[`docs/measurements.md`](docs/measurements.md).
