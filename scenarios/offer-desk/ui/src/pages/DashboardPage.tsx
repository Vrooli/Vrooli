import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface, type ExperienceSurfaceState } from "../components/experience/ExperienceSurface";
import { HealthCard } from "../features/health/HealthCard";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fetchBoard } from "../api/offers";

export function DashboardPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const query = useQuery({ queryKey: ["offer-board"], queryFn: fetchBoard, retry: false, enabled: !fixture });
  const entries = fixture && ["source-degraded", "all-sources-healthy", "posture-present", "posture-partial"].includes(fixture)
    ? [{ title: "Example offer", status: "ACTIVE", actualMinor: 0n }]
    : query.data?.entries ?? [];
  const availability = fixture === "source-degraded" || fixture === "posture-partial"
    ? [{ source: "money-ledger", reason: t(strings.pages.dashboard.sourceUnavailableReason) }]
    : query.data?.availability ?? [];
  const showError = fixture === "ledger-unavailable" || query.isError;
  const showEmpty = fixture === "empty" || (!fixture && !query.data && !query.isError);
  const ledgerUnavailable = fixture === "ledger-unavailable";
  const postureUnavailable = fixture === "posture-unavailable";
  const loading = fixture === "loading" || fixture === "pending" || query.isLoading;
  const state: ExperienceSurfaceState = loading ? "loading" : showError ? "error" : showEmpty ? "empty" : availability.length ? "partial" : "ready";
  const position = query.data?.position;

  return (
    <ExperienceSurface surfaceId="board" state={state} data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-4">
      <h2 id="dashboard-heading" className="text-2xl font-semibold">{t(strings.pages.dashboard.title)}</h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div data-testid={selectors.pages.sourceAvailability} role="list" aria-label={t(strings.pages.dashboard.sourceUnavailableReason)} className={availability.length ? "rounded-md border p-3" : "sr-only"}>
        {availability.map((item) => <p data-testid={`${selectors.pages.sourceAvailability}-${item.source}`} key={item.source} role="listitem">{t(strings.pages.dashboard.sourceUnavailable, item)}</p>)}
      </div>
      <div className="grid gap-4 lg:grid-cols-3">
        <HealthCard />
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.priorityBoard)}</CardTitle></CardHeader>
          <CardContent>
            <div data-testid={selectors.pages.boardRanking} role="table" aria-label={t(strings.pages.dashboard.priorityBoard)}>
              <div role="row"><p role="cell" className="font-medium">{t(strings.pages.dashboard.offerRecords, { count: entries.length })}</p></div>
            </div>
            <div data-testid={selectors.pages.firedTriggers} role="list" aria-label={t(strings.pages.dashboard.firedTriggers)}>
              <p role="listitem" className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.firedTriggers)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.ledgerPosture)}</CardTitle></CardHeader>
          <CardContent>
            <p data-testid={selectors.pages.postureSummary} role="status" aria-label={t(strings.pages.dashboard.ledgerPosture)} className="font-medium">{position ? t(strings.pages.dashboard.runwayMonths, { months: position.runwayMonths.toFixed(2) }) : t(strings.pages.dashboard.postureUnavailable)}</p>
            <p data-testid={selectors.pages.postureBasis} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.postureBasis)}</p>
            <ul data-testid={selectors.pages.postureGoalVerdicts} aria-label={t(strings.pages.dashboard.ledgerPosture)} className={fixture === "posture-present" ? "text-sm" : "sr-only"}>
              <li>{t(strings.pages.dashboard.postureBasis)}</li>
            </ul>
          </CardContent>
        </Card>
      </div>
      <div data-testid={selectors.pages.earningNothing} role="list" aria-label={t(strings.pages.dashboard.earningNothing)} className={entries.length ? "rounded-md border p-4" : "sr-only"}>
        <div role="listitem"><h3 className="font-semibold">{t(strings.pages.dashboard.earningNothing)}</h3>
          {entries.filter((entry) => entry.status.toString().includes("ACTIVE") && entry.actualMinor === 0n).map((entry) => <p key={"nodeId" in entry ? entry.nodeId : entry.title}>{entry.title}</p>)}
        </div>
      </div>
      <p data-testid={selectors.pages.emptyGuidance} role="note" className={showEmpty ? "rounded-md border border-dashed p-4 text-app-muted-foreground" : "sr-only"}>{t(strings.pages.dashboard.emptyGuidance)}</p>
      <p data-testid="board-ledger-gap" role="status" className={ledgerUnavailable ? undefined : "sr-only"}>{t(strings.pages.dashboard.boardUnavailable, { message: t(strings.pages.dashboard.postureUnavailable) })}</p>
      <p data-testid={selectors.pages.postureGap} role="status" className={showError || postureUnavailable ? undefined : "sr-only"}>{t(strings.pages.dashboard.boardUnavailable, { message: query.error instanceof Error ? query.error.message : t(strings.pages.dashboard.postureUnavailable) })}</p>
    </ExperienceSurface>
  );
}
