import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  Boxes,
  CheckCircle2,
  Loader2,
  RefreshCw,
  Search,
  ShieldCheck,
  XCircle,
  type LucideIcon,
} from "lucide-react";

import { Badge } from "../components/ui/Badge";
import { Card, CardBody, CardHeader, CardTitle } from "../components/ui/Card";
import { EmptyState } from "../components/ui/EmptyState";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";
import { fetchHealth } from "../api/health";
import { searchStatus } from "../api/search";
import {
  useActivityFeed,
  type ActivityItem,
} from "../features/dashboard/useActivityFeed";

interface StatCardProps {
  label: string;
  value: string | number;
  testId: string;
}

function StatCard({ label, value, testId }: StatCardProps) {
  return (
    <div
      data-testid={testId}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-app-foreground tabular-nums">{value}</p>
    </div>
  );
}

interface QuickActionProps {
  to: string;
  label: string;
  icon: LucideIcon;
  testId: string;
}

function QuickAction({ to, label, icon: Icon, testId }: QuickActionProps) {
  return (
    <Link
      to={to}
      data-testid={testId}
      className="flex min-h-touch items-center gap-2 rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm text-app-foreground hover:bg-app-surface-muted"
    >
      <Icon aria-hidden className="h-4 w-4" />
      <span>{label}</span>
    </Link>
  );
}

function formatTimestamp(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

export function DashboardPage() {
  const { t } = useTranslation();

  const health = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
  });
  const status = useQuery({
    queryKey: ["search", "status"],
    queryFn: searchStatus,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });
  const activity = useActivityFeed();

  const scenariosValidated = new Set(
    activity
      .filter(
        (a): a is Extract<ActivityItem, { kind: "validation" }> => a.kind === "validation",
      )
      .map((a) => a.run.scenario),
  ).size;
  const surfacesIndexed = status.data?.indexedCount ?? 0;
  const openIssues = activity
    .filter((a): a is Extract<ActivityItem, { kind: "validation" }> => a.kind === "validation")
    .reduce((acc, a) => acc + a.run.errors, 0);

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-1">
        <h2 id="dashboard-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.dashboard.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.description)}
        </p>
      </header>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          label={t(strings.pages.dashboard.stats.scenariosValidated)}
          value={scenariosValidated}
          testId={selectors.dashboard.stats.scenariosValidated}
        />
        <StatCard
          label={t(strings.pages.dashboard.stats.surfacesIndexed)}
          value={surfacesIndexed}
          testId={selectors.dashboard.stats.surfacesIndexed}
        />
        <StatCard
          label={t(strings.pages.dashboard.stats.openIssues)}
          value={openIssues}
          testId={selectors.dashboard.stats.openIssues}
        />
      </div>

      <section aria-labelledby="dashboard-quick-actions" className="flex flex-col gap-2">
        <h3
          id="dashboard-quick-actions"
          className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground"
        >
          {t(strings.pages.dashboard.quickActions.heading)}
        </h3>
        <div className="flex flex-wrap gap-2">
          <QuickAction
            to={ROUTES.search}
            label={t(strings.pages.dashboard.quickActions.search)}
            icon={Search}
            testId={selectors.dashboard.quickActions.search}
          />
          <QuickAction
            to={ROUTES.validation}
            label={t(strings.pages.dashboard.quickActions.validate)}
            icon={ShieldCheck}
            testId={selectors.dashboard.quickActions.validate}
          />
          <QuickAction
            to={ROUTES.reindex}
            label={t(strings.pages.dashboard.quickActions.reindex)}
            icon={RefreshCw}
            testId={selectors.dashboard.quickActions.reindex}
          />
          <QuickAction
            to={ROUTES.inventory}
            label={t(strings.pages.dashboard.quickActions.inventory)}
            icon={Boxes}
            testId={selectors.dashboard.quickActions.inventory}
          />
        </div>
      </section>

      <Card data-testid={selectors.dashboard.apiStatus.card}>
        <CardHeader>
          <CardTitle>{t(strings.pages.dashboard.apiStatus.heading)}</CardTitle>
        </CardHeader>
        <CardBody>
          {health.isLoading ? (
            <p
              className="flex items-center gap-2 text-sm text-app-muted-foreground"
              role="status"
            >
              <Loader2 aria-hidden className="h-4 w-4 animate-spin" />
              {t(strings.pages.dashboard.apiStatus.loading)}
            </p>
          ) : health.error ? (
            <div
              role="alert"
              className="rounded-control border border-app-danger/40 bg-app-danger/10 p-3 text-sm text-app-danger"
            >
              {t(strings.pages.dashboard.apiStatus.error, {
                message:
                  health.error instanceof Error
                    ? health.error.message
                    : String(health.error),
              })}
            </div>
          ) : health.data ? (
            <div className="flex flex-col gap-3">
              <dl className="grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-[max-content_1fr]">
                <DefRow label={t(strings.pages.dashboard.apiStatus.service)}>
                  <span className="font-mono break-all">{health.data.service}</span>
                </DefRow>
                <DefRow label={t(strings.pages.dashboard.apiStatus.version)}>
                  <span className="font-mono break-all">{health.data.version || "—"}</span>
                </DefRow>
                <DefRow label={t(strings.pages.dashboard.apiStatus.uptime)}>
                  <span className="font-mono tabular-nums">
                    {t(strings.pages.dashboard.apiStatus.uptimeSeconds, {
                      seconds: Math.round(health.data.uptimeSeconds),
                    })}
                  </span>
                </DefRow>
                <DefRow label={t(strings.pages.dashboard.apiStatus.readiness)}>
                  {health.data.readiness ? (
                    <Badge tone="success">{t(strings.pages.dashboard.apiStatus.ready)}</Badge>
                  ) : (
                    <Badge tone="error">{t(strings.pages.dashboard.apiStatus.notReady)}</Badge>
                  )}
                </DefRow>
                <DefRow label={t(strings.pages.dashboard.apiStatus.indexedSurfaces)}>
                  <span className="font-mono tabular-nums">{surfacesIndexed}</span>
                </DefRow>
              </dl>

              <div>
                <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
                  {t(strings.pages.dashboard.apiStatus.dependencies)}
                </h4>
                {Object.keys(health.data.dependencies).length === 0 ? (
                  <p className="text-sm text-app-muted-foreground">
                    {t(strings.pages.dashboard.apiStatus.noDependencies)}
                  </p>
                ) : (
                  <ul className="flex flex-col gap-1.5">
                    {Object.entries(health.data.dependencies).map(([name, dep]) => (
                      <li
                        key={name}
                        data-testid={selectors.dashboard.apiStatus.dependency}
                        className="flex items-center justify-between gap-2 rounded-control border border-app-border bg-app-background px-3 py-1.5"
                      >
                        <div className="flex items-center gap-2">
                          {dep.connected ? (
                            <CheckCircle2 aria-hidden className="h-4 w-4 text-app-success" />
                          ) : (
                            <XCircle aria-hidden className="h-4 w-4 text-app-danger" />
                          )}
                          <span className="font-mono text-sm break-all">{name}</span>
                        </div>
                        <span className="text-xs text-app-muted-foreground tabular-nums font-mono">
                          {dep.latencyMs > 0
                            ? t(strings.pages.dashboard.apiStatus.latency, {
                                ms: Math.round(dep.latencyMs),
                              })
                            : ""}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          ) : null}
        </CardBody>
      </Card>

      <section aria-labelledby="dashboard-activity" className="flex flex-col gap-2">
        <h3
          id="dashboard-activity"
          className="text-sm font-semibold uppercase tracking-wide text-app-muted-foreground"
        >
          {t(strings.pages.dashboard.activity.heading)}
        </h3>
        {activity.length === 0 ? (
          <EmptyState
            icon={Activity}
            title={t(strings.pages.dashboard.activity.empty)}
            data-testid={selectors.dashboard.activity.empty}
          />
        ) : (
          <ol
            className="flex flex-col gap-2"
            data-testid={selectors.dashboard.activity.list}
            aria-label={t(strings.pages.dashboard.activity.heading)}
          >
            {activity.map((item, idx) => (
              <li key={item.id}>
                <ActivityRow item={item} index={idx} />
              </li>
            ))}
          </ol>
        )}
      </section>
    </section>
  );
}

function ActivityRow({ item, index }: { item: ActivityItem; index: number }) {
  const { t } = useTranslation();
  if (item.kind === "validation") {
    const r = item.run;
    return (
      <Link
        to={ROUTES.validationDetail(r.scenario)}
        data-testid={selectors.dashboard.activityRow({ index })}
        className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-app-border bg-app-surface p-3 hover:border-app-primary/40"
        aria-label={t(strings.pages.dashboard.activity.openValidation, { scenario: r.scenario })}
      >
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="flex items-center gap-2 text-sm font-medium">
            <ShieldCheck aria-hidden className="h-4 w-4 text-app-muted-foreground" />
            <span>{t(strings.pages.dashboard.activity.kindValidation)}</span>
            <span className="font-mono break-all">{r.scenario}</span>
          </span>
          <span className="text-xs text-app-muted-foreground">
            <time>{formatTimestamp(r.ranAt)}</time>
          </span>
        </div>
        <div className="flex items-center gap-2">
          {r.passed ? (
            <Badge tone="success">{t(strings.pages.dashboard.activity.validationPassed)}</Badge>
          ) : (
            <Badge tone="error">{t(strings.pages.dashboard.activity.validationFailed)}</Badge>
          )}
          {r.errors > 0 ? (
            <Badge tone="error">
              {t(strings.pages.dashboard.activity.errorsLabel, { count: r.errors })}
            </Badge>
          ) : null}
          {r.warnings > 0 ? (
            <Badge tone="warn">
              {t(strings.pages.dashboard.activity.warningsLabel, { count: r.warnings })}
            </Badge>
          ) : null}
        </div>
      </Link>
    );
  }
  const j = item.job;
  return (
    <Link
      to={ROUTES.reindexJob(j.jobId)}
      data-testid={selectors.dashboard.activityRow({ index })}
      className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-app-border bg-app-surface p-3 hover:border-app-primary/40"
      aria-label={t(strings.pages.dashboard.activity.openJob, { jobId: j.jobId })}
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="flex items-center gap-2 text-sm font-medium">
          <RefreshCw aria-hidden className="h-4 w-4 text-app-muted-foreground" />
          <span>{t(strings.pages.dashboard.activity.kindReindex)}</span>
          <span className="font-mono break-all text-app-muted-foreground">
            {j.scenario.length === 0 ? t(strings.pages.reindex.jobs.allScenarios) : j.scenario}
          </span>
        </span>
        <span className="text-xs text-app-muted-foreground">
          <time>{formatTimestamp(j.triggeredAt)}</time>
          {" · "}
          <span className="font-mono">{j.jobId}</span>
        </span>
      </div>
      <div className="flex items-center gap-2">
        {j.dryRun ? <Badge tone="info">{t(strings.pages.reindex.jobs.columns.dryRun)}</Badge> : null}
      </div>
    </Link>
  );
}

function DefRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <>
      <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd className="text-sm text-app-foreground">{children}</dd>
    </>
  );
}
