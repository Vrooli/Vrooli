# Provider corpus — routing-recall test fixture

These descriptors are **test data for the routing-recall gate**, not a production
registration source. Until the Search Self-Tuning System work (plan Phase 2),
search-hub shipped every provider's `ProviderDescriptor` as a `//go:embed`'d seed
in `internal/providers/seeds/`. That embed path was deleted: providers now
**self-register** from their own `.vrooli/search.json` (see
`packages/searchregister-go`), so search-hub no longer carries any per-provider
descriptor in its binary.

The Ollama-gated classifier routing-recall gate
(`internal/routing/classifier_recall_test.go`) still needs a representative
multi-provider landscape to measure routing quality against — that is what these
files are. They are loaded only by tests (`os.ReadFile`), never embedded or
registered. `provider_corpus_test.go` guards that every file here is a valid,
registerable descriptor and pins the live-vs-capability-gap split, so the corpus
cannot silently rot.

A new live provider does NOT need a file here; it self-registers. Add one only to
keep the routing-recall corpus representative of the landscape the classifier
must route across.
