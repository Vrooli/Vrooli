# Switchboard

Reaches an owner's agents on the messaging channels people already use, and
governs what each may do for whom.

An agent that is *somewhere* — reachable the way a person is, and safe to give
out. A message arrives from outside Vrooli (iMessage, SMS, Telegram, Slack, a
call, or this scenario's own in-app thread), Switchboard resolves which agent
that address belongs to, decides what that particular sender is allowed to ask
for, and runs the agent inside those bounds.

The direction that does not exist anywhere else in Vrooli is **inbound**.
`notification-hub` already delivers to a human and even sends iMessage from a
Mac. Nothing turns an *arriving* message into an agent turn. That is this
scenario.

> **Status: documentation-first. No product code exists yet.**
> Generated 2026-09-01 from the `react-vite` template. The charter, requirements
> registry, domain map, dependency contract, design rationale, and experience
> specs are authored; no domain is implemented. `template-manager orient
> switchboard` reports **8 of 9 gates complete** — the two outstanding items both
> require code. Read [`docs/internal/PROGRESS.md`](docs/internal/PROGRESS.md)
> for the true frontier before starting work.

> **Start here:** [`docs/START-HERE.md`](docs/START-HERE.md) owns the
> initialization protocol. Run `make orient` for machine-readable gate status.

## What You Get

**The permanent capability:** being reachable. Every scenario in Vrooli produces
something worth saying; none of them should own a transport, a thread, or a trust
decision. Those live here once.

- **Many channels, one contract.** A channel is a validated JSON descriptor plus
  one adapter. Adding Slack, Discord or Signal is two new files and zero edits
  anywhere else — no change to the core, the console, the funnel, or the tests.
- **A trust guard that fails closed.** Effective scope is the narrowest of the
  sender's tier, the thread's ceiling, and the agent's capability grant, resolved
  *before the agent reads the message*. An owner-only scope is unreachable from a
  lower tier by construction, not by configuration.
- **Group threads that narrow rather than widen.** A room's ceiling is the lowest
  tier present, so adding a stranger lowers the room instead of raising the
  stranger.
- **Cost controls that are structural.** An agent-authored message never starts a
  turn, and every thread has an hourly budget and a spend cap. Two agents in one
  room cannot spend metered inference indefinitely.
- **Custody.** The thread, the agent, and the credentials stay on the owner's
  machine. Reaching Apple means reaching the owner's own Mac through their own
  fleet link — never a vendor's relay.
- **Depth behind the bubble.** The message is a front door onto every scenario
  the ecosystem has, reached through `agent-manager` with no integration written.

Standard template shape underneath: Go API (`api/`), Go CLI (`cli/`), React +
Vite UI (`ui/`), proto contracts, SQLite through the `api-core` storage seam,
lifecycle metadata in `.vrooli/service.json`, and the PWA install surface.

## Documentation Map

| Question | Document |
|---|---|
| What is this and what must it do? | [`PRD.md`](PRD.md), [`requirements/`](requirements/) |
| Where do I start working? | [`docs/START-HERE.md`](docs/START-HERE.md) |
| What has actually been done, and what is next? | [`docs/internal/PROGRESS.md`](docs/internal/PROGRESS.md) |
| What are the bounded contexts and their build order? | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| What is the shape, and where does new code go? | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| What happens in what order? | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) |
| What is stored, retained, deleted? | [`docs/concepts/DATA.md`](docs/concepts/DATA.md) |
| What does this depend on, and how does it fail? | [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Why is it like this? | [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md) |
| What is the attack surface? | [`docs/internal/SECURITY.md`](docs/internal/SECURITY.md) |
| What is known-broken or unowned? | [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) |
| How is it run, watched, recovered? | [`docs/operations/`](docs/operations/) |
| How does it earn its keep? | [`docs/business/`](docs/business/) |
| What are the screens and journeys? | [`experience/`](experience/), [`DESIGN.md`](DESIGN.md) |

## Customize Safely

**The generated scaffold is not the product.** Replace these:

- The `notes` domain (proto, API, CLI, UI feature) — a worked vertical slice to
  copy once, then delete with `template-manager detemplate switchboard`. Do not
  run that until one real domain is green.
- Starter page content and the dashboard metric placeholders.
- Generic PWA icons, once real branding exists.

**Keep these — they are durable infrastructure, not decoration:**

- The responsive shell: `min-h-dvh` sizing, overflow-contained main content,
  fixed safe-area bottom navigation on mobile, desktop sidebar, theme controls,
  Settings-owned locale switching.
- i18n wiring (`SUPPORTED_LOCALES`, `useTranslation`, the locale switcher),
  accessibility primitives (`role`, `aria-*`, `data-testid`), design-token
  plumbing, the governed primitives under `ui/src/components/ui/`, and the
  feature-folder pattern under `ui/src/features/<name>/`.
- The `DESIGN.md` token contract. It is adopted unmodified, and the rationale is
  recorded at the top of that file.

**Scenario-specific rules that are not negotiable:**

1. **Never branch on a channel identifier above the adapter layer.** Ask the
   descriptor. Every `if channel == "..."` is a future edit site and the reason
   this scenario exists in the shape it does.
2. **Build one domain at a time**, in the order fixed by `DOMAINS.md`:
   `channels` → `conversations` → `agents` → `trust` → `turns`. Within a domain,
   follow the `ARCHITECTURE.md` extension order: proto, API, transport, CLI, UI.
   Do not build every API, then every CLI, then every UI.
3. **Never log a message body.** Stricter than "do not log secrets", and easy to
   violate accidentally in a debug statement on the ingress path.
4. **Agent descriptors are held by reference into `prompt-manager`.** Never store
   a copy of a descriptor this scenario does not own.
5. **Ship the in-app adapter and Telegram together**, not sequentially. Two
   adapters on day one is the only cheap proof the contract is channel-neutral.

## Running The Scenario

Always through the lifecycle. Never run a binary directly — that bypasses process
naming, port allocation, and health checks.

```bash
make setup      # first-time dependency and codegen setup
make start      # or: vrooli scenario start switchboard
make status
make logs
make test
make stop
make orient     # initialization gate progress
```

Dependency work flows through `scenario-dependency-analyzer`. Never run a raw
package manager, and never hand-edit the approved-dependencies file.

## Working Rules

- Run test suites with `vrooli scenario test switchboard`. The run is
  server-owned and survives a cancel; to wait, block **once** with
  `test-genie runs wait --json`. Never poll.
- Test the desired behavior, not the current implementation.
- File defects outside this scenario's scope with the `report-bug` skill to
  `scenario-qa`. Record completed non-trivial work with
  `vrooli-memory journal note --kind work-record`.
- `pnpm` everywhere for JavaScript workspaces; never `npm` or `yarn`.
