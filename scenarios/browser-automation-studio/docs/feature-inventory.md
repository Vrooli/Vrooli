# Served feature inventory

BAS's `playwright-driver/scripts/feature-inventory.mjs` builds a retained baseline from a real, served scenario. It uses the driver's pinned `rebrowser-playwright`, crawls same-origin links, and records each visited route, visible action, menu item, semantic region and React component instance with a screenshot crop. React fiber/type/source metadata is used when exposed by the served runtime; otherwise component identity falls back to the stable semantic/test-id identity. Repeated React instances get separate route-local occurrence identities and crops. The inventory records React event prop **names** visible on components and action ancestors; it never reads or invokes handler values. This can reveal declared gestures such as pointer, drag or keyboard handlers, but it cannot establish native listeners, custom event wiring or whether a handler works. Those remain unobserved. Feature identities are based on route and served identity, so adding an unrelated control does not renumber semantic identities.

`identity.build` is computed from the SHA-256 of the first served document body and records its route, state and byte count in `identity.build_source`. This identifies the served app-shell document; it does not hash each loaded JavaScript asset or prove a source revision. The `--build` and `--revision` CLI values are retained as `build_hint` and `source_revision_hint` only. `identity.source_revision` stays null unless a separate trusted source-to-serving proof is available. Never treat a caller-supplied label as attestation.

Run this only against an owner-started scenario, using a local or otherwise authorized URL:

```sh
cd scenarios/browser-automation-studio/playwright-driver
node scripts/feature-inventory.mjs crawl http://localhost:PORT/ /tmp/bas-before \
  --scenario SCENARIO --build BUILD_ID --revision GIT_REVISION
node scripts/feature-inventory.mjs crawl http://localhost:PORT/ /tmp/bas-after \
  --scenario SCENARIO --build BUILD_ID --revision GIT_REVISION
node scripts/feature-inventory.mjs merge /tmp/bas-complete \
  /tmp/bas-before/inventory.json /tmp/bas-settings/inventory.json /tmp/bas-deep-links/inventory.json
node scripts/feature-inventory.mjs diff /tmp/bas-before/inventory.json \
  /tmp/bas-after/inventory.json /tmp/bas-diff.json
```

When a full baseline requires multiple targeted crawls of the same served build, `merge`
unions them into one diff-ready inventory. Every input must match on scenario,
measured document build, base URL, viewport, and synthetic profile. Caller-supplied
build/revision hints do not establish identity. Duplicate
feature identities are deduplicated; conflicting identities fail. Screenshot
variants that differ are retained beside the primary crop, and the merged
coverage keeps each source capture's route states and observed action paths.
Merge captures of one unchanged build; it does not create a before/after result.

Synthetic action paths may be scoped to a named entry state:

```sh
node scripts/feature-inventory.mjs crawl http://localhost:PORT/ /tmp/bas-states \
  --scenario SCENARIO --build BUILD_ID --revision GIT_REVISION \
  --synthetic-profile PROFILE \
  --entrypoint approval='/?approval=synthetic-decision' \
  --safe-action-paths 'default::Settings > Account | approval::Pending'
```

Only exact visible accessible names are clicked. Supported roles include
buttons, radios, tabs, links, and menu items; callers must explicitly allowlist
only local UI transitions after reviewing their effects. Clearing the synthetic
browser profile happens before visiting a query-bearing entrypoint, because an
application may consume and strip its query parameters during boot.

Diff requires both files to use the BAS inventory schema, carry measured served-document build identities, identify the same scenario, and identify distinct builds. It rejects duplicate feature IDs and duplicate kind/route/identity tuples before comparing inventories. Its `totals` report overall before, after, added and removed counts, plus those same measured counts by feature kind. It emits one stable-ID `research-feature-decision/v1` card for every disappeared row. Each card embeds the before crop and the after route image as PNG data, names the screen and taps, explains both keep/drop effects, and gives a keep recommendation. Before writing a successful diff, BAS checks every generated card itself (a title, a screen, evidence with alt text and an image, and a recommendation that is one of the options); an unusable card makes the diff fail. File paths are not accepted by the phone-ready card contract. The inventory and evidence directory are durable caller-owned artifacts; no BAS database or external storage is mutated.

Every inventory includes `coverage`: candidate and observed counts for menu, disclosure and tab states; semantic and React component counts; how many React component rows exposed event prop names; explicit activation or re-identification failures; nested interactions found but not explored; and unattempted drag, hover, keyboard and nested-scroll gestures. Same-origin links exposed in recognized opened states are added to the route crawl. Only declared presentation controls are clicked. Ordinary buttons, links, menu choices and submit controls are never activated. React component visibility is runtime-dependent: production runtimes may hide fiber metadata, and source locations may be omitted. Repeated components use their React key where available; unkeyed repeats use a route-local order number, which may shift when a preceding matching component is added. Missing rows mean "not observed by this crawl," never proof of deletion or full completeness. Screenshot crops document each observed instance; counts and evidence coverage are honest limits, not a completeness claim. Review the coverage object before treating an inventory as complete. The route limit defaults to 50 and is capped at 200.

The scan excludes zero-area, CSS-hidden, fully transparent, `aria-hidden`, or inert nodes from the current action/component view. Element screenshots are rejected when their visible pixels are uniform; BAS retries the live bounding box and then the clipped viewport. The pixel check catches empty textures but does not judge composition, contrast, readability or taste.
