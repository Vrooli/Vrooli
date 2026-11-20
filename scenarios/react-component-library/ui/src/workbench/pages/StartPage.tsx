import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  ArrowUpRight,
  Boxes,
  CircleAlert,
  Layers,
  PenTool,
  Plus,
  ShieldCheck,
} from "lucide-react";

import { getReviewSummary } from "../../api/review";
import { WorkspaceProvenance, forkWorkspace } from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { appRoutes, libraryPath, statsPath, workspacePath } from "../../routes";
import {
  assetAdoptions,
  assetDomain,
  formatNumber,
  formatPercent,
  formatRelative,
  timestampToDate,
  useCatalog,
  useCoverage,
} from "../data";
import {
  Banner,
  Chip,
  Dot,
  Empty,
  Kpi,
  Panel,
  PanelBody,
  PanelHead,
  Ring,
  Skeleton,
  WbButton,
} from "../primitives";
import { SearchSurface } from "../search/SearchSurface";
import { useWorkspaces } from "../workspaces/useWorkspaces";

export function StartPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [forking, setForking] = useState<string | null>(null);
  const [forkError, setForkError] = useState<string | null>(null);
  const catalog = useCatalog();
  const coverage = useCoverage();
  const workspaces = useWorkspaces();
  const review = useQuery({
    queryKey: ["review", "summary"],
    queryFn: getReviewSummary,
    retry: 0,
  });

  const assets = useMemo(() => catalog.data ?? [], [catalog.data]);
  const openFindings = review.data?.open ?? 0;
  const report = coverage.data;
  const production = report?.maturity?.productionReadyCoverage;

  const categories = useMemo(() => {
    const counts = new Map<string, number>();
    for (const asset of assets) {
      const domain = assetDomain(asset);
      counts.set(domain, (counts.get(domain) ?? 0) + 1);
    }
    return [...counts.entries()]
      .sort(([leftName, leftCount], [rightName, rightCount]) =>
        rightCount === leftCount
          ? leftName.localeCompare(rightName)
          : rightCount - leftCount,
      )
      .slice(0, 6);
  }, [assets]);

  const domainCount = useMemo(
    () => new Set(assets.map((asset) => assetDomain(asset))).size,
    [assets],
  );

  const popular = useMemo(
    () =>
      [...assets]
        .sort((left, right) => assetAdoptions(right) - assetAdoptions(left))
        .slice(0, 4),
    [assets],
  );

  const summaries = workspaces.data ?? [];
  const recentWorkspaces = summaries.slice(0, 4);
  const curated = summaries.filter(
    (workspace) => workspace.provenance === WorkspaceProvenance.CURATED,
  );
  const draftCount = assets.filter((asset) => asset.draftVersion).length;

  const forkTemplate = async (id: string) => {
    setForking(id);
    setForkError(null);
    try {
      const workspace = await forkWorkspace(id);
      void navigate(workspacePath(workspace.id));
    } catch (cause) {
      setForkError(
        cause instanceof Error
          ? cause.message
          : t("workbench.start.forkFailed", { defaultValue: "The template could not be forked." }),
      );
    } finally {
      setForking(null);
    }
  };

  return (
    <div className="wb-page" data-experience-surface="start" data-testid="start-page">
      <div className="wb-home-hero">
        <div>
          <p className="wb-kicker">
            {catalog.isLoading
              ? t("workbench.start.loadingKicker", {
                  defaultValue: "Start · loading the library…",
                })
              : t("workbench.start.kicker", {
                  defaultValue: "Start · {{count}} assets · {{drafts}} drafts",
                  count: assets.length,
                  drafts: draftCount,
                })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.start.titleLead", { defaultValue: "What will you" })}{" "}
            <em>{t("workbench.start.titleEmphasis", { defaultValue: "make next?" })}</em>
          </h1>
          <p className="wb-lede">
            {t("workbench.start.lede", {
              defaultValue:
                "Search the shared library, browse by domain, continue a workspace, or start from a curated template.",
            })}
          </p>

          <SearchSurface compact />

          <div className="wb-home-actions">
            <WbButton
              variant="primary"
              size="md"
              onClick={() => void navigate(appRoutes.library)}
            >
              {/* The historical CTA key keeps the library discoverable from Start. */}
              {t("workbench.home.browse", { defaultValue: "Browse library" })}
              <ArrowRight aria-hidden size={15} />
            </WbButton>
            <WbButton variant="ghost" size="md" onClick={() => void navigate(appRoutes.design)}>
              <PenTool aria-hidden size={15} />
              {t("workbench.start.openDesign", { defaultValue: "Open design" })}
            </WbButton>
            <WbButton variant="ghost" size="md" onClick={() => void navigate(appRoutes.create)}>
              <Plus aria-hidden size={15} />
              {t("workbench.start.newAsset", { defaultValue: "New asset" })}
            </WbButton>
          </div>

          <div className="wb-home-chips">
            {popular.map((asset) => (
              <Link key={asset.id} to={`/assets/${encodeURIComponent(asset.id)}`}>
                <Chip>{asset.displayName}</Chip>
              </Link>
            ))}
            <Link to={appRoutes.review}>
              <Chip tone={review.isError ? "ice" : openFindings > 0 ? "amber" : "green"}>
                {review.isLoading
                  ? t("workbench.start.reviewChecking", {
                      defaultValue: "Checking review…",
                    })
                  : review.isError
                    ? t("workbench.start.reviewUnavailable", {
                        defaultValue: "Review unavailable",
                      })
                    : t("workbench.start.needsReview", {
                        defaultValue: "{{findings}} need review",
                        findings: formatNumber(openFindings),
                      })}
              </Chip>
            </Link>
          </div>
        </div>

        <Panel aria-label={t("workbench.start.pulse", { defaultValue: "Library pulse" })}>
          <PanelHead
            title={t("workbench.start.pulse", { defaultValue: "Library pulse" })}
            meta={t("workbench.start.pulseMeta", { defaultValue: "Coverage" })}
            actions={
              <Link to={statsPath("quality")} className="wb-health-link">
                {t("workbench.start.openStats", { defaultValue: "Open stats" })}
                <ArrowUpRight aria-hidden size={12} />
              </Link>
            }
          />
          <PanelBody>
            {coverage.isLoading ? (
              <div data-testid="start-pulse-loading">
                <Skeleton height={76} />
              </div>
            ) : coverage.isError ? (
              <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                {t("workbench.start.coverageUnavailable", {
                  defaultValue:
                    "Coverage did not complete. Nothing is shown as zero; open Stats to retry.",
                })}
              </Banner>
            ) : report ? (
              <div className="wb-health">
                <Ring
                  value={(production?.ratio ?? 0) * 100}
                  size={72}
                  tone={(production?.ratio ?? 0) >= 0.8 ? "green" : "amber"}
                />
                <div className="wb-health-copy">
                  <b>
                    {t("workbench.start.productionCoverage", {
                      defaultValue: "Production coverage",
                    })}
                  </b>
                  <span>
                    {t("workbench.start.productionBody", {
                      defaultValue: "{{ready}} of {{total}} assets meet their target rung.",
                      ready: production?.numerator ?? 0,
                      total: production?.denominator ?? report.rows.length,
                    })}
                  </span>
                  <span className="wb-row" style={{ flexWrap: "wrap", gap: 6 }}>
                    <Chip tone="green">
                      <Dot tone="green" />
                      {production ? formatPercent(production.ratio) : "—"}
                    </Chip>
                    <Chip tone={openFindings > 0 ? "amber" : "green"}>
                      {t("workbench.start.findings", {
                        defaultValue: "{{count}} findings",
                        count: openFindings,
                      })}
                    </Chip>
                  </span>
                </div>
              </div>
            ) : (
              <Empty
                title={t("workbench.start.noReport", { defaultValue: "No report yet" })}
              >
                {t("workbench.start.noReportBody", {
                  defaultValue: "Open Stats when the first coverage pass is ready.",
                })}
              </Empty>
            )}
          </PanelBody>
        </Panel>
      </div>

      <div className="wb-kpis">
        <Kpi
          label={t("workbench.start.kpiAssets", { defaultValue: "Indexed assets" })}
          value={
            catalog.isError ? "—" : catalog.isLoading ? "…" : formatNumber(assets.length)
          }
          detail={
            catalog.isError
              ? t("workbench.start.unavailable", { defaultValue: "Unavailable" })
              : t("workbench.start.kpiAssetsDetail", {
                  defaultValue: "{{count}} drafts",
                  count: draftCount,
                })
          }
        />
        <Kpi
          label={t("workbench.start.kpiDomains", { defaultValue: "Domains" })}
          value={
            catalog.isError ? "—" : catalog.isLoading ? "…" : formatNumber(domainCount)
          }
          detail={
            catalog.isError
              ? t("workbench.start.unavailable", { defaultValue: "Unavailable" })
              : t("workbench.start.kpiDomainsDetail", {
                  defaultValue: "Browse by intent",
                })
          }
        />
        <Kpi
          label={t("workbench.start.kpiWorkspaces", { defaultValue: "Workspaces" })}
          value={
            workspaces.isError
              ? "—"
              : workspaces.isLoading
                ? "…"
                : formatNumber(summaries.length)
          }
          detail={
            workspaces.isError
              ? t("workbench.start.unavailable", { defaultValue: "Unavailable" })
              : t("workbench.start.kpiWorkspacesDetail", {
                  defaultValue: "Continue designing",
                })
          }
        />
        <Kpi
          label={t("workbench.start.kpiProduction", { defaultValue: "Production ready" })}
          value={production ? formatPercent(production.ratio) : coverage.isError ? "—" : "…"}
          tone={production ? (production.ratio >= 0.8 ? "green" : "amber") : undefined}
          detail={t("workbench.start.kpiProductionDetail", {
            defaultValue: "Measured by coverage",
          })}
        />
      </div>

      <div className="wb-home-grid">
        <div className="wb-stack">
          <Panel>
            <PanelHead
              title={t("workbench.start.browseTitle", { defaultValue: "Browse by domain" })}
              meta={t("workbench.start.browseMeta", {
                defaultValue: "{{count}} domains",
                count: domainCount,
              })}
              actions={
                <Link to={appRoutes.library} className="wb-btn quiet sm">
                  {t("workbench.start.viewAll", { defaultValue: "View all" })}
                  <ArrowUpRight aria-hidden size={13} />
                </Link>
              }
            />
            <PanelBody>
              {catalog.isLoading ? (
                <div className="wb-new-grid" data-testid="start-categories-loading">
                  <Skeleton height={96} />
                  <Skeleton height={96} />
                  <Skeleton height={96} />
                  <Skeleton height={96} />
                </div>
              ) : catalog.isError ? (
                <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
                  <b>
                    {t("workbench.start.catalogUnavailableTitle", {
                      defaultValue: "The catalog did not answer.",
                    })}
                  </b>{" "}
                  {t("workbench.start.catalogUnavailableBody", {
                    defaultValue:
                      "Domains cannot be derived without it. The library is not shown as empty.",
                  })}
                </Banner>
              ) : categories.length === 0 ? (
                <Empty
                  title={t("workbench.start.emptyCategoriesTitle", {
                    defaultValue: "No domains indexed yet",
                  })}
                  action={
                    <Link to={appRoutes.create} className="wb-btn ghost sm">
                      {t("workbench.start.emptyCategoriesAction", {
                        defaultValue: "Create the first asset",
                      })}
                    </Link>
                  }
                >
                  {t("workbench.start.emptyCategoriesBody", {
                    defaultValue:
                      "Category cards come from the catalog's own domains; add an asset or reindex to populate them.",
                  })}
                </Empty>
              ) : (
                <div className="wb-new-grid">
                  {categories.map(([domain, count]) => (
                    <Link
                      key={domain}
                      to={libraryPath(domain)}
                      className="wb-new-card"
                      data-testid="start-category-card"
                    >
                      <span className="wb-new-art" aria-hidden>
                        <span className="wb-new-art-bar" />
                      </span>
                      <b>{domain}</b>
                      <small>
                        {t("workbench.start.categoryCount", {
                          defaultValue: "{{count}} assets",
                          count,
                        })}
                      </small>
                    </Link>
                  ))}
                </div>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.start.continue", { defaultValue: "Continue designing" })}
              meta={t("workbench.start.continueMeta", {
                defaultValue: "{{count}} workspaces",
                count: summaries.length,
              })}
              actions={
                <Link to={appRoutes.design} className="wb-btn quiet sm">
                  {t("workbench.start.allWorkspaces", { defaultValue: "All workspaces" })}
                  <ArrowUpRight aria-hidden size={13} />
                </Link>
              }
            />
            <PanelBody flush>
              {workspaces.isLoading ? (
                <div className="wb-stack" style={{ padding: 16 }} data-testid="start-designs-loading">
                  <Skeleton height={42} />
                  <Skeleton height={42} />
                  <Skeleton height={42} />
                </div>
              ) : workspaces.isError ? (
                <div style={{ padding: 16 }}>
                  <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                    {t("workbench.start.designsUnavailable", {
                      defaultValue:
                        "The workspace list is unavailable. Open Design to retry the design service.",
                    })}
                  </Banner>
                </div>
              ) : recentWorkspaces.length === 0 ? (
                <Empty
                  title={t("workbench.start.noDesigns", { defaultValue: "No workspaces yet" })}
                  action={
                    <Link to={appRoutes.design} className="wb-btn ghost sm">
                      {t("workbench.start.openDesignArchive", {
                        defaultValue: "Open the design index",
                      })}
                    </Link>
                  }
                >
                  {t("workbench.start.noDesignsBody", {
                    defaultValue:
                      "The design index lists workspaces and templates once the studio has created one.",
                  })}
                </Empty>
              ) : (
                <ul className="wb-home-list">
                  {recentWorkspaces.map((summary) => (
                    <li key={summary.id}>
                      <div className="wb-home-row">
                        <span className="wb-home-thumb" aria-hidden>
                          <Layers size={14} />
                        </span>
                        <span style={{ minWidth: 0, flex: 1 }}>
                          <b>{summary.name}</b>
                          <small>
                            {t("workbench.start.designDisplays", {
                              defaultValue: "{{count}} displays",
                              count: summary.displayCount,
                            })}
                            {" · "}
                            {formatRelative(timestampToDate(summary.updatedAt))}
                          </small>
                        </span>
                        <Link
                          to={workspacePath(summary.id)}
                          className="wb-btn ghost sm"
                        >
                          {t("workbench.start.openWorkspace", { defaultValue: "Open" })}
                          <ArrowRight aria-hidden size={13} />
                        </Link>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </PanelBody>
          </Panel>
        </div>

        <div className="wb-stack">
          <Panel>
            <PanelHead
              title={t("workbench.start.templates", { defaultValue: "Curated templates" })}
              meta={t("workbench.start.templatesMeta", {
                defaultValue: "{{count}} workspaces",
                count: curated.length,
              })}
            />
            <PanelBody>
              {forkError ? (
                <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                  {forkError}
                </Banner>
              ) : null}
              {workspaces.isLoading ? (
                <div data-testid="start-templates-loading">
                  <Skeleton height={96} />
                </div>
              ) : workspaces.isError ? (
                <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                  {t("workbench.start.designsUnavailable", {
                    defaultValue:
                      "The workspace list is unavailable. Open Design to retry the design service.",
                  })}
                </Banner>
              ) : curated.length === 0 ? (
                <Empty
                  title={t("workbench.start.noTemplates", { defaultValue: "No curated templates yet" })}
                >
                  {t("workbench.start.noTemplatesBody", {
                    defaultValue:
                      "A curated template is a workspace snapshot with provenance curated. It forks on use and never changes the original.",
                  })}
                </Empty>
              ) : (
                <div className="wb-continue-templates">
                  {curated.slice(0, 4).map((workspace) => (
                    <article
                      key={workspace.id}
                      className="wb-continue-template"
                      data-testid="start-template-card"
                    >
                      <span className="wb-new-art" aria-hidden>
                        <span className="wb-new-art-bar" />
                      </span>
                      <b>{workspace.name}</b>
                      <small>
                        {t("workbench.start.templatePages", {
                          defaultValue: "{{pages}} pages · {{displays}} displays",
                          pages: workspace.pageCount,
                          displays: workspace.displayCount,
                        })}
                      </small>
                      <WbButton
                        variant="ghost"
                        size="sm"
                        disabled={forking === workspace.id}
                        onClick={() => void forkTemplate(workspace.id)}
                      >
                        {forking === workspace.id
                          ? t("workbench.start.forking", { defaultValue: "Forking…" })
                          : t("workbench.start.useTemplate", { defaultValue: "Use template" })}
                      </WbButton>
                    </article>
                  ))}
                </div>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.start.shortcuts", { defaultValue: "Shortcuts" })}
              meta={t("workbench.start.shortcutsMeta", { defaultValue: "Go" })}
            />
            <PanelBody flush>
              <ul className="wb-signals">
                <li>
                  <span className="wb-signal-icon is-green" aria-hidden>
                    <ShieldCheck size={13} />
                  </span>
                  <span>
                    <Link to={statsPath("quality")}>
                      <b>{t("workbench.start.shortcutQuality", { defaultValue: "Quality" })}</b>
                    </Link>
                    <small>
                      {t("workbench.start.shortcutQualityBody", {
                        defaultValue: "Maturity, evidence and next work.",
                      })}
                    </small>
                  </span>
                </li>
                <li>
                  <span className="wb-signal-icon is-amber" aria-hidden>
                    <Boxes size={13} />
                  </span>
                  <span>
                    <Link to={appRoutes.library}>
                      <b>{t("workbench.start.shortcutLibrary", { defaultValue: "Library" })}</b>
                    </Link>
                    <small>
                      {t("workbench.start.shortcutLibraryBody", {
                        defaultValue: "Assets, domains and verification state.",
                      })}
                    </small>
                  </span>
                </li>
                <li>
                  <span className="wb-signal-icon is-ice" aria-hidden>
                    <PenTool size={13} />
                  </span>
                  <span>
                    <Link to={appRoutes.design}>
                      <b>{t("workbench.start.shortcutDesign", { defaultValue: "Design" })}</b>
                    </Link>
                    <small>
                      {t("workbench.start.shortcutDesignBody", {
                        defaultValue: "Workspaces, pages and displays.",
                      })}
                    </small>
                  </span>
                </li>
              </ul>
            </PanelBody>
          </Panel>
        </div>
      </div>
    </div>
  );
}
