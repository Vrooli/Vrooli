import { useQuery } from "@tanstack/react-query";
import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";
import { AlertCircle, Archive, CheckCircle2, GitCompare, ListChecks, RefreshCw, TimerReset } from "lucide-react";
import type { ReactNode } from "react";
import { ValidationMode } from "@vrooli/proto-types/template-manager/v1/validation/validation_pb";

import { fetchTemplateDashboard } from "../api/templateDomain";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";
import {
  debtStatusTone,
  driftTone,
  kindLabel,
  modeLabel,
  runStatusTone,
  type Tone,
} from "../lib/templateLabels";

const queryKey = ["template-dashboard"] as const;

export function DashboardPage() {
  const { t } = useTranslation();
  const query = useQuery({ queryKey, queryFn: fetchTemplateDashboard });

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <div className="min-w-0">
        <h2 id="dashboard-heading" className="text-2xl font-semibold">
          {t(strings.pages.dashboard.title)}
        </h2>
        <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.description)}
        </p>
      </div>

      {query.isLoading && <EmptyState title={t(strings.dashboard.loadingTitle)} description={t(strings.dashboard.loadingDescription)} icon={<RefreshCw aria-hidden="true" className="h-5 w-5 animate-spin" />} />}
      {query.isError && <EmptyState title={t(strings.dashboard.errorTitle)} description={t(strings.dashboard.errorDescription)} icon={<AlertCircle aria-hidden="true" className="h-5 w-5" />} />}

      {query.data && <DashboardLoaded data={query.data} />}
    </section>
  );
}

function DashboardLoaded({ data }: { data: Awaited<ReturnType<typeof fetchTemplateDashboard>> }) {
  const { t } = useTranslation();
  const metrics = deriveMetrics(data);

  return (
    <>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
        <HealthCard />
        <MetricCard title={t(strings.dashboard.metrics.openDebt)} value={String(metrics.openDebt)} icon={<AlertCircle aria-hidden="true" className="h-5 w-5" />} tone="danger" />
        <MetricCard title={t(strings.dashboard.metrics.deepStreak)} value={String(metrics.deepStreak)} icon={<CheckCircle2 aria-hidden="true" className="h-5 w-5" />} tone="success" />
        <MetricCard title={t(strings.dashboard.metrics.versionLag)} value={String(metrics.maxLag)} icon={<GitCompare aria-hidden="true" className="h-5 w-5" />} tone={metrics.maxLag > 0 ? "warning" : "success"} />
        <MetricCard title={t(strings.dashboard.metrics.templates)} value={String(data.templates.templates.length)} icon={<Archive aria-hidden="true" className="h-5 w-5" />} tone="info" />
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
        <StandingCard buckets={metrics.standingBuckets} />
        <MonitorCard monitor={data.monitor.status} />
      </div>

      <RegistryCard templates={data.templates.templates} />

      <div className="grid gap-4 xl:grid-cols-3">
        <RunsCard runs={data.runs.runs} />
        <DebtCard entries={data.debt.entries} />
        <DriftCard snapshots={data.drift.snapshots} />
      </div>
    </>
  );
}

function MonitorCard({ monitor }: { monitor?: { enabled: boolean; intervalSeconds: bigint; inFlight: boolean; lastStatus: string; lastRunId: string; greenStreak: bigint; nextRunAt?: Timestamp } }) {
  const { t } = useTranslation();
  const next = monitor?.nextRunAt ? timestampDate(monitor.nextRunAt) : undefined;
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-3">
        <div className="min-w-0">
          <CardTitle>{t(strings.dashboard.monitor.title)}</CardTitle>
          <CardDescription>{t(strings.dashboard.monitor.description)}</CardDescription>
        </div>
        <StatusBadge tone={monitor?.enabled ? "success" : "warning"}>
          <TimerReset aria-hidden="true" className="h-4 w-4" />
        </StatusBadge>
      </CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2">
        <MonitorStat label={t(strings.dashboard.monitor.status)} value={monitor?.inFlight ? "running" : (monitor?.lastStatus ?? "unknown")} />
        <MonitorStat label={t(strings.dashboard.monitor.nextRun)} value={next ? next.toLocaleString() : t(strings.dashboard.monitor.unscheduled)} />
        <MonitorStat label={t(strings.dashboard.monitor.interval)} value={`${Number(monitor?.intervalSeconds ?? 0n)}s`} />
        <MonitorStat label={t(strings.dashboard.monitor.streak)} value={String(monitor?.greenStreak ?? 0n)} />
        <div className="sm:col-span-2">
          <MonitorStat label={t(strings.dashboard.monitor.lastRun)} value={monitor?.lastRunId || "-"} />
        </div>
      </CardContent>
    </Card>
  );
}

function MonitorStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-panel border border-app-border px-3 py-2">
      <p className="text-xs font-medium text-app-muted-foreground">{label}</p>
      <p className="truncate text-sm font-semibold">{value}</p>
    </div>
  );
}

function MetricCard({ title, value, icon, tone }: { title: string; value: string; icon: ReactNode; tone: "success" | "warning" | "danger" | "info" }) {
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-3">
        <CardTitle className="text-sm text-app-muted-foreground">{title}</CardTitle>
        <StatusBadge tone={tone}>{icon}</StatusBadge>
      </CardHeader>
      <CardContent>
        <p className="text-3xl font-semibold tabular-nums">{value}</p>
      </CardContent>
    </Card>
  );
}

function StandingCard({ buckets }: { buckets: Array<{ standing: string; count: bigint }> }) {
  const { t } = useTranslation();
  const total = buckets.reduce((sum, bucket) => sum + Number(bucket.count), 0);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t(strings.dashboard.standing.title)}</CardTitle>
        <CardDescription>{t(strings.dashboard.standing.description)}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {buckets.length === 0 ? (
          <EmptyState title={t(strings.dashboard.empty.title)} description={t(strings.dashboard.empty.description)} icon={<ListChecks aria-hidden="true" className="h-5 w-5" />} />
        ) : (
          buckets.map((bucket) => {
            const count = Number(bucket.count);
            const pct = total === 0 ? 0 : Math.round((count / total) * 100);
            return (
              <div key={bucket.standing} className="min-w-0">
                <div className="mb-1 flex items-center justify-between gap-3 text-sm">
                  <span className="truncate font-medium">{standingLabel(bucket.standing)}</span>
                  <span className="shrink-0 tabular-nums text-app-muted-foreground">{count}</span>
                </div>
                <div className="h-2 overflow-hidden rounded-pill bg-app-surface-muted">
                  <div className="h-full rounded-pill bg-app-primary" style={{ width: `${pct}%` }} />
                </div>
              </div>
            );
          })
        )}
      </CardContent>
    </Card>
  );
}

function RegistryCard({ templates }: { templates: Array<{ id: string; kind: number; version: string; status: string; versionLag?: { lagCount: number } }> }) {
  const { t } = useTranslation();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t(strings.dashboard.registry.title)}</CardTitle>
        <CardDescription>{t(strings.dashboard.registry.description)}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-2">
          {templates.slice(0, 8).map((template) => (
            <Row
              key={template.id}
              primary={template.id}
              secondary={`${kindLabel(template.kind)} · ${template.version}`}
              badge={template.status}
              tone={(template.versionLag?.lagCount ?? 0) > 0 ? "warning" : "success"}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function RunsCard({ runs }: { runs: Array<{ id: string; templateId: string; mode: number; status: string; findings: unknown[] }> }) {
  const { t } = useTranslation();
  return (
    <ListCard title={t(strings.dashboard.runs.title)} description={t(strings.dashboard.runs.description)} empty={t(strings.dashboard.runs.empty)}>
      {runs.slice(0, 6).map((run) => (
        <Row key={run.id} primary={run.id} secondary={`${run.templateId} · ${modeLabel(run.mode)} · ${run.findings.length} findings`} badge={run.status} tone={runStatusTone(run.status)} />
      ))}
    </ListCard>
  );
}

function DebtCard({ entries }: { entries: Array<{ key: string; severity: string; status: string; title: string }> }) {
  const { t } = useTranslation();
  return (
    <ListCard title={t(strings.dashboard.debt.title)} description={t(strings.dashboard.debt.description)} empty={t(strings.dashboard.debt.empty)}>
      {entries.slice(0, 6).map((entry) => (
        <Row key={entry.key} primary={entry.title} secondary={`${entry.key} · ${entry.severity}`} badge={entry.status} tone={debtStatusTone(entry.status)} />
      ))}
    </ListCard>
  );
}

function DriftCard({ snapshots }: { snapshots: Array<{ id: string; templateId: string; target: string; status: string; driftCount: number }> }) {
  const { t } = useTranslation();
  return (
    <ListCard title={t(strings.dashboard.drift.title)} description={t(strings.dashboard.drift.description)} empty={t(strings.dashboard.drift.empty)}>
      {snapshots.slice(0, 6).map((snapshot) => (
        <Row key={snapshot.id} primary={snapshot.target} secondary={`${snapshot.templateId} · ${snapshot.driftCount} drifted`} badge={snapshot.status} tone={driftTone(snapshot.driftCount)} />
      ))}
    </ListCard>
  );
}

function ListCard({ title, description, empty, children }: { title: string; description: string; empty: string; children: ReactNode }) {
  const items = Array.isArray(children) ? children.filter(Boolean) : children;
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {Array.isArray(items) && items.length === 0 ? (
          <EmptyState title={empty} />
        ) : (
          <div className="grid gap-2">{children}</div>
        )}
      </CardContent>
    </Card>
  );
}

function Row({ primary, secondary, badge, tone }: { primary: string; secondary: string; badge: string; tone: Tone }) {
  return (
    <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-panel border border-app-border px-3 py-2">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium">{primary}</p>
        <p className="truncate text-xs text-app-muted-foreground">{secondary}</p>
      </div>
      <StatusBadge tone={tone}>{badge}</StatusBadge>
    </div>
  );
}

function standingLabel(value: string): string {
  return value.split("_").join(" ");
}

function deriveMetrics(data: Awaited<ReturnType<typeof fetchTemplateDashboard>>): {
  openDebt: number;
  deepStreak: number;
  maxLag: number;
  standingBuckets: Array<{ standing: string; count: bigint }>;
} {
  const openDebt = data.debt.entries.filter((entry) => entry.status === "open").length;
  const maxLag = data.templates.templates.reduce((max, template) => Math.max(max, template.versionLag?.lagCount ?? 0), 0);
  let deepStreak = 0;
  for (const run of data.runs.runs.filter((run) => run.mode === ValidationMode.DEEP)) {
    if (run.status !== "passed") {
      break;
    }
    deepStreak++;
  }

  const latestDrift = new Map<string, number>();
  for (const snapshot of data.drift.snapshots) {
    if (!latestDrift.has(snapshot.templateId)) {
      latestDrift.set(snapshot.templateId, snapshot.driftCount);
    }
  }
  const openDebtByTemplate = new Set(data.debt.entries.filter((entry) => entry.status === "open").map((entry) => entry.templateId));
  const counts = new Map<string, number>();
  for (const template of data.templates.templates) {
    const standing = openDebtByTemplate.has(template.id)
      ? "open_debt"
      : (template.versionLag?.lagCount ?? 0) > 0
        ? "version_lag"
        : (latestDrift.get(template.id) ?? 0) > 0
          ? "drift"
          : "current";
    counts.set(standing, (counts.get(standing) ?? 0) + 1);
  }
  return {
    openDebt,
    deepStreak,
    maxLag,
    standingBuckets: [...counts.entries()].map(([standing, count]) => ({ standing, count: BigInt(count) })),
  };
}
