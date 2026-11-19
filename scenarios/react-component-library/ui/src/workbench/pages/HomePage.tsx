import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  ArrowRight,
  ArrowUpRight,
  Boxes,
  CircleAlert,
  Clock3,
  Layers,
  PenTool,
  Plus,
  Search,
  ShieldCheck,
  Sparkles,
} from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import {
  assetAdoptions,
  assetGroup,
  assetVersion,
  formatNumber,
  formatPercent,
  formatRelative,
  useCapabilities,
  useCatalog,
  useCoverage,
  useDesignScenarios,
  findingsFromCoverage,
  timestampToDate,
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
import { cx } from "../cx";
import { TemplateThumb } from "../templates/TemplateThumb";
import { TEMPLATES } from "../templates/library";

function timeOf(asset: { updatedAt?: unknown }): number {
  return timestampToDate(asset.updatedAt)?.getTime() ?? 0;
}

export function HomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const catalog = useCatalog();
  const coverage = useCoverage();
  const capabilities = useCapabilities();
  const scenarios = useDesignScenarios();

  const assets = useMemo(() => catalog.data ?? [], [catalog.data]);
  const findings = useMemo(() => findingsFromCoverage(coverage.data), [coverage.data]);
  const report = coverage.data;

  const versionCount = useMemo(
    () => assets.reduce((sum, asset) => sum + (asset.metrics?.versionCount ?? 1), 0),
    [assets],
  );
  const freshRatio = report?.maturity?.productionReadyCoverage?.ratio ?? 0;
  const readyCount = report?.maturity?.productionReadyCoverage?.numerator ?? 0;

  const recent = useMemo(
    () => [...assets].sort((left, right) => timeOf(right) - timeOf(left)).slice(0, 5),
    [assets],
  );

  const popular = useMemo(
    () => [...assets].sort((left, right) => assetAdoptions(right) - assetAdoptions(left)).slice(0, 5),
    [assets],
  );

  const draftCount = assets.filter((asset) => asset.draftVersion).length;
  const pageCount = (scenarios.data ?? []).reduce((sum, entry) => sum + (entry.pageCount ?? 0), 0);
  const featuredScenario =
    (scenarios.data ?? []).find((entry) => entry.scenario === "react-component-library") ??
    (scenarios.data ?? []).find((entry) => (entry.pageCount ?? 0) > 0);

  const availableCapabilities = (capabilities.data?.states ?? []).filter(
    (state) => state.status === "available",
  ).length;
  const totalCapabilities = capabilities.data ? capabilities.data.states.length : 0;

  const submitSearch = (event: React.FormEvent) => {
    event.preventDefault();
    void navigate(`${appRoutes.components}?q=${encodeURIComponent(query.trim())}`);
  };

  return (
    <div className="wb-page" data-experience-surface="home" data-testid="home-page">
      <div className="wb-home-hero">
        <div>
          <p className="wb-kicker">
            {t("workbench.home.kicker", {
              defaultValue: "The shared UI layer · {{count}} assets",
              count: assets.length,
            })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.home.titleLead", { defaultValue: "Make every surface" })}{" "}
            <em>{t("workbench.home.titleEmphasis", { defaultValue: "feel inevitable." })}</em>
          </h1>
          <p className="wb-lede">
            {t("workbench.home.lede", {
              defaultValue:
                "A living library of versioned React assets, authored stories, usage contracts and captured evidence. Find the surface, inspect the proof, then shape it inside a real page.",
            })}
          </p>

          <form onSubmit={submitSearch} className="wb-home-search">
            <Search aria-hidden size={17} style={{ color: "var(--wb-muted-2)", flex: "none" }} />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              aria-label={t("workbench.home.searchLabel", { defaultValue: "Search the library" })}
              placeholder={t("workbench.home.searchPlaceholder", {
                defaultValue: "Search components, props, pages, or paste a component id",
              })}
            />
            <kbd className="wb-kbd">{t("workbench.home.searchShortcut", { defaultValue: "⌘ K" })}</kbd>
          </form>

          <div className="wb-home-actions">
            <WbButton variant="primary" size="md" onClick={() => void navigate(appRoutes.components)}>
              {t("workbench.home.browse", { defaultValue: "Browse components" })}
              <ArrowRight aria-hidden size={15} />
            </WbButton>
            <WbButton variant="ghost" size="md" onClick={() => void navigate(appRoutes.design)}>
              <PenTool aria-hidden size={15} />
              {t("workbench.home.openStudio", { defaultValue: "Open design studio" })}
            </WbButton>
          </div>

          <div className="wb-home-chips">
            {popular.map((asset) => (
              <Link key={asset.id} to={`/assets/${encodeURIComponent(asset.id)}`}>
                <Chip>{asset.displayName}</Chip>
              </Link>
            ))}
            <Link to={appRoutes.review}>
              <Chip tone={findings.length > 0 ? "amber" : "green"}>
                {t("workbench.home.needsReview", {
                  defaultValue: "{{count}} need review",
                  count: findings.length,
                })}
              </Chip>
            </Link>
          </div>
        </div>

        <div className="wb-specimen">
          <div className="wb-specimen-stage">
            <div className="wb-specimen-label">
              <span className="wb-chip is-green">
                <Dot tone="green" />
                {t("workbench.home.liveSpecimen", { defaultValue: "Live specimen" })}
              </span>
              <span className="wb-tag">{t("workbench.home.buttonAsset", { defaultValue: "Button · 2.2.14" })}</span>
            </div>
            <Card className="wb-specimen-card">
              <CardHeader>
                <CardTitle>
                  <span className="wb-specimen-icon" aria-hidden>
                    <Sparkles size={15} />
                  </span>
                  {t("workbench.home.specimenTitle", { defaultValue: "Publish this version" })}
                </CardTitle>
                <CardDescription>
                  {t("workbench.home.specimenBody", {
                    defaultValue:
                      "Rules, stories and evidence are ready. One primary action, a 44px touch target and a stable width through pending.",
                  })}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="wb-row" style={{ flexWrap: "wrap" }}>
                  <WbButton variant="primary" size="md">
                    {t("workbench.home.specimenAction", { defaultValue: "Publish" })}
                    <ArrowRight aria-hidden size={14} />
                  </WbButton>
                  <WbButton variant="ghost" size="md">
                    {t("workbench.home.specimenReview", { defaultValue: "Review" })}
                  </WbButton>
                  <span className="wb-muted-2 wb-small" style={{ marginLeft: "auto" }}>
                    {t("workbench.home.specimenChecks", { defaultValue: "7/7 checks" })}
                  </span>
                </div>
              </CardContent>
            </Card>
            <div className="wb-specimen-target">
              <span className="wb-tag">{t("workbench.home.target", { defaultValue: "Target 44 × 44" })}</span>
            </div>
          </div>
          <p className="wb-specimen-caption">
            {t("workbench.home.specimenCaption", {
              defaultValue:
                "Rendered from the shared Button asset inside the workbench — the same contract consumers receive.",
            })}
          </p>
        </div>
      </div>

      <div className="wb-kpis">
        <Kpi
          label={t("workbench.home.kpiAssets", { defaultValue: "Indexed assets" })}
          value={catalog.isLoading ? "—" : formatNumber(assets.length)}
          detail={t("workbench.home.kpiAssetsDetail", {
            defaultValue: "{{count}} drafts in flight",
            count: draftCount,
          })}
        />
        <Kpi
          label={t("workbench.home.kpiVersions", { defaultValue: "Versioned sources" })}
          value={catalog.isLoading ? "—" : formatNumber(versionCount)}
          detail={t("workbench.home.kpiVersionsDetail", {
            defaultValue: "{{count}} declared design pages",
            count: pageCount,
          })}
        />
        <Kpi
          label={t("workbench.home.kpiEvidence", { defaultValue: "Production ready" })}
          value={report ? formatPercent(freshRatio) : coverage.isError ? "—" : "…"}
          tone={report ? (freshRatio >= 0.8 ? "green" : "amber") : undefined}
          detail={
            report
              ? t("workbench.home.kpiEvidenceDetail", {
                  defaultValue: "{{count}} assets at target",
                  count: readyCount,
                })
              : t("workbench.home.kpiEvidenceUnavailable", { defaultValue: "Coverage unavailable" })
          }
        />
        <Kpi
          label={t("workbench.home.kpiFindings", { defaultValue: "Open findings" })}
          value={coverage.isError ? "—" : formatNumber(findings.length)}
          tone={findings.some((finding) => finding.severity === "blocking") ? "red" : "amber"}
          detail={t("workbench.home.kpiFindingsDetail", {
            defaultValue: "{{count}} blocking",
            count: findings.filter((finding) => finding.severity === "blocking").length,
          })}
        />
      </div>

      {coverage.isError ? (
        <div style={{ marginBottom: 18 }}>
          <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
            <b>{t("workbench.home.coverageDownTitle", { defaultValue: "Coverage service did not complete." })}</b>{" "}
            {t("workbench.home.coverageDownBody", {
              defaultValue:
                "Health, findings and maturity are shown as unavailable rather than zero. Retry from the Coverage page.",
            })}
          </Banner>
        </div>
      ) : null}

      <div className="wb-home-grid">
        <div className="wb-stack">
          <Panel>
            <PanelHead
              title={t("workbench.home.recent", { defaultValue: "Recently touched" })}
              meta={t("workbench.home.recentMeta", { defaultValue: "Continue where you left off" })}
              actions={
                <Link to={appRoutes.components} className="wb-btn quiet sm">
                  {t("workbench.home.viewAll", { defaultValue: "View all" })}
                  <ArrowUpRight aria-hidden size={13} />
                </Link>
              }
            />
            <PanelBody flush>
              {catalog.isLoading ? (
                <div className="wb-stack" style={{ padding: 16 }}>
                  <Skeleton height={42} />
                  <Skeleton height={42} />
                  <Skeleton height={42} />
                </div>
              ) : recent.length === 0 ? (
                <Empty title={t("workbench.home.noAssets", { defaultValue: "No assets indexed yet" })}>
                  {t("workbench.home.noAssetsBody", {
                    defaultValue: "Create the first component or reindex the library to begin.",
                  })}
                </Empty>
              ) : (
                <ul className="wb-home-list">
                  {recent.map((asset) => (
                    <li key={asset.id}>
                      <Link to={`/assets/${encodeURIComponent(asset.id)}`} className="wb-home-row">
                        <span className="wb-home-thumb" aria-hidden>
                          <Boxes size={14} />
                        </span>
                        <span style={{ minWidth: 0, flex: 1 }}>
                          <b>{asset.displayName}</b>
                          <small>{asset.catalogId || asset.libraryId}</small>
                        </span>
                        <span className="wb-mono wb-small wb-muted">{assetVersion(asset)}</span>
                        <Chip tone={asset.draftVersion ? "amber" : "green"}>
                          {asset.draftVersion
                            ? t("workbench.home.draft", { defaultValue: "Draft" })
                            : t("workbench.home.released", { defaultValue: "Released" })}
                        </Chip>
                        <span className="wb-muted-2 wb-small">
                          {formatRelative(new Date(timeOf(asset)))}
                        </span>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.home.continue", { defaultValue: "Continue designing" })}
              meta={t("workbench.home.continueMeta", { defaultValue: "{{count}} pages", count: pageCount })}
            />
            <PanelBody>
              {scenarios.isError ? (
                <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                  {t("workbench.home.designUnavailable", {
                    defaultValue:
                      "The design page inventory is unavailable. Open the studio to retry the page service.",
                  })}
                </Banner>
              ) : (
                <div className="wb-row" style={{ alignItems: "stretch", gap: 14 }}>
                  <div className="wb-continue-card">
                    <span className="wb-caption">
                      {t("workbench.home.lastScenario", { defaultValue: "Scenario" })}
                    </span>
                    <b>{featuredScenario?.scenario ?? t("workbench.home.noScenario", { defaultValue: "No pages yet" })}</b>
                    <span className="wb-muted wb-small">
                      {t("workbench.home.pageCount", {
                        defaultValue: "{{count}} declared pages ready to compose",
                        count: featuredScenario?.pageCount ?? 0,
                      })}
                    </span>
                    <WbButton
                      variant="primary"
                      size="sm"
                      onClick={() =>
                        void navigate(
                          featuredScenario
                            ? `/design/${encodeURIComponent(featuredScenario.scenario)}`
                            : appRoutes.design,
                        )
                      }
                    >
                      {t("workbench.home.resume", { defaultValue: "Resume studio" })}
                      <ArrowRight aria-hidden size={14} />
                    </WbButton>
                  </div>
                  <div className="wb-continue-templates">
                    {TEMPLATES.slice(0, 2).map((template) => (
                      <Link
                        key={template.id}
                        to={`${appRoutes.design}?template=${template.id}`}
                        className="wb-continue-template"
                      >
                        <TemplateThumb template={template} />
                        <span>{template.name}</span>
                      </Link>
                    ))}
                  </div>
                </div>
              )}
            </PanelBody>
          </Panel>
        </div>

        <div className="wb-stack">
          <Panel>
            <PanelHead
              title={t("workbench.home.health", { defaultValue: "Library health" })}
              meta={t("workbench.home.healthMeta", { defaultValue: "Measured" })}
            />
            <PanelBody>
              {report ? (
                <div className="wb-health">
                  <Ring value={freshRatio * 100} size={76} tone={freshRatio >= 0.8 ? "green" : "amber"} />
                  <div className="wb-health-copy">
                    <b>{t("workbench.home.healthTitle", { defaultValue: "Production coverage" })}</b>
                    <span>
                      {t("workbench.home.healthBody", {
                        defaultValue:
                          "{{ready}} of {{total}} assets meet their declared target rung.",
                        ready: readyCount,
                        total: report.maturity?.total ?? assets.length,
                      })}
                    </span>
                    <Link to={appRoutes.coverage} className="wb-health-link">
                      {t("workbench.home.openCoverage", { defaultValue: "Open coverage" })}
                      <ArrowUpRight aria-hidden size={12} />
                    </Link>
                  </div>
                </div>
              ) : coverage.isLoading ? (
                <Skeleton height={76} />
              ) : (
                <Empty title={t("workbench.home.coverageUnavailable", { defaultValue: "Coverage unavailable" })}>
                  {t("workbench.home.coverageUnavailableBody", {
                    defaultValue: "The service did not complete the report. Nothing is rendered as zero.",
                  })}
                </Empty>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.home.signals", { defaultValue: "Signals" })}
              meta={t("workbench.home.signalsMeta", { defaultValue: "Today" })}
            />
            <PanelBody flush>
              <ul className="wb-signals">
                <li>
                  <span className="wb-signal-icon is-green" aria-hidden>
                    <ShieldCheck size={13} />
                  </span>
                  <span>
                    <b>
                      {t("workbench.home.signalCapabilities", {
                        defaultValue: "{{available}}/{{total}} capabilities available",
                        available: availableCapabilities,
                        total: totalCapabilities || 0,
                      })}
                    </b>
                    <small>
                      {t("workbench.home.signalCapabilitiesBody", {
                        defaultValue: "Agent Manager, Experience Manager and UI Health are optional.",
                      })}
                    </small>
                  </span>
                </li>
                <li>
                  <span className="wb-signal-icon is-amber" aria-hidden>
                    <Layers size={13} />
                  </span>
                  <span>
                    <b>
                      {t("workbench.home.signalDrafts", {
                        defaultValue: "{{count}} assets have a draft version",
                        count: draftCount,
                      })}
                    </b>
                    <small>
                      {t("workbench.home.signalDraftsBody", {
                        defaultValue: "Drafts are not releases; publish from the asset workspace.",
                      })}
                    </small>
                  </span>
                </li>
                <li>
                  <span className="wb-signal-icon is-ice" aria-hidden>
                    <Clock3 size={13} />
                  </span>
                  <span>
                    <b>
                      {t("workbench.home.signalRecent", {
                        defaultValue: "{{count}} assets touched recently",
                        count: recent.length,
                      })}
                    </b>
                    <small>
                      {t("workbench.home.signalRecentBody", {
                        defaultValue: "Ordered by the catalog's own update timestamp.",
                      })}
                    </small>
                  </span>
                </li>
              </ul>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.home.collection", { defaultValue: "New in the collection" })}
              meta={`${assets.filter((asset) => assetGroup(asset) === "components").length}`}
            />
            <PanelBody>
              {assets.length === 0 ? (
                <Empty title={t("workbench.home.emptyCollection", { defaultValue: "The collection is empty" })}>
                  {t("workbench.home.emptyCollectionBody", {
                    defaultValue: "Create a component, extract one from a scenario, or adopt an asset.",
                  })}
                </Empty>
              ) : (
                <div className="wb-new-grid">
                  {assets.slice(0, 4).map((asset) => (
                    <Link
                      key={asset.id}
                      to={`/assets/${encodeURIComponent(asset.id)}`}
                      className="wb-new-card"
                    >
                      <span className="wb-new-art" aria-hidden>
                        <span className={cx("wb-new-art-bar", asset.draftVersion && "is-draft")} />
                      </span>
                      <b>{asset.displayName}</b>
                      <small>{asset.catalogId || asset.libraryId}</small>
                    </Link>
                  ))}
                </div>
              )}
              <Link to={appRoutes.create} className="wb-create-link">
                <Plus aria-hidden size={14} />
                {t("workbench.home.addAsset", { defaultValue: "Add to the library" })}
              </Link>
            </PanelBody>
          </Panel>
        </div>
      </div>
    </div>
  );
}
