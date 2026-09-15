import { Suspense, lazy, type ReactNode } from "react";
import { Route, Routes } from "react-router-dom";

import { ErrorBoundary } from "./components/ErrorBoundary";
import { RouteSkeleton } from "./components/ui/RouteSkeleton";
import { strings } from "./consts/strings";
import { useTranslation } from "./i18n";
import { AppShell } from "./layout/AppShell";
import { ROUTE_PATTERNS } from "./routes.generated";

const DashboardPage = lazy(() => import("./pages/DashboardPage").then((m) => ({ default: m.DashboardPage })));
const SettingsPage = lazy(() => import("./pages/SettingsPage").then((m) => ({ default: m.SettingsPage })));
const NotFoundPage = lazy(() => import("./pages/NotFoundPage").then((m) => ({ default: m.NotFoundPage })));
const ValidationListPage = lazy(() =>
  import("./features/validation/ValidationListPage").then((m) => ({ default: m.ValidationListPage })),
);
const ValidationDetailPage = lazy(() =>
  import("./features/validation/ValidationDetailPage").then((m) => ({ default: m.ValidationDetailPage })),
);
const CaptureGalleryPage = lazy(() =>
  import("./features/captures/CaptureGalleryPage").then((m) => ({ default: m.CaptureGalleryPage })),
);
const SearchPage = lazy(() => import("./features/search/SearchPage").then((m) => ({ default: m.SearchPage })));
const InventoryPage = lazy(() =>
  import("./features/inventory/InventoryPage").then((m) => ({ default: m.InventoryPage })),
);
const SurfaceDetailPage = lazy(() =>
  import("./features/inventory/SurfaceDetailPage").then((m) => ({ default: m.SurfaceDetailPage })),
);
const ReindexPage = lazy(() => import("./features/reindex/ReindexPage").then((m) => ({ default: m.ReindexPage })));
const JobDetailPage = lazy(() =>
  import("./features/reindex/JobDetailPage").then((m) => ({ default: m.JobDetailPage })),
);

function Page({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  return (
    <ErrorBoundary>
      <Suspense fallback={<RouteSkeleton label={t(strings.route.loading)} />}>{children}</Suspense>
    </ErrorBoundary>
  );
}

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path={ROUTE_PATTERNS.dashboard} element={<Page><DashboardPage /></Page>} />
        <Route path={ROUTE_PATTERNS.validation} element={<Page><ValidationListPage /></Page>} />
        <Route path={ROUTE_PATTERNS.validationDetail} element={<Page><ValidationDetailPage /></Page>} />
        <Route path={ROUTE_PATTERNS.captures} element={<Page><CaptureGalleryPage /></Page>} />
        <Route path={ROUTE_PATTERNS.search} element={<Page><SearchPage /></Page>} />
        <Route path={ROUTE_PATTERNS.inventory} element={<Page><InventoryPage /></Page>} />
        <Route path={ROUTE_PATTERNS.surfaceDetail} element={<Page><SurfaceDetailPage /></Page>} />
        <Route path={ROUTE_PATTERNS.reindex} element={<Page><ReindexPage /></Page>} />
        <Route path={ROUTE_PATTERNS.reindexJob} element={<Page><JobDetailPage /></Page>} />
        <Route path={ROUTE_PATTERNS.settings} element={<Page><SettingsPage /></Page>} />
        <Route path={ROUTE_PATTERNS.notFound} element={<Page><NotFoundPage /></Page>} />
      </Route>
    </Routes>
  );
}
