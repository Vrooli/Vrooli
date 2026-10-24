# Vega presentation capture

This fixture harness captures **the actual, unchanged Vega frontend**, not a
reconstruction of its UI. It supplies authored responses at the browser network
boundary. It does not start a scenario, bind a server, collect host observations,
query live APIs, or write a database. Use it only for explicitly staged UI demos;
keep a visible `Demo data` label in compositions. It proves frontend interactions
and rendering only. Backend claims require independent real-runtime evidence.

The operator authorized fictional/mock presentation data on 2026-09-22 while
protecting Aquila and all remote-control infrastructure. Normal presentation
startup and Baseline Modes traversed optional live dependency reconciliation in
that assessment; Agent Manager was degraded. Reassess current code and runtime
before any later lifecycle action. This alternative preserves real UI without taking that
startup path. It is not a replacement lifecycle or a fake running scenario.

Copy `ui/` to an evidence directory (exclude node_modules, dist, coverage), point
its node_modules at the existing governed installation, and use the installed
Vite build with `--configLoader runner` in that copy. Build output stays in the
copy. Do not install packages or build shared packages. Hash source and output.

Run `capture.mjs <copied-ui/dist> <new-output-dir> <puppeteer-module-path>` using
an already installed Puppeteer and Chrome. Requests for fixture procedures are
answered in memory; local static assets are read from the supplied bundle; every
other request fails closed and is recorded. A fresh browser profile and disabled
service workers prevent persistence or proxy routing into live data. Capture
actions use the product's existing controls, navigation, hover, and scrolling.
Add `record` as the final argument to retain moving captures. The harness uses a
3200×2000 viewport at 200% browser presentation zoom; it leaves the app source
unchanged. Inspect the pixels, not just encoded dimensions: CDP screencasting with
deviceScaleFactor alone can produce padding instead of additional source detail.
The capture records timestamps and encodes at 30 fps; it does not guarantee 30
distinct captured frames each second.

Fonts must already be bundled locally. `dist/capture-font-map.json` maps each
original font request URL to `{ "file": "relative/local/font", "contentType":
"font/woff2" }`. Keep original font licenses. The browser never fetches them.
Bundle `validate.mjs` with the existing governed esbuild installation (resolve it
through Vite); set `nodePaths` to the original UI's node_modules. Run that bundle
on `responses.json` to check the current generated Protobuf response schemas.
Use `node --test world.test.mjs` for consistency and fail-closed regressions.

`world.mjs` authors one coherent fictional workstation. Changes to values must
remain internally consistent and use the current generated proto schemas.
Retain fixture responses, network audit, source/bundle hashes, and raw captures.
Never label these as real host metrics or end-to-end investigation results.
