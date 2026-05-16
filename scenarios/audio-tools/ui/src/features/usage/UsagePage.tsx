import { useQuery } from "@tanstack/react-query";
import { Panel } from "../../components/ui/panel";
import { Badge } from "../../components/ui/badge";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { Card, CardDescription, CardTitle } from "../../components/ui/card";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { getSummary, listRecent } from "../../services/usage";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

export function UsagePage() {
  const { t } = useTranslation();
  const summary = useQuery({ queryKey: ["usage", "summary", 86400], queryFn: () => getSummary(86400) });
  const recent = useQuery({ queryKey: ["usage", "recent", 50], queryFn: () => listRecent(86400, 50) });

  const dash = t(strings.common.dash);

  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader title={t(strings.usage.title)} description={t(strings.usage.description)} />

      <div className="grid gap-3 md:grid-cols-3">
        <Stat title={t(strings.usage.operations)} value={summary.data?.ok ? summary.data.data.operationsTotal : dash} />
        <Stat title={t(strings.usage.creditsCharged)} value={summary.data?.ok ? summary.data.data.creditsTotal : dash} />
        <Stat
          title={t(strings.usage.errors)}
          value={summary.data?.ok ? summary.data.data.errorCount : dash}
          tone={summary.data?.ok && summary.data.data.errorCount > 0 ? "danger" : "neutral"}
        />
      </div>

      <Panel title={t(strings.usage.distributionTitle)} description={t(strings.usage.distributionDescription)}>
        {summary.isLoading ? (
          <LoadingRows rows={3} label={t(strings.common.loading)} />
        ) : summary.data && !summary.data.ok ? (
          <ApiErrorState error={summary.data.error} onRetry={() => void summary.refetch()} />
        ) : summary.data?.ok && summary.data.data.distribution.length === 0 ? (
          <p className="text-sm text-app-muted-foreground">{t(strings.usage.noUsage)}</p>
        ) : summary.data?.ok ? (
          <ul className="flex flex-col gap-2">
            {summary.data.data.distribution.map((d) => (
              <DistributionBar
                key={`${d.providerTier}-${d.providerId}`}
                label={`${d.providerTier} · ${d.providerId}`}
                value={d.count}
                total={summary.data.ok ? summary.data.data.operationsTotal || 1 : 1}
              />
            ))}
          </ul>
        ) : null}
      </Panel>

      <Panel title={t(strings.usage.recentTitle)} bodyless>
        {recent.isLoading ? (
          <div className="p-4"><LoadingRows rows={6} label={t(strings.common.loading)} /></div>
        ) : recent.data && !recent.data.ok ? (
          <div className="p-4"><ApiErrorState error={recent.data.error} onRetry={() => void recent.refetch()} /></div>
        ) : recent.data?.ok && recent.data.data.length === 0 ? (
          <div className="p-4 text-sm text-app-muted-foreground">{t(strings.usage.recentEmpty)}</div>
        ) : recent.data?.ok ? (
          <Table>
            <THead>
              <TR>
                <TH>{t(strings.usage.colTime)}</TH>
                <TH>{t(strings.usage.colCapability)}</TH>
                <TH>{t(strings.usage.colOperation)}</TH>
                <TH>{t(strings.usage.colProvider)}</TH>
                <TH>{t(strings.usage.colModel)}</TH>
                <TH className="text-right">{t(strings.usage.colLatency)}</TH>
                <TH className="text-right">{t(strings.usage.colCredits)}</TH>
                <TH>{t(strings.usage.colStatus)}</TH>
              </TR>
            </THead>
            <TBody>
              {recent.data.data.map((r) => (
                <TR key={r.operationId}>
                  <TD className="whitespace-nowrap font-mono text-xs">{formatTime(r.emittedAt)}</TD>
                  <TD><Badge variant="info">{r.capability}</Badge></TD>
                  <TD className="text-xs">{r.operation}</TD>
                  <TD className="text-xs">{r.providerTier}/{r.providerId}</TD>
                  <TD className="font-mono text-xs">{r.modelId || dash}</TD>
                  <TD className="text-right text-xs">{t(strings.common.millisSuffix, { ms: Math.round(r.latencyMs) })}</TD>
                  <TD className="text-right text-xs">{r.creditsCharged}</TD>
                  <TD>
                    {r.error ? (
                      <Badge variant="danger">{t(strings.status.error)}</Badge>
                    ) : r.fallbackReason ? (
                      <Badge variant="warning">{t(strings.status.fallback)}</Badge>
                    ) : (
                      <Badge variant="success">{t(strings.status.ok)}</Badge>
                    )}
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        ) : null}
      </Panel>
    </div>
  );
}

function Stat({ title, value, tone = "neutral" }: { title: string; value: string | number; tone?: "neutral" | "danger" }) {
  const { t } = useTranslation();
  return (
    <Card padding="md">
      <CardTitle>{title}</CardTitle>
      <p className={"mt-2 text-2xl font-semibold " + (tone === "danger" ? "text-app-danger" : "text-app-foreground")}>
        {value}
      </p>
      <CardDescription className="mt-1">{t(strings.common.last24h)}</CardDescription>
    </Card>
  );
}

function DistributionBar({ label, value, total }: { label: string; value: number; total: number }) {
  const pct = total > 0 ? Math.round((value / total) * 100) : 0;
  return (
    <li>
      <div className="mb-1 flex items-center justify-between text-xs text-app-muted-foreground">
        <span>{label}</span>
        <span>{value} · {pct}%</span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-pill bg-app-surface-muted">
        <div className="h-full bg-app-primary" style={{ width: `${pct}%` }} aria-hidden="true" />
      </div>
    </li>
  );
}

function formatTime(iso: string): string {
  if (!iso) return "—";
  try {
    return new Date(iso).toLocaleTimeString();
  } catch {
    return iso;
  }
}
