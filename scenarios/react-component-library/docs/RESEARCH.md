# Research — React Component Library

Notes on substrate decisions that don't rise to a permanent decision in
`docs/internal/DECISIONS.md` but need a written record so future agents
don't relitigate them.

## Preview harness bundling (Phase 4 slice 3, 2026-05-12)

The library has to execute real React inside an iframe so the user
sees a live preview of the component they edited. That means the API
needs a way to turn a TSX source file on disk into something the
browser can load as an ES module.

### Options considered

1. **esbuild Go API on-the-fly** — call `github.com/evanw/esbuild/pkg/api`
   from the API process; transform the TSX file (loader `tsx`, format
   `esm`); serve the result as `/preview/{id}/bundle.js`. React /
   ReactDOM resolved client-side via an importmap pointing at esm.sh.
2. **Vite SSR / dev server proxy** — run a Vite dev server child
   process; proxy `/preview/...` to it. Vite handles bundling.
3. **Pre-bundle all components at scenario boot** — walk the registry,
   produce one bundle per component up-front, serve as static files.

### Decision: option 1 (esbuild Go on-the-fly)

| Axis | Why option 1 wins |
|---|---|
| Runtime topology | Pure Go; no Node child process; matches the rest of the API. |
| Cold-start latency | esbuild ≈10ms per small TSX file — well under the PRD's <1s warm-preview budget. |
| Determinism | One process, one transform; nothing async-cached behind the scenes. |
| Greenfield fit | No bundler config files, no `vite.config.*` to babysit. |
| Editing UX | Save → re-transform on the next iframe load. Cache-buster query forces reload. |
| Failure surface | esbuild surface errors are structured and easy to return as Connect `InvalidArgument`. |

Option 2 was rejected because spawning a Node-side process duplicates
the substrate and pushes us toward two languages of bundler config
for the same problem. Option 3 was rejected because it forces a full
re-bundle on every save and hides the "the file you edited is what's
running" semantics that make live preview believable.

### Resolved contracts

- **Externals**: `react`, `react-dom`, `react-dom/client` are marked
  external in the esbuild call; the harness HTML carries an importmap
  that maps them to `https://esm.sh/react@<pinned>` etc. The pin is
  read from a constant in `internal/preview/`; bumping it is a
  source-edit, not a runtime concern.
- **Working directory**: esbuild's `ResolveDir` is the configured
  component source root (the same root the FSContentStore guards).
  Relative imports inside a component resolve relative to that root.
- **Harness HTML**: served at `GET /preview/{id}/harness.html`. The
  shell renders the component's default export into a `<div id="root">`
  via `ReactDOM.createRoot`. The component bundle is fetched at
  `/preview/{id}/bundle.js`.
- **Cache-busting**: the host iframe wrapper appends a `?v=<sha256>`
  query reflecting the latest content sha; saves cause the sha to
  change which causes the iframe to reload.
- **iframe-bridge child** (req 03 mentions `initIframeBridgeChild()`):
  full bridge wiring (HELLO/READY/INSPECT) is **deferred** to req 06
  (element selection). For now the harness posts a minimal
  `preview-ready` message so the host can detect first-paint without
  pulling the bridge package into the bundle yet. Tracked in
  `docs/internal/PROBLEMS.md`.

### Revisit triggers

- esbuild transform exceeds the warm-preview latency budget on any
  realistic component.
- esm.sh becomes an unacceptable network dependency for offline use —
  if so, vendor React into the API binary as embedded files and serve
  from `/preview/runtime/react@*.js` instead of the importmap CDN.
- iframe-bridge wiring becomes blocking for any P0 feature — promote
  the bundling of `@vrooli/iframe-bridge/child` into the esbuild call
  (it's just another module to bundle in alongside the component).

### Behaviors worth preserving (skimmed from the stashed pre-rewrite tree)

The stashed `/tmp/react-component-library-pre-rewrite-2026-05-12/`
tree shipped a placeholder HTML preview, so there's nothing to lift.
The behavioral lesson is that "iframe with text content" is not a
preview — execution has to be real or the feature has no value.

## Preview runtime range handling (2026-07-08)

Phase 5 diagnosis confirmed the first preview-runtime inconsistency:
the harness bundled component TSX as ESM but then discarded every
declared dependency whose name started with `react`, installing a
fixed React/ReactDOM 18.3.1 import map instead. A component declaring
React 17 therefore transformed successfully, but the iframe executed
against React 18 at runtime.

The fix keeps a supported preview-runtime candidate list and resolves
declared `react` / `react-dom` ranges to the newest satisfying
candidate. When a React runtime range is unresolvable, the harness
keeps the default 18.3.1 mapping and renders an in-iframe import-map
diagnostic rather than silently previewing with an unexplained pin.

The follow-up offline/degraded-path diagnosis found a second blank-pane
cause: top-level static imports from `react-dom/client` and the inlined
component module can fail before harness code reaches its render
`try/catch`. The harness now dynamically imports `react`,
`react-dom/client`, and the data-URL component module inside a single
guarded block. CDN or module-resolution failures therefore write a
visible `preview: render failed` diagnostic into `#preview-error`.
