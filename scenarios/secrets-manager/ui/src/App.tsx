import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, CheckCircle2, Database, RefreshCcw, ShieldAlert, ShieldCheck, TerminalSquare } from "lucide-react";
import { Button } from "./components/ui/button";
import {
  fetchHealth,
  fetchVaultStatus,
  fetchCompliance,
  fetchVulnerabilities,
  type VaultResourceStatus,
  type SecurityVulnerability
} from "./lib/api";

const severityAccent: Record<string, string> = {
  critical: "border-red-500/60 bg-red-500/10 text-red-200",
  high: "border-amber-400/60 bg-amber-400/10 text-amber-100",
  medium: "border-yellow-300/40 bg-yellow-300/5 text-yellow-100",
  low: "border-emerald-400/50 bg-emerald-400/5 text-emerald-100"
};

type Intent = "good" | "warn" | "danger" | "info";

const intentAccent: Record<Intent, string> = {
  good: "border-emerald-500/30 bg-emerald-500/5 text-emerald-100",
  warn: "border-amber-400/40 bg-amber-400/10 text-amber-100",
  danger: "border-red-500/40 bg-red-500/10 text-red-100",
  info: "border-cyan-400/40 bg-cyan-400/10 text-cyan-100"
};

const formatTimestamp = (value?: string) => {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

const percentage = (found: number, total: number) => {
  if (!total) return 0;
  return Math.round((found / total) * 100);
};

const StatusTile = ({
  icon: Icon,
  label,
  value,
  meta,
  intent
}: {
  icon: typeof Activity;
  label: string;
  value: string;
  meta?: string;
  intent: Intent;
}) => (
  <div className={`rounded-2xl border px-4 py-5 shadow-lg shadow-black/20 ${intentAccent[intent]}`}>
    <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-white/70">
      <span>{label}</span>
      <Icon className="h-4 w-4" />
    </div>
    <p className="mt-3 text-2xl font-semibold">{value}</p>
    {meta ? <p className="text-sm text-white/70">{meta}</p> : null}
  </div>
);

const StatCard = ({ label, value, description }: { label: string; value: string; description?: string }) => (
  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
    <p className="text-xs uppercase tracking-[0.2em] text-white/60">{label}</p>
    <p className="mt-2 text-3xl font-semibold">{value}</p>
    {description ? <p className="text-sm text-white/60">{description}</p> : null}
  </div>
);

const SeverityBadge = ({ severity }: { severity: string }) => {
  const key = severityAccent[severity]
    ? severity
    : severity === "healthy"
    ? "low"
    : severity === "degraded"
    ? "high"
    : severity === "invalid"
    ? "medium"
    : severity;

  return (
    <span
      className={`rounded-full border px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ${
        severityAccent[key] ?? severityAccent.low
      }`}
    >
      {severity}
    </span>
  );
};

const SecretsRow = ({ status }: { status: VaultResourceStatus }) => {
  const configuredPercent = percentage(status.secrets_found, status.secrets_total);
  const tone: Intent =
    status.health_status === "healthy"
      ? "good"
      : status.health_status === "critical"
      ? "danger"
      : status.health_status === "degraded"
      ? "warn"
      : "info";
  const progressColor =
    tone === "good"
      ? "bg-emerald-400"
      : tone === "warn"
      ? "bg-amber-400"
      : tone === "danger"
      ? "bg-red-500"
      : "bg-cyan-400";

  return (
    <div className="flex items-center justify-between rounded-xl border border-white/5 bg-white/5 px-4 py-3">
      <div>
        <p className="text-sm font-semibold text-white">{status.resource_name}</p>
        <p className="text-xs text-white/70">{status.secrets_found}/{status.secrets_total} configured</p>
      </div>
      <div className="flex w-1/2 flex-col items-end gap-2">
        <div className="h-2 w-full rounded-full bg-white/10">
          <div className={`h-2 rounded-full ${progressColor}`} style={{ width: `${configuredPercent}%` }} />
        </div>
        <div className="flex items-center gap-2 text-sm">
          <span>{configuredPercent}% ready</span>
          <SeverityBadge severity={status.health_status} />
        </div>
      </div>
    </div>
  );
};

const VulnerabilityItem = ({ vuln }: { vuln: SecurityVulnerability }) => (
  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-semibold">{vuln.title}</p>
        <p className="text-xs text-white/60">{vuln.component_name} · {vuln.file_path}:{vuln.line_number}</p>
      </div>
      <SeverityBadge severity={vuln.severity} />
    </div>
    <p className="mt-2 text-sm text-white/70">{vuln.description}</p>
    <p className="mt-2 text-xs text-white/50">Recommendation: {vuln.recommendation}</p>
  </div>
);

export default function App() {
  const [componentType, setComponentType] = useState("");
  const [componentFilter, setComponentFilter] = useState("");
  const [severityFilter, setSeverityFilter] = useState("");

  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 60000
  });

  const vaultQuery = useQuery({
    queryKey: ["vault-status"],
    queryFn: () => fetchVaultStatus(),
    refetchInterval: 60000
  });

  const complianceQuery = useQuery({
    queryKey: ["compliance"],
    queryFn: fetchCompliance,
    refetchInterval: 60000
  });

  const vulnerabilityQuery = useQuery({
    queryKey: ["vulnerabilities", componentType, componentFilter, severityFilter],
    queryFn: () =>
      fetchVulnerabilities({
        componentType: componentType || undefined,
        component: componentFilter || undefined,
        severity: severityFilter || undefined
      }),
    refetchInterval: 90000
  });

  const isRefreshing =
    healthQuery.isFetching || vaultQuery.isFetching || complianceQuery.isFetching || vulnerabilityQuery.isFetching;

  const componentOptions = useMemo(() => {
    const set = new Set<string>();
    vaultQuery.data?.resource_statuses?.forEach((status) => set.add(status.resource_name));
    vulnerabilityQuery.data?.vulnerabilities?.forEach((vuln) => set.add(vuln.component_name));
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [vaultQuery.data, vulnerabilityQuery.data]);

  const vulnerabilitySummary = complianceQuery.data?.vulnerability_summary ?? {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0
  };

  const missingSecrets = vaultQuery.data?.missing_secrets ?? [];
  const resourceStatuses = vaultQuery.data?.resource_statuses ?? [];
  const vulnerabilities = vulnerabilityQuery.data?.vulnerabilities ?? [];

  const refreshAll = () => {
    healthQuery.refetch();
    vaultQuery.refetch();
    complianceQuery.refetch();
    vulnerabilityQuery.refetch();
  };

  return (
    <div className="relative min-h-screen bg-slate-950 text-slate-50">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(16,185,129,0.15),_transparent_50%),radial-gradient(circle_at_bottom,_rgba(59,130,246,0.15),_transparent_60%)]" />
      <main className="relative mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
        <header className="rounded-3xl border border-white/10 bg-black/30 p-6 shadow-2xl shadow-emerald-500/10">
          <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Security Vault Access Terminal</p>
          <div className="mt-3 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h1 className="text-3xl font-semibold tracking-tight text-white">Secrets Manager</h1>
              <p className="mt-2 max-w-2xl text-sm text-white/70">
                Discover, validate, and provision secrets across the Vrooli resource graph. Monitor vault readiness,
                identify critical vulnerabilities, and trigger remediation from one unified control room.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <div className="rounded-full border border-emerald-400/40 bg-emerald-400/10 px-4 py-2 text-xs uppercase tracking-[0.2em] text-emerald-100">
                {vulnerabilityQuery.data ? `${vulnerabilityQuery.data.total_count} Findings` : "Scanning"}
              </div>
              <Button variant="outline" size="sm" onClick={refreshAll} disabled={isRefreshing} className="gap-2">
                <RefreshCcw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
                Refresh data
              </Button>
            </div>
          </div>
        </header>

        <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <StatusTile
            icon={TerminalSquare}
            label="API Terminal"
            value={(healthQuery.data?.status ?? "unknown").toUpperCase()}
            meta={healthQuery.data?.service}
            intent={healthQuery.data?.status === "healthy" ? "good" : "warn"}
          />
          <StatusTile
            icon={Database}
            label="Database"
            value={(healthQuery.data?.dependencies?.database ?? "unknown").toUpperCase()}
            meta={`v${healthQuery.data?.version ?? "1.0"}`}
            intent={healthQuery.data?.dependencies?.database === "connected" ? "good" : "warn"}
          />
          <StatusTile
            icon={ShieldCheck}
            label="Vault Coverage"
            value={`${vaultQuery.data?.configured_resources ?? 0}/${vaultQuery.data?.total_resources ?? 0}`}
            meta={`${missingSecrets.length} missing`}
            intent={missingSecrets.length > 0 ? "warn" : "good"}
          />
          <StatusTile
            icon={Activity}
            label="Last Scan"
            value={formatTimestamp(complianceQuery.data?.last_updated ?? vaultQuery.data?.last_updated)}
            meta={`${vulnerabilityQuery.data?.scan_duration ?? 0} ms`}
            intent={vulnerabilityQuery.data?.risk_score && vulnerabilityQuery.data.risk_score > 60 ? "danger" : "info"}
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <div className="space-y-4 rounded-3xl border border-white/5 bg-white/5 p-6 lg:col-span-2">
            <div className="flex flex-wrap items-center gap-4">
              <StatCard label="Overall Score" value={`${complianceQuery.data?.overall_score ?? 0}%`} />
              <StatCard label="Configured Components" value={`${complianceQuery.data?.configured_components ?? 0}`} />
              <StatCard label="Security Score" value={`${complianceQuery.data?.remediation_progress?.security_score ?? 0}%`} />
              <StatCard label="Vault Health" value={`${complianceQuery.data?.vault_secrets_health ?? 0}%`} />
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Vulnerability Mix</p>
                <div className="mt-4 space-y-3">
                  {(["critical", "high", "medium", "low"] as const).map((level) => (
                    <div key={level} className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <SeverityBadge severity={level} />
                        <span className="capitalize text-white/80">{level}</span>
                      </div>
                      <span className="text-lg font-semibold">
                        {vulnerabilitySummary[level] ? vulnerabilitySummary[level].toString() : "0"}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
              <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Missing Secrets</p>
                {missingSecrets.length === 0 ? (
                  <div className="mt-4 flex items-center gap-3 rounded-2xl border border-emerald-500/40 bg-emerald-500/10 px-4 py-4 text-sm text-emerald-100">
                    <CheckCircle2 className="h-5 w-5" />
                    All required secrets are configured.
                  </div>
                ) : (
                  <div className="mt-3 space-y-3">
                    {missingSecrets.slice(0, 4).map((secret) => (
                      <div key={`${secret.resource_name}-${secret.secret_name}`} className="rounded-xl border border-amber-400/30 bg-amber-400/5 p-3">
                        <p className="text-sm font-semibold text-white">{secret.resource_name}</p>
                        <p className="text-xs text-white/70">{secret.secret_name}</p>
                        <p className="mt-1 text-xs text-amber-100">{secret.description}</p>
                      </div>
                    ))}
                    {missingSecrets.length > 4 ? (
                      <p className="text-xs text-white/60">+{missingSecrets.length - 4} more secrets need attention</p>
                    ) : null}
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="rounded-3xl border border-white/10 bg-black/30 p-6">
            <p className="text-xs uppercase tracking-[0.2em] text-white/60">Scan Inputs</p>
            <div className="mt-4 space-y-4">
              <div>
                <label htmlFor="componentType" className="text-xs text-white/60">
                  Component type
                </label>
                <select
                  id="componentType"
                  className="mt-2 w-full rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                  value={componentType}
                  onChange={(event) => setComponentType(event.target.value)}
                >
                  <option value="">All components</option>
                  <option value="resource">Resources</option>
                  <option value="scenario">Scenarios</option>
                </select>
              </div>
              <div>
                <label htmlFor="component" className="text-xs text-white/60">
                  Component
                </label>
                <select
                  id="component"
                  className="mt-2 w-full rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                  value={componentFilter}
                  onChange={(event) => setComponentFilter(event.target.value)}
                >
                  <option value="">All</option>
                  {componentOptions.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label htmlFor="severity" className="text-xs text-white/60">
                  Severity
                </label>
                <select
                  id="severity"
                  className="mt-2 w-full rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                  value={severityFilter}
                  onChange={(event) => setSeverityFilter(event.target.value)}
                >
                  <option value="">All levels</option>
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-white/70">
                <p>
                  Scan ID: <span className="font-mono text-white">{vulnerabilityQuery.data?.scan_id ?? "pending"}</span>
                </p>
                <p>Risk score: {vulnerabilityQuery.data?.risk_score ?? 0}</p>
                <p>Duration: {vulnerabilityQuery.data?.scan_duration ?? 0} ms</p>
              </div>
            </div>
          </div>
        </section>

        <section className="grid gap-6 lg:grid-cols-2">
          <div className="rounded-3xl border border-white/5 bg-white/5 p-6">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">Component Security Status</h2>
              <div className="flex items-center gap-2 text-xs text-white/60">
                <ShieldCheck className="h-4 w-4" />
                Live data from vault and file scans
              </div>
            </div>
            <div className="mt-4 space-y-3">
              {resourceStatuses.length === 0 && !vaultQuery.isLoading ? (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
                  No resource scan results available yet.
                </div>
              ) : null}
              {resourceStatuses.slice(0, 6).map((status) => (
                <SecretsRow key={status.resource_name} status={status} />
              ))}
            </div>
          </div>

          <div className="rounded-3xl border border-white/5 bg-white/5 p-6">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">Security Findings</h2>
              <div className="flex items-center gap-2 text-xs text-white/60">
                <ShieldAlert className="h-4 w-4" />
                Showing {vulnerabilities.length} results
              </div>
            </div>
            <div className="mt-4 space-y-3">
              {vulnerabilityQuery.isLoading ? (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
                  Scanning filesystem for vulnerabilities…
                </div>
              ) : null}
              {!vulnerabilityQuery.isLoading && vulnerabilities.length === 0 ? (
                <div className="rounded-2xl border border-emerald-500/40 bg-emerald-500/10 p-6 text-center text-sm text-emerald-100">
                  <ShieldCheck className="mx-auto mb-2 h-6 w-6" />
                  No vulnerabilities found for the selected filters.
                </div>
              ) : null}
              {vulnerabilities.slice(0, 6).map((vuln) => (
                <VulnerabilityItem key={vuln.id} vuln={vuln} />
              ))}
            </div>
          </div>
        </section>

        <footer className="pb-6 text-center text-xs text-white/40">
          Powered by the Vrooli lifecycle · API base: {import.meta.env.VITE_API_BASE_URL || "lifecycle-managed"}
        </footer>
      </main>
    </div>
  );
}
