import { ReactNode } from "react";
import { Link, useLocation } from "react-router-dom";
import { RefreshCw, Database, ArrowLeft } from "lucide-react";
import { Button } from "./ui/button";
import { HealthIndicator, type HealthStatus } from "./ui/HealthIndicator";

// ─────────────────────────────────────────────────────────────────────────────
// Layout Component
// [REQ:P0-001] Reference Scenario Registry - Shared layout structure
// ─────────────────────────────────────────────────────────────────────────────
//
// This component provides consistent header/main/footer structure across all pages.
// The header adapts based on whether we're on a detail page (shows back button)
// or the root dashboard (shows brand).
//
// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe-safe layout                                    ║
// ║                                                                          ║
// ║  Uses h-full instead of h-screen/min-h-screen because:                   ║
// ║  - h-screen compiles to 100vh which can refer to the OUTER window's      ║
// ║    viewport inside an iframe                                             ║
// ║  - h-full (100%) correctly inherits from parent (iframe)                 ║
// ║                                                                          ║
// ║  See: UI Interop skill §4.5 (Iframe-Safe Scroll & Viewport)              ║
// ╚══════════════════════════════════════════════════════════════════════════╝
// ─────────────────────────────────────────────────────────────────────────────

interface LayoutProps {
  /** Page title displayed in header */
  title: string;
  /** Optional subtitle or breadcrumb text */
  subtitle?: string;
  /** Health status for the indicator */
  healthStatus: HealthStatus;
  /** Whether data is currently loading */
  isLoading: boolean;
  /** Callback when refresh button is clicked */
  onRefresh: () => void;
  /** Page content */
  children: ReactNode;
  /** Optional test ID prefix for the page */
  testIdPrefix?: string;
}

export function Layout({
  title,
  subtitle,
  healthStatus,
  isLoading,
  onRefresh,
  children,
  testIdPrefix = "page"
}: LayoutProps) {
  const location = useLocation();
  const isDetailPage = location.pathname !== "/";

  return (
    <div className="h-full bg-slate-950 text-slate-50 flex flex-col overflow-hidden">
      {/* Header */}
      <header className="shrink-0 border-b border-white/10 bg-slate-950/80 backdrop-blur-sm">
        <div className="mx-auto max-w-6xl px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              {isDetailPage ? (
                <>
                  <Button
                    variant="ghost"
                    size="sm"
                    asChild
                    className="hover:bg-white/5"
                  >
                    <Link to="/" data-testid={`${testIdPrefix}-back-button`}>
                      <ArrowLeft className="h-4 w-4 mr-2" />
                      Back
                    </Link>
                  </Button>
                  <div className="h-6 w-px bg-white/10" />
                </>
              ) : (
                <Database className="h-6 w-6 text-indigo-400" />
              )}
              <div className="min-w-0">
                <h1 data-testid={`${testIdPrefix}-title`} className="text-xl font-semibold truncate">
                  {title}
                </h1>
                {subtitle && (
                  <p className="text-sm text-slate-400 truncate">{subtitle}</p>
                )}
              </div>
            </div>

            <div className="flex items-center gap-4">
              <HealthIndicator
                status={healthStatus}
                isLoading={isLoading}
                testId={`${testIdPrefix}-health-status`}
              />

              <Button
                data-testid={`${testIdPrefix}-refresh-button`}
                variant="outline"
                size="sm"
                onClick={onRefresh}
                disabled={isLoading}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
                Refresh
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <div className="mx-auto max-w-6xl px-6 py-8">
          {children}
        </div>
      </main>

      {/* Footer */}
      <footer className="shrink-0 border-t border-white/5 bg-slate-950/50">
        <div className="mx-auto max-w-6xl px-6 py-3 flex items-center justify-between">
          {isDetailPage ? (
            <Link
              to="/"
              className="text-xs text-slate-500 hover:text-slate-400 transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500/50 rounded"
            >
              ← Back to dashboard
            </Link>
          ) : (
            <span className="text-xs text-slate-500" />
          )}
          <p className="text-xs text-slate-500">
            Development Toolchain Validator
          </p>
        </div>
      </footer>
    </div>
  );
}
