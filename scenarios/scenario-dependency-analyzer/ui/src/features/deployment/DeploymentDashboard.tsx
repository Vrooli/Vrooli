import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Info } from "lucide-react";

import { buildScenarioApiUrl, fetchDeploymentReport } from "../../api/client";
import { Card, CardContent } from "../../components/ui/card";
import { getApiBaseUrl } from "../../lib/utils";
import type { ScenarioSummary } from "../../types";
import {
  aggregateDeploymentGaps,
  buildDeploymentStatus,
  type DeploymentTierOption,
  type ScenarioDeploymentStatus
} from "./deploymentStatus";
import { DeploymentDetailsPanel } from "./DeploymentDetailsPanel";
import { DeploymentReadinessIntro } from "./DeploymentReadinessIntro";
import { DeploymentStatusList } from "./DeploymentStatusList";
import { DeploymentSummaryCards } from "./DeploymentSummaryCards";
import { MetadataGapsPanel } from "./MetadataGapsPanel";
import { RecommendedFlowPanel } from "./RecommendedFlowPanel";

interface DeploymentDashboardProps {
  scenarios: ScenarioSummary[];
  loading: boolean;
  onRefresh: () => void;
  onScanScenario: (scenarioName: string, apply?: boolean) => void | Promise<void>;
  onSelectScenario: (scenarioName: string, options?: { openCatalog?: boolean }) => void;
}

const tierOptions: DeploymentTierOption[] = [
  { value: "desktop", label: "Tier 2 · Desktop" },
  { value: "local_dev", label: "Tier 1 · Local / Dev" },
  { value: "mobile", label: "Tier 3 · Mobile" },
  { value: "saas", label: "Tier 4 · SaaS / Cloud" },
  { value: "enterprise", label: "Tier 5 · Enterprise" }
];

const deploymentStatusOrder = { critical: 0, issues: 1, "not-scanned": 2, ready: 3 };

export function DeploymentDashboard({ scenarios, loading, onRefresh, onScanScenario, onSelectScenario }: DeploymentDashboardProps) {
  const [statuses, setStatuses] = useState<Map<string, ScenarioDeploymentStatus>>(new Map());
  const [loadingReports, setLoadingReports] = useState(false);
  const [showDagHelp, setShowDagHelp] = useState(false);
  const [selectedScenario, setSelectedScenario] = useState<ScenarioDeploymentStatus | null>(null);
  const [targetTier, setTargetTier] = useState<string>("desktop");
  const [search, setSearch] = useState<string>("");
  const [apiError, setApiError] = useState<string | null>(null);
  const apiBase = useMemo(() => getApiBaseUrl(), []);
  const statusRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (scenarios.length === 0) return;

    const loadReports = async () => {
      setLoadingReports(true);
      const newStatuses = new Map<string, ScenarioDeploymentStatus>();
      const batchSize = 5;

      for (let i = 0; i < scenarios.length; i += batchSize) {
        const batch = scenarios.slice(i, i + batchSize);
        const reports = await Promise.all(
          batch.map(async (scenario) => {
            const report = await fetchDeploymentReport(scenario.name);
            if (!report) {
              setApiError(`API is unreachable. Is scenario-dependency-analyzer running? UI base: ${apiBase || "unknown"}`);
            }
            return report;
          })
        );

        batch.forEach((scenario, idx) => {
          newStatuses.set(scenario.name, buildDeploymentStatus(scenario, reports[idx] ?? null));
        });
      }

      setStatuses(newStatuses);
      setLoadingReports(false);
    };

    void loadReports();
  }, [apiBase, scenarios]);

  const statusArray = useMemo(() => Array.from(statuses.values()), [statuses]);
  const aggregatedGaps = useMemo(() => aggregateDeploymentGaps(statuses.values()), [statuses]);

  const statusCounts = useMemo(
    () => ({
      critical: statusArray.filter((s) => s.status === "critical").length,
      issues: statusArray.filter((s) => s.status === "issues").length,
      notScanned: statusArray.filter((s) => s.status === "not-scanned").length,
      ready: statusArray.filter((s) => s.status === "ready").length
    }),
    [statusArray]
  );

  const filteredStatusArray = useMemo(() => {
    const normalizedSearch = search.trim().toLowerCase();
    return statusArray
      .filter((s) => {
        if (!normalizedSearch) return true;
        return (
          s.scenario.name.toLowerCase().includes(normalizedSearch) ||
          s.scenario.display_name.toLowerCase().includes(normalizedSearch)
        );
      })
      .sort((a, b) => {
        if (deploymentStatusOrder[a.status] !== deploymentStatusOrder[b.status]) {
          return deploymentStatusOrder[a.status] - deploymentStatusOrder[b.status];
        }
        return (b.blockersCount || 0) - (a.blockersCount || 0);
      });
  }, [search, statusArray]);

  const handleScanAllNonReady = useCallback(() => {
    statusArray
      .filter((s) => s.status !== "ready")
      .forEach((s) => onScanScenario(s.scenario.name, false));
  }, [onScanScenario, statusArray]);

  const handleSelectScenario = useCallback(
    (status: ScenarioDeploymentStatus) => {
      setSelectedScenario(status);
      onSelectScenario(status.scenario.name);
    },
    [onSelectScenario]
  );

  const handleScan = useCallback(
    async (scenarioName: string, apply?: boolean) => {
      setStatuses((prev) => {
        const next = new Map(prev);
        const entry = next.get(scenarioName);
        if (entry) {
          next.set(scenarioName, { ...entry, loading: true });
        }
        return next;
      });

      try {
        await Promise.resolve(onScanScenario(scenarioName, apply));
        const refreshed = await fetchDeploymentReport(scenarioName);
        if (!refreshed) {
          setApiError(`API is unreachable. Is scenario-dependency-analyzer running? UI base: ${apiBase || "unknown"}`);
        } else {
          setApiError(null);
        }
        setStatuses((prev) => {
          const next = new Map(prev);
          const entry = next.get(scenarioName);
          if (entry) {
            next.set(scenarioName, buildDeploymentStatus(entry.scenario, refreshed, false));
          }
          return next;
        });
      } catch (error) {
        console.error("Scan failed", error);
        setApiError("Scan failed. Ensure the API is running (vrooli scenario run scenario-dependency-analyzer) and the UI points to the correct port.");
        setStatuses((prev) => {
          const next = new Map(prev);
          const entry = next.get(scenarioName);
          if (entry) {
            next.set(scenarioName, { ...entry, loading: false });
          }
          return next;
        });
      }
    },
    [apiBase, onScanScenario]
  );

  const handleExportDag = useCallback(
    async (scenarioName?: string) => {
      const name = scenarioName || selectedScenario?.scenario.name;
      if (!name) {
        setShowDagHelp(true);
        return;
      }
      try {
        const url = buildScenarioApiUrl(`/scenarios/${name}/dag/export?recursive=true`);
        const response = await fetch(url);
        if (!response.ok) {
          setShowDagHelp(true);
          return;
        }
        const data: unknown = await response.json();
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
        const downloadUrl = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = downloadUrl;
        link.download = `${name}-dag-export.json`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(downloadUrl);
      } catch (e) {
        console.error("Failed to export DAG", e);
        setShowDagHelp(true);
      }
    },
    [selectedScenario]
  );

  const handleOpenCatalog = useCallback(
    (scenarioName: string) => onSelectScenario(scenarioName, { openCatalog: true }),
    [onSelectScenario]
  );

  return (
    <div className="space-y-6">
      <DeploymentReadinessIntro
        apiError={apiError}
        targetTier={targetTier}
        tierOptions={tierOptions}
        onSelectTargetTier={setTargetTier}
      />

      <RecommendedFlowPanel
        onScanAll={handleScanAllNonReady}
        onExportDAG={() => handleExportDag()}
        onJumpToStatus={() => statusRef.current?.scrollIntoView({ behavior: "smooth", block: "start" })}
        dagHelp={showDagHelp ? "dag" : null}
        onOpenDocs={() => window.open("/docs/deployment", "_blank")}
        targetTierLabel={tierOptions.find((t) => t.value === targetTier)?.label}
      />

      <DeploymentSummaryCards
        criticalCount={statusCounts.critical}
        issuesCount={statusCounts.issues}
        notScannedCount={statusCounts.notScanned}
        readyCount={statusCounts.ready}
      />

      <Card className="border border-border/40 bg-background/40">
        <CardContent className="py-4 text-xs text-muted-foreground">
          <div className="flex items-start gap-2">
            <Info className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            <p className="font-medium text-foreground">How to read this page</p>
          </div>
          <p className="mt-1">
            Ready = metadata present, no blockers. Issues = gaps or blockers detected. Critical = missing all deployment
            metadata. Not scanned = run Scan to populate data. Fitness is best/worst tier score; blockers are dependencies
            that fail a tier. Scan &amp; Apply writes updates to service.json. Use the tier buttons above to see fitness
            and blockers for the platform you are targeting.
          </p>
        </CardContent>
      </Card>

      {aggregatedGaps && <MetadataGapsPanel gaps={aggregatedGaps} />}

      <DeploymentStatusList
        loading={loading}
        loadingReports={loadingReports}
        onRefresh={onRefresh}
        onScan={handleScan}
        onSearchChange={setSearch}
        onSelectScenario={handleSelectScenario}
        search={search}
        selectedScenarioName={selectedScenario?.scenario.name}
        statuses={filteredStatusArray}
        statusRef={statusRef}
        targetTier={targetTier}
      />

      {selectedScenario && (
        <DeploymentDetailsPanel
          onExportDag={handleExportDag}
          onOpenCatalog={handleOpenCatalog}
          onScan={handleScan}
          selectedScenario={selectedScenario}
          targetTier={targetTier}
          tierOptions={tierOptions}
        />
      )}
    </div>
  );
}
