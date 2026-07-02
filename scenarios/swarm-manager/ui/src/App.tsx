/**
 * Main Application Component
 *
 * Wraps the app with:
 * - Error Boundary: Catches runtime errors and shows recovery UI
 * - BrowserRouter: Client-side routing
 * - 404 handler: Catches unknown routes
 *
 * Fullscreen experiences are first-class routes. The graph, entity details,
 * Command Post, and Decision Stream all participate in browser history.
 */

import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route, Navigate, useLocation } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { ErrorBoundary } from "./components/ui/error-boundary";
import { PageErrorBoundary } from "./components/ui/page-error-boundary";
import { NotFoundPage } from "./pages/NotFoundPage";
import { AppShell } from "./app/shell/AppShell";

/**
 * Redirect that carries the query string along — old /operations filter
 * links (status/lane/owner_type/q/window_seconds/view) use the same param
 * names the Plan board reads, so shared bookmarks keep their filters.
 */
function RedirectPreservingSearch({ to }: { to: string }) {
  const location = useLocation();
  return <Navigate to={`${to}${location.search}`} replace />;
}

const GraphWorkspace = lazy(() =>
  import("./surfaces/graph/components/GraphWorkspace").then((m) => ({
    default: m.GraphWorkspace,
  })),
);
const BacklogDetailsPage = lazy(() =>
  import("./pages/BacklogDetailsPage").then((m) => ({ default: m.BacklogDetailsPage })),
);
const ScenarioDetailsPage = lazy(() =>
  import("./pages/ScenarioDetailsPage").then((m) => ({ default: m.ScenarioDetailsPage })),
);
const ExecutionDetailsPage = lazy(() =>
  import("./pages/ExecutionDetailsPage").then((m) => ({ default: m.ExecutionDetailsPage })),
);
const InitiativeDetailsPage = lazy(() =>
  import("./pages/InitiativeDetailsPage").then((m) => ({ default: m.InitiativeDetailsPage })),
);
const CaptureDetailsPage = lazy(() =>
  import("./pages/CaptureDetailsPage").then((m) => ({ default: m.CaptureDetailsPage })),
);
const RecordsPage = lazy(() =>
  import("./pages/RecordsPage").then((m) => ({ default: m.RecordsPage })),
);
const RecordDetailsPage = lazy(() =>
  import("./pages/RecordDetailsPage").then((m) => ({ default: m.RecordDetailsPage })),
);
const SessionDetailsPage = lazy(() =>
  import("./pages/SessionDetailsPage").then((m) => ({ default: m.SessionDetailsPage })),
);
const OperatingModeDetailsPage = lazy(() =>
  import("./pages/OperatingModeDetailsPage").then((m) => ({ default: m.OperatingModeDetailsPage })),
);

/**
 * Compute the BrowserRouter basename from proxy context.
 */
function getRouterBasename(): string {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  if (proxyPath) {
    return proxyPath.replace(/\/+$/, "");
  }
  return "";
}

function handleError(error: Error, _errorInfo: React.ErrorInfo) {
  if (import.meta.env.PROD) {
    console.error("[App] Production error:", error.message);
  }
}

export default function App() {
  const basename = getRouterBasename();

  return (
    <ErrorBoundary onError={handleError}>
      <BrowserRouter basename={basename}>
        <Suspense
          fallback={
            <div className="flex h-screen items-center justify-center bg-slate-950 text-slate-400">
              Loading...
            </div>
          }
        >
          <Routes>
            <Route element={<AppShell />}>
              <Route path="/graph" element={<PageErrorBoundary pageName="Graph"><GraphWorkspace /></PageErrorBoundary>} />
              <Route path="/graph/:lens" element={<PageErrorBoundary pageName="Graph"><GraphWorkspace /></PageErrorBoundary>} />

              <Route index element={<Navigate to="/graph/plan" replace />} />

              <Route path="backlog/:kind/:name" element={<PageErrorBoundary pageName="Backlog Details"><BacklogDetailsPage /></PageErrorBoundary>} />
              <Route path="scenarios/:name" element={<PageErrorBoundary pageName="Scenario Details"><ScenarioDetailsPage /></PageErrorBoundary>} />
              <Route path="executions/:executionId" element={<PageErrorBoundary pageName="Execution Details"><ExecutionDetailsPage /></PageErrorBoundary>} />
              <Route path="initiatives/:name" element={<PageErrorBoundary pageName="Initiative Details"><InitiativeDetailsPage /></PageErrorBoundary>} />
              <Route path="captures/:captureId" element={<PageErrorBoundary pageName="Capture Details"><CaptureDetailsPage /></PageErrorBoundary>} />
              <Route path="records" element={<PageErrorBoundary pageName="Records"><RecordsPage /></PageErrorBoundary>} />
              <Route path="records/:recordId" element={<PageErrorBoundary pageName="Record Details"><RecordDetailsPage /></PageErrorBoundary>} />
              <Route path="sessions/:sessionId" element={<PageErrorBoundary pageName="Session Details"><SessionDetailsPage /></PageErrorBoundary>} />
              <Route path="operating-modes/:mode" element={<PageErrorBoundary pageName="Operating Mode Details"><OperatingModeDetailsPage /></PageErrorBoundary>} />
              {/* Retired surfaces — absorbed by the Plan lens board. Old
                  deep links and bookmarks redirect rather than 404. */}
              <Route path="command-post" element={<Navigate to="/graph/plan" replace />} />
              <Route path="command-post/decisions" element={<Navigate to="/graph/plan?drawer=decisions" replace />} />
              <Route path="operations" element={<RedirectPreservingSearch to="/graph/plan" />} />
            </Route>

            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
