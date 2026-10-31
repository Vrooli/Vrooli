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
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 lg:grid-cols-3">
        <HealthCard />
        <MetricPlaceholder label={t(strings.pages.dashboard.statPlaceholderLabel)} />
        <MetricPlaceholder label={t(strings.pages.dashboard.statPlaceholderLabel)} />
      </div>
    </section>
  );
}

function MetricPlaceholder({ label }: { label: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm uppercase text-app-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">--</p>
      </CardContent>
    </Card>
  );
}
