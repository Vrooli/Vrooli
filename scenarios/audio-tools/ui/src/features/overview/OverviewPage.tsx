import { useQuery } from "@tanstack/react-query";
import { Activity, Mic, Server, Settings } from "lucide-react";
import { Link } from "react-router-dom";
import { Panel } from "../../components/ui/panel";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { fetchHealth } from "../../api/health";
import { getProviderConfig } from "../../services/settings";
import { listRecent } from "../../services/usage";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { SummaryCard } from "./components/SummaryCard";

export function OverviewPage() {
  const { t } = useTranslation();
  const health = useQuery({ queryKey: ["health"], queryFn: fetchHealth, refetchInterval: 30_000 });
  const provider = useQuery({ queryKey: ["settings", "provider"], queryFn: getProviderConfig });
  const recent = useQuery({ queryKey: ["usage", "recent", 5], queryFn: () => listRecent(60 * 60, 5) });

  const apiTone: "danger" | "success" | "warning" = health.error
    ? "danger"
    : health.data?.status.toLowerCase() === "ok"
      ? "success"
      : "warning";
  const apiValue = health.data
    ? health.data.status
    : health.isLoading
      ? t(strings.status.checking)
      : t(strings.status.offline);

  const tiers = provider.data?.ok
    ? [
        provider.data.data.byokEnabled ? "BYOK" : null,
        provider.data.data.vrooliEnabled ? "Vrooli" : null,
        provider.data.data.localEnabled ? "Local" : null,
      ]
        .filter(Boolean)
        .join(" · ") || t(strings.common.dash)
    : provider.data
      ? t(strings.common.dash)
      : t(strings.status.checking);

  const recentCount = recent.data?.ok
    ? String(recent.data.data.length)
    : recent.data
      ? t(strings.common.dash)
      : t(strings.status.checking);

  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader
        title={t(strings.app.title)}
        description={t(strings.app.description)}
        actions={
          <Button variant="outline" size="sm" asChild>
            <Link to="/diagnostics">
              <Mic className="h-4 w-4" aria-hidden="true" />
              {t(strings.overview.openDiagnostics)}
            </Link>
          </Button>
        }
      />

      <div className="grid gap-3 md:grid-cols-3">
        <SummaryCard
          icon={<Server className="h-4 w-4" aria-hidden="true" />}
          title={t(strings.overview.summaryApi)}
          value={apiValue}
          tone={apiTone}
          hint={health.data?.service ?? ""}
          statusLabel={
            apiTone === "danger"
              ? t(strings.status.issue)
              : apiTone === "warning"
                ? t(strings.status.degraded)
                : t(strings.status.healthy)
          }
        />
        <SummaryCard
          icon={<Activity className="h-4 w-4" aria-hidden="true" />}
          title={t(strings.overview.summaryRecent)}
          value={recentCount}
          tone="info"
          hint={t(strings.common.lastHour)}
          statusLabel={t(strings.status.live)}
        />
        <SummaryCard
          icon={<Settings className="h-4 w-4" aria-hidden="true" />}
          title={t(strings.overview.summaryProviderTiers)}
          value={tiers}
          tone="neutral"
          hint={t(strings.overview.precedenceHint)}
          statusLabel={t(strings.status.live)}
        />
      </div>

      <Panel
        title={t(strings.overview.recentTitle)}
        description={t(strings.overview.recentDescription)}
      >
        {recent.isLoading ? (
          <LoadingRows rows={4} label={t(strings.common.loading)} />
        ) : recent.data && !recent.data.ok ? (
          <ApiErrorState error={recent.data.error} onRetry={() => void recent.refetch()} />
        ) : recent.data?.ok && recent.data.data.length === 0 ? (
          <p className="text-sm text-app-muted-foreground">{t(strings.overview.noOperations)}</p>
        ) : recent.data?.ok ? (
          <ul className="divide-y divide-app-border text-sm">
            {recent.data.data.map((row) => (
              <li key={row.operationId} className="flex flex-wrap items-center justify-between gap-2 py-2">
                <span className="flex items-center gap-2">
                  <Badge variant={row.error ? "danger" : "neutral"}>{row.capability}</Badge>
                  <span className="font-mono text-xs text-app-muted-foreground">{row.operation}</span>
                </span>
                <span className="flex items-center gap-3 text-xs text-app-muted-foreground">
                  <span>{row.providerTier}/{row.providerId}</span>
                  <span>{t(strings.common.millisSuffix, { ms: Math.round(row.latencyMs) })}</span>
                  <span>{t(strings.common.creditsSuffix, { count: row.creditsCharged })}</span>
                </span>
              </li>
            ))}
          </ul>
        ) : null}
      </Panel>
    </div>
  );
}
