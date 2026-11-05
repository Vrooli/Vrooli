# Testing — Backdrop Studio

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## The integration lane — and why unit tests are not enough here

**Run it:**

```bash
cd scenarios/backdrop-studio
make integration          # every seeded style through a really running image-tools
make integration-evidence # capture into managed test_runs storage; review candidate corpus before updating testdata
```

**What the lane asserts, beyond "it rendered":**

| Assertion | Guards against |
|---|---|
| Every seeded style renders, bound and unbound, at two geometries | The twelve-broken-styles failure below |
| Seam conformance against the real executor | A fake that has drifted from the wire |
| Every treatment changes its input | A treatment silently doing nothing — this caught `pixel_sort` and `bloom` returning their input untouched on a flat source |
| The known-bad case still fails the perceptual gate | A loosened threshold, or a reverted mark-width fix, quietly re-admitting the original defect |
| The repaired style still clears its bar | A gate that rejects everything and therefore looks healthy |
| Every metric within 0.05 of the recorded corpus | A style drifting toward its floor between releases. Renders are deterministic, so movement is a change in the code or the catalog, never noise |

**Read this before trusting a green unit suite.** On 2026-08-12 twelve of
sixteen seeded styles were unrenderable while every Go unit test passed. The
cause was structural, not careless:

- `backdrop-studio` tests against a fake executor that never reaches
  image-tools' REST edge.
- `image-tools` tests its treatments below the wire.
- Neither side could see the boundary between them.

The one test that did cross the boundary resolved parameters **against a bound
brand** — the single path a CLI caller never takes — so it watched ten styles
ship broken with a literal `$brand.primary` on the wire.

The lane exists to make that class of defect impossible to miss. It has already
caught three things no unit test could:

| Found by the lane | What it was |
|---|---|
| Candidate bytes were JPEG | A field named `image_png` carried JPEG, because a model backend answers in whatever format it likes. |
| Recorded geometry was fiction | A candidate reported 1600x1000 for a 2048x2048 image. |
| Every model render was billable | Hardcoded `allowByok` routed to a paid cloud provider while an installed local GPU sat idle. |

### What it needs

| Requirement | Why |
|---|---|
| `backdrop-studio` running and built from the working tree | The lane compares `/api/v1/build` against a fingerprint it computes from `api/`, and refuses to render on a mismatch. Two audits in two days drew false conclusions from a stale binary. |
| `image-tools` running | Unreachable is reported as a named dependency failure, never as "no styles render". |
| `asset-studio` running | Only for the release assertions. |
| An installed image model | Optional. Model-backed styles skip with `SKIP(no-image-model)` when none is available. |

Set `BACKDROP_STUDIO_LANE_BRAND=<brand-id>` to run the bound pass against a real
Brand Manager brand; without it the bound pass binds tokens directly, which
exercises the same code path.

### Why it is not a Test Genie phase

Test Genie's phase registry is provider-backed and closed — there is no
`integration` phase for a scenario to register a Go lane against, and
`playbooks` is a deprecated alias for the BAS workflow phase. So the lane is a
first-class `make` target documented here and in
[`EVIDENCE.md`](EVIDENCE.md) rather than a registered phase. Run it before
closing any change that touches a contract.

## Cross-references

- **Seams definition + adding new seams**: [`SEAMS.md`](SEAMS.md).
- **Skill bundle for testing-related work** (load before substantial test changes):
  ```bash
    Load the `seam-discovery-and-enforcement` skill through prompt-manager before
    making substantial test changes.
  ```
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
