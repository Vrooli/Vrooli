import { Suspense, lazy, type ReactNode } from "react";
import { Route, Routes } from "react-router-dom";

import { ApplicationShell as AppShell } from "./components/ui/ApplicationShell";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { useHostShortcutRelay } from "./hooks/useHostShortcutRelay";
import { useTranslation } from "./i18n";
import { CatalogBrowser } from "./features/catalog/CatalogBrowser";
import { appRoutes } from "./routes";

const ComponentDetailPage = lazy(() =>
  import("./pages/ComponentDetailPage").then((m) => ({ default: m.ComponentDetailPage })),
);
const SettingsPage = lazy(() =>
  import("./pages/SettingsPage").then((m) => ({ default: m.SettingsPage })),
);
const CoveragePage = lazy(() =>
  import("./pages/CoveragePage").then((m) => ({ default: m.CoveragePage })),
);
const CapabilitiesPage = lazy(() =>
  import("./pages/CapabilitiesPage").then((m) => ({ default: m.CapabilitiesPage })),
);
const NotFoundPage = lazy(() =>
  import("./pages/NotFoundPage").then((m) => ({ default: m.NotFoundPage })),
);
const PreviewPopoutPage = lazy(() =>
  import("./pages/PreviewPopoutPage").then((m) => ({ default: m.PreviewPopoutPage })),
);

function RouteSkeleton() {
  const { t } = useTranslation();
  return (
    <div
      data-testid="route-skeleton"
      role="status"
      aria-live="polite"
      className="flex items-center gap-space-2xs text-sm text-app-muted-foreground"
    >
      <span className="inline-block h-3 w-3 animate-pulse rounded-pill bg-app-surface-muted" />
      {t("route.loading", { defaultValue: "Loading…" })}
    </div>
  );
}

function Page({ children }: { children: ReactNode }) {
  return (
    <ErrorBoundary>
      <Suspense fallback={<RouteSkeleton />}>{children}</Suspense>
    </ErrorBoundary>
  );
}

export default function App() {
  useHostShortcutRelay();

  return (
    <Routes>
      <Route
        path={appRoutes.preview}
        element={
          <Page>
            <PreviewPopoutPage />
          </Page>
        }
      />
      <Route element={<AppShell />}>
        <Route
          path={appRoutes.catalog}
          element={
            <Page>
              <CatalogBrowser surfaceId="catalog-results" />
            </Page>
          }
        />
        <Route
          path={appRoutes.assetCatalog}
          element={
            <Page>
              <CatalogBrowser surfaceId="catalog-results" />
            </Page>
          }
        />
        <Route
          path={appRoutes.asset}
          element={
            <Page>
              <ComponentDetailPage />
            </Page>
          }
        />
        <Route
          path={appRoutes.coverage}
          element={
            <Page>
              <CoveragePage />
            </Page>
          }
        />
        <Route
          path={appRoutes.capabilities}
          element={
            <Page>
              <CapabilitiesPage />
            </Page>
          }
        />
        <Route
          path={appRoutes.settings}
          element={
            <Page>
              <SettingsPage />
            </Page>
          }
        />
        <Route
          path="*"
          element={
            <Page>
              <NotFoundPage />
            </Page>
          }
        />
      </Route>
    </Routes>
  );
}
