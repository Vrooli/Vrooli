# Ollama Known Problems

## Work ladder

- **W3 managed-service migration:** Complete. The default and only Ollama
  runtime is the checksum-verified, Vrooli-supervised native artifact. Phase 6
  removed the Docker runtime declaration and compatibility override after
  verifying the exact container, image tags, and unused volume were gone.
- **Evidence (2026-08-15):** Native Ollama restarted with Docker stopped,
  passed `health-ready` and GPU health after a model was loaded, and served the
  unchanged 19-model inventory totaling 62,434,233,666 logical bytes.
  `vrooli scenario test search-hub` passed 21/21 phases with Docker stopped.

## Deliberately unfixed

- **Conditional target artifacts (owner: resource-platform maintainers):**
  Linux arm64 and Windows server bundles are not staged, and macOS native
  target smoke evidence is not yet recorded. The deployment contract therefore
  keeps those targets conditional or unsupported rather than claiming release
  support.
- **First-run model registry dependency (owner: Ollama resource maintainers):**
  model pulls still require registry/network availability on a cold data root;
  the native server is bundled, but model weights remain regenerable external
  data. This is an explicit operational constraint, not a lifecycle defect.
