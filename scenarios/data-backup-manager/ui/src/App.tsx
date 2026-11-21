import { BaseStyles } from "@vrooli/react-component-library/BaseStyles";

import { Providers } from "./app/providers";
import { AppRouter } from "./app/routes";

/**
 * Top-level app composition. The shell, routing, theme provider, and pages
 * live in `app/`, `layout/`, `pages/`, and `theme/`. This file is intentionally
 * tiny so scenarios that rip out the default routes can do so in one place.
 *
 * `BaseStyles` publishes the component library's canonical design tokens
 * (`--space-*`, `--control-height`, the type ramp, focus ring, reduced-motion
 * policy). The scenario design kit only defines colors and radii; without this
 * mount every library component renders with undefined spacing and collapses.
 * It renders no DOM node and is idempotent, so it belongs at the app root.
 */
export default function App() {
  return (
    <>
      <BaseStyles />
      <Providers>
        <AppRouter />
      </Providers>
    </>
  );
}
