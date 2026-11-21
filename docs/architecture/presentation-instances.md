# Presentation Instances

A presentation instance is a named non-live instance of a scenario (`<scenario>@presentation`)
that a recording, screenshot, or demo runs against instead of the operator's live instance.
It exists so launch media shows the real product without showing the operator's real data.

This page states what a presentation instance guarantees today, what it does not, and the
design for the one remaining gap: data that arrives from other scenarios.

Validated live on 2026-09-17: `web-console@presentation` ran beside live with its own
ports, storage and empty session table (0 against live's 647), carried none of the caller's
session environment, produced a clean product capture, failed closed on a listed dependency
that was not running, and left live healthy when it was stopped.

## What the instance owns

| Concern | Owner | Guarantee |
|---|---|---|
| SQLite / Postgres / Redis / Qdrant | `api-core/storage` namespace helpers | `<scenario>_<variant>` names |
| Config, data, cache, logs, state directories | `api-core/storage` `Resolver.Resolve` | `<class-root>/vrooli/<scenario>_<variant>` |
| Ports, registry rows, records, locks | `internal/scenarioruntime` `InstanceKey` | one row per instance; a variant never claims a manifest fixed port |
| Process environment | `packages/envkit-go` | caller-session variables stay with the caller |
| Source, build outputs, process working directory | `internal/lifecycle` `effectiveSourceDir` / `Scenario.SourcePath()` | routed per instance; `Path` stays the manifest origin |

Two rules follow from the table and are enforced in code:

- **Fixed ports are live-only.** `liveFixedPorts` gates both start-time orphan cleanup and
  stop-time teardown, so stopping a variant cannot kill the live instance's fixed-port
  process.
- **A variant never rebuilds shared outputs.** Without a Baseline Modes engagement every
  instance builds from the working tree, so `admitVariantBuild` refuses a variant's setup
  while another instance of that scenario is running. Restart live first, then start the
  variant. Editing a shared package is enough to trigger this, because it makes every
  dependent scenario stale, so a variant start needs a quiet working tree.

## What still arrives live

A presentation instance calls its dependencies, and the lifecycle starts dependencies as
live instances (`internal/lifecycle/dependencies.go`). Three classes of data therefore
still reach the screen:

1. **Duplicable scenario dependencies** — a dependency that can run as its own instance.
2. **Single-instance services** — the control plane and the node fleet Bridge reports.
   There is one of each per host; they cannot be duplicated per presentation.
3. **Host facts** — the hostname in a shell prompt, real paths, real device names.

### 1. Duplicable dependencies: the follow list

A non-live instance may resolve named dependencies at its **own** variant:

```bash
vrooli scenario start web-console --instance presentation \
  --variant-dependencies vrooli-bridge,audio-tools
```

- The lifecycle validates the names against `dependencies.scenarios`, refuses the flag for a
  live instance, and publishes `VROOLI_VARIANT_DEPENDENCIES` to the instance.
- `cli-core/cliutil` resolves a listed dependency to `<dependency>@<own variant>`; an explicit
  address still wins, and an unlisted dependency stays live.
- `api-core/discovery` **fails closed** for a listed dependency: when that instance is not
  running the lookup returns an error instead of falling back to live. A fallback would
  deliver exactly the data the list exists to keep out of a recording.

The list is opt-in per dependency because following every dependency automatically would
cascade through each dependency's own graph and reach services that cannot be duplicated.

### 2. Single-instance services: fixture stubs (design, not built)

**Measured 2026-09-17: the fleet is not one of these.** `vrooli-bridge` starts as a
presentation instance with its own (empty) node database while the live Bridge keeps
serving the real fleet, and a `web-console@presentation` following it offers only
"This machine" in its machine picker — no node names, no fleet. The fleet view is
therefore already covered by the follow list in §1. A stub would only be needed to
*populate* a demo fleet for a richer shot, which is an aesthetic want, not a safety
requirement. Confirm this per scenario before assuming a dependency needs a stub.

For a service that genuinely has no second instance, the presentation instance needs an
**answering surface that is not the live one**.

Shape:

- A stub serves the same Connect procedures as the real service, from authored fixture
  responses — demo machines, demo nodes — validated against the service's proto messages so
  a fixture cannot drift into a shape the client rejects.
- Discovery routes to it through the same follow list: the stub registers as
  `<scenario>@presentation`, so no client code changes and no scenario learns a demo mode.
- Fixtures live with the presentation material they serve, and are reviewed like any other
  published content.

Decided 2026-09-17, to apply whenever a stub is actually built:

- **Where it runs: its own scenario.** A process the capture tool starts dies with the tool
  and cannot be inspected; a mode of a test surface drags test-mode leasing into something
  that must serve a person for the length of a recording. A scenario gets ports, health, and
  lifecycle, which is what lets the follow list address it at all.
- **How fixtures are authored: by hand from the proto, reviewed like published content.**
  Sanitizing a recording leaks whatever the author forgot to look for; hand-authoring cannot
  leak by omission. A recorder is a later convenience, never the trusted path.
- **An unfixtured procedure refuses**, matching the follow list. An empty result looks like a
  working product with no data, which is exactly the frame that reaches a video unnoticed.

### 3. Host facts

Not a routing problem. A staged session sets its own prompt and works in a scratch
directory; nothing in the product needs to change.

## What a presentation instance is not

- It is not a Baseline Modes engagement. An engagement splits code between a frozen copy and
  the working tree so a candidate can be developed; a presentation instance runs the same
  code as live and only differs in data.
- It is not a test-mode lease. `X-Vrooli-Test-Mode` routes one request to a leased pool for
  the length of an automated run; a presentation instance serves a person for as long as a
  recording takes.
