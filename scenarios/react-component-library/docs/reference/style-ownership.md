# Style ownership

`BaseStyles` is the single owner of the published control reset, focus-visible ring, reduced-motion
policy, forced-colors policy, control-size tokens, direct-child icon scale, and visually-hidden
utility. It mounts through `useLibraryStyleSheet` in `document.head`.

An asset may add selectors for its own named state or anatomy. It must not repeat those shared
concerns, emit a `<style>` element from component output, or rely on render order to beat a
consumer class. Asset-specific CSS belongs in a module-level stylesheet mounted with a stable
asset/version key. The key is derived by the `StyleSheet` foundation as
`<normalised libraryId>-<exact version>`; for example,
`react-component-library:MessageList@1.1.1` becomes
`react-component-library-messagelist-1.1.1`. This keeps two reachable versions of one asset from
sharing a page-global `<style data-rcl-sheet>` node. Consumer overrides remain ordinary classes
loaded after the library foundation.

`StyleSheet.name` remains accepted for one compatibility release and logs a development warning.
New sources must pass `libraryId` and `version`; the `catalog.stylesheet-key` and
`catalog.stylesheet-key-duplicate` gates enforce the migration. Remove `name` in the first
StyleSheet foundation major release after this correction is adopted by out-of-tree consumers.

## Strict token diagnosis

`ui-health capture --strict-tokens` runs a same-session browser evaluation before the computed
DOM snapshot. It sets every CSS custom property declared by the loaded style sheets to the
sentinel `#ff00ff`. The capture envelope records the number of elements that rendered a sentinel
colour and samples any element that still rendered a non-sentinel colour. A non-sentinel sample is
an explicit token-boundary gap, not a visual pass.

Literal `var(--token, value)` fallbacks are blocking by default. The
`catalog.token-fallback-literal` gate reads the named `x-token-fallback-exemptions` entries in
`catalog/config.json`; each exemption must state why the property is a host/runtime contract.
Unregistered design properties are not exemptions and therefore fail visibly when their
declaration is absent.

## Utility-class prohibition

A published component emits no library-owned utility class. Its runtime appearance comes from its
module stylesheet and semantic custom properties, so a consumer does not need Tailwind—or the
library's Tailwind theme—to render it correctly. This includes palette, layout, spacing, sizing,
typography, state/viewport variants, arbitrary values, and custom utilities such as
`touch-target`.

Class-bearing props are pass-through seams, not an exception for library defaults. `className`,
`panelClassName`, `contentClassName`, and `backdropClassName` may carry a value supplied by the
consumer; the component must not concatenate its own utility strings onto that value. The
`SidebarShell/2.0.0` implementation is the reference shape: library geometry is stylesheet-owned,
while consumer classes win through the public prop.

The `catalog.utility-class` gate enforces this boundary across shipped runtime source. Its dated,
shrink-only allowlist records migration debt and cannot grow. Ingest applies the same detector and
refuses scenario-local source that would recreate the portability defect.

## Multiple style fragments and migration safety

One `libraryId`/version key owns one complete CSS string. Combine base styles and
component rules before calling `useLibraryStyleSheet`; registering two different
strings under the same key retains only the first string and reports a collision.
`Tabs/1.3.1` demonstrates a combined stylesheet while retaining compact density.

The stylesheet-key codemod only visits an asset's active governed draft. It never
rewrites released versions. Multiple injection sites require manual review and
combination before the codemod can assign an owner/version key. Begin drafts
through `components draft-begin`; do not update release attestations to approve
unreviewed source changes.
