import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface, type ExperienceSurfaceState } from "../components/experience/ExperienceSurface";
import { HealthCard } from "../features/health/HealthCard";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { configuredBookId, fetchPosition } from "../api/ledger";

interface PositionView {
  runwayMonths: number;
  runwayAvailable?: boolean;
  partial: boolean;
  availability: Array<{ adapterId: string; reason: string }>;
}

export function DashboardPage() {
  const { t } = useTranslation();
  const bookId = configuredBookId();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const query = useQuery({
    queryKey: ["position", bookId],
    queryFn: fetchPosition,
    retry: false,
    enabled: Boolean(bookId) && !fixture,
  });

  const fixtureData: PositionView | null = fixture && fixture !== "empty" && fixture !== "request-error"
    ? {
        runwayMonths: 4.2,
        runwayAvailable: fixture !== "loading",
        partial: ["partial", "stale", "position-partial", "goal-unmet"].includes(fixture),
        availability: ["partial", "stale", "position-partial"].includes(fixture)
          ? [{ adapterId: "bank-feed", reason: "source is stale or unavailable" }]
          : [],
      }
    : null;
  const data = fixtureData ?? query.data;
  const showError = fixture === "request-error" || query.isError;
  const showEmpty = fixture === "empty" || (!fixture && !data && !query.isError);
  const loading = fixture === "loading" || fixture === "pending" || query.isLoading;
  const surfaceState: ExperienceSurfaceState = loading ? "loading" : showError ? "error" : showEmpty ? "empty" : data?.partial ? "partial" : "ready";
  const positionStatus = showError
    ? t(strings.pages.dashboard.positionUnavailable, { message: query.error instanceof Error ? query.error.message : t(strings.pages.dashboard.sourceUnavailable) })
    : data?.partial
      ? t(strings.pages.dashboard.partial)
      : t(strings.pages.dashboard.complete);
  const runwayText = data?.runwayAvailable === false || (data && !data.runwayAvailable)
    ? t(strings.pages.dashboard.runwayUndefined)
    : data
      ? t(strings.pages.dashboard.runwayMonths, { months: data.runwayMonths.toFixed(2) })
      : t(strings.pages.dashboard.runwayUndefined);

  return (
    <ExperienceSurface
      surfaceId="dashboard"
      state={surfaceState}
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
      statusMessage={showError ? t(strings.pages.dashboard.positionUnavailable, { message: "source unavailable" }) : undefined}
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">{t(strings.pages.dashboard.title)}</h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 lg:grid-cols-3">
        <HealthCard />
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.runwayLabel)}</CardTitle></CardHeader>
          <CardContent>
            <p data-testid={selectors.pages.runwayFigure} role="status" className="text-2xl font-semibold tabular-nums">{runwayText}</p>
            <p data-testid={selectors.pages.runwayBasis} role="note" aria-label={t(strings.pages.dashboard.runwayBasis)} className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.runwayBasis)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">{t(strings.pages.dashboard.completenessLabel)}</CardTitle></CardHeader>
          <CardContent>
            <p data-testid={selectors.pages.completeness} role="status" aria-label={t(strings.pages.dashboard.completenessLabel)} className="font-medium">{data?.partial ? t(strings.pages.dashboard.partial) : data ? t(strings.pages.dashboard.complete) : t(strings.pages.dashboard.notConfigured)}</p>
            {data?.availability.map((item) => <p data-testid={selectors.pages.missingAdapter} role="note" aria-label={item.adapterId} key={item.adapterId} className="text-sm text-amber-700">{t(strings.pages.dashboard.missingAdapter, item)}</p>)}
          </CardContent>
        </Card>
      </div>
      <p data-testid={selectors.pages.positionError} role={showError ? "alert" : "status"} aria-live={showError ? "assertive" : "off"} className={showError ? undefined : "sr-only"}>{positionStatus}</p>
      {showEmpty && <p data-testid={selectors.pages.emptyGuidance} role="note" className="rounded-md border border-dashed p-4 text-app-muted-foreground">{t(strings.pages.dashboard.emptyGuidance)}</p>}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>{t(strings.pages.dashboard.goalsTitle)}</CardTitle></CardHeader>
          <CardContent>
            <ul data-testid={selectors.pages.goalList} aria-label={t(strings.pages.dashboard.goalsTitle)}>
              <li>{t(strings.pages.dashboard.goalsDescription)}</li>
            </ul>
            <meter data-testid={selectors.pages.sustainProgress} min={0} max={3} value={fixture === "goal-unmet" ? 1 : data ? 3 : 0} className="mt-3 block w-full" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>{t(strings.pages.dashboard.changeTitle)}</CardTitle></CardHeader>
          <CardContent><p data-testid={selectors.pages.positionDelta} className="tabular-nums">{t(strings.pages.dashboard.changeDescription)}</p></CardContent>
        </Card>
      </div>
    </ExperienceSurface>
  );
}
