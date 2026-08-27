# Style ownership

`BaseStyles` is the single owner of the published control reset, focus-visible ring, reduced-motion
policy, forced-colors policy, control-size tokens, direct-child icon scale, and visually-hidden
utility. It mounts through `useLibraryStyleSheet` in `document.head`.

An asset may add selectors for its own named state or anatomy. It must not repeat those shared
concerns, emit a `<style>` element from component output, or rely on render order to beat a
consumer class. Asset-specific CSS belongs in a module-level stylesheet mounted with a stable
asset/version key. Consumer overrides remain ordinary classes loaded after the library foundation.
