# Experience readiness contracts

`scenario-experience-spec/v1` is the authored source for meaningful UI
readiness. A readiness profile is a deterministic projection of that spec; no
consumer owns a second hand-authored readiness registry.

## Authoring a region

Pages may declare `regions`. A region is an independently rendered or
asynchronously resolved surface such as a result list, inspector, preview, or
test-history panel. Do not declare passive buttons, labels, or layout wrappers:
they inherit their parent surface lifecycle.

Each region has a stable semantic id, a purpose, a local or exact pinned library
component reference, lifecycle contract, and a volatile runtime binding in
`bindings.regions`.

```json
{
  "id": "results",
  "purpose": "Show query results once the data is available.",
  "component": {"local": "result-table"},
  "lifecycle": {"kind": "async", "states": ["loading", "ready", "empty", "error"]}
}
```

`required` defaults to `true`. Required unsettled surfaces block readiness;
optional failures can yield `partial` while the primary task remains usable.
Static surfaces declare `{"kind":"static","states":["static"]}`.

## Lifecycle vocabulary

Use only `loading`, `ready`, `empty`, `partial`, `error`, and `static` for a
region lifecycle. Existing page state identifiers remain page-level UX states;
they are not duplicated runtime lifecycle metadata.

At runtime, the RCL `ExperienceSurface` primitive exposes the contract through
`data-experience-surface=<region-id>` and
`data-experience-state=<lifecycle-state>`. It provides semantic hooks and
accessible state announcements, not layout.

## Ownership and consumption

- Experience Manager parses and validates authored contracts, then compiles the
  versioned readiness profile.
- React Component Library stores reusable component contracts next to the exact
  component version. A scenario refers to a canonical component with an exact
  version pin, or declares a local component explicitly.
- A local component may instead be an explicit wrapper using
  `component.extends: {"component":"…","version":"…"}` and a
  `component.extension.purpose`. Wrappers may add new behavior, but Experience
  Manager rejects any canonical state or claim identifier they attempt to
  redefine.
- Workflow Health maps workflow coverage to declared routes, region bindings,
  and lifecycle states; selectors remain implementation hooks rather than a
  semantic registry.
- UI Health treats the iframe bridge handshake as shell boot only. It records
  instrumented surfaces separately when checking functional lifecycle, for
  every route declared by the compiled profile.
- Browser Automation Studio uses an explicit caller wait first. Declared
  readiness is used only when a resolved profile is available; external and
  undeclared URLs retain generic navigation behavior.

## Troubleshooting

- An unbound region means `bindings.regions` lacks a `testid` or `selector` for
  the region id.
- An unresolved local component must be added to the same scenario's
  `experience/index.json` component registry.
- A pinned library reference needs a kebab-case component id and exact semver.
- A wrapper pin needs the canonical `experience-contract.json` at that exact
  RCL version and must not reuse its state or claim ids.
- A static region may only declare the `static` lifecycle state; async regions
  may not declare `static`.

## Presentation assets

`ExperienceSurface` is the semantic boundary. `AsyncPanel` is the optional
RCL presentation asset for loading, ready, empty, partial, and error states;
it preserves the surface's attributes and live announcements while leaving
cards, grids, and page shells to the consuming design. Use it for a meaningful
async region rather than adding ad-hoc lifecycle data attributes to children.

## Enforcement and interpretation

For `schemaVersion` 1.1.0 and later, a page that declares loading or error
states must declare a required async region and bind it at runtime. This makes
the primary asynchronous work observable instead of leaving automation to infer
readiness from a shell selector. Existing 1.0.0 documents remain descriptive
until deliberately migrated.

`loading` means mounted but unsettled. It is never a functional-ready result.
For async regions, `ready`, `empty`, `partial`, and `error` are terminal
states: BAS may stop waiting at any of them but reports the observed outcome;
required `error` remains a failed health result. UI Health waits for a declared
terminal state before collecting runtime evidence.
