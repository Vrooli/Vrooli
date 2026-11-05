# Provenance search contract

Git Control Tower owns provenance identity and evidence standing. Search Hub
owns federation and ranking; it must not copy provenance rows into its own
store.

`RepoService/SearchProvenance` accepts a bounded request:

```json
{"query":"run-123 api/review.go","repoId":42,"limit":20}
```

Federated Search Hub calls may provide `scope:"repo:42"` instead of `repoId`.
An omitted scope is refused; the provider manifest does not invent a default
repository.

The repository identity is explicit. A missing or non-positive `repoId` is a
refusal, so a caller cannot accidentally search the UI's active repository.
The response has `available=false` when Workspace Sandbox cannot provide the
authoritative source. An available source with no matching rows returns an
empty `results` array. These states must remain distinct in Search Hub.

Each result has a stable opaque ID, exact run/sandbox/path fields, and an
evidence standing. Path overlap is only a search match; it is not committed
authorship. Private owner data is not added to the public snippet.

The current endpoint is intentionally lexical and bounded. Exact IDs and
revision-aware lookup are the next owner-owned extension; semantic ranking
belongs to Search Hub after this source binding is activated with explicit
repository scope.
