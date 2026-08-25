# @vrooli/react-component-library

This governed in-repository package exposes the React Component Library as a
single file dependency. Every published library version has a stable subpath:

```tsx
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
```

The unversioned component subpath resolves to the newest live version. Run
`pnpm sync-exports` after adding or retiring a library version; CI treats a
stale export map as a build failure.
