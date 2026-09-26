# Tunnel Manager UX Direction

## Status

This is the design brief for the next Tunnel Manager UX pass. It is a planning
document, not a claim that the current UI already provides these interactions.
The machine-readable contract is under [`../../experience/`](../../experience/)
and currently marks all pages and journeys as `draft`.

## Product promise

Tunnel Manager should let an operator answer three questions quickly:

1. What is publicly reachable right now?
2. Is each route trustworthy from local service through the public URL?
3. What is the safest next action if a route is missing, expiring, or unhealthy?

The primary repeated action is exposing a scenario for a bounded period. The
primary safety concern is making the publicness and authentication boundary
understandable before the operator confirms that action.

## Recommended direction: Exposure Command Center

The product should be exposure-first rather than settings-first or metrics-
first. The Overview is a command center with a large operational visualization
and an action-oriented findings rail. The Exposure surface owns the detailed
route inventory and the guided expose flow.

The intended destination structure is:

```text
Overview
Exposures
  Active / Expiring soon / Not exposed
Diagnostics
  Route health / Tunnel metrics / Recovery events
Governance
  Port compliance / Ingress drift / External routes
Settings
  Cloudflare readiness / /public/* policy / Appearance
```

The current route set remains registered in the experience index while this
direction is implemented: `/`, `/exposure`, `/recovery`, `/metrics`, `/audit`,
`/drift`, and `/settings`. Navigation consolidation is a later implementation
decision and must not leave stale experience pages behind.

## Signature visual: Exposure Constellation

The Overview should show a central Cloudflare tunnel node with scenario nodes
around it. Every visual mark must correspond to real state:

| Visual | Meaning |
|---|---|
| Central tunnel node | Cloudflared and tunnel readiness |
| Scenario node | One exposed route or eligible route candidate |
| Ring segment 1 | Local service is reachable |
| Ring segment 2 | Tunnel ingress is configured |
| Ring segment 3 | External URL probe passes |
| Ring segment 4 | Access policy is reconciled |
| Solid connection | Active and healthy route |
| Pulsing connection | Provisioning, reconciling, or probing |
| Amber connection | Degraded, stale, or expiring route |
| Dashed connection | Configured but not currently verified |
| Lease arc | Remaining time on a leased route |
| Selected node | Opens route detail and next actions |

The ring is analogous to the meaningful control-ladder visualization used by
Infrastructure Manager. It should provide an explanation, not merely a score.
If a summary score is introduced, it must be decomposable into the route and
signal counts that produced it.

## Publicness boundary

Every route detail and exposure confirmation must state:

```text
Main application: protected
/public/* assets: anonymous only when explicitly enabled
Whole-host bypass: never allowed by Tunnel Manager
```

The UI must distinguish disabled, enabled and reconciled, pending sync, missing
Access credentials, insufficient scope, and policy-reconciliation failure.
The `/public/*` capability is remote-mode only and requires Cloudflare Access
Apps and Policies Edit permission. A missing API token and an insufficient
token scope are different operator problems and need different remediation.

## Guided expose flow

The flow is a three-step review:

1. **Choose** — select a discoverable scenario, not a raw scenario-name text
   field. Show running state, fixed-port compliance, prior exposure, and any
   credential or lifecycle blocker.
2. **Configure** — choose a bounded duration such as one hour, one day, seven
   days, or a custom value. Show the proposed hostname, target, tier, primary-
   app authentication, and explicit `/public/*` opt-in.
3. **Confirm** — repeat the exact consequence: scenario, URL, duration,
   expiry, local target, authentication boundary, and whether the action may
   start the scenario or change tunnel ingress.

After submission, show:

```text
Read manifest → validate port → reconcile ingress → reconcile policy →
probe local target → probe public URL → publish result
```

The success state includes the public URL, expiry, policy scope, and local,
tunnel, and external verification results.

## Route detail

Selecting a route should open a drawer or inspector with a left-to-right route
journey:

```text
localhost target → Cloudflare ingress → public URL
```

Each stage names its signal and last verification time. The inspector provides
open/copy URL, extend lease, disable exposure, run probes, or exact remediation
actions.

## Required operational states

All async surfaces must intentionally design loading, ready, empty, partial,
stale, request-error, success, and retry behavior. Tunnel Manager also needs
explicit states for local mode, remote mode, missing API token, insufficient
scope, fixed-port non-compliance, provisioning, probe pending, expiring lease,
expired lease, recovery backoff or breaker open, and Access policy pending or
failed.

An empty route list means “nothing is exposed.” It must not look like a failed
request or an unavailable route inventory.

## Visual polish principles

- Use a calm dark operational shell with bright semantic signal colors.
- Keep the constellation sparse, legible, and useful at a glance.
- Use restrained motion only for real lifecycle changes.
- Keep route nodes keyboard reachable and provide a text/table equivalent.
- Do not use geographic maps, fake traffic particles, unexplained percentages,
  or decorative network lines.
- Preserve light, dark, system, reduced-motion, RTL, and mobile behavior.
- On mobile, transform the constellation into a route list with the same legend
  and a bottom-sheet route detail.

## Component-library adoption plan

Adopt reusable assets only after their stories are reviewed in the Tunnel
Manager context. Candidate assets include `SidebarShell`, `CommandCenterShell`,
`DataTable`, `ResourceCollection`, `DrawerShell`, `Dialog`, `EmptyState`,
`StatusIndicator`, and `AsyncBoundary`.

Adoption must preserve i18n, accessibility selectors, semantic status labels,
and the feature-folder architecture. Each adopted asset needs desktop/mobile,
light/dark, loading, empty, error, stale, and keyboard evidence where those
states apply.

## Contract and implementation boundary

`experience/` records intended communication and evidence requirements.
`PRD.md` remains the capability and operational-target authority. `DESIGN.md`
defines the visual language. `docs/concepts/UI-ARCHITECTURE.md` defines source
tree and adoption seams. React implementation must reconcile with all three;
none should be weakened to make validation pass.
