import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";
import { useQuery } from "@tanstack/react-query";
import { fetchCoverage, fetchCondition, fetchFocus } from "../api/reliability";
import { ExperienceSurface, type ExperienceSurfaceState } from "../components/experience/ExperienceSurface";

/**
 * Dashboard / home page. The board composes the three reliability domains;
 * detail routes retain the full evidence and rationale for each domain.
 */
export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 lg:grid-cols-4">
        <HealthCard />
        <ReliabilitySummary />
      </div>
    </section>
  );
}

function ReliabilitySummary() {
  const coverage = useQuery({ queryKey: ["reliability", "coverage"], queryFn: fetchCoverage });
  const condition = useQuery({ queryKey: ["reliability", "trust"], queryFn: fetchCondition });
  const focus = useQuery({ queryKey: ["reliability", "focus"], queryFn: fetchFocus });
  const projections = coverage.data?.projections ?? [];
  const missing = projections.reduce((sum, projection) => sum + projection.missingCount, 0);
  const readings = condition.data?.readings.length ?? 0;
  const findings = focus.data?.findings.length ?? 0;
  const state: ExperienceSurfaceState = coverage.isLoading || condition.isLoading || focus.isLoading ? "loading" : coverage.error || condition.error || focus.error ? "partial" : "ready";
  return <>
    <ExperienceSurface surfaceId="confidence-header" state={state} data-testid="board-confidence-header" statusMessage={state === "loading" ? "Loading board confidence." : undefined}>
      <SummaryCard label="Coverage gaps" value={coverage.isLoading ? "…" : String(missing)} detail="dated missing cells" />
    </ExperienceSurface>
    <ExperienceSurface surfaceId="ranked-findings" state={state} data-testid="board-ranked-findings">
      <SummaryCard label="Trusted readings" value={condition.isLoading ? "…" : String(readings)} detail="current condition observations" />
    </ExperienceSurface>
    <ExperienceSurface surfaceId="source-availability" state={state} data-testid="board-source-availability">
      <SummaryCard label="Next findings" value={focus.isLoading ? "…" : String(findings)} detail="ranked, read-only priorities" />
    </ExperienceSurface>
  </>;
}

function SummaryCard({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm uppercase text-app-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">{value}</p>
        <p className="text-xs text-app-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}
