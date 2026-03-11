import { useQuery } from "@tanstack/react-query";
import { RefreshCw, Database, CheckCircle, XCircle, AlertCircle, FolderCode, Tag } from "lucide-react";
import { Button } from "./components/ui/button";
import { fetchHealth, fetchReferences, type Reference } from "./lib/api";

// ─────────────────────────────────────────────────────────────────────────────
// Reference Card Component
// ─────────────────────────────────────────────────────────────────────────────

interface ReferenceCardProps {
  reference: Reference;
}

function ReferenceCard({ reference }: ReferenceCardProps) {
  const formattedDate = new Date(reference.updated_at).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric"
  });

  return (
    <div
      data-testid={`reference-card-${reference.slug}`}
      className="rounded-xl border border-white/10 bg-white/5 p-5 transition-colors hover:bg-white/8"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h3 className="text-lg font-medium text-slate-50 truncate">{reference.name}</h3>
          <p className="mt-1 text-sm text-slate-400 font-mono">{reference.slug}</p>
        </div>
        <span
          data-testid={`reference-template-${reference.slug}`}
          className="shrink-0 inline-flex items-center gap-1.5 rounded-full bg-indigo-500/20 px-3 py-1 text-xs font-medium text-indigo-300"
        >
          <Tag className="h-3 w-3" />
          {reference.template}
        </span>
      </div>

      <div
        data-testid={`reference-path-${reference.slug}`}
        className="mt-4 flex items-center gap-2 text-sm text-slate-400"
      >
        <FolderCode className="h-4 w-4 shrink-0" />
        <span className="truncate font-mono text-xs">{reference.path}</span>
      </div>

      {reference.description && (
        <p className="mt-3 text-sm text-slate-300 line-clamp-2">{reference.description}</p>
      )}

      <div className="mt-4 flex items-center justify-between text-xs text-slate-500">
        <span>Updated {formattedDate}</span>
        <span className="font-mono">{reference.id.slice(0, 8)}</span>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Main App Component
// [REQ:P0-002] Reference Scenario API Endpoints - UI integration
// ─────────────────────────────────────────────────────────────────────────────

export default function App() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000 // Refresh health every 30 seconds
  });

  const referencesQuery = useQuery({
    queryKey: ["references"],
    queryFn: () => fetchReferences(),
    enabled: healthQuery.isSuccess // Only fetch references when API is healthy
  });

  const isHealthy = healthQuery.isSuccess;
  const isLoading = healthQuery.isLoading || referencesQuery.isLoading;
  const hasError = healthQuery.isError || referencesQuery.isError;
  const references = referencesQuery.data ?? [];

  const handleRefresh = () => {
    healthQuery.refetch();
    referencesQuery.refetch();
  };

  // ╔══════════════════════════════════════════════════════════════╗
  // ║  INTEROP-CRITICAL: Iframe-safe layout                        ║
  // ║                                                              ║
  // ║  Uses h-full instead of h-screen/min-h-screen because:       ║
  // ║  - h-screen compiles to 100vh which can refer to the OUTER   ║
  // ║    window's viewport inside an iframe                        ║
  // ║  - h-full (100%) correctly inherits from parent (iframe)     ║
  // ║                                                              ║
  // ║  See: UI Interop skill §4.5 (Iframe-Safe Scroll & Viewport)  ║
  // ╚══════════════════════════════════════════════════════════════╝
  return (
    <div className="h-full bg-slate-950 text-slate-50 flex flex-col overflow-hidden">
      {/* Header */}
      <header className="shrink-0 border-b border-white/10 bg-slate-950/80 backdrop-blur-sm">
        <div className="mx-auto max-w-6xl px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Database className="h-6 w-6 text-indigo-400" />
              <h1 data-testid="dashboard-title" className="text-xl font-semibold">
                Development Toolchain Validator
              </h1>
            </div>

            <div className="flex items-center gap-4">
              {/* Health status indicator */}
              <div
                data-testid="dashboard-health-status"
                className="flex items-center gap-2 text-sm"
              >
                {isLoading ? (
                  <AlertCircle className="h-4 w-4 text-amber-400" />
                ) : isHealthy ? (
                  <CheckCircle className="h-4 w-4 text-emerald-400" />
                ) : (
                  <XCircle className="h-4 w-4 text-red-400" />
                )}
                <span className={isHealthy ? "text-emerald-400" : isLoading ? "text-amber-400" : "text-red-400"}>
                  {isLoading ? "Checking..." : isHealthy ? "Connected" : "Disconnected"}
                </span>
              </div>

              <Button
                data-testid="dashboard-refresh-button"
                variant="outline"
                size="sm"
                onClick={handleRefresh}
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
          {/* Section header */}
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-lg font-medium">Reference Scenarios</h2>
              <p className="mt-1 text-sm text-slate-400">
                Known-good implementations for validating steer skills and development tools
              </p>
            </div>
            <div
              data-testid="dashboard-reference-count"
              className="text-sm text-slate-400"
            >
              {references.length} {references.length === 1 ? "reference" : "references"}
            </div>
          </div>

          {/* Loading state */}
          {isLoading && (
            <div
              data-testid="dashboard-loading-state"
              className="flex flex-col items-center justify-center py-16"
            >
              <RefreshCw className="h-8 w-8 text-slate-400 animate-spin mb-4" />
              <p className="text-slate-400">Loading references...</p>
            </div>
          )}

          {/* Error state */}
          {hasError && !isLoading && (
            <div
              data-testid="dashboard-error-state"
              className="rounded-xl border border-red-500/20 bg-red-500/10 p-6 text-center"
            >
              <XCircle className="h-8 w-8 text-red-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-red-300">Unable to connect</h3>
              <p className="mt-2 text-sm text-slate-400">
                Make sure the scenario is running with <code className="font-mono bg-slate-800 px-1.5 py-0.5 rounded">vrooli scenario start development-toolchain-validator</code>
              </p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={handleRefresh}
              >
                Try Again
              </Button>
            </div>
          )}

          {/* Empty state */}
          {!isLoading && !hasError && references.length === 0 && (
            <div
              data-testid="dashboard-empty-state"
              className="rounded-xl border border-white/10 bg-white/5 p-8 text-center"
            >
              <Database className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <h3 className="text-lg font-medium">No references registered</h3>
              <p className="mt-2 text-sm text-slate-400 max-w-md mx-auto">
                Reference scenarios serve as ground truth for validating steer skills and development tools.
                Use the CLI to register your first reference.
              </p>
              <div className="mt-6 bg-slate-800/50 rounded-lg p-4 max-w-lg mx-auto">
                <code className="text-sm text-slate-300 font-mono">
                  dtv reference create --slug reference-react-vite --name &quot;React Vite Reference&quot; --template react-vite --path /path/to/scenario
                </code>
              </div>
            </div>
          )}

          {/* Reference list */}
          {!isLoading && !hasError && references.length > 0 && (
            <div
              data-testid="references-list"
              className="grid gap-4 md:grid-cols-2"
            >
              {references.map((ref) => (
                <ReferenceCard key={ref.id} reference={ref} />
              ))}
            </div>
          )}
        </div>
      </main>

      {/* Footer */}
      <footer className="shrink-0 border-t border-white/5 bg-slate-950/50">
        <div className="mx-auto max-w-6xl px-6 py-3">
          <p className="text-xs text-slate-500 text-center">
            Validates cross-steer skill interoperability, development tooling correctness, and scenario quality infrastructure
          </p>
        </div>
      </footer>
    </div>
  );
}
