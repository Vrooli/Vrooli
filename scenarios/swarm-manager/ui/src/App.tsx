/**
 * Main Application Component
 *
 * Wraps the app with:
 * - Error Boundary: Catches runtime errors and shows recovery UI
 * - BrowserRouter: Client-side routing
 * - 404 handler: Catches unknown routes
 *
 * The primary route is /graph (GraphWorkspace), which replaces the
 * old 5-tab MainLayout. Legacy routes redirect to /graph with
 * appropriate query params.
 */

import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { ErrorBoundary } from "./components/ui/error-boundary";
import { PageErrorBoundary } from "./components/ui/page-error-boundary";
import { NotFoundPage } from "./pages/NotFoundPage";
import {
  BacklogRedirect,
  BacklogDetailsRedirect,
  ScenariosRedirect,
  ScenarioDetailsRedirect,
  ExecutionRedirect,
  PromptsRedirect,
  SettingsRedirect,
} from "./surfaces/graph/components/LegacyRedirect";

const GraphWorkspace = lazy(() =>
  import("./surfaces/graph/components/GraphWorkspace").then((m) => ({
    default: m.GraphWorkspace,
  })),
);
const BacklogDetailsPage = lazy(() =>
  import("./pages/BacklogDetailsPage").then((m) => ({
    default: m.BacklogDetailsPage,
  })),
);
const ScenarioDetailsPage = lazy(() =>
  import("./pages/ScenarioDetailsPage").then((m) => ({
    default: m.ScenarioDetailsPage,
  })),
);
const ExecutionDetailsPage = lazy(() =>
  import("./pages/ExecutionDetailsPage").then((m) => ({
    default: m.ExecutionDetailsPage,
  })),
);
const InitiativeDetailsPage = lazy(() =>
  import("./pages/InitiativeDetailsPage").then((m) => ({
    default: m.InitiativeDetailsPage,
  })),
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
            {/* Primary route: graph workspace */}
            <Route path="/graph" element={<PageErrorBoundary pageName="Graph"><GraphWorkspace /></PageErrorBoundary>} />

            {/* Root redirects to /graph */}
            <Route index element={<Navigate to="/graph" replace />} />

            {/* Detail pages (accessible from graph inspector) */}
            <Route path="/details/backlog/:kind/:name" element={<PageErrorBoundary pageName="Backlog Details"><BacklogDetailsPage /></PageErrorBoundary>} />
            <Route path="/details/scenario/:name" element={<PageErrorBoundary pageName="Scenario Details"><ScenarioDetailsPage /></PageErrorBoundary>} />
            <Route path="/details/execution/:executionId" element={<PageErrorBoundary pageName="Execution Details"><ExecutionDetailsPage /></PageErrorBoundary>} />
            <Route path="/details/execution/:executionId/prompt-trace" element={<PageErrorBoundary pageName="Execution Details"><ExecutionDetailsPage /></PageErrorBoundary>} />
            <Route path="/details/initiative/:name" element={<PageErrorBoundary pageName="Initiative Details"><InitiativeDetailsPage /></PageErrorBoundary>} />

            {/* Legacy route redirects */}
            <Route path="backlog" element={<BacklogRedirect />} />
            <Route path="backlog/:kind/:name" element={<BacklogDetailsRedirect />} />
            <Route path="scenarios" element={<ScenariosRedirect />} />
            <Route path="scenarios/:name" element={<ScenarioDetailsRedirect />} />
            <Route path="execution" element={<ExecutionRedirect />} />
            <Route path="prompts" element={<PromptsRedirect />} />
            <Route path="settings" element={<SettingsRedirect />} />

            {/* 404 handler */}
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
