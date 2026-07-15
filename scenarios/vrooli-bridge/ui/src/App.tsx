import { Providers } from "./app/providers";
import { AppRouter } from "./app/routes";
import { AppGate } from "./features/session/AppGate";

/**
 * Top-level app composition. The shell, routing, theme provider, and pages
 * live in `app/`, `layout/`, `pages/`, and `theme/`. This file is intentionally
 * tiny so scenarios that rip out the default routes can do so in one place.
 *
 * The whole console is owner-gated, so `AppGate` renders the sign-in surface
 * until an owner token is present, then hands off to the routed shell.
 */
export default function App() {
  return (
    <Providers>
      <AppGate>
        <AppRouter />
      </AppGate>
    </Providers>
  );
}
