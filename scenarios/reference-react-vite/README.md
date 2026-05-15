# Reference React Vite

Golden reference scenario demonstrating a fully-developed React+Vite application that adheres to all applicable steer skills.

## Purpose

This is **NOT** a product. It exists solely as a ground-truth test bed for:

1. **development-toolchain-validator**: Maps steer skill expectations against this reference to detect cross-steer conflicts and validate tooling correctness.
2. **prompt-manager meta optimization team**: Uses this as the "known-good" target to validate skill improvements.
3. **AI agents**: Concrete example of what steer skill guidance produces when fully applied.
4. **Human developers**: Living documentation of Vrooli best practices for react-vite scenarios.

## Design Principles

- **Simple domain, exemplary architecture**: The business logic is trivially simple (task/project management). The value is in the architectural patterns.
- **Every pattern exercised**: If a steer skill describes a pattern, this reference includes a concrete example.
- **Fully exercisable by tooling**: Every lifecycle command, test phase, and auditor rule works against this scenario.
- **Maintenance is a feature**: Updated as steer skills evolve. development-toolchain-validator detects drift.

## Quality Targets

- `scenario-auditor audit reference-react-vite` → Zero violations
- `test-genie execute reference-react-vite --preset comprehensive` → All 11 phases pass
- `scenario-completeness-scoring score reference-react-vite` → Score 96+ (Production Ready)

## Applicable Steer Skills

**Core**: api-steer, storage-steer, cli-steer, interoperability-steer, unit-testing-architecture-steer

**Quality**: documentation-health, screaming-architecture-audit, react-coherence, react-stability, code-cleanup, refactor, domain-compression, cognitive-load-reduction

**Testing**: test, e2e-testing, performance, security

**UX**: ux, experience-architecture-audit, navigation-integrity-audit, polish

**Specialized**: error-semantics-recovery-path-design, failure-topography-and-graceful-degradation, boundary-of-responsibility-enforcement, seam-discovery-and-enforcement, invariant-discovery-and-enforcement

## Stack

- **API**: Go with standard library HTTP server, gorilla/mux
- **UI**: React + TypeScript + Vite, Tailwind CSS
- **CLI**: Go with cli-core
- **Storage**: PostgreSQL (direct SQL, no ORM)

## Setup

```bash
cd scenarios/reference-react-vite && make start
```

## Navigation contract

`ui/flow/navigation.json` is the single source of truth for this template's URL
surface, layout containers (top nav bar, bottom nav, hamburger drawer, user-menu
popover, auth footer), affordances, and reachability invariants. The spec is
validated, verified, and reconciled with the React code by flow-verifier; the
URL constants in `ui/src/routes.generated.ts` are emitted from the same spec.

```bash
make navigation        # validate + verify + reconcile + emit routes.generated.ts
```

### Template example surfaces — review on generation

When generating a new scenario from this template, replace the following
illustrative surfaces with real product content (or remove them outright):

- **Auth context (`src/contexts/AppContext.tsx`)** — client-only `auth` /
  `role` / `feature_beta` toggles for demoing the gating predicates. Swap for a
  real auth provider before shipping.
- **Login + Forgot-password pages** — placeholder forms, no credentials wired.
- **`/admin/users`, `/beta`** — illustrative role / feature-flag gated routes.
- **Settings sub-pages** (`/settings/{display,notifications,about}`) — stub
  copy only; replace with real preferences UI.
- **Responsive containers** in `src/components/Layout.tsx` (TopNavBar,
  BottomNav, HamburgerMenu, UserMenu) — kept generic; swap branded markup in.

## Related

- [development-toolchain-validator](../development-toolchain-validator/README.md) — Validates this reference against steer skills
- [prompt-manager](../prompt-manager/README.md) — Source of steer skills this reference implements
- [PRD](PRD.md) — Full product requirements

## Progress & Issues

- [Progress Log](docs/PROGRESS.md)
