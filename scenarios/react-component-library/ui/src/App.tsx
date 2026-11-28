import { Suspense, lazy, type ReactNode } from "react";
import { Navigate, Route, Routes, useLocation, useParams } from "react-router-dom";

import { ErrorBoundary } from "./components/ErrorBoundary";
import { useHostShortcutRelay } from "./hooks/useHostShortcutRelay";
import { useTranslation } from "./i18n";
import { strings } from "./consts/strings";
import { WorkbenchShell } from "./workbench/WorkbenchShell";
import { appRoutes, designPath, workspacePath, type StatsView } from "./routes";
import { useWorkspaces } from "./workbench/workspaces/useWorkspaces";

const StartPage = lazy(() =>
  import("./workbench/pages/StartPage").then((m) => ({ default: m.StartPage })),
);
const LibraryPage = lazy(() =>
  import("./workbench/pages/LibraryPage").then((m) => ({ default: m.LibraryPage })),
);
const ComponentPage = lazy(() =>
  import("./workbench/pages/ComponentPage").then((m) => ({ default: m.ComponentPage })),
);
const DesignIndexPage = lazy(() =>
  import("./workbench/pages/DesignIndexPage").then((m) => ({ default: m.DesignIndexPage })),
);
const AdoptExperiencePage = lazy(() =>
  import("./workbench/pages/AdoptExperiencePage").then((m) => ({ default: m.AdoptExperiencePage })),
);
const ComponentContextRoute = lazy(() =>
  import("./workbench/pages/ComponentContextRoute").then((m) => ({
    default: m.ComponentContextRoute,
  })),
);
const WorkspaceStudioPage = lazy(() =>
  import("./workbench/pages/WorkspaceStudioPage").then((m) => ({
    default: m.WorkspaceStudioPage,
  })),
);
const ReviewPage = lazy(() =>
  import("./workbench/pages/ReviewPage").then((m) => ({ default: m.ReviewPage })),
);
const StatsPage = lazy(() =>
  import("./workbench/pages/StatsPage").then((m) => ({ default: m.StatsPage })),
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
const WorkspaceStateCapturePage = lazy(() =>
  import("./workbench/pages/WorkspaceStateCapturePage").then((m) => ({ default: m.WorkspaceStateCapturePage })),
);
const NotFoundPage = lazy(() =>
  import("./workbench/pages/NotFoundPage").then((m) => ({ default: m.NotFoundPage })),
);

/**
 * The URL is the asset identity. Remounting on a different asset is the only
 * reset that is guaranteed to run before the first query, so a version or
 * pinned story from the previous component can never reach the next one.
 */
function AssetPageRoute() {
  const { id = "" } = useParams();
  return <ComponentPage key={id} />;
}

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
      <span style={{ display: "block", marginTop: 12 }}>
        {t("route.loading", { defaultValue: "Loading…" })}
      </span>
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

/** `/components` and `/catalog` were the asset catalog. Preserve their query string. */
function LibraryRedirect() {
  const location = useLocation();
  return <Navigate to={`${appRoutes.library}${location.search}`} replace />;
}

/** The three measurement routes are Stats views; preserve unrelated params. */
function StatsRedirect({ view }: { view: StatsView }) {
  const location = useLocation();
  const search = new URLSearchParams(location.search);
  search.set("view", view);
  return <Navigate to={`${appRoutes.stats}?${search.toString()}`} replace />;
}

/** The legacy sketch route keeps scenario/page context on the studio path. */
function SketchRedirect() {
  const location = useLocation();
  const parts = location.pathname.split("/").filter(Boolean);
  const scenario = parts[1];
  const page = parts[2];
  const search = new URLSearchParams(location.search);
  search.delete("scenario");
  search.delete("page");
  const target = designPath(scenario, page);
  const serialized = search.toString();
  return <Navigate to={serialized ? `${target}?${serialized}` : target} replace />;
}

/**
 * A page binding is the durable link between a scenario page and the workspace
 * that composes it. Looking it up lets the old `/design/:scenario/:page` URL
 * open the workspace studio once a workspace owns that page. An unbound
 * experience page gets an explicit adoption screen; it never opens the legacy
 * composition owner.
 */
function BoundDesignRoute({ scenario, page }: { scenario: string; page: string }) {
  const { t } = useTranslation();
  const workspaces = useWorkspaces();
  const bound = (workspaces.data ?? []).find(
    (entry) => entry.bindingScenario === scenario && entry.bindingPage === page,
  );
  if (bound) return <Navigate to={workspacePath(bound.id)} replace />;
  if (workspaces.isLoading) return <RouteSkeleton />;
  if (workspaces.isError) {
    return (
      <div className="wb-page" role="alert">
        {t(strings.workbench.adopt.bindingUnavailable)}
      </div>
    );
  }
  return <AdoptExperiencePage />;
}

/**
 * `/design` is the workspace index. A bookmarked component-context URL is
 * converted into a fork of its matching curated workspace, with component
 * identity saved into the workspace slot before opening the editor.
 */
function DesignRoute() {
  const location = useLocation();
  const search = new URLSearchParams(location.search);
  const workspaceId = search.get("workspace");
  if (workspaceId) {
    return (
      <Page>
        <WorkspaceStudioPage workspaceId={workspaceId} />
      </Page>
    );
  }
  const scenario = search.get("scenario");
  const page = search.get("page");
  if (scenario && page) {
    return (
      <Page>
        <BoundDesignRoute scenario={scenario} page={page} />
      </Page>
    );
  }
  const componentContext = search.has("component");
  return <Page>{componentContext ? <ComponentContextRoute /> : <DesignIndexPage />}</Page>;
}

/** The path form of the same binding lookup. */
function BoundDesignPage() {
  const { scenario = "", page = "" } = useParams<{ scenario?: string; page?: string }>();
  if (!page) return <Navigate to={appRoutes.design} replace />;
  return (
    <Page>
      <BoundDesignRoute scenario={scenario} page={page} />
    </Page>
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
            <PreviewPage />
          </Page>
        }
      />
      <Route
        path={appRoutes.workspaceCapture}
        element={<Page><WorkspaceStateCapturePage /></Page>}
      />
      <Route element={<WorkbenchShell />}>
        <Route
          path={appRoutes.start}
          element={
            <Page>
              <StartPage />
            </Page>
          }
        />
        <Route
          path={appRoutes.library}
          element={
            <Page>
              <LibraryPage />
            </Page>
          }
        />
        <Route path={appRoutes.components} element={<LibraryRedirect />} />
        <Route path={appRoutes.assetCatalog} element={<LibraryRedirect />} />
        <Route
          path={appRoutes.asset}
          element={
            <Page>
              <AssetPageRoute />
            </Page>
          }
        />
        <Route path={appRoutes.design} element={<DesignRoute />} />
        {[appRoutes.designScenario, appRoutes.designPage].map((path) => (
          <Route key={path} path={path} element={<BoundDesignPage />} />
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
          path={appRoutes.stats}
          element={
            <Page>
              <StatsPage />
            </Page>
          }
        />
        <Route path={appRoutes.coverage} element={<StatsRedirect view="quality" />} />
        <Route path={appRoutes.capabilities} element={<StatsRedirect view="services" />} />
        <Route path={appRoutes.activity} element={<StatsRedirect view="activity" />} />
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
