import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";
import { Link } from "react-router-dom";

/**
 * Dashboard / home page. It is the launch point for the four governed
 * operator surfaces; it never pretends an empty generation store is a metric.
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
        <SurfaceLink href="/variation" title="Variation Board" body="Review measured candidate spreads without taste ranking." />
        <SurfaceLink href="/document" title="Document Workspace" body="Compose long-form prose section by section." />
      </div>
      <div className="grid gap-4 md:grid-cols-2"><SurfaceLink href="/styles" title="Style Library" body="Manage versioned voices and conformance targets." /><SurfaceLink href="/declarations" title="Declaration Registry" body="Validate consumer-owned files and provenance." /></div>
    </section>
  );
}

function SurfaceLink({ href, title, body }: { href: string; title: string; body: string }) {
  return (
    <Card className="transition hover:border-app-primary">
      <CardHeader>
        <CardTitle><Link className="flex min-h-11 items-center focus-visible:rounded-control" to={href}>{title}</Link></CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-app-muted-foreground">{body}</p>
      </CardContent>
    </Card>
  );
}
