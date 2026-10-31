import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  CheckCircle2,
  FileWarning,
  Filter,
  Hammer,
  RefreshCw,
  Search,
  ShieldCheck,
  Wrench,
} from "lucide-react";

import { auditClient } from "../../api/audit";
import type {
  AuditQualityResponse,
  ExplainFindingResponse,
  FixConfigResponse,
} from "../../api/audit";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const DEFAULT_SCENARIO = "quality-health";
const ALL_FILTER = "all";

type RemediationGroup = "autofix" | "agent" | "manual";

const normalize = (value: string) => value.trim().toLowerCase();

const findingGroup = (finding: AuditQualityResponse["findings"][number]): RemediationGroup => {
  if (finding.autofixAvailable) return "autofix";
  if (normalize(finding.remediation).includes("manual")) return "manual";
  return "agent";
};

const severityWeight = (severity: string) => {
  switch (normalize(severity)) {
    case "error":
      return 0;
    case "warning":
      return 1;
    default:
      return 2;
  }
};

const statusTone = (status: string) => {
  switch (normalize(status)) {
    case "passed":
      return "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
    case "failed":
    case "error":
      return "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300";
    case "degraded":
      return "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300";
    default:
      return "border-app-border bg-app-surface-muted text-app-muted-foreground";
  }
};

const severityTone = (severity: string) => {
  switch (normalize(severity)) {
    case "error":
      return "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300";
    case "warning":
      return "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300";
    default:
      return "border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300";
  }
};

const shortPath = (path: string) => path || "unknown";

export function ScenarioAuditWorkbench() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState(DEFAULT_SCENARIO);
  const [submittedScenario, setSubmittedScenario] = useState(DEFAULT_SCENARIO);
  const [severityFilter, setSeverityFilter] = useState(ALL_FILTER);
  const [surfaceFilter, setSurfaceFilter] = useState(ALL_FILTER);
  const [ruleFilter, setRuleFilter] = useState(ALL_FILTER);
  const [selectedFindingId, setSelectedFindingId] = useState("");
  const [fixResult, setFixResult] = useState<FixConfigResponse | undefined>();

  const auditQuery = useQuery({
    queryKey: ["quality-audit", submittedScenario],
    queryFn: () =>
      auditClient.auditQuality({
        scenario: submittedScenario,
        includeCommandExecution: true,
        includeAutofixPreview: true,
        useCache: true,
      }),
    enabled: submittedScenario.length > 0,
  });

  const findings = useMemo(() => {
    const data = auditQuery.data;
    if (!data) return [];
    return [...data.findings]
      .sort((a, b) => severityWeight(a.severity) - severityWeight(b.severity))
      .filter((finding) => {
        const severityMatches =
          severityFilter === ALL_FILTER || normalize(finding.severity) === severityFilter;
        const surfaceMatches =
          surfaceFilter === ALL_FILTER || finding.surfaceId === surfaceFilter;
        const ruleMatches = ruleFilter === ALL_FILTER || finding.ruleId === ruleFilter;
        return severityMatches && surfaceMatches && ruleMatches;
      });
  }, [auditQuery.data, ruleFilter, severityFilter, surfaceFilter]);

  const selectedFinding = findings.find((finding) => finding.id === selectedFindingId) ?? findings[0];

  const explanationQuery = useQuery<ExplainFindingResponse>({
    queryKey: ["quality-explain", submittedScenario, selectedFinding?.id, selectedFinding?.ruleId],
    queryFn: () =>
      auditClient.explainFinding({
        scenario: submittedScenario,
        findingId: selectedFinding?.id ?? "",
        ruleId: selectedFinding?.ruleId ?? "",
      }),
    enabled: Boolean(selectedFinding),
  });

  const fixMutation = useMutation({
    mutationFn: (apply: boolean) =>
      (apply ? auditClient.applyFixConfig : auditClient.previewFixConfig)({
        scenario: submittedScenario,
        ruleIds: selectedFinding?.ruleId ? [selectedFinding.ruleId] : [],
        apply,
      }),
    onSuccess: (result) => setFixResult(result),
  });

  const groupedFindings = useMemo(
    () => ({
      autofix: findings.filter((finding) => findingGroup(finding) === "autofix"),
      agent: findings.filter((finding) => findingGroup(finding) === "agent"),
      manual: findings.filter((finding) => findingGroup(finding) === "manual"),
    }),
    [findings],
  );

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const nextScenario = scenario.trim();
    if (!nextScenario) return;
    setSubmittedScenario(nextScenario);
    setSelectedFindingId("");
    setFixResult(undefined);
  };

  const data = auditQuery.data;
  const surfaces = data?.surfaces ?? [];
  const rules = Array.from(new Set((data?.findings ?? []).map((finding) => finding.ruleId))).sort();

  return (
    <section
      data-testid={selectors.qualityWorkbench.root}
      aria-labelledby="quality-workbench-heading"
      className="flex flex-col gap-5"
    >
      <div className="flex flex-col gap-3 border-b border-app-border pb-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="max-w-3xl">
          <p
            data-testid={selectors.app.eyebrow}
            className="text-xs font-semibold uppercase text-app-muted-foreground"
          >
            {t(strings.app.eyebrow)}
          </p>
          <h2 id="quality-workbench-heading" className="mt-1 text-2xl font-semibold">
            {t(strings.quality.title)}
          </h2>
          <p
            data-testid={selectors.app.description}
            className="mt-2 text-sm text-app-muted-foreground"
          >
            {t(strings.app.description)}
          </p>
        </div>
        <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-2">
          <label className="flex min-w-64 flex-col gap-1 text-sm">
            <span className="text-app-muted-foreground">{t(strings.quality.scenarioLabel)}</span>
            <Input
              data-testid={selectors.qualityWorkbench.scenarioInput}
              value={scenario}
              onChange={(event) => setScenario(event.target.value)}
              aria-label={t(strings.quality.scenarioLabel)}
              placeholder={t(strings.quality.scenarioPlaceholder)}
            />
          </label>
          <Button
            data-testid={selectors.qualityWorkbench.runButton}
            type="submit"
            disabled={auditQuery.isFetching}
          >
            <Search aria-hidden="true" className="h-4 w-4" />
            {auditQuery.isFetching ? t(strings.quality.running) : t(strings.quality.runAudit)}
          </Button>
          <Button
            data-testid={selectors.qualityWorkbench.refreshButton}
            type="button"
            variant="outline"
            onClick={() => void auditQuery.refetch()}
            disabled={auditQuery.isFetching}
          >
            <RefreshCw aria-hidden="true" className="h-4 w-4" />
            {t(strings.quality.refresh)}
          </Button>
        </form>
      </div>

      {auditQuery.isLoading && (
        <div
          data-testid={selectors.qualityWorkbench.loading}
          className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground"
        >
          {t(strings.quality.loading)}
        </div>
      )}

      {auditQuery.error && (
        <div
          data-testid={selectors.qualityWorkbench.error}
          className="rounded-panel border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-300"
        >
          {errorMessage(auditQuery.error, t)}
        </div>
      )}

      {data && (
        <>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
            <Metric
              testId={selectors.qualityWorkbench.status}
              label={t(strings.quality.status)}
              value={data.status || t(strings.quality.unknown)}
              tone={statusTone(data.status)}
              icon={<ShieldCheck aria-hidden="true" className="h-4 w-4" />}
            />
            <Metric
              testId={selectors.qualityWorkbench.maturity}
              label={t(strings.quality.maturity)}
              value={data.maturity ? `R${data.maturity.rung} ${data.maturity.label}` : t(strings.quality.unknown)}
              icon={<CheckCircle2 aria-hidden="true" className="h-4 w-4" />}
            />
            <Metric
              testId={selectors.qualityWorkbench.counts}
              label={t(strings.quality.findings)}
              value={t(strings.quality.countSummary, {
                errors: data.counts?.errors ?? 0,
                warnings: data.counts?.warnings ?? 0,
                infos: data.counts?.infos ?? 0,
              })}
              icon={<FileWarning aria-hidden="true" className="h-4 w-4" />}
            />
            <Metric
              testId={selectors.qualityWorkbench.surfaces}
              label={t(strings.quality.surfaces)}
              value={String(data.counts?.surfaces ?? surfaces.length)}
              icon={<Filter aria-hidden="true" className="h-4 w-4" />}
            />
            <Metric
              testId={selectors.qualityWorkbench.commands}
              label={t(strings.quality.commands)}
              value={String(data.commandResults.length)}
              icon={<Hammer aria-hidden="true" className="h-4 w-4" />}
            />
          </div>

          {data.degradedReason && (
            <div
              data-testid={selectors.qualityWorkbench.degraded}
              className="rounded-panel border border-amber-500/40 bg-amber-500/10 p-4 text-sm text-amber-700 dark:text-amber-300"
            >
              {data.degradedReason}
            </div>
          )}

          <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.65fr)]">
            <div className="flex flex-col gap-4">
              <Panel title={t(strings.quality.surfaceBreakdown)} testId={selectors.qualityWorkbench.surfaceBreakdown}>
                {surfaces.length === 0 ? (
                  <p className="text-sm text-app-muted-foreground">{t(strings.quality.noSurfaces)}</p>
                ) : (
                  <div className="grid gap-2 md:grid-cols-2">
                    {surfaces.map((surface) => (
                      <div
                        key={surface.id}
                        data-testid={selectors.qualityWorkbench.surfaceCard({ id: surface.id })}
                        className="rounded-control border border-app-border bg-app-surface-muted p-3"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <p className="font-medium">{surface.id}</p>
                          <span className={`rounded-control border px-2 py-0.5 text-xs ${statusTone(surface.status)}`}>
                            {surface.status || t(strings.quality.unknown)}
                          </span>
                        </div>
                        <p className="mt-1 text-xs text-app-muted-foreground">
                          {[surface.kind, surface.language, surface.framework].filter(Boolean).join(" / ")}
                        </p>
                        <p className="mt-2 truncate text-xs text-app-muted-foreground">{surface.rootPath}</p>
                      </div>
                    ))}
                  </div>
                )}
              </Panel>

              <Panel title={t(strings.quality.findingsWorkbench)} testId={selectors.qualityWorkbench.findingsWorkbench}>
                <div className="grid gap-2 md:grid-cols-3">
                  <SelectFilter
                    label={t(strings.quality.severityFilter)}
                    value={severityFilter}
                    onChange={setSeverityFilter}
                    options={[
                      [ALL_FILTER, t(strings.quality.allSeverities)],
                      ["error", t(strings.quality.errorSeverity)],
                      ["warning", t(strings.quality.warningSeverity)],
                      ["info", t(strings.quality.infoSeverity)],
                    ]}
                  />
                  <SelectFilter
                    label={t(strings.quality.surfaceFilter)}
                    value={surfaceFilter}
                    onChange={setSurfaceFilter}
                    options={[
                      [ALL_FILTER, t(strings.quality.allSurfaces)],
                      ...surfaces.map((surface) => [surface.id, surface.id] as const),
                    ]}
                  />
                  <SelectFilter
                    label={t(strings.quality.ruleFilter)}
                    value={ruleFilter}
                    onChange={setRuleFilter}
                    options={[
                      [ALL_FILTER, t(strings.quality.allRules)],
                      ...rules.map((rule) => [rule, rule] as const),
                    ]}
                  />
                </div>

                <div className="mt-4 grid gap-3 lg:grid-cols-3">
                  <FindingGroup title={t(strings.quality.autofixGroup)} findings={groupedFindings.autofix} />
                  <FindingGroup title={t(strings.quality.agentGroup)} findings={groupedFindings.agent} />
                  <FindingGroup title={t(strings.quality.manualGroup)} findings={groupedFindings.manual} />
                </div>

                <div className="mt-4 flex flex-col gap-2">
                  {findings.length === 0 ? (
                    <p data-testid={selectors.qualityWorkbench.empty} className="text-sm text-app-muted-foreground">
                      {t(strings.quality.noFindings)}
                    </p>
                  ) : (
                    findings.map((finding) => (
                      <button
                        key={finding.id}
                        type="button"
                        data-testid={selectors.qualityWorkbench.findingRow({ id: finding.id })}
                        onClick={() => setSelectedFindingId(finding.id)}
                        className={[
                          "rounded-control border p-3 text-left transition hover:bg-app-surface-muted",
                          selectedFinding?.id === finding.id
                            ? "border-app-primary bg-app-surface-muted"
                            : "border-app-border bg-app-surface",
                        ].join(" ")}
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <span className={`rounded-control border px-2 py-0.5 text-xs ${severityTone(finding.severity)}`}>
                            {finding.severity}
                          </span>
                          <span className="text-xs font-medium text-app-muted-foreground">{finding.ruleId}</span>
                          <span className="text-xs text-app-muted-foreground">{shortPath(finding.filePath)}</span>
                        </div>
                        <p className="mt-2 text-sm font-medium">{finding.message}</p>
                        <p className="mt-1 line-clamp-2 text-xs text-app-muted-foreground">{finding.evidence}</p>
                      </button>
                    ))
                  )}
                </div>
              </Panel>
            </div>

            <aside className="flex flex-col gap-4">
              <Panel title={t(strings.quality.contractDetail)} testId={selectors.qualityWorkbench.contractDetail}>
                {selectedFinding ? (
                  <div className="flex flex-col gap-3 text-sm">
                    <DetailRow label={t(strings.quality.rule)} value={selectedFinding.ruleId} />
                    <DetailRow label={t(strings.quality.expected)} value={selectedFinding.expected || t(strings.quality.none)} />
                    <DetailRow label={t(strings.quality.observed)} value={selectedFinding.observed || t(strings.quality.none)} />
                    <DetailBlock label={t(strings.quality.whyItMatters)} value={explanationQuery.data?.whyItMatters || selectedFinding.whyItMatters} />
                    <DetailBlock label={t(strings.quality.remediation)} value={explanationQuery.data?.remediation || selectedFinding.remediation} />
                  </div>
                ) : (
                  <p className="text-sm text-app-muted-foreground">{t(strings.quality.selectFinding)}</p>
                )}
              </Panel>

              <Panel title={t(strings.quality.autofixPreview)} testId={selectors.qualityWorkbench.autofixPreview}>
                {selectedFinding?.autofixAvailable ? (
                  <div className="flex flex-col gap-3">
                    <div className="flex flex-wrap gap-2">
                      <Button
                        data-testid={selectors.qualityWorkbench.previewFixButton}
                        type="button"
                        variant="outline"
                        onClick={() => fixMutation.mutate(false)}
                        disabled={fixMutation.isPending}
                      >
                        <Wrench aria-hidden="true" className="h-4 w-4" />
                        {t(strings.quality.previewFix)}
                      </Button>
                      <Button
                        data-testid={selectors.qualityWorkbench.applyFixButton}
                        type="button"
                        onClick={() => fixMutation.mutate(true)}
                        disabled={fixMutation.isPending}
                      >
                        <AlertTriangle aria-hidden="true" className="h-4 w-4" />
                        {t(strings.quality.applyFix)}
                      </Button>
                    </div>
                    {fixMutation.error && (
                      <p className="text-sm text-red-700 dark:text-red-300">{errorMessage(fixMutation.error, t)}</p>
                    )}
                    {(fixResult?.candidates ?? data.autofixCandidates).slice(0, 3).map((candidate) => (
                      <div
                        key={`${candidate.ruleId}:${candidate.filePath}`}
                        data-testid={selectors.qualityWorkbench.autofixCandidate({ ruleId: candidate.ruleId })}
                        className="rounded-control border border-app-border bg-app-surface-muted p-3 text-xs"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <p className="font-medium">{candidate.filePath}</p>
                          <span>{candidate.applied ? t(strings.quality.applied) : t(strings.quality.dryRun)}</span>
                        </div>
                        <p className="mt-2 text-app-muted-foreground">{candidate.description}</p>
                        <div className="mt-2 grid gap-2">
                          <pre className="max-h-28 overflow-auto rounded-control bg-app-background p-2">{candidate.before}</pre>
                          <pre className="max-h-28 overflow-auto rounded-control bg-app-background p-2">{candidate.after}</pre>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-app-muted-foreground">{t(strings.quality.noAutofix)}</p>
                )}
              </Panel>

              <Panel title={t(strings.quality.commandsTitle)} testId={selectors.qualityWorkbench.commandResults}>
                {data.commandResults.length === 0 ? (
                  <p className="text-sm text-app-muted-foreground">{t(strings.quality.noCommands)}</p>
                ) : (
                  <div className="flex flex-col gap-2">
                    {data.commandResults.map((command) => (
                      <div key={`${command.name}:${command.workingDirectory}`} className="rounded-control border border-app-border p-3 text-xs">
                        <div className="flex items-center justify-between gap-2">
                          <p className="font-medium">{command.name}</p>
                          <span className={`rounded-control border px-2 py-0.5 ${statusTone(command.status)}`}>
                            {command.status}
                          </span>
                        </div>
                        <p className="mt-1 truncate text-app-muted-foreground">{command.command}</p>
                        {command.failureReason && (
                          <p className="mt-2 text-red-700 dark:text-red-300">{command.failureReason}</p>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </Panel>
            </aside>
          </div>
        </>
      )}
    </section>
  );
}

function Metric({
  label,
  value,
  testId,
  icon,
  tone = "border-app-border bg-app-surface text-app-foreground",
}: {
  label: string;
  value: string;
  testId: string;
  icon: ReactNode;
  tone?: string;
}) {
  return (
    <div data-testid={testId} className={`rounded-panel border p-4 ${tone}`}>
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs font-semibold uppercase">{label}</p>
        {icon}
      </div>
      <p className="mt-3 text-xl font-semibold">{value}</p>
    </div>
  );
}

function Panel({ title, testId, children }: { title: string; testId: string; children: ReactNode }) {
  return (
    <section data-testid={testId} className="rounded-panel border border-app-border bg-app-surface p-4">
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">{title}</h3>
      <div className="mt-3">{children}</div>
    </section>
  );
}

function SelectFilter({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: ReadonlyArray<readonly [string, string]>;
}) {
  return (
    <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
      {label}
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="rounded-control border border-app-border bg-app-surface px-2 py-2 text-sm text-app-foreground"
      >
        {options.map(([optionValue, optionLabel]) => (
          <option key={optionValue} value={optionValue}>
            {optionLabel}
          </option>
        ))}
      </select>
    </label>
  );
}

function FindingGroup({
  title,
  findings,
}: {
  title: string;
  findings: AuditQualityResponse["findings"];
}) {
  return (
    <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-semibold uppercase text-app-muted-foreground">{title}</p>
        <span className="text-sm font-semibold">{findings.length}</span>
      </div>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-semibold uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-1 break-words font-medium">{value}</p>
    </div>
  );
}

function DetailBlock({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-semibold uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-1 text-app-muted-foreground">{value || "—"}</p>
    </div>
  );
}
