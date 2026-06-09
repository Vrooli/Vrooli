# searchregister-go

The shared **self-registration bridge** between a scenario's `.vrooli/search.json`
SSOT and search-hub's `RegistryService`. A scenario that owns a searchable corpus
calls `Register` at boot to push its provider descriptor(s) to search-hub — so
search-hub can route to it and (later) run quality sweeps against it.

This replaces the old model where search-hub shipped every provider's descriptor
as a `//go:embed`'d build-time seed. Registration is now **scenario-owned**: the
provider declares itself, search-hub stores whatever registers.

## Why a separate module

```
aisearch-go ──parses──▶ search.json (descriptor + tuning + tests)
   │  (stays free of registry/transport vocab)
   ▼
searchregister-go  ──maps descriptor──▶ registry proto ──RegisterProvider──▶ search-hub
   ▲  (knows BOTH sides — that coupling is its whole job)
```

- `aisearch-go` keeps the `search.json` shape but **must not** import the
  registry/transport proto (it would bloat a deliberately lean library that many
  scenarios embed).
- `api-core` already has the proto + Connect + discovery deps, but it is
  foundational infrastructure and **must not** carry one scenario's contract
  vocabulary.
- So the bridge lives here: a thin search-domain client lib (parallel to
  `aisearch-go`), allowed to depend on both.

## API

```go
// Pure mapping: search.json provider block → registry ProviderDescriptor.
func Descriptor(p aisearch.ProviderConfig) (*registryv1.ProviderDescriptor, error)
func Descriptors(f aisearch.SearchFile) ([]*registryv1.ProviderDescriptor, error)

// Boot-time self-registration: read search.json, map, upsert with bounded retry
// and graceful degrade. Call from a background goroutine — search-hub is OPTIONAL,
// so this must never block or fail the scenario's own startup.
func Register(ctx context.Context, cfg Config) []Result
```

`Config` exposes seams (`ResolveBaseURL`, `NewClient`, `Retry`) that default to
production wiring (api-core discovery + a Connect client + a short boot-friendly
retry) and are overridden in tests. The `RegistryClient` interface is the narrow
RPC seam unit tests fake.

## What it carries

The descriptor maps the full `search.json` provider block — including the
`tuning` block (so search-hub reads the incumbent factors for sweeps) — and the
`tests` block is mirrored into search-hub's eval store as `<provider_id>.primary`
via `RegisterSuite` (`corpusStoreMirrorsFile`: the store is a cache of the file,
re-healed on every boot).

The per-provider **control token** flows both ways: `OnControlToken` receives the
token search-hub mints/echoes so the scenario can cache it (in memory) to gate its
own override / reindex / config-write verbs, and `ControlToken` supplies the
cached token back to Register so it is echoed as ownership proof on
re-registration (an empty token — first boot, holder not yet populated — is the
normal first-contact case).

## Consumers

`scenarios/cli-health/api` (`cli-health.commands`) and
`scenarios/knowledge-observatory/api` (`knowledge-observatory.docs`). Any scenario
that adopts `.vrooli/search.json` gains self-registration (and, downstream,
auto-tuning) by calling `Register` at boot.
