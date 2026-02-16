/**
 * Main Application Component
 *
 * Wraps the app with:
 * - Error Boundary: Catches runtime errors and shows recovery UI
 * - BrowserRouter: Client-side routing
 * - 404 handler: Catches unknown routes
 *
 * ╔════════════════════════════════════════════════════════════════╗
 * ║  ERROR HANDLING LAYERS                                         ║
 * ║                                                                ║
 * ║  1. ErrorBoundary (App) - catches catastrophic errors         ║
 * ║  2. PageErrorBoundary - isolates page-level crashes           ║
 * ║  3. API Client (ApiError) - catches HTTP/network errors       ║
 * ║  4. ErrorState component - displays user-friendly messages    ║
 * ║  5. NotFoundPage - handles invalid routes                     ║
 * ╚════════════════════════════════════════════════════════════════╝
 */

import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { ErrorBoundary } from "./components/ui/error-boundary";
import { PageErrorBoundary } from "./components/ui/page-error-boundary";
import { MainLayout } from "./components/layout/MainLayout";
import { BacklogPage } from "./pages/BacklogPage";
import { BacklogDetailsPage } from "./pages/BacklogDetailsPage";
import { ScenariosPage } from "./pages/ScenariosPage";
import { ScenarioDetailsPage } from "./pages/ScenarioDetailsPage";
import { ExecutionPage } from "./pages/ExecutionPage";
import { PromptsPage } from "./pages/PromptsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { NotFoundPage } from "./pages/NotFoundPage";

/**
 * Compute the BrowserRouter basename from proxy context.
 *
 * When the UI is served through a proxy (e.g., app-monitor at
 * /apps/swarm-manager/proxy/), React Router needs the proxy path
 * as its basename so that all navigation targets
 * (navigate("/backlog"), <Link to="/settings">, etc.) resolve
 * relative to the proxy prefix instead of the domain root.
 */
function getRouterBasename(): string {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  if (proxyPath) {
    return proxyPath.replace(/\/+$/, "");
  }
  return "";
}

/**
 * Optional error logging callback.
 * In production, this could send errors to a logging service.
 */
function handleError(error: Error, _errorInfo: React.ErrorInfo) {
  // In development, errors are already logged by ErrorBoundary
  // In production, this could send to a service like Sentry
  if (import.meta.env.PROD) {
    // Future: send to error tracking service
    console.error("[App] Production error:", error.message);
  }
}

export default function App() {
  const basename = getRouterBasename();

  return (
    <ErrorBoundary onError={handleError}>
      <BrowserRouter basename={basename}>
        <Routes>
          <Route path="/" element={<MainLayout />}>
            <Route index element={<Navigate to="/backlog" replace />} />
            <Route
              path="backlog"
              element={
                <PageErrorBoundary pageName="Backlog">
                  <BacklogPage />
                </PageErrorBoundary>
              }
            />
            <Route
              path="backlog/:kind/:name"
              element={
                <PageErrorBoundary pageName="Backlog Details">
                  <BacklogDetailsPage />
                </PageErrorBoundary>
              }
            />
            <Route
              path="scenarios"
              element={
                <PageErrorBoundary pageName="Scenarios">
                  <ScenariosPage />
                </PageErrorBoundary>
              }
            />
            <Route
              path="scenarios/:name"
              element={
                <PageErrorBoundary pageName="Scenario Details">
                  <ScenarioDetailsPage />
                </PageErrorBoundary>
              }
            />
            <Route
              path="execution"
              element={
                <PageErrorBoundary pageName="Execution">
                  <ExecutionPage />
                </PageErrorBoundary>
              }
            />
            <Route
              path="prompts"
              element={
                <PageErrorBoundary pageName="Prompts">
                  <PromptsPage />
                </PageErrorBoundary>
              }
            />
            <Route
              path="settings"
              element={
                <PageErrorBoundary pageName="Settings">
                  <SettingsPage />
                </PageErrorBoundary>
              }
            />
            {/* 404 handler - catches all unknown routes */}
            <Route path="*" element={<NotFoundPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
