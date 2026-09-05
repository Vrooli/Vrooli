# @vrooli/react-component-library

This governed in-repository package exposes the React Component Library as a
single file dependency. Use the major-scoped subpath as the house style:

```tsx
import { Button } from "@vrooli/react-component-library/Button/2";
```

It resolves to the newest non-deprecated release in that major. Use an exact
subpath such as `Button/2.2.1` only when pinning a reproduction. Exact aliases
remain available for deprecated releases, while floating aliases never select
them. Use the bare `Button` form only inside the library gallery; it follows
the manifest's `latest` pointer and may cross a major boundary.

Run `pnpm sync-exports` after adding, deprecating, or retiring a library
version; CI treats a stale export map as a build failure. Consumers install
this package through a `file:` dependency, which package managers materialize
as a copy. Rebuild/reinstall the governed dependency before diagnosing a
consumer that still resolves an older alias map.
