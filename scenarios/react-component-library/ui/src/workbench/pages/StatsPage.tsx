import { Link, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowUpRight, CircleAlert } from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes, statsPath, statsView, type StatsView } from "../../routes";
import { SectionState, getLibraryHealth } from "../../api/libraryHealth";
import {
  formatNumber,
  formatPercent,
  useCapabilities,
  useCoverage,
} from "../data";
import {
  Banner,
  Chip,
  Kpi,
  Panel,
  PanelBody,
  PanelHead,
  Ring,
  Skeleton,
  Tabs,
} from "../primitives";
import { ActivityPage } from "./ActivityPage";
import { CapabilitiesPage } from "./CapabilitiesPage";
import { CoveragePage } from "./CoveragePage";

export function StatsPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const view = statsView(searchParams);
  const tabs: Array<{ id: StatsView; label: string }> = [
    {
      id: "overview",
      label: t("workbench.stats.tabOverview", { defaultValue: "Overview" }),
    },
    {
      id: "quality",
      label: t("workbench.stats.tabQuality", { defaultValue: "Quality" }),
    },
    {
      id: "activity",
      label: t("workbench.stats.tabActivity", { defaultValue: "Activity" }),
    },
    {
      id: "services",
      label: t("workbench.stats.tabServices", { defaultValue: "Services" }),
    },
  ];
  const activeTab = tabs.find((tab) => tab.id === view);

  const selectView = (next: StatsView) => {
    const search = new URLSearchParams(searchParams);
    if (next === "overview") search.delete("view");
    else search.set("view", next);
    setSearchParams(search, { replace: true });
  };

  return (
    <div className="wb-page" data-testid="stats-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">
            {t("workbench.stats.kicker", { defaultValue: "Library · Stats" })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.stats.title", { defaultValue: "Is the library healthy?" })}
          </h1>
          <p className="wb-lede">
            {t("workbench.stats.lede", {
              defaultValue:
                "One place for quality, change and service readiness. A section that cannot answer says so instead of rendering zero.",
            })}
          </p>
        </div>
      </div>

      <Tabs
        label={t("workbench.stats.tabs", { defaultValue: "Stats views" })}
        value={view}
        onChange={selectView}
        items={tabs}
      />

      <div
        role="tabpanel"
        aria-label={t("workbench.stats.panel", {
          defaultValue: "{{view}} stats",
          view: activeTab?.label ?? view,
        })}
        style={{ marginTop: 18 }}
      >
        {view === "overview" ? (
          <StatsOverview />
        ) : view === "quality" ? (
          <CoveragePage embedded />
        ) : view === "activity" ? (
          <ActivityPage embedded />
        ) : (
          <CapabilitiesPage embedded />
        )}
      </div>
    </div>
  );
}

/**
 * Overview composes compact signals from owned sources. Counts come from
 * LibraryHealthService; quality and services keep their richer views below.
 * A section that cannot answer says so instead of rendering zero.
 */
function StatsOverview() {
  const { t } = useTranslation();
  const coverage = useCoverage();
  const capabilities = useCapabilities();
  const health = useQuery({
    queryKey: ["wb", "library-health", 7],
    queryFn: () => getLibraryHealth(7),
    retry: 0,
  });

  const report = coverage.data;
  const production = report?.maturity?.productionReadyCoverage;
  const states = capabilities.data?.states ?? [];
  const available = states.filter((state) => state.status === "available").length;
  const degraded = states.filter((state) => state.status === "degraded").length;
  const unavailable = states.filter((state) => state.status === "unavailable").length;
  const counts = health.data?.counts;
  const activity = health.data?.activity;
  const activityMetrics = activity?.metrics ?? [];

  return (
    <div className="wb-stack" data-testid="stats-overview">
      {health.isError ? (
        <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
          {t("workbench.stats.healthUnavailable", {
            defaultValue:
              "Library health did not answer. Counts are shown as unavailable, not zero.",
          })}
        </Banner>
      ) : null}

      <div className="wb-kpis">
        <Kpi
          label={t("workbench.stats.kpiProduction", { defaultValue: "Production ready" })}
          value={production ? formatPercent(production.ratio) : coverage.isError ? "—" : "…"}
          tone={production ? (production.ratio >= 0.5 ? "green" : "amber") : undefined}
          detail={
            production
              ? t("workbench.stats.kpiProductionDetail", {
                  defaultValue: "{{count}} of {{total}} assets at target",
                  count: production.numerator,
                  total: production.denominator,
                })
              : t("workbench.stats.unavailable", { defaultValue: "Unavailable" })
          }
        />
        <Kpi
          label={t("workbench.stats.kpiAssets", { defaultValue: "Assets" })}
          value={counts ? formatNumber(counts.assets) : health.isError ? "—" : "…"}
          detail={
            counts
              ? t("workbench.stats.kpiAssetsDetail", {
                  defaultValue: "{{published}} published versions",
                  published: formatNumber(counts.published),
                })
              : t("workbench.stats.unavailable", { defaultValue: "Unavailable" })
          }
        />
        <Kpi
          label={t("workbench.stats.kpiWorkspaces", { defaultValue: "Workspaces" })}
          value={counts ? formatNumber(counts.workspaces) : health.isError ? "—" : "…"}
          detail={t("workbench.stats.kpiWorkspacesDetail", {
            defaultValue: "Composed workspaces",
          })}
        />
        <Kpi
          label={t("workbench.stats.kpiFindings", { defaultValue: "Open findings" })}
          value={counts ? formatNumber(counts.findingsOpen) : health.isError ? "—" : "…"}
          tone={counts && counts.findingsOpen > 0 ? "amber" : undefined}
          detail={t("workbench.stats.kpiFindingsDetail", {
            defaultValue: "Server-owned review queue",
          })}
        />
      </div>

      <div className="wb-grid-2">
        <Panel>
          <PanelHead
            title={t("workbench.stats.qualityCard", { defaultValue: "Quality" })}
            meta={t("workbench.stats.qualityMeta", { defaultValue: "Coverage" })}
          />
          <PanelBody>
            {coverage.isLoading ? (
              <Skeleton height={76} />
            ) : coverage.isError ? (
              <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                {t("workbench.stats.qualityUnavailable", {
                  defaultValue:
                    "Coverage did not complete. No ratio or distribution is shown as zero.",
                })}
              </Banner>
            ) : report ? (
              <div className="wb-health">
                <Ring
                  value={(production?.ratio ?? 0) * 100}
                  size={66}
                  tone={(production?.ratio ?? 0) >= 0.5 ? "green" : "amber"}
                />
                <div className="wb-health-copy">
                  <b>
                    {t("workbench.stats.qualityTitle", {
                      defaultValue: "{{ready}} of {{total}} at target",
                      ready: production?.numerator ?? 0,
                      total: production?.denominator ?? report.rows.length,
                    })}
                  </b>
                  <Link to={statsPath("quality")} className="wb-health-link">
                    {t("workbench.stats.openQuality", { defaultValue: "Open quality" })}
                    <ArrowUpRight aria-hidden size={12} />
                  </Link>
                </div>
              </div>
            ) : (
              <p className="wb-muted wb-small">
                {t("workbench.stats.noQuality", {
                  defaultValue: "The report returned no rows.",
                })}
              </p>
            )}
          </PanelBody>
        </Panel>

        <Panel>
          <PanelHead
            title={t("workbench.stats.activityCard", { defaultValue: "Activity" })}
            meta={t("workbench.stats.activityMeta", { defaultValue: "Last 7 days" })}
          />
          <PanelBody>
            {health.isLoading ? (
              <Skeleton height={76} />
            ) : health.isError || activity?.state === SectionState.UNAVAILABLE ? (
              <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                {activity?.reason ||
                  t("workbench.stats.activityUnavailable", {
                    defaultValue: "The activity log is unavailable.",
                  })}
              </Banner>
            ) : (
              <div className="wb-stack tight">
                {activityMetrics.length === 0 ? (
                  <p className="wb-muted wb-small">
                    {t("workbench.stats.activityEmpty", {
                      defaultValue: "The log answered with zero events in this window.",
                    })}
                  </p>
                ) : (
                  activityMetrics.map((metric) => (
                    <div key={metric.key} className="wb-row-between">
                      <span className="wb-small wb-muted">{metric.label || metric.key}</span>
                      <b className="wb-mono">{formatNumber(metric.value)}</b>
                    </div>
                  ))
                )}
                <Link to={statsPath("activity")} className="wb-health-link">
                  {t("workbench.stats.openActivity", { defaultValue: "Open activity" })}
                  <ArrowUpRight aria-hidden size={12} />
                </Link>
              </div>
            )}
          </PanelBody>
        </Panel>

        <Panel>
          <PanelHead
            title={t("workbench.stats.servicesCard", { defaultValue: "Services" })}
            meta={t("workbench.stats.servicesMeta", { defaultValue: "Capabilities" })}
          />
          <PanelBody>
            {capabilities.isLoading ? (
              <Skeleton height={76} />
            ) : capabilities.isError ? (
              <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                {t("workbench.stats.servicesUnavailable", {
                  defaultValue:
                    "The capability registry did not answer; nothing is assumed available.",
                })}
              </Banner>
            ) : states.length === 0 ? (
              <p className="wb-muted wb-small">
                {t("workbench.stats.noServices", {
                  defaultValue: "The registry returned no integrations.",
                })}
              </p>
            ) : (
              <div className="wb-stack tight">
                <div className="wb-row" style={{ flexWrap: "wrap" }}>
                  <Chip tone="green">{available}</Chip>
                  <Chip tone="amber">{degraded}</Chip>
                  <Chip tone="red">{unavailable}</Chip>
                </div>
                <span className="wb-small wb-muted">
                  {t("workbench.stats.servicesBreakdown", {
                    defaultValue: "{{available}} available · {{degraded}} degraded · {{unavailable}} unavailable",
                    available,
                    degraded,
                    unavailable,
                  })}
                </span>
                <Link to={statsPath("services")} className="wb-health-link">
                  {t("workbench.stats.openServices", { defaultValue: "Open services" })}
                  <ArrowUpRight aria-hidden size={12} />
                </Link>
              </div>
            )}
          </PanelBody>
        </Panel>

        <Panel>
          <PanelHead
            title={t("workbench.stats.reviewCard", { defaultValue: "Needs a person" })}
            meta={t("workbench.stats.reviewMeta", { defaultValue: "Review" })}
          />
          <PanelBody>
            <p className="wb-small wb-muted">
              {t("workbench.stats.reviewBody", {
                defaultValue:
                  "Review is a work queue, not a measurement. It keeps its own destination so triage never hides inside a chart.",
              })}
            </p>
            <Link to={appRoutes.review} className="wb-health-link">
              {t("workbench.stats.openReview", { defaultValue: "Open review" })}
              <ArrowUpRight aria-hidden size={12} />
            </Link>
          </PanelBody>
        </Panel>
      </div>
    </div>
  );
}
