import { Providers } from "./app/providers";
import { AppRouter } from "./app/routes";

/**
 * Top-level app composition. The shell, routing, theme provider, and pages
 * live in `app/`, `layout/`, `pages/`, and `theme/`. This file is intentionally
 * tiny so scenarios that rip out the default routes can do so in one place.
 */
export default function App() {
  return (
    <Providers>
      <AppRouter />
    </Providers>
  );
}
