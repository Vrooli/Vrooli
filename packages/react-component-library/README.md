# @vrooli/react-component-library

This governed in-repository package exposes the React Component Library as a
single file dependency. Use the major-scoped subpath as the house style:

```tsx
import { Button } from "@vrooli/react-component-library/Button/2";
```

It resolves to the newest non-deprecated release in that major. Use an exact
subpath such as `Button/2.2.1` only when pinning a reproduction. Exact aliases
remain available for deprecated releases. A historical major with no active
release can retain a compatibility alias until it is explicitly retired. Use the bare `Button` form only inside the library gallery; it follows
the manifest's `latest` pointer and may cross a major boundary.

Run `pnpm sync-exports` after adding, deprecating, or retiring a library
version; CI treats a stale export map as a build failure. Consumers install
this package through a `file:` dependency, which package managers materialize
as a copy. Rebuild/reinstall the governed dependency before diagnosing a
consumer that still resolves an older alias map.

Withdraw an obsolete major alias through the owner command:

```sh
react-component-library components manifest-update react-component-library:SidebarShell --retired-major-aliases 1
```

The manifest's `retiredMajorAliases` removes the named major alias while retaining
exact historical releases. The current major cannot be retired. Omitted metadata
preserves the decision; `--clear-retired-major-aliases` explicitly reverses it.
Regenerate exports and rebuild the package after changing this metadata.
SidebarShell/1 is retired; use SidebarShell/2 for current compositions.
