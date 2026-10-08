import { useQuery } from "@tanstack/react-query";
import { Activity, AlertTriangle, Database, ShieldCheck } from "lucide-react";
import type { ReactNode } from "react";

import { listProviderRoles, listRouteEvidence, scanScenario } from "../api/gateway";
import { StatusChip } from "../components/StatusChip";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

const failedEvents = (events: { status: string }[]) =>
  events.filter((event) => event.status.toLowerCase() !== "ok").length;

export function DashboardPage() {
  const { t } = useTranslation();
  const rolesQuery = useQuery({
    queryKey: ["dashboard", "provider-roles"],
    queryFn: () => listProviderRoles(),
  });
  const evidenceQuery = useQuery({
    queryKey: ["dashboard", "route-evidence"],
    queryFn: () => listRouteEvidence(8),
  });
  const conformanceQuery = useQuery({
    queryKey: ["dashboard", "conformance", "ai-gateway"],
    queryFn: () => scanScenario("ai-gateway"),
  });

  const roleCount = rolesQuery.data?.roles.length ?? 0;
  const providers = new Set((rolesQuery.data?.roles ?? []).map((role) => role.provider)).size;
  const failures = failedEvents(evidenceQuery.data?.events ?? []);
  const findings = conformanceQuery.data?.findings.length ?? 0;
  const hasError = rolesQuery.isError || evidenceQuery.isError || conformanceQuery.isError;

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-2">
        <p className="text-xs font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.dashboard.eyebrow)}
        </p>
        <h2 id="dashboard-heading" className="text-2xl font-semibold">
          {t(strings.pages.dashboard.title)}
        </h2>
        <p className="max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.description)}
        </p>
      </header>

      {hasError ? (
        <div data-testid={selectors.dashboard.error} className="rounded-panel border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          {rolesQuery.isError ? errorMessage(rolesQuery.error, t) : null}
          {evidenceQuery.isError ? errorMessage(evidenceQuery.error, t) : null}
          {conformanceQuery.isError ? errorMessage(conformanceQuery.error, t) : null}
        </div>
      ) : null}

      <div
        data-testid={selectors.dashboard.summary}
        role="region"
        aria-label={t(strings.pages.dashboard.summaryLabel)}
        className="grid gap-3 md:grid-cols-4"
      >
        <MetricCard
          icon={<Database aria-hidden="true" size={18} />}
          label={t(strings.pages.dashboard.metrics.providers)}
          value={rolesQuery.isLoading ? t(strings.states.loading) : String(providers)}
          detail={t(strings.pages.dashboard.metrics.roles, { count: roleCount })}
          tone={providers > 0 ? "success" : "warning"}
        />
        <MetricCard
          icon={<Activity aria-hidden="true" size={18} />}
          label={t(strings.pages.dashboard.metrics.routes)}
          value={evidenceQuery.isLoading ? t(strings.states.loading) : String(evidenceQuery.data?.events.length ?? 0)}
          detail={t(strings.pages.dashboard.metrics.failures, { count: failures })}
          tone={failures === 0 ? "success" : "danger"}
        />
        <MetricCard
          icon={<AlertTriangle aria-hidden="true" size={18} />}
          label={t(strings.pages.dashboard.metrics.findings)}
          value={conformanceQuery.isLoading ? t(strings.states.loading) : String(findings)}
          detail={conformanceQuery.data?.maturityLevel || t(strings.states.unknown)}
          tone={findings === 0 ? "success" : "warning"}
        />
        <MetricCard
          icon={<ShieldCheck aria-hidden="true" size={18} />}
          label={t(strings.pages.dashboard.metrics.evidence)}
          value={t(strings.pages.dashboard.metrics.metadataOnly)}
          detail={t(strings.pages.dashboard.metrics.redaction)}
          tone="info"
        />
      </div>

      <div className="grid gap-5 xl:grid-cols-[1fr_1fr]">
        <section
          data-testid={selectors.dashboard.routeEvents}
          className="rounded-panel border border-app-border bg-app-surface p-4"
          aria-labelledby="route-activity-heading"
        >
          <h3 id="route-activity-heading" className="font-semibold">
            {t(strings.pages.dashboard.routeActivity)}
          </h3>
          <div className="mt-3 overflow-hidden rounded-control border border-app-border">
            {(evidenceQuery.data?.events.length ?? 0) > 0 ? (
              <table className="w-full min-w-[620px] text-left text-sm">
                <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                  <tr>
                    <th className="px-3 py-2">{t(strings.pages.dashboard.columns.scenario)}</th>
                    <th className="px-3 py-2">{t(strings.pages.dashboard.columns.role)}</th>
                    <th className="px-3 py-2">{t(strings.pages.dashboard.columns.provider)}</th>
                    <th className="px-3 py-2">{t(strings.pages.dashboard.columns.status)}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-app-border">
                  {evidenceQuery.data?.events.map((event) => (
                    <tr key={event.eventId}>
                      <td className="px-3 py-2 font-mono text-xs">{event.scenario}</td>
                      <td className="px-3 py-2 font-mono text-xs">{event.role}</td>
                      <td className="px-3 py-2">{event.selectedProvider}</td>
                      <td className="px-3 py-2">
                        <StatusChip tone={event.status.toLowerCase() === "ok" ? "success" : "danger"}>
                          {event.status || t(strings.states.unknown)}
                        </StatusChip>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="p-3 text-sm text-app-muted-foreground">
                {evidenceQuery.isLoading ? t(strings.states.loading) : t(strings.pages.dashboard.noEvidence)}
              </p>
            )}
          </div>
        </section>

        <section
          data-testid={selectors.dashboard.conformanceDebt}
          className="rounded-panel border border-app-border bg-app-surface p-4"
          aria-labelledby="policy-gaps-heading"
        >
          <h3 id="policy-gaps-heading" className="font-semibold">
            {t(strings.pages.dashboard.policyGaps)}
          </h3>
          <div className="mt-3 grid gap-2">
            {conformanceQuery.data?.findings.length ? (
              conformanceQuery.data.findings.slice(0, 5).map((finding) => (
                <div key={`${finding.ruleId}-${finding.path}`} className="rounded-control border border-app-border bg-app-surface-muted p-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <StatusChip tone="warning">{finding.severity}</StatusChip>
                    <span className="font-mono text-xs">{finding.ruleId}</span>
                  </div>
                  <p className="mt-2 text-sm">{finding.message}</p>
                  <p className="mt-1 font-mono text-xs text-app-muted-foreground">{finding.path}</p>
                </div>
              ))
            ) : (
              <p className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm text-app-muted-foreground">
                {conformanceQuery.isLoading ? t(strings.states.loading) : t(strings.pages.dashboard.noFindings)}
              </p>
            )}
          </div>
        </section>
      </div>
    </section>
  );
}

interface MetricCardProps {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
  tone: "success" | "warning" | "danger" | "info";
}

function MetricCard({ icon, label, value, detail, tone }: MetricCardProps) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <div className="flex items-center justify-between gap-3">
        <span className="inline-flex size-9 items-center justify-center rounded-control bg-app-surface-muted text-app-muted-foreground">
          {icon}
        </span>
        <StatusChip tone={tone}>{detail}</StatusChip>
      </div>
      <p className="mt-4 text-xs uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}
