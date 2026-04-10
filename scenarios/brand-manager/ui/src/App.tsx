import { Suspense, lazy } from "react";
import { useQuery } from "@tanstack/react-query";
import { Palette, Search, Shield } from "lucide-react";
import { useRouter } from "./lib/router";
import { fetchHealth } from "./lib/api";
import { HEALTH_CHECK_RETRY, HEALTH_CHECK_INTERVAL_MS } from "./config/constants";
import { ErrorBoundary } from "./components/error-boundary";
import { cn } from "./lib/utils";

const BrandListPage = lazy(() => import("./pages/BrandListPage"));
const BrandDetailPage = lazy(() => import("./pages/BrandDetailPage"));
const BrandFormPage = lazy(() => import("./pages/BrandFormPage"));
const ScannerPage = lazy(() => import("./pages/ScannerPage"));
const StandardsPage = lazy(() => import("./pages/StandardsPage"));

// [REQ:BM-REQ-UI-DASHBOARD]

export default function App() {
  const { route, navigate } = useRouter();

  const { data: health } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    retry: HEALTH_CHECK_RETRY,
    refetchInterval: HEALTH_CHECK_INTERVAL_MS,
  });

  const isBrandsSection = route.page === "brands" || route.page === "brand-detail" || route.page === "brand-create" || route.page === "brand-edit";
  const isScannerActive = route.page === "scanner";
  const isStandardsActive = route.page === "standards";

  return (
    <div className="h-full flex flex-col overflow-hidden bg-slate-950 text-slate-50" data-testid="app-root">
      {/* Top bar */}
      <header className="border-b border-white/10 bg-slate-950/80 backdrop-blur shrink-0 z-10">
        <div className="max-w-4xl mx-auto px-6 py-3 flex items-center justify-between">
          <button
            onClick={() => navigate("/brands")}
            className={cn(
              "flex items-center gap-2 transition-colors",
              isBrandsSection ? "text-white" : "text-slate-400 hover:text-white",
            )}
            data-testid="nav-home"
          >
            <Palette className="h-5 w-5" />
            <span className="font-semibold">Brand Manager</span>
          </button>
          <nav className="flex items-center gap-3">
            <button
              onClick={() => navigate("/scanner")}
              className={cn(
                "flex items-center gap-1 text-xs transition-colors",
                isScannerActive ? "text-white font-medium" : "text-slate-400 hover:text-slate-50",
              )}
              data-testid="nav-scanner"
              aria-current={isScannerActive ? "page" : undefined}
            >
              <Search className="h-3 w-3" />
              Scanner
            </button>
            <button
              onClick={() => navigate("/standards")}
              className={cn(
                "flex items-center gap-1 text-xs transition-colors",
                isStandardsActive ? "text-white font-medium" : "text-slate-400 hover:text-slate-50",
              )}
              data-testid="nav-standards"
              aria-current={isStandardsActive ? "page" : undefined}
            >
              <Shield className="h-3 w-3" />
              Standards
            </button>
            <div className="flex items-center gap-2 text-xs text-slate-500">
              <span
                className={`h-2 w-2 rounded-full ${health ? "bg-emerald-500" : "bg-red-500"}`}
                data-testid="health-indicator"
              />
              {health ? "API Connected" : "API Offline"}
            </div>
          </nav>
        </div>
      </header>

      {/* Main content — scrollable area contained within iframe */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-4xl mx-auto px-6 py-8">
          <Suspense fallback={<div className="text-center text-slate-400 py-12">Loading...</div>}>
            <ErrorBoundary section="Brand Library">
              {route.page === "brands" && <BrandListPage onNavigate={navigate} />}
            </ErrorBoundary>
            <ErrorBoundary section="Brand Details">
              {route.page === "brand-detail" && <BrandDetailPage brandId={route.id} onNavigate={navigate} />}
            </ErrorBoundary>
            <ErrorBoundary section="Brand Form">
              {route.page === "brand-create" && <BrandFormPage onNavigate={navigate} />}
            </ErrorBoundary>
            <ErrorBoundary section="Brand Form">
              {route.page === "brand-edit" && <BrandFormPage brandId={route.id} onNavigate={navigate} />}
            </ErrorBoundary>
            <ErrorBoundary section="Scanner">
              {route.page === "scanner" && <ScannerPage onNavigate={navigate} />}
            </ErrorBoundary>
            <ErrorBoundary section="Standards">
              {route.page === "standards" && <StandardsPage onNavigate={navigate} />}
            </ErrorBoundary>
          </Suspense>
        </div>
      </main>
    </div>
  );
}
