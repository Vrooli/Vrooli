import { lazy, Suspense } from "react";
import { Providers } from "./app/providers";
import { AppRouter } from "./app/routes";

const RuntimeContractProbe = lazy(() =>
  import("./RuntimeContractProbe.test").then((module) => ({
    default: module.RuntimeContractProbe,
  })),
);

/**
 * Top-level app composition. The shell, routing, theme provider, and pages
 * live in `app/`, `layout/`, `pages/`, and `theme/`. This file is intentionally
 * tiny so scenarios that rip out the default routes can do so in one place.
 */
export default function App() {
  const showRuntimeContractProbe =
    import.meta.env.DEV &&
    new URLSearchParams(window.location.search).get("rcl-contract-probe") ===
      "1";
  return (
    <Providers>
      <AppRouter />
      {showRuntimeContractProbe && (
        <Suspense fallback={null}>
          <RuntimeContractProbe />
        </Suspense>
      )}
    </Providers>
  );
}
