import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { fetchRuns } from "../api/inventory";
import { HealthCard } from "../features/health/HealthCard";
import { TimelineCard } from "../features/timeline/TimelineCard";
import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";

export function DashboardPage() {
  const { t } = useTranslation();
  const recent = useQuery({
    queryKey: ["runs", "recent"],
    queryFn: () => fetchRuns({ limit: 5 }),
  });

  return (
    <div data-testid="dashboard-page" className="flex flex-col gap-6">
      <header className="flex flex-wrap items-baseline justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-app-foreground">
            {t("dashboard.title", { defaultValue: "Operational dashboard" })}
          </h1>
          <p className="mt-1 text-sm text-app-muted-foreground">
            {t("dashboard.subtitle", {
              defaultValue: "Verification health, recent runs, and trend across all flows.",
            })}
          </p>
        </div>
        <Link
          to={ROUTES.flowsInventory}
          data-testid="dashboard-cta-verify"
          className="inline-flex h-9 items-center rounded-control bg-app-primary px-4 text-sm font-medium text-app-primary-foreground hover:brightness-95"
        >
          {t("dashboard.cta", { defaultValue: "Browse flows" })}
        </Link>
      </header>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <HealthCard />
        <section
          data-testid="dashboard-recent-runs"
          aria-label={t("dashboard.recentRuns", { defaultValue: "Recent runs" })}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h2 className="text-sm font-semibold text-app-foreground">
            {t("dashboard.recentRuns", { defaultValue: "Recent runs" })}
          </h2>
          {recent.isLoading && (
            <p data-testid="dashboard-recent-loading" className="mt-2 text-xs text-app-muted-foreground">
              {t("dashboard.loading", { defaultValue: "Loading…" })}
            </p>
          )}
          {recent.error && (
            <p data-testid="dashboard-recent-error" className="mt-2 text-xs text-app-danger">
              {t("dashboard.error", { defaultValue: "Failed to load recent runs." })}
            </p>
          )}
          {!recent.isLoading && !recent.error && (recent.data?.length ?? 0) === 0 && (
            <p data-testid="dashboard-recent-empty" className="mt-2 text-xs text-app-muted-foreground">
              {t("dashboard.empty", { defaultValue: "No runs yet — verify a flow to see results here." })}
            </p>
          )}
          <ul className="mt-2 flex flex-col gap-1">
            {(recent.data ?? []).map((r) => (
              <li key={r.id} className="flex items-center gap-2 text-xs">
                <StatusDot status={r.status} />
                <Link
                  to={ROUTES.runDetail(r.id)}
                  data-testid={`dashboard-recent-${r.id}`}
                  className="truncate text-app-foreground hover:underline"
                >
                  {r.flowId}
                </Link>
                <span className="ms-auto text-app-muted-foreground">
                  {new Date(r.finishedAt).toLocaleTimeString()}
                </span>
              </li>
            ))}
          </ul>
        </section>
      </div>

      <TimelineCard />
    </div>
  );
}

function StatusDot({ status }: { status: "passed" | "failed" | "error" }) {
  const cls =
    status === "passed"
      ? "bg-app-success"
      : status === "failed"
        ? "bg-app-danger"
        : "bg-app-warning";
  return <span aria-hidden className={`inline-block h-2 w-2 rounded-pill ${cls}`} />;
}
