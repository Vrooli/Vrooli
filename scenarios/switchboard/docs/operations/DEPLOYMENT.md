# Deployment — Switchboard

## Purpose Of This Document

How this scenario is packaged, where it can run, and what must be true of the
host before it starts. Use it to answer: which tiers are supported, what the
runtime needs, and what a release and a rollback involve.

## Supported Tiers

| Tier | Support | Notes |
|---|---|---|
| **Tier 1 — local scenario** | **Primary** | The intended deployment. API, CLI, and console on the owner's own machine |
| Tier 1 on a macOS fleet node | **Required for iMessage** | A second install on a Mac, reached through `vrooli-bridge`. This is not an optional nicety — it is how the most valuable channel works, and it is why no resource whose macOS support is unsupported or unproven may ever be added |
| Tier 2 — desktop app | Possible, unplanned | The console would package, but a desktop bundle running an inbound listener needs an always-on story this scenario does not have yet |
| Tier 3 — hosted | **Rejected for the default product** | Hosting the inbound path means the operator's private conversations transit somebody else's machine, which forfeits the custody promise the scenario exists to make. Only a deliberately labelled relay offering could ever be hosted |

## Runtime Requirements

| Requirement | Needed for | Notes |
|---|---|---|
| Go toolchain, Node toolchain | Build | Per the template |
| Writable scenario storage root | Always | SQLite database plus media blobs. Media growth is the storage line item to watch |
| Outbound HTTPS | Any external channel | Telegram and Slack are outbound-initiated; no inbound port needed for either |
| A public origin via `tunnel-manager` | Webhook-ingress channels only | Poll-ingress and in-app channels need none. Declared per descriptor as `requires: public_origin` |
| A reachable Mac fleet node | iMessage only | Registered with `vrooli-bridge`, running the node agent, signed into Messages |
| Full Disk Access on that Mac | iMessage ingress only | Apple provides no supported inbound interface. Granted by the operator, never by an installer |
| `agent-manager`, `prompt-manager`, `scenario-authenticator` running | Everything | Required dependencies; the scenario starts but refuses turns without them |
| **No Docker** | — | Deliberate. Zero resource dependencies at P0 |

## Packaging

Standard template packaging: a Go API binary, a Go CLI binary, and a built
console served by the scenario. Two additions specific to this scenario:

- **Channel descriptors ship as data**, in `data/channels/`. They are part of the
  release artifact and are versioned with it, but an operator may add their own
  file without rebuilding anything. That is the point of the mechanism.
- **The installed console is a progressive web app.** The seeded
  `site.webmanifest`, `sw.js`, maskable icons, relative install asset URLs and
  safe-area tokens are kept valid; generic template icons are replaced when real
  branding exists.

## Release Checklist

1. `make test` green, including the `channel-conformance` phase for every
   registered adapter.
2. `vrooli scenario requirements validate switchboard --json` → `PASSED`.
3. Every descriptor in `data/channels/` validates, and each carries a
   `schemaVersion` the running build recognises.
4. `.vrooli/service.json` dependencies still agree with
   `docs/concepts/INTEGRATIONS.md`. A disagreement is a release blocker, not a
   documentation lag.
5. Migration story confirmed: greenfield declarative schemas only, or a stated
   reason otherwise.
6. If a Mac node is in the fleet, confirm it is on the same project revision —
   a version-skewed node fails a dispatch in a way that reads as a channel
   outage.
7. Secrets resolve by reference. No credential value appears in any
   configuration file, log line, or read endpoint response.
8. `docs/internal/PROGRESS.md` updated with what shipped and the true frontier.

## Rollback

| Change | Rollback | Risk |
|---|---|---|
| API or CLI binary | Redeploy the previous binary | Low. In-flight turns fail and are visible on their threads |
| A descriptor | Restore the previous file and restart | Low, but a descriptor that widened a limit may have allowed sends the previous one rejects — check the thread, not just the file |
| A schema change | **No automatic path while greenfield** | Declarative schemas do not roll back. This is acceptable only until production data exists, at which point versioned migrations are earned |
| A channel adapter | Remove the adapter; the descriptor remains and reports `unimplemented` | Low, and deliberately visible rather than hidden |
| A Mac node going away | Host-bound channels report unavailable with a reason | None to data. Threads and history are retained |

**The rollback that does not exist:** there is no way to un-send a message. A
defect that causes the agent to say something wrong to a real person is not
recoverable by deploying a fix, which is why the trust and budget controls are
fail-closed rather than fail-open.

## Cross-References

- `docs/operations/RUNBOOK.md` — start, stop, and incident response
- `docs/operations/OBSERVABILITY.md` — what to watch after a release
- `docs/concepts/INTEGRATIONS.md` — the dependency contract checked in step 4
- `docs/concepts/DATA.md` — the migration story referenced in step 5
- `docs/internal/DECISIONS.md` — why no resource with weak macOS evidence is permitted
