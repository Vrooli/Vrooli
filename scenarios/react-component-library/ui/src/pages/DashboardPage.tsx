/**
 * DashboardPage — overview surface.
 *
 * Today: health card + indexed-component count. As capabilities land
 * (adoption tracking, recent versions, drift alerts) they slot into
 * new card components here. The dashboard is intentionally
 * card-composed so growth is additive.
 */
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { componentsClient } from "../api/components";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

export function DashboardPage() {
  const { t } = useTranslation();

  const { data, isLoading } = useQuery({
    queryKey: ["components", "list", "dashboard"],
    queryFn: () => componentsClient.listComponents({ limit: 200 }),
    staleTime: 30_000,
  });

  const count = data?.components?.length ?? 0;

  return (
    <div data-testid="dashboard-page" className="flex flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t("dashboard.title", { defaultValue: "Dashboard" })}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t("dashboard.subtitle", {
            defaultValue:
              "Overview of the component library and the local API.",
          })}
        </p>
      </header>

      <div className="grid gap-4 md:grid-cols-2">
        <section
          data-testid="dashboard-components-tile"
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h2 className="text-sm font-semibold text-app-foreground">
            {t("dashboard.componentsTitle", { defaultValue: "Indexed components" })}
          </h2>
          <p className="mt-3 text-3xl font-semibold text-app-foreground">
            {isLoading
              ? t("dashboard.loading", { defaultValue: "—" })
              : count}
          </p>
          <Link
            to="/components"
            data-testid="dashboard-components-link"
            className="mt-3 inline-flex h-8 items-center rounded-control border border-app-border bg-app-surface px-3 text-xs text-app-foreground hover:bg-app-surface-muted"
          >
            {t("dashboard.openLibrary", { defaultValue: "Open library" })}
          </Link>
        </section>

        <HealthCard />
      </div>
    </div>
  );
}
