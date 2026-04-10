import { HashRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Layout } from "./components/Layout";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { StreamPage } from "./pages/StreamPage";
import { AnalyticsPage } from "./pages/AnalyticsPage";
import { EventLogPage } from "./pages/EventLogPage";
import { SettingsPage } from "./pages/SettingsPage";
import { ScenarioMetricsPage } from "./pages/ScenarioMetricsPage";
import { CorrelationTracePage } from "./pages/CorrelationTracePage";
import { PoliciesPage } from "./pages/PoliciesPage";
import { PolicyEditorPage } from "./pages/PolicyEditorPage";
import { CircuitBreakerPage } from "./pages/CircuitBreakerPage";
import { SubscriptionsPage } from "./pages/SubscriptionsPage";
import { SubscriptionHealthPage } from "./pages/SubscriptionHealthPage";
import { CompliancePage } from "./pages/CompliancePage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10000),
      staleTime: 5000,
    },
  },
});

export function AppRouter() {
  return (
    <ErrorBoundary fallbackMessage="Vrooli Events failed to load">
      <QueryClientProvider client={queryClient}>
        <HashRouter>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<Navigate to="/stream" replace />} />
              <Route path="stream" element={<StreamPage />} />
              <Route path="analytics" element={<AnalyticsPage />} />
              <Route path="scenarios" element={<ScenarioMetricsPage />} />
              <Route path="traces" element={<CorrelationTracePage />} />
              <Route path="events" element={<EventLogPage />} />
              <Route path="policies" element={<PoliciesPage />} />
              <Route path="policies/:id/edit" element={<PolicyEditorPage />} />
              <Route path="circuit-breakers" element={<CircuitBreakerPage />} />
              <Route path="subscriptions" element={<SubscriptionsPage />} />
              <Route path="subscriptions/:id/health" element={<SubscriptionHealthPage />} />
              <Route path="compliance" element={<CompliancePage />} />
              <Route path="settings" element={<SettingsPage />} />
              <Route path="*" element={<Navigate to="/stream" replace />} />
            </Route>
          </Routes>
        </HashRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

import type { Route as RouteId } from "./lib/router";
export type { RouteId };
