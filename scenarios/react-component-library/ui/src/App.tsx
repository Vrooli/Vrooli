import { Suspense, lazy, type ReactNode } from "react";
import { Route, Routes } from "react-router-dom";

import { AppShell } from "./components/AppShell";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { useTranslation } from "./i18n";

const DashboardPage = lazy(() =>
  import("./pages/DashboardPage").then((m) => ({ default: m.DashboardPage })),
);
const ComponentsPage = lazy(() =>
  import("./pages/ComponentsPage").then((m) => ({ default: m.ComponentsPage })),
);
const ComponentDetailPage = lazy(() =>
  import("./pages/ComponentDetailPage").then((m) => ({ default: m.ComponentDetailPage })),
);
const AdoptionsPage = lazy(() =>
  import("./pages/AdoptionsPage").then((m) => ({ default: m.AdoptionsPage })),
);
const SettingsPage = lazy(() =>
  import("./pages/SettingsPage").then((m) => ({ default: m.SettingsPage })),
);
const NotFoundPage = lazy(() =>
  import("./pages/NotFoundPage").then((m) => ({ default: m.NotFoundPage })),
);

function RouteSkeleton() {
  const { t } = useTranslation();
  return (
    <div
      data-testid="route-skeleton"
      role="status"
      aria-live="polite"
      className="flex items-center gap-2 text-sm text-app-muted-foreground"
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
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<Page><DashboardPage /></Page>} />
        <Route path="/components" element={<Page><ComponentsPage /></Page>} />
        <Route path="/components/:id" element={<Page><ComponentDetailPage /></Page>} />
        <Route path="/adoptions" element={<Page><AdoptionsPage /></Page>} />
        <Route path="/settings" element={<Page><SettingsPage /></Page>} />
        <Route path="*" element={<Page><NotFoundPage /></Page>} />
      </Route>
    </Routes>
  );
}
