# Ecosystem Manager

**The autonomous generation-and-improvement control plane for the Vrooli
ecosystem.**

Ecosystem Manager creates new scenarios and resources and drives existing
ones toward an objective by running steer-skill agent loops — the
"auto-steer" control loop — through `agent-manager`, with a Trello-style
board for visibility and control.

> **Read [`docs/concepts/CONTROL-MODEL.md`](docs/concepts/CONTROL-MODEL.md)
> first.** It is the canonical mental model for how this scenario decides
> to improve a target: a closed-loop controller (diagnose → select →
> execute → measure → learn), with profiles as objective functions. The
> rest of the code and docs serve that model.

## What You Get

- **One control plane for four operations** — scenario/resource ×
  generator/improver — instead of four separate tools.
- **Auto-steer improvement loops** — apply steer skills across iterations
  with metric-gated stop conditions and quality gates, executed via
  `agent-manager`. (Today an open-loop schedule; being reframed as a
  closed-loop controller — see the control model.)
- **A Kanban board UI** — pending / in-progress / completed / failed /
  blocked, with steering configuration and execution history.
- **Persistent run history and metrics** — Postgres-backed analytics on
  throughput, success rate, and PRD-completion improvement.
- **A REST API and a thin CLI** — drive tasks, queue, steering, and logs
  programmatically.

Local surfaces (lifecycle-managed; ports are allocated by the lifecycle,
with `30500` as the dashboard/proxy URL):

- Dashboard: `http://localhost:30500`
- Health: `GET /health`

## Documentation Map

Full documentation lives under [`docs/`](docs/) and is registered in
[`docs/manifest.json`](docs/manifest.json) (scenario-docs v2 contract).

| Start here | Document |
|---|---|
| **The improvement-loop model (read first)** | [`docs/concepts/CONTROL-MODEL.md`](docs/concepts/CONTROL-MODEL.md) |
| Orientation for picking up work | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Run it locally | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| System shape and surfaces | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Capabilities and ownership | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Vocabulary | [`docs/concepts/GLOSSARY.md`](docs/concepts/GLOSSARY.md) |
| Control-loop state machine | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) |
| Data and storage | [`docs/concepts/DATA.md`](docs/concepts/DATA.md) |
| Dependencies | [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| API / CLI / config reference | [`docs/reference/`](docs/reference/) |
| Run, deploy, observe | [`docs/operations/`](docs/operations/) |
| Seams, testing, errors, decisions, problems | [`docs/internal/`](docs/internal/) |
| Known issues and the controller roadmap | [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) |

## Customize Safely

- **Change the improvement loop?** Update
  [`docs/concepts/CONTROL-MODEL.md`](docs/concepts/CONTROL-MODEL.md)
  first — it is the design authority — then the auto-steer code under
  `api/pkg/autosteer/`.
- **Change a profile?** Edit `profiles/<id>/profile.json` (filesystem,
  version-controlled). Profiles are configuration, not code.
- **Add an endpoint or capability?** Follow the extension rules in
  [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) and
  update the matching reference docs.
- **Run and manage it** only through the lifecycle —
  `vrooli scenario {start,stop,restart,status} ecosystem-manager` or the
  scenario `Makefile` (`make start|stop|logs|test`). Never execute the
  API binary directly.
- **Note the transport deviation:** this scenario serves REST/JSON, not
  proto/Connect-RPC. New work should respect that reality and the
  migration note in [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md);
  see [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md).

---

Ecosystem Manager is the engine that produces and hardens the rest of the
Vrooli portfolio. Making it a genuinely intelligent controller raises the
leverage of everything it touches — which is why the
[control model](docs/concepts/CONTROL-MODEL.md) is the most important
document in this scenario.
