import { Suspense, lazy, type ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";

import { ErrorBoundary } from "./components/ErrorBoundary";
import { useHostShortcutRelay } from "./hooks/useHostShortcutRelay";
import { useTranslation } from "./i18n";
import { WorkbenchShell } from "./workbench/WorkbenchShell";
import { appRoutes } from "./routes";

const HomePage = lazy(() =>
  import("./workbench/pages/HomePage").then((m) => ({ default: m.HomePage })),
);
const ComponentsPage = lazy(() =>
  import("./workbench/pages/ComponentsPage").then((m) => ({ default: m.ComponentsPage })),
);
const ComponentPage = lazy(() =>
  import("./workbench/pages/ComponentPage").then((m) => ({ default: m.ComponentPage })),
);
const DesignStudioPage = lazy(() =>
  import("./workbench/pages/DesignStudioPage").then((m) => ({ default: m.DesignStudioPage })),
);
const ReviewPage = lazy(() =>
  import("./workbench/pages/ReviewPage").then((m) => ({ default: m.ReviewPage })),
);
const CoveragePage = lazy(() =>
  import("./workbench/pages/CoveragePage").then((m) => ({ default: m.CoveragePage })),
);
const CapabilitiesPage = lazy(() =>
  import("./workbench/pages/CapabilitiesPage").then((m) => ({ default: m.CapabilitiesPage })),
);
const ActivityPage = lazy(() =>
  import("./workbench/pages/ActivityPage").then((m) => ({ default: m.ActivityPage })),
);
const SettingsPage = lazy(() =>
  import("./workbench/pages/SettingsPage").then((m) => ({ default: m.SettingsPage })),
);
const CreatePage = lazy(() =>
  import("./workbench/pages/CreatePage").then((m) => ({ default: m.CreatePage })),
);
const PreviewPage = lazy(() =>
  import("./workbench/pages/PreviewPage").then((m) => ({ default: m.PreviewPage })),
);
const NotFoundPage = lazy(() =>
  import("./workbench/pages/NotFoundPage").then((m) => ({ default: m.NotFoundPage })),
);

function RouteSkeleton() {
  const { t } = useTranslation();
  return (
    <div
      data-testid="route-skeleton"
      role="status"
      aria-live="polite"
      className="wb-page"
      style={{ color: "var(--wb-muted)" }}
    >
      <span className="wb-skeleton" style={{ display: "block", width: 180, height: 14 }} />
      <span style={{ display: "block", marginTop: 12 }}>{t("route.loading", { defaultValue: "Loading…" })}</span>
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

/** `/catalog` was the asset catalog. Preserve its query string on the new route. */
function CatalogRedirect() {
  const location = useLocation();
  return <Navigate to={`${appRoutes.components}${location.search}`} replace />;
}

/** The three design routes collapse into one studio; keep scenario/page context. */
function SketchRedirect() {
  const location = useLocation();
  const parts = location.pathname.split("/").filter(Boolean);
  const scenario = parts[1];
  const page = parts[2];
  const search = new URLSearchParams(location.search);
  if (scenario) search.set("scenario", scenario);
  if (page) search.set("page", page);
  return <Navigate to={`${appRoutes.design}?${search.toString()}`} replace />;
}

export default function App() {
  useHostShortcutRelay();

  return (
    <Routes>
      <Route
        path={appRoutes.preview}
        element={
          <Page>
            <PreviewPage />
          </Page>
        }
      />
      <Route element={<WorkbenchShell />}>
        <Route
          path={appRoutes.home}
          element={
            <Page>
              <HomePage />
            </Page>
          }
        />
        <Route path={appRoutes.assetCatalog} element={<CatalogRedirect />} />
        <Route
          path={appRoutes.components}
          element={
            <Page>
              <ComponentsPage />
            </Page>
          }
        />
        <Route
          path={appRoutes.asset}
          element={
            <Page>
              <ComponentPage />
            </Page>
          }
        />
        {[appRoutes.design, appRoutes.designScenario, appRoutes.designPage].map((path) => (
          <Route
            key={path}
            path={path}
            element={
              <Page>
                <DesignStudioPage />
              </Page>
            }
          />
        ))}
        <Route path={appRoutes.legacySketch} element={<SketchRedirect />} />
        <Route
          path={appRoutes.review}
          element={
            <Page>
              <ReviewPage />
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
          path={appRoutes.activity}
          element={
            <Page>
              <ActivityPage />
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
          path={appRoutes.create}
          element={
            <Page>
              <CreatePage />
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
