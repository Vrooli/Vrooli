import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import type { ReactNode } from "react";
import { ROUTE_PATTERNS } from "./routes.generated";
import { AppShell } from "./shared/components/AppShell";
import { ErrorBoundary } from "./shared/ui/composites/ErrorBoundary";
import { GoldensIndex } from "./surfaces/goldens/GoldensIndex";
import { GoldenDetail } from "./surfaces/goldens/GoldenDetail";
import { SkillsPlaceholder } from "./surfaces/placeholders/SkillsPlaceholder";
import { ManifestsPlaceholder } from "./surfaces/placeholders/ManifestsPlaceholder";
import { Settings } from "./surfaces/settings/Settings";

/**
 * Root application component. Routes are sourced from
 * `ui/flow/navigation.json` via `routes.generated.ts`. No raw path
 * strings live here — every pattern flows through `ROUTE_PATTERNS.*`.
 *
 * The shell wraps every surface in its own `<ErrorBoundary>` (inside
 * AppShell's `<Outlet />`), so a crash in one surface keeps the rest of
 * the app responsive. The outermost boundary in `main.tsx` is still the
 * global safety net.
 *
 * Basename note: served via app-monitor's proxy at /apps/<name>/proxy/.
 * Using "." (relative) keeps local dev, proxy, and tunnel contexts all
 * resolving correctly.
 */
/**
 * Routes-only component. Exposed for tests to mount inside a
 * MemoryRouter; production uses `<App />` which adds the BrowserRouter
 * and outer ErrorBoundary.
 */
export function AppRoutes(): ReactNode {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path={ROUTE_PATTERNS.goldensIndex} element={<GoldensIndex />} />
        <Route path={ROUTE_PATTERNS.goldenDetail} element={<GoldenDetail />} />
        <Route path={ROUTE_PATTERNS.skillsIndex} element={<SkillsPlaceholder />} />
        <Route path={ROUTE_PATTERNS.manifestsIndex} element={<ManifestsPlaceholder />} />
        <Route path={ROUTE_PATTERNS.settings} element={<Settings />} />
        <Route path="*" element={<Navigate to={ROUTE_PATTERNS.goldensIndex} replace />} />
      </Route>
    </Routes>
  );
}

export default function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter basename=".">
        <AppRoutes />
      </BrowserRouter>
    </ErrorBoundary>
  );
}
