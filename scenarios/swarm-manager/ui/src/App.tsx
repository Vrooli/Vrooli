/**
 * Main Application Component
 *
 * Wraps the app with:
 * - Error Boundary: Catches runtime errors and shows recovery UI
 * - BrowserRouter: Client-side routing
 * - 404 handler: Catches unknown routes
 *
 * The primary route is /graph (GraphWorkspace), which hosts detail pages
 * as state-driven overlays. Legacy routes redirect to /graph with
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
            {/* Primary route: graph workspace (detail pages render as overlays inside) */}
            <Route path="/graph" element={<PageErrorBoundary pageName="Graph"><GraphWorkspace /></PageErrorBoundary>} />

            {/* Root redirects to /graph */}
            <Route index element={<Navigate to="/graph" replace />} />

            {/* Legacy route redirects */}
            <Route path="backlog" element={<BacklogRedirect />} />
            <Route path="backlog/:kind/:name" element={<BacklogDetailsRedirect />} />
            <Route path="scenarios" element={<ScenariosRedirect />} />
            <Route path="scenarios/:name" element={<ScenarioDetailsRedirect />} />
            <Route path="execution" element={<ExecutionRedirect />} />
            <Route path="prompts" element={<PromptsRedirect />} />
            <Route path="settings" element={<SettingsRedirect />} />

            {/* Legacy detail page routes redirect to graph with detail params */}
            <Route path="details/backlog/:kind/:name" element={<BacklogDetailsRedirect />} />
            <Route path="details/scenario/:name" element={<ScenarioDetailsRedirect />} />
            <Route path="details/execution/:executionId" element={<Navigate to="/graph" replace />} />
            <Route path="details/initiative/:name" element={<Navigate to="/graph" replace />} />

            {/* 404 handler */}
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
