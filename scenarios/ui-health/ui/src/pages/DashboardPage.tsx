import { Link } from "react-router-dom";
import { Boxes, RefreshCw, Search, ShieldCheck, type LucideIcon } from "lucide-react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";

interface StatCardProps {
  label: string;
  value: string;
}

function StatCard({ label, value }: StatCardProps) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-app-foreground">{value}</p>
    </div>
  );
}

interface QuickActionProps {
  to: string;
  label: string;
  icon: LucideIcon;
}

function QuickAction({ to, label, icon: Icon }: QuickActionProps) {
  return (
    <Link
      to={to}
      className="flex items-center gap-2 rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm text-app-foreground hover:bg-app-surface-muted"
    >
      <Icon aria-hidden className="h-4 w-4" />
      <span>{label}</span>
    </Link>
  );
}

export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-1">
        <h2 id="dashboard-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.dashboard.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      </header>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard label={t(strings.pages.dashboard.stats.scenariosValidated)} value="—" />
        <StatCard label={t(strings.pages.dashboard.stats.surfacesIndexed)} value="—" />
        <StatCard label={t(strings.pages.dashboard.stats.openIssues)} value="—" />
      </div>

      <section aria-labelledby="dashboard-quick-actions" className="flex flex-col gap-2">
        <h3 id="dashboard-quick-actions" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.dashboard.quickActions.heading)}
        </h3>
        <div className="flex flex-wrap gap-2">
          <QuickAction to={ROUTES.search} label={t(strings.pages.dashboard.quickActions.search)} icon={Search} />
          <QuickAction
            to={ROUTES.validation}
            label={t(strings.pages.dashboard.quickActions.validate)}
            icon={ShieldCheck}
          />
          <QuickAction
            to={ROUTES.reindex}
            label={t(strings.pages.dashboard.quickActions.reindex)}
            icon={RefreshCw}
          />
          <QuickAction to={ROUTES.inventory} label={t(strings.layout.nav.inventory)} icon={Boxes} />
        </div>
      </section>

      <section aria-labelledby="dashboard-activity" className="flex flex-col gap-2">
        <h3 id="dashboard-activity" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.dashboard.activity.heading)}
        </h3>
        <div className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.activity.empty)}
        </div>
      </section>
    </section>
  );
}
