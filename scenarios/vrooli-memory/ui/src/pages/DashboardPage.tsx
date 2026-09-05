import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
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
      <p className="text-app-muted-foreground">Harness state only. Journal, recall, frontier, and vocabulary operations live in Source Ledger.</p>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <HealthCard />
        <HarnessCard title="Projection status" value="Managed block ready" />
        <HarnessCard title="Import runs" value="Available per runtime" />
        <HarnessCard title="Capture & maintenance" value="Monitored" />
      </div>
    </section>
  );
}

function HarnessCard({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm uppercase text-app-muted-foreground">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-lg font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}
