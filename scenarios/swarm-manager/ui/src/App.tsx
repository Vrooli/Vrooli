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
import { ErrorBoundary } from "./components/ui/error-boundary";
import { PageErrorBoundary } from "./components/ui/page-error-boundary";
import { MainLayout } from "./components/layout/MainLayout";
import { IdeasPage } from "./pages/IdeasPage";
import { IdeaDetailsPage } from "./pages/IdeaDetailsPage";
import { ScenariosPage } from "./pages/ScenariosPage";
import { ScenarioDetailsPage } from "./pages/ScenarioDetailsPage";
import { RecommendationsPage } from "./pages/RecommendationsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { NotFoundPage } from "./pages/NotFoundPage";

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
  return (
    <ErrorBoundary onError={handleError}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<MainLayout />}>
            <Route index element={<Navigate to="/ideas" replace />} />
            <Route
              path="ideas"
              element={
                <PageErrorBoundary pageName="Ideas">
                  <IdeasPage />
                </PageErrorBoundary>
              }
            />
            <Route
              path="ideas/:name"
              element={
                <PageErrorBoundary pageName="Idea Details">
                  <IdeaDetailsPage />
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
              path="recommendations"
              element={
                <PageErrorBoundary pageName="Recommendations">
                  <RecommendationsPage />
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
