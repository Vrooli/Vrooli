---
title: "Coherence Notes"
description: "Where the codebase + docs are intentionally out of step, and why"
category: "internal"
order: 101
audience: ["developers"]
internal: true
---

# Coherence Notes

This file lists places where the **physical structure** and the **documented mental model** are intentionally not aligned, with the reason and the planned resolution. Each entry should either be closed (deleted) or marked resolved when the gap is fixed.

## API: domain boundaries are file-level, not package-level

- **Documented model:** `docs/concepts/ARCHITECTURE.md` lists ~8 logical domains (`landing`, `billing`, `downloads`, `ai`, `metrics`, `admin`, `remote-profile`, `user-auth`) and a per-domain `register*Routes` composer.
- **Actual code:** All API source lives in `package main` under `api/`. Domain boundaries are enforced by *file naming* + the `register*Routes` grouping in `api/routes.go`, not by Go package boundaries.
- **Why:** Extracting `api/domain/<name>/` subpackages requires (a) moving the per-domain test files (most >500 LOC, currently relying on unexported package-internal helpers), (b) exporting cross-cutting helpers (`logStructured`, `requireAdmin`, `helpers.go`, `httphelpers.go`, `dbhelpers.go`, session middleware), and (c) deciding whether to introduce an `api/shared/` (or similar) package to break the resulting cycles. A tactical mid-execution split risks circular imports and broken test coverage.
- **Resolution:** Tracked as backlog item `qa-deep-landing-page-business-suite-api-domain-subpackages-20260424` (`execute`, `acceptance_allow = scenarios/landing-page-business-suite/**`). That item's plan is the canonical place to design any new shared package and to sequence per-domain moves.

## UI: surface barrels exist for `user-auth` only

- **Documented model:** Each UI surface (`public-landing`, `admin-portal`, `user-auth`) is independently importable.
- **Actual code:** `surfaces/user-auth/index.ts` exports a barrel; `public-landing` and `admin-portal` do not. `App.tsx` deep-imports their route components by file path.
- **Why:** No business reason — historical accretion. Barrels for the other two surfaces would tighten boundaries but require touching every route module.
- **Resolution:** Open. Low-priority cleanup, not a correctness issue.

## Docs: `docs/reference/api/README.md` is an in-section overview

- **Documented model:** Manifest registers it as `path: reference/api/README.md`.
- **Actual code:** The file is `README.md` literally — by convention some doc tooling treats `README.md` as the index of its parent. Some upstream docs-health checks flag this as misplaced.
- **Resolution:** Open. The file is correct in place; rename to `reference/api/index.md` if/when docs tooling enforces a strict naming convention.

---

When you fix one of the entries above, **delete it** rather than marking "done." This file should reflect *current* drift, not a changelog.
