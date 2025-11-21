import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  Activity,
  CheckCircle2,
  Compass,
  Database,
  Layers,
  Map,
  RefreshCcw,
  ShieldAlert,
  ShieldCheck,
  Target,
  TerminalSquare
} from "lucide-react";
import { Button } from "./components/ui/button";
import {
  fetchHealth,
  fetchVaultStatus,
  fetchCompliance,
  fetchVulnerabilities,
  fetchOrientationSummary,
  generateDeploymentManifest,
  fetchResourceDetail,
  updateResourceSecret,
  updateSecretStrategy,
  updateVulnerabilityStatus,
  type VaultResourceStatus,
  type SecurityVulnerability,
  type JourneyCard as JourneyCardData,
  type OrientationSummary,
  type DeploymentManifestRequest,
  type DeploymentManifestResponse,
  type ResourceDetail,
  type ResourceSecretDetail,
  type UpdateResourceSecretPayload,
  type UpdateSecretStrategyPayload
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

const classificationTone: Record<string, string> = {
  infrastructure: "bg-sky-400/15 text-sky-100 border-sky-500/40",
  service: "bg-purple-400/10 text-purple-100 border-purple-500/40",
  user: "bg-amber-400/10 text-amber-100 border-amber-500/40"
};

type JourneyId = "configure-secrets" | "fix-vulnerabilities" | "prep-deployment";

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

const Skeleton = ({ className = "", variant = "default" }: { className?: string; variant?: "default" | "text" | "circular" }) => {
  const baseClasses = "animate-pulse bg-white/10";
  const variantClasses = variant === "circular" ? "rounded-full" : variant === "text" ? "rounded" : "rounded-2xl";
  return <div className={`${baseClasses} ${variantClasses} ${className}`} />;
};

const LoadingStatCard = () => (
  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
    <Skeleton className="h-3 w-20" variant="text" />
    <Skeleton className="mt-2 h-9 w-16" variant="text" />
    <Skeleton className="mt-1 h-4 w-32" variant="text" />
  </div>
);

const LoadingStatusTile = () => (
  <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-5">
    <div className="flex items-center justify-between">
      <Skeleton className="h-3 w-24" variant="text" />
      <Skeleton className="h-4 w-4" variant="circular" />
    </div>
    <Skeleton className="mt-3 h-8 w-32" variant="text" />
    <Skeleton className="mt-1 h-4 w-24" variant="text" />
  </div>
);

const LoadingResourceCard = () => (
  <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
    <div className="flex items-center justify-between">
      <div className="flex-1">
        <Skeleton className="h-6 w-32" variant="text" />
        <Skeleton className="mt-1 h-4 w-40" variant="text" />
      </div>
      <Skeleton className="h-8 w-20" variant="text" />
    </div>
    <div className="mt-3 space-y-2">
      <Skeleton className="h-16 w-full" />
      <Skeleton className="h-16 w-full" />
    </div>
  </div>
);

const LoadingSecretsRow = () => (
  <div className="flex items-center justify-between rounded-xl border border-white/5 bg-white/5 px-4 py-3">
    <div className="flex-1">
      <Skeleton className="h-4 w-32" variant="text" />
      <Skeleton className="mt-1 h-3 w-24" variant="text" />
    </div>
    <div className="flex w-1/2 flex-col items-end gap-2">
      <Skeleton className="h-2 w-full" />
      <Skeleton className="h-5 w-24" variant="text" />
    </div>
  </div>
);

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
  const [activeJourney, setActiveJourney] = useState<JourneyId | null>(null);
  const [journeyStep, setJourneyStep] = useState(0);
  const [deploymentScenario, setDeploymentScenario] = useState("picker-wheel");
  const [deploymentTier, setDeploymentTier] = useState("tier-2-desktop");
  const [resourceInput, setResourceInput] = useState("");
  const [activeResource, setActiveResource] = useState<string | null>(null);
  const [selectedSecretKey, setSelectedSecretKey] = useState<string | null>(null);
  const [strategyTier, setStrategyTier] = useState("tier-2-desktop");
  const [strategyHandling, setStrategyHandling] = useState("prompt");
  const [strategyPrompt, setStrategyPrompt] = useState("Desktop pairing prompt");
  const [strategyDescription, setStrategyDescription] = useState("Prompt operator for credentials during install");

  const openResourcePanel = useCallback((resourceName?: string) => {
    if (!resourceName) return;
    setActiveResource(resourceName);
    setSelectedSecretKey(null);
  }, []);

  const closeResourcePanel = useCallback(() => {
    setActiveResource(null);
    setSelectedSecretKey(null);
  }, []);

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

  const orientationQuery = useQuery({
    queryKey: ["orientation-summary"],
    queryFn: fetchOrientationSummary,
    refetchInterval: 120000
  });

  const isRefreshing =
    healthQuery.isFetching || vaultQuery.isFetching || complianceQuery.isFetching || vulnerabilityQuery.isFetching;

  const isInitialLoading =
    healthQuery.isLoading || vaultQuery.isLoading || complianceQuery.isLoading || vulnerabilityQuery.isLoading || orientationQuery.isLoading;

  const manifestMutation = useMutation<DeploymentManifestResponse, Error, DeploymentManifestRequest>({
    mutationFn: (payload) => generateDeploymentManifest(payload)
  });

  const resourceDetailQuery = useQuery<ResourceDetail>({
    queryKey: ["resource-detail", activeResource],
    queryFn: () => fetchResourceDetail(activeResource as string),
    enabled: Boolean(activeResource),
    refetchOnWindowFocus: false
  });

  const updateSecretMutation = useMutation({
    mutationFn: ({ resource, secret, payload }: { resource: string; secret: string; payload: UpdateResourceSecretPayload }) =>
      updateResourceSecret(resource, secret, payload),
    onSuccess: () => {
      resourceDetailQuery.refetch();
    }
  });

  const updateStrategyMutation = useMutation({
    mutationFn: ({ resource, secret, payload }: { resource: string; secret: string; payload: UpdateSecretStrategyPayload }) =>
      updateSecretStrategy(resource, secret, payload),
    onSuccess: () => {
      resourceDetailQuery.refetch();
    }
  });

  const updateVulnerabilityStatusMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { status: string; assigned_to?: string } }) =>
      updateVulnerabilityStatus(id, payload),
    onSuccess: () => {
      resourceDetailQuery.refetch();
    }
  });

  useEffect(() => {
    setJourneyStep(0);
  }, [activeJourney]);

  useEffect(() => {
    if (!resourceDetailQuery.data || !resourceDetailQuery.data.secrets.length) {
      setSelectedSecretKey(null);
      return;
    }
    const exists = resourceDetailQuery.data.secrets.some((secret) => secret.secret_key === selectedSecretKey);
    if (!exists) {
      setSelectedSecretKey(resourceDetailQuery.data.secrets[0].secret_key);
    }
  }, [resourceDetailQuery.data, selectedSecretKey]);

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
    orientationQuery.refetch();
  };

  const orientationData = orientationQuery.data;
  const heroStats = orientationData?.hero_stats;
  const journeyCards = orientationData?.journeys ?? [];
  const tierReadiness = orientationData?.tier_readiness ?? [];
  const resourceInsights = orientationData?.resource_insights ?? [];

  const topResourceNeedingAttention = useMemo(() => {
    if (resourceInsights.length > 0) {
      return resourceInsights[0].resource_name;
    }
    return resourceStatuses.find((status) => status.secrets_missing > 0)?.resource_name;
  }, [resourceInsights, resourceStatuses]);

  const handleJourneySelect = (journey: JourneyId) => {
    setActiveJourney(journey);
  };

  const parsedResources = useMemo(() => {
    if (!resourceInput.trim()) return undefined;
    return resourceInput
      .split(/[\n,]+/)
      .map((value) => value.trim())
      .filter(Boolean);
  }, [resourceInput]);

  const selectedSecret = useMemo<ResourceSecretDetail | undefined>(() => {
    if (!resourceDetailQuery.data || resourceDetailQuery.data.secrets.length === 0) {
      return undefined;
    }
    if (!selectedSecretKey) {
      return resourceDetailQuery.data.secrets[0];
    }
    return resourceDetailQuery.data.secrets.find((secret) => secret.secret_key === selectedSecretKey) ??
      resourceDetailQuery.data.secrets[0];
  }, [resourceDetailQuery.data, selectedSecretKey]);

  const handleSecretUpdate = useCallback(
    (secretKey: string, payload: UpdateResourceSecretPayload) => {
      if (!activeResource) return;
      updateSecretMutation.mutate({ resource: activeResource, secret: secretKey, payload });
    },
    [activeResource, updateSecretMutation]
  );

  const handleStrategyApply = useCallback(() => {
    if (!activeResource || !selectedSecret) return;
    const payload: UpdateSecretStrategyPayload = {
      tier: strategyTier,
      handling_strategy: strategyHandling,
      requires_user_input: strategyHandling === "prompt",
      prompt_label: strategyPrompt,
      prompt_description: strategyDescription
    };
    updateStrategyMutation.mutate({ resource: activeResource, secret: selectedSecret.secret_key, payload });
  }, [activeResource, selectedSecret, strategyTier, strategyHandling, strategyPrompt, strategyDescription, updateStrategyMutation]);

  const handleVulnerabilityStatus = useCallback(
    (id: string, status: string) => {
      updateVulnerabilityStatusMutation.mutate({ id, payload: { status } });
    },
    [updateVulnerabilityStatusMutation]
  );

  const handleManifestRequest = () => {
    manifestMutation.mutate({
      scenario: deploymentScenario,
      tier: deploymentTier,
      resources: parsedResources,
      include_optional: false
    });
  };

  const journeySteps = useMemo(() => {
    if (!activeJourney) return [];
    const steps = [] as Array<{ title: string; description: string; content?: React.ReactNode }>;
    if (activeJourney === "configure-secrets") {
      steps.push({
        title: "Audit coverage",
        description: "Review vault readiness and identify resources with missing requirements.",
        content: (
          <ul className="mt-2 space-y-1 text-sm text-white/80">
            <li>Configured resources: {heroStats?.vault_configured ?? 0}</li>
            <li>Total resources tracked: {heroStats?.vault_total ?? 0}</li>
            <li>Missing secrets: {heroStats?.missing_secrets ?? 0}</li>
          </ul>
        )
      });
      steps.push({
        title: "Prioritize fixes",
        description: "Work the highest-risk resources first using the insights table below.",
        content: (
          <div className="space-y-3">
            <p className="text-sm text-white/70">
              Start with degraded resources in the Workbench section. Each resource card exposes inline actions to
              manage its secrets and deployment strategies.
            </p>
            <Button size="sm" onClick={() => openResourcePanel(topResourceNeedingAttention)} disabled={!topResourceNeedingAttention}>
              {topResourceNeedingAttention ? `Open ${topResourceNeedingAttention}` : "No targets"}
            </Button>
          </div>
        )
      });
      steps.push({
        title: "Provision",
        description: "Use the CLI/API provisioning hooks or jump into secrets-manager CLI to apply changes.",
        content: (
          <div className="space-y-2 text-sm text-white/70">
            <p>Recommended command:</p>
            <code className="block rounded-xl border border-white/20 bg-black/40 px-3 py-2 font-mono text-xs text-emerald-200">
              secrets-manager plan --scenario picker-wheel --tier tier-2-desktop | jq
            </code>
          </div>
        )
      });
    }
    if (activeJourney === "fix-vulnerabilities") {
      steps.push({
        title: "Review findings",
        description: "Filter findings by severity and assign owners before remediation.",
        content: (
          <p className="text-sm text-white/70">Risk score: {orientationData?.hero_stats.risk_score ?? 0}</p>
        )
      });
      steps.push({
        title: "Plan remediation",
        description: "Trigger claude-code fixer runs for repetitive issues or create app-issue-tracker tickets.",
        content: (
          <p className="text-sm text-white/70">
            Use the vulnerability list below to copy file references and recommendations. When fixes are ready, run the
            automated tests to confirm stability.
          </p>
        )
      });
      steps.push({
        title: "Verify",
        description: "Re-run scans and confirm compliance deltas.",
        content: (
          <Button variant="outline" size="sm" onClick={() => vulnerabilityQuery.refetch()}>
            Re-run scan
          </Button>
        )
      });
    }
    if (activeJourney === "prep-deployment") {
      steps.push({
        title: "Assess tier readiness",
        description: "Confirm which tiers have full secret strategies.",
        content: (
          <ul className="mt-2 space-y-1 text-sm text-white/80">
            {tierReadiness.slice(0, 3).map((tier) => (
              <li key={tier.tier}>
                {tier.label}: {tier.ready_percent}% ({tier.strategized}/{tier.total})
              </li>
            ))}
          </ul>
        )
      });
      steps.push({
        title: "Generate manifest",
        description: "Select scenario, tier, and optional resource filters to emit a bundle manifest.",
        content: (
          <div className="space-y-3 text-sm text-white/80">
            <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
              Scenario
              <input
                type="text"
                value={deploymentScenario}
                onChange={(event) => setDeploymentScenario(event.target.value)}
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
              Tier
              <select
                value={deploymentTier}
                onChange={(event) => setDeploymentTier(event.target.value)}
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              >
                {tierReadiness.map((tier) => (
                  <option key={tier.tier} value={tier.tier}>
                    {tier.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
              Resources (optional)
              <textarea
                value={resourceInput}
                onChange={(event) => setResourceInput(event.target.value)}
                placeholder="postgres, vault"
                rows={2}
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </label>
            <Button variant="secondary" size="sm" className="w-full" onClick={handleManifestRequest} disabled={manifestMutation.isLoading}>
              {manifestMutation.isLoading ? "Generating…" : "Generate manifest"}
            </Button>
            {manifestMutation.isError ? (
              <p className="text-xs text-red-300">{manifestMutation.error.message}</p>
            ) : null}
          </div>
        )
      });
      steps.push({
        title: "Review manifest",
        description: "Inspect the resulting manifest and hand it off to deployment-manager or scenario-to-*.",
        content: manifestMutation.data ? (
          <div className="space-y-2 text-xs text-white/70">
            <p>
              Secrets covered: {manifestMutation.data.summary.strategized_secrets}/{manifestMutation.data.summary.total_secrets}
            </p>
            <p>Blocking: {manifestMutation.data.summary.requires_action}</p>
            <div className="rounded-xl border border-white/10 bg-black/40 p-3 font-mono">
              <pre className="overflow-x-auto text-[10px] text-emerald-100">
{JSON.stringify(manifestMutation.data.summary, null, 2)}
              </pre>
            </div>
          </div>
        ) : (
          <p className="text-sm text-white/60">No manifest generated yet.</p>
        )
      });
    }
    return steps;
  }, [
    activeJourney,
    heroStats,
    orientationData,
    tierReadiness,
    deploymentScenario,
    deploymentTier,
    resourceInput,
    manifestMutation.data,
    manifestMutation.isLoading,
    manifestMutation.isError,
    openResourcePanel,
    topResourceNeedingAttention
  ]);

  const activeJourneyCard: JourneyCardData | undefined = journeyCards.find((card) => card.id === activeJourney);
  const activeStep = journeySteps[journeyStep];

  return (
    <div className="relative min-h-screen bg-slate-950 text-slate-50">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(16,185,129,0.15),_transparent_50%),radial-gradient(circle_at_bottom,_rgba(59,130,246,0.15),_transparent_60%)]" />

      {/* Initial loading overlay */}
      {isInitialLoading && !healthQuery.data && !vaultQuery.data && !complianceQuery.data && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/95 backdrop-blur-sm">
          <div className="rounded-3xl border border-emerald-400/40 bg-emerald-400/5 px-8 py-6 shadow-2xl shadow-emerald-500/20">
            <div className="flex items-center gap-4">
              <RefreshCcw className="h-8 w-8 animate-spin text-emerald-400" />
              <div>
                <p className="text-lg font-semibold text-white">Loading Security Dashboard</p>
                <p className="text-sm text-white/60">Fetching vault status, compliance data, and vulnerability scans...</p>
              </div>
            </div>
          </div>
        </div>
      )}

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
              {isInitialLoading ? (
                <div className="flex items-center gap-2 rounded-full border border-cyan-400/40 bg-cyan-400/10 px-4 py-2 text-xs uppercase tracking-[0.2em] text-cyan-100">
                  <RefreshCcw className="h-3 w-3 animate-spin" />
                  Loading data...
                </div>
              ) : (
                <div className="rounded-full border border-emerald-400/40 bg-emerald-400/10 px-4 py-2 text-xs uppercase tracking-[0.2em] text-emerald-100">
                  {vulnerabilityQuery.data ? `${vulnerabilityQuery.data.total_count} Findings` : "Ready"}
                </div>
              )}
              <Button variant="outline" size="sm" onClick={refreshAll} disabled={isRefreshing} className="gap-2">
                <RefreshCcw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
                Refresh data
              </Button>
            </div>
          </div>
        </header>

        <section className="grid gap-6 lg:grid-cols-[2fr,1fr]">
          <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Orientation Hub</p>
                <h2 className="mt-1 text-2xl font-semibold text-white">Readiness Snapshot</h2>
              </div>
              <div className="rounded-full border border-white/10 px-3 py-1 text-xs text-white/70">
                {orientationQuery.isLoading ? (
                  <Skeleton className="h-3 w-16" variant="text" />
                ) : (
                  <>Updated {orientationData ? new Date(orientationData.updated_at).toLocaleTimeString() : "—"}</>
                )}
              </div>
            </div>
            {orientationQuery.isLoading ? (
              <div className="mt-4 grid gap-4 md:grid-cols-3">
                <LoadingStatCard />
                <LoadingStatCard />
                <LoadingStatCard />
              </div>
            ) : (
              <>
                <div className="mt-4 grid gap-4 md:grid-cols-3">
                  <StatCard label="Overall" value={`${heroStats?.overall_score ?? 0}%`} description={heroStats?.readiness_label} />
                  <StatCard label="Risk" value={`${heroStats?.risk_score ?? 0}`} description="Security score" />
                  <StatCard
                    label="Confidence"
                    value={`${Math.round((heroStats?.confidence ?? 0) * 100)}%`}
                    description="Best tier coverage"
                  />
                </div>
                <div className="mt-6 flex flex-wrap gap-3">
                  {(orientationData?.vulnerability_insights ?? []).map((highlight) => (
                    <span
                      key={highlight.severity}
                      className="rounded-full border border-white/10 bg-black/30 px-3 py-1 text-xs uppercase tracking-[0.2em] text-white/70"
                    >
                      {highlight.message}
                    </span>
                  ))}
                </div>
              </>
            )}
          </div>
          <div className="space-y-4">
            <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold uppercase tracking-[0.3em] text-white/70">Journeys</h3>
                <Compass className="h-4 w-4 text-white/60" />
              </div>
              <div className="mt-3 grid gap-3">
                {journeyCards.map((card) => (
                  <button
                    key={card.id}
                    onClick={() => handleJourneySelect(card.id as JourneyId)}
                    className={`w-full rounded-2xl border px-4 py-3 text-left transition hover:border-white/40 ${
                      activeJourney === card.id ? "border-emerald-400 bg-emerald-500/10" : "border-white/10 bg-black/30"
                    }`}
                  >
                    <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-white/60">
                      <span>{card.badge}</span>
                      <span>{card.primers[0]}</span>
                    </div>
                    <p className="mt-1 text-base font-semibold text-white">{card.title}</p>
                    <p className="text-sm text-white/60">{card.description}</p>
                  </button>
                ))}
              </div>
            </div>
            <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
              <div className="flex items-center justify-between text-xs uppercase tracking-[0.3em] text-white/60">
                <span>Guided Flow</span>
                <Map className="h-4 w-4 text-white/60" />
              </div>
              {!activeJourney || journeySteps.length === 0 ? (
                <p className="mt-3 text-sm text-white/60">Select a journey to start a guided experience.</p>
              ) : (
                <div className="mt-3 space-y-3">
                  <p className="text-xs text-white/50">
                    Step {journeyStep + 1} of {journeySteps.length} — {activeJourneyCard?.title}
                  </p>
                  <h4 className="text-lg font-semibold text-white">{activeStep?.title}</h4>
                  <p className="text-sm text-white/70">{activeStep?.description}</p>
                  {activeStep?.content}
                  <div className="flex items-center justify-between pt-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setActiveJourney(null)}
                      className="text-xs uppercase tracking-[0.2em]"
                    >
                      Exit
                    </Button>
                    <div className="space-x-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setJourneyStep((value) => Math.max(0, value - 1))}
                        disabled={journeyStep === 0}
                      >
                        Back
                      </Button>
                      <Button
                        size="sm"
                        onClick={() => setJourneyStep((value) => Math.min(journeySteps.length - 1, value + 1))}
                        disabled={journeyStep >= journeySteps.length - 1}
                      >
                        Next
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="grid gap-4 md:grid-cols-3">
          {orientationQuery.isLoading ? (
            <>
              <LoadingStatCard />
              <LoadingStatCard />
              <LoadingStatCard />
            </>
          ) : (
            tierReadiness.map((tier) => (
              <div key={tier.tier} className="rounded-3xl border border-white/10 bg-white/5 p-4">
                <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-white/60">
                  <span>{tier.label}</span>
                  <Layers className="h-4 w-4 text-white/60" />
                </div>
                <p className="mt-2 text-3xl font-semibold text-white">{tier.ready_percent}%</p>
                <p className="text-sm text-white/60">
                  {tier.strategized}/{tier.total} required secrets covered
                </p>
                {tier.blocking_secret_sample.length ? (
                  <p className="mt-2 text-xs text-white/50">Blockers: {tier.blocking_secret_sample.join(", ")}</p>
                ) : null}
              </div>
            ))
          )}
        </section>

        <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-white/60">Resource Workbench</p>
              <h2 className="mt-1 text-2xl font-semibold text-white">Per-Resource Control</h2>
            </div>
            <Target className="h-5 w-5 text-white/60" />
          </div>
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            {orientationQuery.isLoading ? (
              <>
                <LoadingResourceCard />
                <LoadingResourceCard />
                <LoadingResourceCard />
                <LoadingResourceCard />
              </>
            ) : resourceInsights.length === 0 ? (
              <div className="col-span-2 rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
                No resource insights available. API data is still loading or no resources are configured.
              </div>
            ) : (
              resourceInsights.map((resource) => (
                <div key={resource.resource_name} className="rounded-2xl border border-white/10 bg-black/30 p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-lg font-semibold text-white">{resource.resource_name}</p>
                      <p className="text-xs text-white/60">
                        {resource.valid_secrets}/{resource.total_secrets} valid · Missing {resource.missing_secrets}
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-[10px] uppercase tracking-[0.2em]"
                      onClick={() => openResourcePanel(resource.resource_name)}
                    >
                      Manage
                    </Button>
                  </div>
                  <div className="mt-3 space-y-2">
                    {resource.secrets.map((secret) => (
                      <div key={secret.secret_key} className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white/80">
                        <div className="flex items-center justify-between">
                          <span className="font-mono text-xs text-white">{secret.secret_key}</span>
                          <span
                            className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.2em] ${
                              classificationTone[secret.classification] ?? "border-white/20"
                            }`}
                          >
                            {secret.classification}
                          </span>
                        </div>
                        <p className="text-xs text-white/50">{secret.secret_type}</p>
                        {Object.keys(secret.tier_strategies || {}).length ? (
                          <p className="text-[10px] text-white/50">
                            Strategies: {Object.entries(secret.tier_strategies || {})
                              .map(([tier, value]) => `${tier}:${value}`)
                              .join(" · ")}
                          </p>
                        ) : (
                          <p className="text-[10px] text-amber-300">No tier strategies defined</p>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {healthQuery.isLoading ? (
            <LoadingStatusTile />
          ) : (
            <StatusTile
              icon={TerminalSquare}
              label="API Terminal"
              value={String(healthQuery.data?.status ?? "unknown").toUpperCase()}
              meta={healthQuery.data?.service}
              intent={healthQuery.data?.status === "healthy" ? "good" : "warn"}
            />
          )}
          {healthQuery.isLoading ? (
            <LoadingStatusTile />
          ) : (
            <StatusTile
              icon={Database}
              label="Database"
              value={
                healthQuery.data?.dependencies?.database?.connected
                  ? "CONNECTED"
                  : "DISCONNECTED"
              }
              meta={`v${healthQuery.data?.version ?? "1.0"}`}
              intent={
                healthQuery.data?.dependencies?.database?.connected ? "good" : "warn"
              }
            />
          )}
          {vaultQuery.isLoading ? (
            <LoadingStatusTile />
          ) : (
            <StatusTile
              icon={ShieldCheck}
              label="Vault Coverage"
              value={`${vaultQuery.data?.configured_resources ?? 0}/${vaultQuery.data?.total_resources ?? 0}`}
              meta={`${missingSecrets.length} missing`}
              intent={missingSecrets.length > 0 ? "warn" : "good"}
            />
          )}
          {complianceQuery.isLoading || vulnerabilityQuery.isLoading ? (
            <LoadingStatusTile />
          ) : (
            <StatusTile
              icon={Activity}
              label="Last Scan"
              value={formatTimestamp(complianceQuery.data?.last_updated ?? vaultQuery.data?.last_updated)}
              meta={`${vulnerabilityQuery.data?.scan_duration ?? 0} ms`}
              intent={vulnerabilityQuery.data?.risk_score && vulnerabilityQuery.data.risk_score > 60 ? "danger" : "info"}
            />
          )}
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <div className="space-y-4 rounded-3xl border border-white/5 bg-white/5 p-6 lg:col-span-2">
            {complianceQuery.isLoading ? (
              <div className="flex flex-wrap items-center gap-4">
                <LoadingStatCard />
                <LoadingStatCard />
                <LoadingStatCard />
                <LoadingStatCard />
              </div>
            ) : (
              <div className="flex flex-wrap items-center gap-4">
                <StatCard label="Overall Score" value={`${complianceQuery.data?.overall_score ?? 0}%`} />
                <StatCard label="Configured Components" value={`${complianceQuery.data?.configured_components ?? 0}`} />
                <StatCard label="Security Score" value={`${complianceQuery.data?.remediation_progress?.security_score ?? 0}%`} />
                <StatCard label="Vault Health" value={`${complianceQuery.data?.vault_secrets_health ?? 0}%`} />
              </div>
            )}
            <div className="grid gap-4 md:grid-cols-2">
              <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Vulnerability Mix</p>
                {complianceQuery.isLoading ? (
                  <div className="mt-4 space-y-3">
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-full" />
                  </div>
                ) : (
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
                )}
              </div>
              <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Missing Secrets</p>
                {vaultQuery.isLoading ? (
                  <div className="mt-4 space-y-3">
                    <Skeleton className="h-20 w-full" />
                    <Skeleton className="h-20 w-full" />
                  </div>
                ) : missingSecrets.length === 0 ? (
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
                {vaultQuery.isLoading ? <Skeleton className="h-3 w-32" variant="text" /> : "Live data from vault and file scans"}
              </div>
            </div>
            <div className="mt-4 space-y-3">
              {vaultQuery.isLoading ? (
                <>
                  <LoadingSecretsRow />
                  <LoadingSecretsRow />
                  <LoadingSecretsRow />
                </>
              ) : resourceStatuses.length === 0 ? (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
                  No resource scan results available yet.
                </div>
              ) : (
                resourceStatuses.slice(0, 6).map((status) => (
                  <SecretsRow key={status.resource_name} status={status} />
                ))
              )}
            </div>
          </div>

          <div className="rounded-3xl border border-white/5 bg-white/5 p-6">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">Security Findings</h2>
              <div className="flex items-center gap-2 text-xs text-white/60">
                <ShieldAlert className="h-4 w-4" />
                {vulnerabilityQuery.isLoading ? <Skeleton className="h-3 w-24" variant="text" /> : `Showing ${vulnerabilities.length} results`}
              </div>
            </div>
            <div className="mt-4 space-y-3">
              {vulnerabilityQuery.isLoading ? (
                <>
                  <Skeleton className="h-32 w-full" />
                  <Skeleton className="h-32 w-full" />
                  <Skeleton className="h-32 w-full" />
                </>
              ) : vulnerabilities.length === 0 ? (
                <div className="rounded-2xl border border-emerald-500/40 bg-emerald-500/10 p-6 text-center text-sm text-emerald-100">
                  <ShieldCheck className="mx-auto mb-2 h-6 w-6" />
                  No vulnerabilities found for the selected filters.
                </div>
              ) : (
                vulnerabilities.slice(0, 6).map((vuln) => (
                  <VulnerabilityItem key={vuln.id} vuln={vuln} />
                ))
              )}
            </div>
          </div>
        </section>

        <footer className="pb-6 text-center text-xs text-white/40">
          Powered by the Vrooli lifecycle · API base: {import.meta.env.VITE_API_BASE_URL || "lifecycle-managed"}
        </footer>
      </main>
      {activeResource ? (
        <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/80 px-4 py-10">
          <div className="w-full max-w-5xl rounded-3xl border border-white/10 bg-slate-950/95 p-6 shadow-2xl shadow-emerald-500/20">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Resource Workbench</p>
                <h3 className="text-2xl font-semibold text-white">{activeResource}</h3>
                <p className="text-sm text-white/60">
                  {resourceDetailQuery.data?.valid_secrets ?? 0}/{resourceDetailQuery.data?.total_secrets ?? 0} secrets valid · Missing {resourceDetailQuery.data?.missing_secrets ?? 0}
                </p>
              </div>
              <Button variant="ghost" onClick={closeResourcePanel} className="text-white/70">
                Close
              </Button>
            </div>
            <div className="mt-6 grid gap-6 lg:grid-cols-[1.2fr,1fr]">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm uppercase tracking-[0.3em] text-white/60">Secrets</h4>
                  {resourceDetailQuery.isFetching ? (
                    <span className="text-xs text-white/40">Syncing…</span>
                  ) : null}
                </div>
                <div className="space-y-2 rounded-2xl border border-white/10 bg-black/30 p-3 max-h-[55vh] overflow-y-auto">
                  {resourceDetailQuery.isLoading ? (
                    <p className="text-sm text-white/60">Loading secrets…</p>
                  ) : resourceDetailQuery.data?.secrets.length ? (
                    resourceDetailQuery.data.secrets.map((secret) => (
                      <button
                        key={secret.id}
                        onClick={() => setSelectedSecretKey(secret.secret_key)}
                        className={`w-full rounded-xl border px-3 py-2 text-left ${
                          selectedSecretKey === secret.secret_key
                            ? "border-emerald-500/60 bg-emerald-500/10"
                            : "border-white/10 bg-white/5"
                        }`}
                      >
                        <p className="font-mono text-xs text-white">{secret.secret_key}</p>
                        <p className="text-[11px] text-white/60">{secret.description || secret.secret_type}</p>
                        <div className="mt-2 flex items-center justify-between text-[11px] text-white/60">
                          <span>
                            {secret.classification} · {secret.required ? "Required" : "Optional"}
                          </span>
                          <span>{secret.validation_state}</span>
                        </div>
                      </button>
                    ))
                  ) : (
                    <p className="text-sm text-white/60">No secrets tracked for this resource yet.</p>
                  )}
                </div>
              </div>
              <div className="space-y-4">
                <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
                  <h4 className="text-sm uppercase tracking-[0.3em] text-white/60">Secret Detail</h4>
                  {selectedSecret ? (
                    <div className="mt-3 space-y-4">
                      <div>
                        <p className="font-mono text-sm text-white">{selectedSecret.secret_key}</p>
                        <p className="text-xs text-white/60">{selectedSecret.description || selectedSecret.secret_type}</p>
                      </div>
                      <label className="text-xs uppercase tracking-[0.2em] text-white/60">
                        Classification
                        <select
                          value={selectedSecret.classification}
                          onChange={(event) => handleSecretUpdate(selectedSecret.secret_key, { classification: event.target.value })}
                          className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                        >
                          <option value="infrastructure">Infrastructure</option>
                          <option value="service">Service</option>
                          <option value="user">User</option>
                        </select>
                      </label>
                      <div className="flex items-center justify-between rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white">
                        <span>Required secret</span>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleSecretUpdate(selectedSecret.secret_key, { required: !selectedSecret.required })}
                        >
                          {selectedSecret.required ? "Mark optional" : "Mark required"}
                        </Button>
                      </div>
                      <div>
                        <p className="text-xs uppercase tracking-[0.2em] text-white/60">Tier strategies</p>
                        {Object.entries(selectedSecret.tier_strategies || {}).length ? (
                          <ul className="mt-2 space-y-1 text-xs text-white/70">
                            {Object.entries(selectedSecret.tier_strategies || {}).map(([tier, strategy]) => (
                              <li key={`${tier}-${strategy}`}>{tier}: {strategy}</li>
                            ))}
                          </ul>
                        ) : (
                          <p className="mt-1 text-xs text-amber-200">No tier strategies recorded</p>
                        )}
                      </div>
                      <div className="space-y-2">
                        <p className="text-xs uppercase tracking-[0.2em] text-white/60">Add / update strategy</p>
                        <div className="grid gap-2">
                          <select
                            value={strategyTier}
                            onChange={(event) => setStrategyTier(event.target.value)}
                            className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                          >
                            {tierReadiness.map((tier) => (
                              <option key={tier.tier} value={tier.tier}>
                                {tier.label}
                              </option>
                            ))}
                          </select>
                          <select
                            value={strategyHandling}
                            onChange={(event) => setStrategyHandling(event.target.value)}
                            className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                          >
                            <option value="prompt">Prompt</option>
                            <option value="generate">Generate</option>
                            <option value="strip">Strip</option>
                            <option value="delegate">Delegate</option>
                          </select>
                          <input
                            value={strategyPrompt}
                            onChange={(event) => setStrategyPrompt(event.target.value)}
                            className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                            placeholder="Prompt label"
                          />
                          <textarea
                            value={strategyDescription}
                            onChange={(event) => setStrategyDescription(event.target.value)}
                            className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                            placeholder="Prompt description"
                            rows={2}
                          />
                          <Button size="sm" onClick={handleStrategyApply} disabled={updateStrategyMutation.isLoading}>
                            {updateStrategyMutation.isLoading ? "Saving…" : "Apply strategy"}
                          </Button>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <p className="mt-3 text-sm text-white/60">Select a secret to view details.</p>
                  )}
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
                  <h4 className="text-sm uppercase tracking-[0.3em] text-white/60">Open Vulnerabilities</h4>
                  <div className="mt-3 space-y-3">
                    {resourceDetailQuery.data?.open_vulnerabilities?.length ? (
                      resourceDetailQuery.data.open_vulnerabilities.map((vuln) => (
                        <div key={vuln.id} className="rounded-xl border border-white/10 bg-white/5 p-3">
                          <div className="flex items-center justify-between text-xs text-white/60">
                            <span>{vuln.title}</span>
                            <SeverityBadge severity={vuln.severity} />
                          </div>
                          <p className="mt-1 text-xs text-white/70">{vuln.description}</p>
                          <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-white/60">
                            <Button size="sm" variant="outline" onClick={() => handleVulnerabilityStatus(vuln.id, "in_progress")}>
                              Track
                            </Button>
                            <Button size="sm" onClick={() => handleVulnerabilityStatus(vuln.id, "resolved")}>
                              Resolve
                            </Button>
                          </div>
                        </div>
                      ))
                    ) : (
                      <p className="text-xs text-white/60">No outstanding vulnerabilities for this resource.</p>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
