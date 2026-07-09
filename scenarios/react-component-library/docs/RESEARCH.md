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
   from the API process; bundle the TSX file (loader `tsx`, format
   `esm`); serve the result through the preview harness. React,
   ReactDOM, and declared package dependencies resolve client-side via
   a same-origin importmap.
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

- **Externals**: bare package imports are marked external in the
  esbuild Build call; the harness HTML carries an importmap. React
  runtime imports are same-origin URLs under
  `/preview/runtime/react@<version>/...` and
  `/preview/runtime/react-dom@<version>/...`, served by the API from
  vendored UI workspace packages bundled to ESM on demand. Non-React
  declared dependencies resolve to same-origin
  `/preview/runtime/npm/<package>@<version>/...` URLs when that package
  version is present in the local UI workspace.
- **Relative imports**: esbuild Build runs with `ResolveDir` rooted at
  the component source directory, so `./local` and `../local` imports
  are folded into the component module. Bare package imports stay
  external so each preview gets its own importmap and conflicting
  declared versions do not share a process-global resolution.
- **Harness HTML**: served at `GET /preview/{id}/harness.html`. The
  shell renders the component's default export into a `<div id="root">`
  via `ReactDOM.createRoot`. The transformed component module is
  embedded as a data URL in the harness so the iframe needs no second
  component fetch.
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
- A component needs a package version that is not installed in the UI
  workspace. The current offline contract surfaces an import-map
  diagnostic instead of fetching a CDN fallback; adding more vendored
  versions should happen through the scenario dependency-governance
  flow, not ad hoc package-manager commands.
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

Phase 3 of the renderer redesign removed the CDN dependency for the
React runtime itself. The importmap now points React and ReactDOM to
same-origin `/preview/runtime/...` URLs. The API serves those routes by
bundling the installed UI workspace React package entrypoints into ESM
with esbuild. Only React 18.3.1 is currently vendored; declared ranges
that resolve to another supported candidate produce a diagnostic and
fall back to the vendored default rather than emitting a dead URL.

The harness also inlines the scenario UI's compiled stylesheet when
available, with a token/utility fallback for test and minimal dev
environments. This gives previewed components access to the
`--color-*` design tokens and `bg-app-*` / `text-app-*` Tailwind
utilities they use in source.

Phase 4 completed the module-resolution side of the redesign. The
preview bundler now uses esbuild Build rather than Transform, deriving
`ResolveDir` from the indexed source path so relative imports are
bundled into the component module. Bare package imports remain
external and are resolved by the harness importmap. Non-React
dependencies use `deps.ResolveRangeToLatest` against locally installed
UI workspace package versions and map to same-origin
`/preview/runtime/npm/<package>@<version>/...` runtime URLs. If a
declared dependency range cannot be resolved locally, the harness
renders an import-map diagnostic instead of emitting an `esm.sh` URL or
silently dropping the dependency.

The Phase 5 live-browser regression had a different root cause from
bundling: the host iframe correctly requested `/preview/{id}/harness.html`,
but the production UI server only proxied Connect-RPC paths. Because
`/preview/...` was not proxied before the SPA fallback, `ui/server.js`
served the React Component Library `index.html` into the iframe, making
the preview recursively render the library UI. The UI server now proxies
both `/preview/{id}/harness.html` and `/preview/runtime/...` paths to the
Go API unchanged, before static assets and SPA fallback run.

The follow-up browser run exposed the actual runtime error that the
route-level tests still missed: bundling React CommonJS entrypoints with
esbuild produced default-only ESM, while compiled JSX imports named
exports from `react/jsx-runtime`. ReactDOM's client bundle also cannot
leave `react` as a dynamic CommonJS require in a browser ESM module. The
runtime handler now wraps the vendored React and ReactDOM entrypoints
with explicit ESM named exports (`jsx`, `jsxs`, `createRoot`, hooks,
etc.) and bundles ReactDOM with React included. `pnpm run
test:preview-e2e` drives Chrome through the real UI, clicks Preview for
the StatusBadge component, asserts the iframe URL is `/preview/...`,
checks the rendered DOM, and fails on the browser errors that caused the
visible preview failure. The BAS playbook
`preview-renders-component` covers the same user-visible contract at the
test-genie workflow layer by asserting the host reaches the Rendered
badge after switching the known component editor into Preview mode.
