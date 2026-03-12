import { Link, useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw, FolderCode, Tag, Link2, Terminal, Clock, Hash, XCircle } from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { CopyableCode } from "../components/ui/CopyableCode";
import { Layout } from "../components/Layout";
import { useNavigationShortcuts, useHealthStatus } from "../hooks";
import { fetchReferenceBySlug, fetchConnectionsByReference, type SkillConnection } from "../lib/api";
import { formatDate } from "../lib/utils";

// ─────────────────────────────────────────────────────────────────────────────
// Skill Connection Card
// [REQ:P0-003] Skill Connection Management - Connection display
// ─────────────────────────────────────────────────────────────────────────────

interface SkillConnectionCardProps {
  connection: SkillConnection;
}

function SkillConnectionCard({ connection }: SkillConnectionCardProps) {
  return (
    <div
      data-testid={`connection-card-${connection.skill_id}`}
      className="rounded-lg border border-white/10 bg-white/5 p-4 hover:border-white/20 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-indigo-500/20">
            <Link2 className="h-4 w-4 text-indigo-400" />
          </div>
          <div>
            <span className="font-medium text-slate-200">{connection.skill_id}</span>
            {connection.skill_version && (
              <span className="ml-2 text-xs text-slate-500 font-mono">v{connection.skill_version}</span>
            )}
          </div>
        </div>
      </div>
      <div className="mt-3 flex items-center gap-4 text-xs text-slate-500">
        <div className="flex items-center gap-1.5">
          <Clock className="h-3 w-3" />
          <span>Connected {formatDate(connection.connected_at)}</span>
        </div>
        {connection.skill_content_hash && (
          <div className="flex items-center gap-1.5 font-mono">
            <Hash className="h-3 w-3" />
            <span>{connection.skill_content_hash.slice(0, 12)}</span>
          </div>
        )}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Reference Detail Page
// [REQ:P0-001] Reference Scenario Registry - Detail view with connections
// ─────────────────────────────────────────────────────────────────────────────

export default function ReferenceDetail() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();

  // Centralized health status management
  const { isHealthy, healthStatus, refetch: refetchHealth } = useHealthStatus();

  const referenceQuery = useQuery({
    queryKey: ["reference", slug],
    queryFn: () => fetchReferenceBySlug(slug ?? ""),
    enabled: !!slug && isHealthy
  });

  const connectionsQuery = useQuery({
    queryKey: ["connections", referenceQuery.data?.id],
    queryFn: () => fetchConnectionsByReference(referenceQuery.data?.id ?? ""),
    enabled: !!referenceQuery.data?.id
  });

  const isLoading = !isHealthy || referenceQuery.isLoading;
  const hasError = !isHealthy || referenceQuery.isError;
  const reference = referenceQuery.data;
  const connections = connectionsQuery.data ?? [];

  const handleRefresh = () => {
    refetchHealth();
    referenceQuery.refetch();
    connectionsQuery.refetch();
  };

  const handleBack = () => {
    navigate("/");
  };

  // Register keyboard shortcuts for common actions
  // Press "r" to refresh, "Escape" or "h" to go back to dashboard
  useNavigationShortcuts({
    onRefresh: handleRefresh,
    onBack: handleBack,
    onHome: handleBack
  });

  return (
    <Layout
      title={reference?.name ?? slug ?? "Reference"}
      subtitle={reference?.slug}
      healthStatus={healthStatus}
      isLoading={isLoading}
      onRefresh={handleRefresh}
      testIdPrefix="reference-detail"
    >
      {/* Loading state */}
      {isLoading && (
        <div
          data-testid="reference-detail-loading"
          className="flex flex-col items-center justify-center py-16"
        >
          <RefreshCw className="h-8 w-8 text-slate-400 animate-spin mb-4" />
          <p className="text-slate-400">Loading reference...</p>
        </div>
      )}

      {/* Error state */}
      {hasError && !isLoading && (
        <div
          data-testid="reference-detail-error"
          className="rounded-xl border border-red-500/20 bg-red-500/10 p-6 text-center"
        >
          <XCircle className="h-8 w-8 text-red-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-red-300">
            {referenceQuery.isError ? "Reference not found" : "Unable to connect"}
          </h3>
          <p className="mt-2 text-sm text-slate-400">
            {referenceQuery.isError
              ? `No reference found with slug "${slug}"`
              : "Make sure the scenario is running"}
          </p>
          <div className="mt-4 flex items-center justify-center gap-3">
            <Button variant="outline" asChild>
              <Link to="/">Back to Dashboard</Link>
            </Button>
            <Button variant="outline" onClick={handleRefresh}>
              Try Again
            </Button>
          </div>
        </div>
      )}

      {/* Reference detail content */}
      {!isLoading && !hasError && reference && (
        <div className="space-y-8">
          {/* Reference info card */}
          <div
            data-testid={`reference-detail-info-${reference.slug}`}
            className="rounded-xl border border-white/10 bg-white/5 p-6"
          >
            <div className="flex items-start justify-between mb-6">
              <div>
                <h2 className="text-2xl font-semibold">{reference.name}</h2>
                <p className="mt-1 text-slate-400 font-mono">{reference.slug}</p>
              </div>
              <Badge variant="primary" size="lg">
                <Tag className="h-4 w-4" />
                {reference.template}
              </Badge>
            </div>

            {reference.description && (
              <p className="text-slate-300 mb-6">{reference.description}</p>
            )}

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="rounded-lg bg-white/3 border border-white/5 p-4">
                <div className="flex items-center gap-2 text-xs text-slate-500 uppercase tracking-wide mb-2">
                  <FolderCode className="h-3 w-3" />
                  Path
                </div>
                <code className="text-sm text-slate-300 font-mono break-all">{reference.path}</code>
              </div>
              <div className="rounded-lg bg-white/3 border border-white/5 p-4">
                <div className="flex items-center gap-2 text-xs text-slate-500 uppercase tracking-wide mb-2">
                  <Hash className="h-3 w-3" />
                  ID
                </div>
                <code className="text-sm text-slate-300 font-mono break-all">{reference.id}</code>
              </div>
            </div>

            <div className="mt-4 flex items-center justify-between text-sm text-slate-500">
              <span>Created: {formatDate(reference.created_at, { includeTime: true })}</span>
              <span>Updated: {formatDate(reference.updated_at, { includeTime: true })}</span>
            </div>
          </div>

          {/* Connected Skills section */}
          <div data-testid={`reference-detail-connections-${reference.slug}`}>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <h3 className="text-lg font-medium">Connected Skills</h3>
                <Badge variant="success">
                  <Link2 className="h-3 w-3" />
                  {connections.length}
                </Badge>
              </div>
              {connectionsQuery.isLoading && (
                <RefreshCw className="h-4 w-4 text-slate-400 animate-spin" />
              )}
            </div>

            {connectionsQuery.isLoading && (
              <div
                data-testid={`reference-connections-loading-${reference.slug}`}
                className="rounded-lg border border-white/10 bg-white/5 p-8 text-center"
              >
                <RefreshCw className="h-6 w-6 text-slate-400 animate-spin mx-auto mb-2" />
                <p className="text-sm text-slate-400">Loading connections...</p>
              </div>
            )}

            {!connectionsQuery.isLoading && connections.length === 0 && (
              <div
                data-testid={`reference-no-connections-${reference.slug}`}
                className="rounded-lg border border-white/10 bg-white/5 p-8 text-center"
              >
                <Link2 className="h-10 w-10 text-slate-500 mx-auto mb-3" />
                <h4 className="font-medium text-slate-300">No skills connected</h4>
                <p className="mt-1 text-sm text-slate-400 max-w-md mx-auto">
                  Connect steer skills to this reference to validate their behavior against a known-good implementation.
                </p>
                <div className="mt-4 max-w-lg mx-auto">
                  <CopyableCode
                    code={`dtv connection connect --reference-id ${reference.id.slice(0, 8)}... --skill-id <skill-id>`}
                    size="sm"
                    testId="reference-connect-command"
                  />
                </div>
              </div>
            )}

            {!connectionsQuery.isLoading && connections.length > 0 && (
              <div className="grid gap-3 sm:grid-cols-2">
                {connections.map((conn) => (
                  <SkillConnectionCard key={conn.id} connection={conn} />
                ))}
              </div>
            )}
          </div>

          {/* CLI Quick Reference */}
          <div className="rounded-lg border border-white/10 bg-slate-800/30 p-4">
            <div className="flex items-center gap-2 mb-4">
              <Terminal className="h-4 w-4 text-slate-400" />
              <span className="text-sm font-medium text-slate-300">CLI Quick Reference</span>
            </div>
            <div className="space-y-3">
              <CopyableCode
                code={`dtv reference update ${reference.slug} --name "New Name"`}
                label="Update"
                size="sm"
                testId="cli-update-command"
              />
              <CopyableCode
                code={`dtv connection connect --reference-id ${reference.id.slice(0, 8)}... --skill-id <id>`}
                label="Connect"
                size="sm"
                testId="cli-connect-command"
              />
              <CopyableCode
                code={`dtv validate --reference ${reference.slug}`}
                label="Validate"
                size="sm"
                testId="cli-validate-command"
              />
            </div>
          </div>
        </div>
      )}
    </Layout>
  );
}
