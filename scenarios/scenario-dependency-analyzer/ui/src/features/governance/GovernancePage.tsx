import { useMemo, useState } from "react";
import { RefreshCcw } from "lucide-react";

import type { DependencyUsageGroup } from "../../api/governance";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "../../components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "../../components/ui/tabs";
import { selectors } from "../../consts/selectors";
import { matchesFinding, matchesRecord, matchesUsageGroup } from "./governanceFilters";
import { DependencyDecisionDrawer, VulnerabilityRemediationDrawer } from "./DependencyDecisionDrawer";
import { DependencyUsagePanel } from "./DependencyUsagePanel";
import { GovernanceDependenciesTable, GovernanceFindingsTable } from "./GovernanceFindingsTable";
import { GovernanceFleetSummary } from "./GovernanceFleetSummary";
import { GovernanceTriagePanel } from "./GovernanceTriagePanel";
import { defaultGovernanceFilters, type GovernanceFilters, type GovernanceView } from "./governanceTypes";
import { useGovernanceData } from "./useGovernanceData";
import { useGovernanceMutations } from "./useGovernanceMutations";

type DecisionState = "approved" | "denied" | "deprecated" | "needs_review";

export function GovernancePage() {
  const [policyMode, setPolicyMode] = useState<"advisory" | "strict" | "review_gate">("advisory");
  const [view, setView] = useState<GovernanceView>("triage");
  const [filters, setFilters] = useState<GovernanceFilters>(defaultGovernanceFilters);
  const [selectedDependency, setSelectedDependency] = useState<DependencyUsageGroup | null>(null);
  const [decisionOpen, setDecisionOpen] = useState(false);
  const [decisionState, setDecisionState] = useState<DecisionState>("approved");
  const [remediationOpen, setRemediationOpen] = useState(false);
  const [pendingVulnerabilityId, setPendingVulnerabilityId] = useState("");

  const { fleet, records, recordsGuidance, triage, securityGaps, loading, error, refresh } = useGovernanceData(policyMode);
  const mutations = useGovernanceMutations();

  const ecosystems = useMemo(() => {
    const values = new Set<string>();
    fleet?.usageGroups.forEach((group) => values.add(group.ecosystem));
    records.forEach((record) => values.add(record.ecosystem));
    return ["all", ...Array.from(values).sort()];
  }, [fleet?.usageGroups, records]);

  const scenarios = useMemo(() => {
    const values = new Set<string>();
    fleet?.scenarios.forEach((scenario) => values.add(scenario.scenario));
    return ["all", ...Array.from(values).sort()];
  }, [fleet?.scenarios]);

  const filteredFindings = useMemo(
    () => (fleet?.findings ?? []).filter((finding) => matchesFinding(finding, filters)),
    [fleet?.findings, filters]
  );

  const filteredUsageGroups = useMemo(
    () => (fleet?.usageGroups ?? []).filter((group) => matchesUsageGroup(group, filters)),
    [fleet?.usageGroups, filters]
  );

  const filteredRecords = useMemo(
    () => records.filter((record) => matchesRecord(record, filters)),
    [filters, records]
  );

  const selectedKey = selectedDependency
    ? `${selectedDependency.ecosystem}/${selectedDependency.packageName}`
    : null;

  const selectDependency = (ecosystem: string, packageName: string) => {
    const group = fleet?.usageGroups.find(
      (candidate) => candidate.ecosystem === ecosystem && candidate.packageName === packageName
    );
    if (group) {
      setSelectedDependency(group);
    }
  };

  const openDecision = (group: DependencyUsageGroup, state: DecisionState) => {
    mutations.previewDecision.reset();
    mutations.applyDecision.reset();
    setSelectedDependency(group);
    setDecisionState(state);
    setDecisionOpen(true);
  };

  const openRemediation = (group: DependencyUsageGroup) => {
    mutations.previewRemediation.reset();
    mutations.denyVulnerable.reset();
    setSelectedDependency(group);
    setPendingVulnerabilityId("");
    setRemediationOpen(true);
  };

  const openRemediationForDependency = (ecosystem: string, packageName: string, vulnerabilityId = "") => {
    const group =
      fleet?.usageGroups.find(
        (candidate) => candidate.ecosystem === ecosystem && candidate.packageName === packageName
      ) ?? null;
    if (!group) return;
    mutations.previewRemediation.reset();
    mutations.denyVulnerable.reset();
    setSelectedDependency(group);
    setPendingVulnerabilityId(vulnerabilityId);
    setRemediationOpen(true);
  };

  const setFilter = <K extends keyof GovernanceFilters>(key: K, value: GovernanceFilters[K]) => {
    setFilters((current) => ({ ...current, [key]: value }));
  };

  const mutationError =
    errorMessage(mutations.previewDecision.error) ??
    errorMessage(mutations.applyDecision.error);

  const remediationError =
    errorMessage(mutations.previewRemediation.error) ??
    errorMessage(mutations.denyVulnerable.error);

  return (
    <main className="grid gap-5" data-testid={selectors.governance.root}>
      <GovernanceFleetSummary
        summary={fleet?.summary}
        passed={fleet?.passed ?? false}
        guidance={fleet?.guidance || recordsGuidance}
      />

      <section className="rounded-lg border border-border/50 bg-card/70 p-3">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <Tabs value={view} onValueChange={(value) => setView(value as GovernanceView)}>
            <TabsList className="h-auto flex-wrap justify-start">
              <TabsTrigger value="triage">Triage</TabsTrigger>
              <TabsTrigger value="findings">Findings</TabsTrigger>
              <TabsTrigger value="dependencies">Dependencies</TabsTrigger>
              <TabsTrigger value="scenarios">Scenarios</TabsTrigger>
              <TabsTrigger value="records">Records</TabsTrigger>
            </TabsList>
          </Tabs>

          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-[minmax(180px,1fr)_150px_150px_160px_150px_auto]">
            <Input
              aria-label="Search governance data"
              placeholder="Search packages, rationale, scenarios"
              value={filters.query}
              onChange={(event) => setFilter("query", event.target.value)}
            />
            <FilterSelect label="Ecosystem" value={filters.ecosystem} values={ecosystems} onChange={(value) => setFilter("ecosystem", value)} />
            <FilterSelect label="Severity" value={filters.severity} values={["all", "error", "warning", "info"]} onChange={(value) => setFilter("severity", value as GovernanceFilters["severity"])} />
            <FilterSelect label="State" value={filters.state} values={["all", "approved", "approved_with_constraints", "needs_review", "denied", "deprecated", "unrecorded"]} onChange={(value) => setFilter("state", value as GovernanceFilters["state"])} />
            <FilterSelect label="Scenario" value={filters.scenario} values={scenarios} onChange={(value) => setFilter("scenario", value)} />
            <FilterSelect label="Policy" value={policyMode} values={["advisory", "strict", "review_gate"]} onChange={(value) => setPolicyMode(value as typeof policyMode)} />
          </div>
        </div>

        <div className="mt-3 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>
            Showing {rowCountForView(view, {
              findings: filteredFindings.length,
              dependencies: filteredUsageGroups.length,
              records: filteredRecords.length,
              scenarios: fleet?.scenarios.length ?? 0,
              triage: triageGroupCount(triage) + (securityGaps?.gaps.length ?? 0)
            })} rows
          </span>
          <Button
            data-testid={selectors.governance.refreshButton}
            disabled={loading}
            onClick={() => void refresh()}
            size="sm"
            type="button"
            variant="outline"
          >
            <RefreshCcw className="mr-2 h-4 w-4" aria-hidden="true" />
            Refresh
          </Button>
        </div>
      </section>

      {error ? (
        <p className="rounded-lg border border-rose-400/30 bg-rose-500/10 p-3 text-sm text-rose-100">
          {errorMessage(error) ?? "Governance data could not be loaded."}
        </p>
      ) : null}

      {loading ? (
        <div className="rounded-lg border border-border/50 bg-card/70 p-8 text-sm text-muted-foreground" aria-live="polite">
          Loading dependency governance data...
        </div>
      ) : (
        <section className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <div>
            {view === "triage" ? (
              <GovernanceTriagePanel
                triage={triage}
                securityGaps={securityGaps}
                onSelectDependency={selectDependency}
                onStartRemediation={openRemediationForDependency}
              />
            ) : null}
            {view === "findings" ? (
              <GovernanceFindingsTable findings={filteredFindings} onSelectDependency={selectDependency} />
            ) : null}
            {view === "dependencies" ? (
              <GovernanceDependenciesTable
                groups={filteredUsageGroups}
                selectedKey={selectedKey}
                onSelect={setSelectedDependency}
              />
            ) : null}
            {view === "scenarios" ? <ScenarioRollup scenarios={fleet?.scenarios ?? []} /> : null}
            {view === "records" ? <RecordsTable records={filteredRecords} /> : null}
          </div>
          <DependencyUsagePanel
            group={selectedDependency}
            onStartDecision={openDecision}
            onStartRemediation={openRemediation}
          />
        </section>
      )}

      <DependencyDecisionDrawer
        group={selectedDependency}
        initialState={decisionState}
        open={decisionOpen}
        preview={mutations.previewDecision.data ?? null}
        applied={mutations.applyDecision.data ?? null}
        busy={mutations.previewDecision.isPending || mutations.applyDecision.isPending}
        error={mutationError}
        onClose={() => setDecisionOpen(false)}
        onPreview={(record) => mutations.previewDecision.mutate(record)}
        onApply={(record) => mutations.applyDecision.mutate(record)}
      />

      <VulnerabilityRemediationDrawer
        group={selectedDependency}
        initialVulnerabilityId={pendingVulnerabilityId}
        open={remediationOpen}
        preview={mutations.previewRemediation.data ?? null}
        result={mutations.denyVulnerable.data ?? null}
        busy={mutations.previewRemediation.isPending || mutations.denyVulnerable.isPending}
        error={remediationError}
        onClose={() => setRemediationOpen(false)}
        onPreview={(vulnerabilityId) => {
          if (!selectedDependency) return;
          mutations.previewRemediation.mutate({
            ecosystem: selectedDependency.ecosystem,
            packageName: selectedDependency.packageName,
            vulnerabilityId
          });
        }}
        onApply={(input) => {
          if (!selectedDependency) return;
          mutations.denyVulnerable.mutate({
            ecosystem: selectedDependency.ecosystem,
            packageName: selectedDependency.packageName,
            vulnerabilityId: input.vulnerabilityId,
            affectedRange: input.affectedRange,
            fixedRange: input.fixedRange,
            rationale: input.rationale,
            approvedBy: input.approvedBy,
            dryRun: false
          });
        }}
      />
    </main>
  );
}

function FilterSelect({
  label,
  value,
  values,
  onChange
}: {
  label: string;
  value: string;
  values: string[];
  onChange: (value: string) => void;
}) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger aria-label={label}>
        <SelectValue placeholder={label} />
      </SelectTrigger>
      <SelectContent>
        {values.map((option) => (
          <SelectItem key={option} value={option}>
            {option.replaceAll("_", " ")}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function ScenarioRollup({
  scenarios
}: {
  scenarios: NonNullable<ReturnType<typeof useGovernanceData>["fleet"]>["scenarios"];
}) {
  if (scenarios.length === 0) {
    return <div className="rounded-lg border border-border/50 bg-card/70 p-8 text-center text-sm text-muted-foreground">No scenario results are available.</div>;
  }
  return (
    <div className="overflow-hidden rounded-lg border border-border/50 bg-card/70">
      <table className="w-full text-left text-sm" aria-label="Scenario governance rollup">
        <thead className="border-b border-border/50 bg-background/45 text-xs uppercase text-muted-foreground">
          <tr>
            <th className="px-3 py-2 font-medium">Scenario</th>
            <th className="px-3 py-2 text-right font-medium">Findings</th>
            <th className="px-3 py-2 text-right font-medium">Errors</th>
            <th className="px-3 py-2 text-right font-medium">Warnings</th>
          </tr>
        </thead>
        <tbody>
          {scenarios.map((scenario) => (
            <tr key={scenario.scenario} className="border-b border-border/30 last:border-0">
              <td className="px-3 py-3 font-medium">{scenario.scenario}</td>
              <td className="px-3 py-3 text-right">{scenario.summary?.findingCount ?? scenario.findings.length}</td>
              <td className="px-3 py-3 text-right">{scenario.summary?.errorCount ?? 0}</td>
              <td className="px-3 py-3 text-right">{scenario.summary?.warningCount ?? 0}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function RecordsTable({
  records
}: {
  records: NonNullable<ReturnType<typeof useGovernanceData>["records"]>;
}) {
  if (records.length === 0) {
    return <div className="rounded-lg border border-border/50 bg-card/70 p-8 text-center text-sm text-muted-foreground">No records match the current filters.</div>;
  }
  return (
    <div className="overflow-hidden rounded-lg border border-border/50 bg-card/70">
      <div className="overflow-auto">
        <table className="w-full min-w-[820px] text-left text-sm" aria-label="Approved dependency records">
          <thead className="border-b border-border/50 bg-background/45 text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-3 py-2 font-medium">Dependency</th>
              <th className="px-3 py-2 font-medium">State</th>
              <th className="px-3 py-2 font-medium">Range</th>
              <th className="px-3 py-2 font-medium">Rationale</th>
              <th className="px-3 py-2 font-medium">Review</th>
            </tr>
          </thead>
          <tbody>
            {records.map((record) => (
              <tr key={`${record.ecosystem}/${record.packageName}`} className="border-b border-border/30 last:border-0">
                <td className="px-3 py-3 break-all font-medium">{record.ecosystem}/{record.packageName}</td>
                <td className="px-3 py-3">{record.state}</td>
                <td className="px-3 py-3"><code className="break-all text-xs">{record.versionRange || "*"}</code></td>
                <td className="px-3 py-3 text-muted-foreground">{record.rationale || "-"}</td>
                <td className="px-3 py-3 text-muted-foreground">{record.lastReviewed || record.approvedDate || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function rowCountForView(
  view: GovernanceView,
  counts: Record<GovernanceView, number>
): number {
  return counts[view];
}

function triageGroupCount(triage: ReturnType<typeof useGovernanceData>["triage"]): number {
  if (!triage) return 0;
  return (
    triage.securityActions.length +
    triage.registrySeeding.length +
    triage.rangePolicy.length +
    triage.scenarioHotspots.length +
    triage.staleOrExpiredReviews.length
  );
}

function errorMessage(error: unknown): string | null {
  if (!error) return null;
  return error instanceof Error ? error.message : String(error);
}
