# Provider descriptor integrity fixtures

These descriptors are **test data for registry-contract validation**, not a
production registration source. Providers self-register from their own
`.vrooli/search.json` (see `packages/searchregister-go`), so search-hub does not
ship these fixtures as provider descriptors.

The former Ollama-gated classifier routing-recall test no longer exists. These
files remain only because `provider_corpus_test.go` validates descriptor JSON,
registry eligibility, IDs, and the live-vs-capability-gap split. They are loaded
only by tests (`os.ReadFile`), never embedded or registered.

A new live provider does NOT need a file here; it self-registers. Add a fixture
only when the registry-contract test needs a representative schema case.
