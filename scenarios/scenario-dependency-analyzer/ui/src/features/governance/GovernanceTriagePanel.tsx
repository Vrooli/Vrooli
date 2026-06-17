import { AlertTriangle, ClipboardList, GitBranch, ShieldAlert, TimerReset } from "lucide-react";
import type React from "react";

import type {
  ApprovedDependencyTriageResponse,
  DependencyGovernanceTriageGroup,
  SecurityGovernanceGap,
  SecurityGovernanceGapsResponse
} from "../../api/governance";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { DecisionStatusBadge } from "./DecisionStatusBadge";

export function GovernanceTriagePanel({
  triage,
  securityGaps,
  onSelectDependency,
  onStartRemediation
}: {
  triage: ApprovedDependencyTriageResponse | null;
  securityGaps: SecurityGovernanceGapsResponse | null;
  onSelectDependency: (ecosystem: string, packageName: string) => void;
  onStartRemediation: (ecosystem: string, packageName: string, vulnerabilityId?: string) => void;
}) {
  if (!triage && !securityGaps) {
    return <EmptyPanel title="No triage data is available." />;
  }

  return (
    <div className="grid gap-4" data-testid={selectors.governance.triagePanel}>
      <TriageHeader triage={triage} securityGaps={securityGaps} />
      <SecurityGapsPanel
        response={securityGaps}
        onSelectDependency={onSelectDependency}
        onStartRemediation={onStartRemediation}
      />
      <TriageSection
        title="Security actions"
        icon={<ShieldAlert className="h-4 w-4" aria-hidden="true" />}
        groups={triage?.securityActions ?? []}
        empty="No security governance actions are queued."
        onSelectDependency={onSelectDependency}
      />
      <TriageSection
        title="Registry seeding"
        icon={<ClipboardList className="h-4 w-4" aria-hidden="true" />}
        groups={triage?.registrySeeding ?? []}
        empty="No high-priority unrecorded direct dependencies are queued."
        onSelectDependency={onSelectDependency}
      />
      <TriageSection
        title="Range policy"
        icon={<GitBranch className="h-4 w-4" aria-hidden="true" />}
        groups={triage?.rangePolicy ?? []}
        empty="No range policy decisions need review."
        onSelectDependency={onSelectDependency}
      />
      <div className="grid gap-4 lg:grid-cols-2">
        <TriageSection
          title="Scenario hotspots"
          icon={<AlertTriangle className="h-4 w-4" aria-hidden="true" />}
          groups={triage?.scenarioHotspots ?? []}
          empty="No concentrated scenario hotspots are queued."
          onSelectDependency={onSelectDependency}
        />
        <TriageSection
          title="Stale reviews"
          icon={<TimerReset className="h-4 w-4" aria-hidden="true" />}
          groups={triage?.staleOrExpiredReviews ?? []}
          empty="No stale or expired reviews are queued."
          onSelectDependency={onSelectDependency}
        />
      </div>
    </div>
  );
}

function TriageHeader({
  triage,
  securityGaps
}: {
  triage: ApprovedDependencyTriageResponse | null;
  securityGaps: SecurityGovernanceGapsResponse | null;
}) {
  const summary = triage?.summary;
  return (
    <section className="rounded-lg border border-border/50 bg-card/70 p-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        <Metric label="Status" value={summary?.status || "unknown"} />
        <Metric label="Findings" value={summary?.findingCount ?? 0} />
        <Metric label="Needs review" value={summary?.needsReview ?? 0} />
        <Metric label="Security gaps" value={securityGaps?.uncoveredCount ?? 0} />
        <Metric label="Approved overlap" value={securityGaps?.approvedOverlapCount ?? 0} />
      </div>
      {(triage?.guidance || securityGaps?.guidance) ? (
        <p className="mt-3 text-sm text-muted-foreground">{triage?.guidance || securityGaps?.guidance}</p>
      ) : null}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-md border border-border/40 bg-background/35 p-3">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-1 break-all text-lg font-semibold">{value}</dd>
    </div>
  );
}

function SecurityGapsPanel({
  response,
  onSelectDependency,
  onStartRemediation
}: {
  response: SecurityGovernanceGapsResponse | null;
  onSelectDependency: (ecosystem: string, packageName: string) => void;
  onStartRemediation: (ecosystem: string, packageName: string, vulnerabilityId?: string) => void;
}) {
  const gaps = response?.gaps ?? [];
  return (
    <section className="rounded-lg border border-border/50 bg-card/70">
      <SectionHeader
        title="Security gaps"
        subtitle={`${response?.uncoveredCount ?? 0} uncovered, ${response?.deniedCoveredCount ?? 0} already denied`}
      />
      {response?.warnings.length ? (
        <div className="border-t border-border/40 px-4 py-3 text-sm text-amber-100">
          {response.warnings.slice(0, 2).join(" ")}
        </div>
      ) : null}
      {gaps.length === 0 ? (
        <EmptyPanel title="No uncovered vulnerable dependency gaps match the current evidence." nested />
      ) : (
        <div className="divide-y divide-border/40">
          {gaps.map((gap) => (
            <SecurityGapRow
              key={gap.gapId || `${gap.ecosystem}/${gap.packageName}/${gap.observedVersion}/${gap.vulnerabilityIds.join(",")}`}
              gap={gap}
              onSelectDependency={onSelectDependency}
              onStartRemediation={onStartRemediation}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function SecurityGapRow({
  gap,
  onSelectDependency,
  onStartRemediation
}: {
  gap: SecurityGovernanceGap;
  onSelectDependency: (ecosystem: string, packageName: string) => void;
  onStartRemediation: (ecosystem: string, packageName: string, vulnerabilityId?: string) => void;
}) {
  const vulnerabilityId = gap.vulnerabilityIds[0] ?? "";
  return (
    <article className="grid gap-3 p-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-start">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <DecisionStatusBadge value={gap.normalizedSeverity || gap.severity || "warning"} />
          <button
            className="break-all text-left font-medium text-primary hover:underline"
            onClick={() => onSelectDependency(gap.ecosystem, gap.packageName)}
            type="button"
          >
            {gap.ecosystem}/{gap.packageName}
          </button>
          <code className="rounded bg-background/50 px-1.5 py-0.5 text-xs">{gap.observedVersion || "unknown"}</code>
        </div>
        <p className="mt-2 text-sm text-muted-foreground">{gap.remediation || "Review the vulnerable range and create a denied governance record."}</p>
        <MetaList
          values={[
            ["Vulnerabilities", gap.vulnerabilityIds.join(", ")],
            ["Affected", gap.affectedRanges.slice(0, 2).join(", ")],
            ["Fixed", gap.fixedRanges.slice(0, 2).join(", ")],
            ["Scenarios", summarizeList(gap.scenarios)]
          ]}
        />
      </div>
      <div className="flex flex-wrap gap-2 lg:justify-end">
        <Button size="sm" onClick={() => onStartRemediation(gap.ecosystem, gap.packageName, vulnerabilityId)}>
          Deny range
        </Button>
      </div>
      {gap.suggestedCommand ? (
        <code className="block break-all rounded-md border border-border/40 bg-background/35 p-2 text-xs lg:col-span-2">
          {gap.suggestedCommand}
        </code>
      ) : null}
    </article>
  );
}

function TriageSection({
  title,
  icon,
  groups,
  empty,
  onSelectDependency
}: {
  title: string;
  icon: React.ReactNode;
  groups: DependencyGovernanceTriageGroup[];
  empty: string;
  onSelectDependency: (ecosystem: string, packageName: string) => void;
}) {
  return (
    <section className="rounded-lg border border-border/50 bg-card/70">
      <SectionHeader title={title} subtitle={`${groups.length} groups`} icon={icon} />
      {groups.length === 0 ? (
        <EmptyPanel title={empty} nested />
      ) : (
        <div className="divide-y divide-border/40">
          {groups.map((group) => (
            <TriageGroupRow
              key={group.groupId || `${group.section}/${group.ecosystem}/${group.packageName}/${group.actionType}`}
              group={group}
              onSelectDependency={onSelectDependency}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function TriageGroupRow({
  group,
  onSelectDependency
}: {
  group: DependencyGovernanceTriageGroup;
  onSelectDependency: (ecosystem: string, packageName: string) => void;
}) {
  return (
    <article className="grid gap-3 p-4">
      <div className="flex flex-wrap items-center gap-2">
        <DecisionStatusBadge value={group.highestSeverity || "info"} />
        <button
          className="break-all text-left font-medium text-primary hover:underline"
          onClick={() => onSelectDependency(group.ecosystem, group.packageName)}
          type="button"
        >
          {group.ecosystem}/{group.packageName}
        </button>
        <span className="rounded bg-background/50 px-1.5 py-0.5 text-xs text-muted-foreground">
          {group.actionType.replaceAll("_", " ")}
        </span>
      </div>
      <div>
        <p className="text-sm font-medium">{group.title}</p>
        <p className="mt-1 text-sm text-muted-foreground">{group.rationale}</p>
      </div>
      <MetaList
        values={[
          ["Findings", group.findingCount.toString()],
          ["Scenarios", group.scenarioCount.toString()],
          ["Usages", group.usageCount.toString()],
          ["Versions", summarizeList(group.observedVersions)]
        ]}
      />
      {group.recommendedCommand ? (
        <code className="block break-all rounded-md border border-border/40 bg-background/35 p-2 text-xs">
          {group.recommendedCommand}
        </code>
      ) : null}
    </article>
  );
}

function SectionHeader({
  title,
  subtitle,
  icon
}: {
  title: string;
  subtitle: string;
  icon?: React.ReactNode;
}) {
  return (
    <header className="flex flex-wrap items-center justify-between gap-2 border-b border-border/40 px-4 py-3">
      <div className="flex items-center gap-2">
        {icon}
        <h2 className="text-sm font-semibold">{title}</h2>
      </div>
      <span className="text-xs text-muted-foreground">{subtitle}</span>
    </header>
  );
}

function MetaList({ values }: { values: Array<[string, string]> }) {
  return (
    <dl className="grid gap-2 text-xs text-muted-foreground sm:grid-cols-2 lg:grid-cols-4">
      {values.map(([label, value]) => (
        <div key={label} className="min-w-0">
          <dt>{label}</dt>
          <dd className="mt-0.5 break-all text-foreground">{value || "-"}</dd>
        </div>
      ))}
    </dl>
  );
}

function EmptyPanel({ title, nested = false }: { title: string; nested?: boolean }) {
  return (
    <div className={nested ? "p-4 text-sm text-muted-foreground" : "rounded-lg border border-border/50 bg-card/70 p-8 text-sm text-muted-foreground"}>
      {title}
    </div>
  );
}

function summarizeList(values: string[]): string {
  if (values.length <= 3) return values.join(", ");
  return `${values.slice(0, 3).join(", ")} +${values.length - 3} more`;
}
