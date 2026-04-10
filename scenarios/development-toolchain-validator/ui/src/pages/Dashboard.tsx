import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw, Database, FolderCode, Tag, ChevronRight, Link2, Eye, XCircle } from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { CopyableCode } from "../components/ui/CopyableCode";
import { Layout } from "../components/Layout";
import { useNavigationShortcuts, useHealthStatus } from "../hooks";
import { fetchReferences, fetchConnectionsByReference, type Reference } from "../lib/api";
import { formatDate } from "../lib/utils";

// ─────────────────────────────────────────────────────────────────────────────
// Reference Card Component
// [REQ:P0-001] Reference Scenario Registry - Card with navigation to detail
// ─────────────────────────────────────────────────────────────────────────────

interface ReferenceCardProps {
  reference: Reference;
}

function ReferenceCard({ reference }: ReferenceCardProps) {
  // Pre-fetch connections to show count
  const connectionsQuery = useQuery({
    queryKey: ["connections", reference.id],
    queryFn: () => fetchConnectionsByReference(reference.id),
    staleTime: 30000 // Cache for 30 seconds
  });

  const connections = connectionsQuery.data ?? [];
  const connectionCount = connections.length;

  return (
    <div
      data-testid={`reference-card-${reference.slug}`}
      className="rounded-xl border border-white/10 bg-white/5 transition-colors hover:border-indigo-500/30 hover:bg-white/8"
    >
      <Link
        to={`/references/${reference.slug}`}
        data-testid={`reference-card-link-${reference.slug}`}
        className="block p-5 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:ring-inset rounded-xl"
      >
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3 min-w-0 flex-1">
            <ChevronRight className="h-5 w-5 text-slate-400 shrink-0" />
            <div className="min-w-0 flex-1">
              <h3 className="text-lg font-medium text-slate-50 truncate">{reference.name}</h3>
              <p className="mt-1 text-sm text-slate-400 font-mono">{reference.slug}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {connectionCount > 0 && (
              <Badge
                data-testid={`reference-connection-count-${reference.slug}`}
                variant="success"
              >
                <Link2 className="h-3 w-3" />
                {connectionCount}
              </Badge>
            )}
            <Badge
              data-testid={`reference-template-${reference.slug}`}
              variant="primary"
            >
              <Tag className="h-3 w-3" />
              {reference.template}
            </Badge>
          </div>
        </div>

        <div
          data-testid={`reference-path-${reference.slug}`}
          className="mt-4 ml-8 flex items-center gap-2 text-sm text-slate-400"
        >
          <FolderCode className="h-4 w-4 shrink-0" />
          <span className="truncate font-mono text-xs">{reference.path}</span>
        </div>

        {reference.description && (
          <p className="mt-3 ml-8 text-sm text-slate-300 line-clamp-2">{reference.description}</p>
        )}

        <div className="mt-4 ml-8 flex items-center justify-between text-xs text-slate-500">
          <span>Updated {formatDate(reference.updated_at)}</span>
          <div className="flex items-center gap-1.5 text-indigo-400">
            <Eye className="h-3 w-3" />
            <span>View details</span>
          </div>
        </div>
      </Link>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Dashboard Page
// [REQ:P0-001] Reference Scenario Registry - Main dashboard view
// ─────────────────────────────────────────────────────────────────────────────

export default function Dashboard() {
  // Centralized health status management
  const { isHealthy, healthStatus, refetch: refetchHealth } = useHealthStatus();

  const referencesQuery = useQuery({
    queryKey: ["references"],
    queryFn: () => fetchReferences(),
    enabled: isHealthy // Only fetch references when API is healthy
  });

  const isLoading = !isHealthy || referencesQuery.isLoading;
  const hasError = !isHealthy || referencesQuery.isError;
  const references = referencesQuery.data ?? [];

  const handleRefresh = () => {
    refetchHealth();
    referencesQuery.refetch();
  };

  // Register keyboard shortcuts for common actions
  // Press "r" to refresh data
  useNavigationShortcuts({
    onRefresh: handleRefresh
  });

  return (
    <Layout
      title="Development Toolchain Validator"
      healthStatus={healthStatus}
      isLoading={isLoading}
      onRefresh={handleRefresh}
      testIdPrefix="dashboard"
    >
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
            Make sure the scenario is running:
          </p>
          <div className="mt-4 max-w-xl mx-auto">
            <CopyableCode
              code="vrooli scenario start development-toolchain-validator"
              size="sm"
              testId="dashboard-error-command"
            />
          </div>
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
          <div className="mt-6 max-w-lg mx-auto">
            <CopyableCode
              code='dtv reference create --slug reference-react-vite --name "React Vite Reference" --template react-vite --path /path/to/scenario'
              size="sm"
              testId="dashboard-empty-command"
            />
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
    </Layout>
  );
}
