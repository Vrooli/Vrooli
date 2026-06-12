# Invariants

- Code Facts does not parse Go or TypeScript source directly.
- Fact-family filtering must not fabricate unsupported facts as success.
- Provider failures must be visible as `unsupported` or `unknown`.
- Cache keys must be deterministic and inspectable.
- CLI JSON output must match API response shapes.
- The generated `notes` domain is not product vocabulary.
