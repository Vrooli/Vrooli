# Style ownership

`BaseStyles` is the single owner of the published control reset, focus-visible ring, reduced-motion
policy, forced-colors policy, control-size tokens, direct-child icon scale, and visually-hidden
utility. It mounts through `useLibraryStyleSheet` in `document.head`.

An asset may add selectors for its own named state or anatomy. It must not repeat those shared
concerns, emit a `<style>` element from component output, or rely on render order to beat a
consumer class. Asset-specific CSS belongs in a module-level stylesheet mounted with a stable
asset/version key. Consumer overrides remain ordinary classes loaded after the library foundation.

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
