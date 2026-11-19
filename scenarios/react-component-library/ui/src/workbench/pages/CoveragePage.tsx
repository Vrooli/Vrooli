import { useMemo } from "react";
import { Link } from "react-router-dom";
import { ArrowUpRight, CircleAlert, RefreshCw } from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import {
  formatNumber,
  formatPercent,
  formatRelative,
  isCoverageWarming,
  useCatalog,
  useCoverage,
  useNextWork,
} from "../data";
import {
  Banner,
  Chip,
  Empty,
  Kpi,
  Meter,
  Panel,
  PanelBody,
  PanelHead,
  Skeleton,
  WbButton,
} from "../primitives";

const BANDS = [
  { id: "missing", label: "Missing" },
  { id: "scaffolded", label: "Scaffolded" },
  { id: "implemented", label: "Implemented" },
  { id: "verified", label: "Verified" },
  { id: "production-ready", label: "Production" },
] as const;

export function CoveragePage() {
  const { t } = useTranslation();
  const coverage = useCoverage();
  const nextWork = useNextWork(10);
  const catalog = useCatalog();
  const report = coverage.data;

  const matrix = useMemo(() => {
    if (!report) return [];
    const byDomain = new Map<string, { order: number; bands: Record<string, number>; total: number }>();
    for (const row of report.rows) {
      const domain = row.domain || "uncategorized";
      const entry = byDomain.get(domain) ?? {
        order: row.domainOrder || Number.MAX_SAFE_INTEGER,
        bands: { missing: 0, scaffolded: 0, implemented: 0, verified: 0, "production-ready": 0 },
        total: 0,
      };
      const band = BANDS.some((item) => item.id === row.achieved) ? row.achieved : "missing";
      entry.bands[band] = (entry.bands[band] ?? 0) + 1;
      entry.total += 1;
      byDomain.set(domain, entry);
    }
    return [...byDomain.entries()]
      .sort(([, left], [, right]) => left.order - right.order)
      .map(([domain, entry]) => ({ domain, ...entry }));
  }, [report]);

  const production = report?.maturity?.productionReadyCoverage;
  const nextRows = useMemo(() => {
    const fromService = nextWork.data?.rows ?? [];
    if (fromService.length > 0) return fromService;
    return [...(report?.rows ?? [])]
      .filter((row) => row.failedGates.length > 0)
      .sort((left, right) => left.assetScore - right.assetScore || right.blocksDownstream - left.blocksDownstream)
      .slice(0, 10);
  }, [nextWork.data?.rows, report?.rows]);

  const measuredAt = report
    ? report.rows
        .map((row) => (row.newestEvidence ? new Date(row.newestEvidence).getTime() : 0))
        .reduce((max, value) => Math.max(max, value), 0)
    : 0;

  if (!report && isCoverageWarming(coverage.error)) {
    return (
      <div className="wb-page" data-testid="coverage-page">
        <div className="wb-pagehead">
          <div className="wb-pagehead-copy">
            <p className="wb-kicker">
              {t("workbench.coverage.kicker", { defaultValue: "Library · Coverage" })}
            </p>
            <h1 className="wb-h1">
              {t("workbench.coverage.title", { defaultValue: "How mature is the library?" })}
            </h1>
          </div>
          <div className="wb-actions">
            <Chip tone="amber">{t("workbench.coverage.preparingChip", { defaultValue: "Preparing" })}</Chip>
          </div>
        </div>
        <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
          <b>{t("workbench.coverage.preparingTitle", { defaultValue: "Coverage is being prepared." })}</b>{" "}
          {t("workbench.coverage.preparingBody", {
            defaultValue:
              "The first full gate pass runs the TypeScript toolchain and the capture evidence join; it can take a few minutes on a cold cache. This page retries automatically. Nothing is rendered as zero while it waits.",
          })}
        </Banner>
        <Panel>
          <PanelHead
            title={t("workbench.coverage.preparingPanel", { defaultValue: "What is being measured" })}
            meta={t("workbench.coverage.preparingMeta", { defaultValue: "64 gates · 3 tiers" })}
          />
          <PanelBody>
            <p className="wb-muted">
              {t("workbench.coverage.preparingDetail", {
                defaultValue:
                  "Every catalog asset is checked against release integrity, asset quality and consumer use. The report stores which rung each asset achieved, which gates failed, and how much downstream work each failure blocks.",
              })}
            </p>
          </PanelBody>
        </Panel>
      </div>
    );
  }

  return (
    <div className="wb-page" data-testid="coverage-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t("workbench.coverage.kicker", { defaultValue: "Library · Coverage" })}</p>
          <h1 className="wb-h1">{t("workbench.coverage.title", { defaultValue: "How mature is the library?" })}</h1>
          <p className="wb-lede">
            {t("workbench.coverage.lede", {
              defaultValue:
                "Maturity distribution, freshness and ranked next work. A page that says “unavailable” is better than a page that implies zero.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          {report ? (
            <Chip tone="green">
              {t("workbench.coverage.measured", {
                defaultValue: "Measured {{age}}",
                age: formatRelative(measuredAt ? new Date(measuredAt) : undefined),
              })}
            </Chip>
          ) : (
            <Chip tone="amber">{t("workbench.coverage.notMeasured", { defaultValue: "Not measured" })}</Chip>
          )}
          <WbButton variant="ghost" size="md" onClick={() => void coverage.refetch()} disabled={coverage.isFetching}>
            <RefreshCw aria-hidden size={14} />
            {t("workbench.coverage.refresh", { defaultValue: "Refresh" })}
          </WbButton>
        </div>
      </div>

      {coverage.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
          <b>{t("workbench.coverage.unavailableTitle", { defaultValue: "Coverage unavailable." })}</b>{" "}
          {t("workbench.coverage.unavailableBody", {
            defaultValue:
              "The coverage service did not complete the report. No ratio, distribution or ranking is rendered as zero.",
          })}
        </Banner>
      ) : null}

      <div className="wb-kpis">
        <Kpi
          label={t("workbench.coverage.kpiFresh", { defaultValue: "Production ready" })}
          value={production ? formatPercent(production.ratio) : "—"}
          tone={production ? (production.ratio >= 0.5 ? "green" : production.ratio > 0 ? "amber" : "red") : undefined}
          detail={
            production
              ? t("workbench.coverage.kpiFreshDetail", {
                  defaultValue: "{{count}} of {{total}} assets at target",
                  count: production.numerator,
                  total: production.denominator,
                })
              : t("workbench.coverage.unavailableShort", { defaultValue: "Unavailable" })
          }
        />
        <Kpi
          label={t("workbench.coverage.kpiAssets", { defaultValue: "Assets" })}
          value={report ? formatNumber(report.maturity?.total ?? report.rows.length) : "—"}
          detail={t("workbench.coverage.kpiAssetsDetail", {
            defaultValue: "{{count}} indexed components",
            count: catalog.data?.length ?? 0,
          })}
        />
        <Kpi
          label={t("workbench.coverage.kpiAtTarget", { defaultValue: "At or above target" })}
          value={report ? formatNumber(report.maturity?.atOrAboveTarget ?? 0) : "—"}
          detail={t("workbench.coverage.kpiAtTargetDetail", {
            defaultValue: "{{percent}} of the catalog",
            percent: report?.maturity?.total
              ? formatPercent(report.maturity.atOrAboveTarget / report.maturity.total)
              : "—",
          })}
        />
        <Kpi
          label={t("workbench.coverage.kpiNext", { defaultValue: "Next work" })}
          value={nextRows.length}
          tone="signal"
          detail={t("workbench.coverage.kpiNextDetail", { defaultValue: "Ranked by blast radius" })}
        />
      </div>

      <div className="wb-coverage-layout">
        <div className="wb-stack">
          <Panel>
            <PanelHead
              title={t("workbench.coverage.matrix", { defaultValue: "Maturity by domain" })}
              meta={t("workbench.coverage.matrixMeta", {
                defaultValue: "{{count}} assets · {{bands}} bands",
                count: report?.rows.length ?? 0,
                bands: BANDS.length,
              })}
            />
            <PanelBody flush>
              {coverage.isLoading ? (
                <div style={{ padding: 14 }}>
                  <Skeleton height={180} />
                </div>
              ) : matrix.length === 0 ? (
                <Empty title={t("workbench.coverage.noMatrix", { defaultValue: "No maturity rows" })}>
                  {t("workbench.coverage.noMatrixBody", {
                    defaultValue: "The report returned no rows. That is not the same as a library with no assets.",
                  })}
                </Empty>
              ) : (
                <div className="wb-matrix-scroll">
                <div className="wb-matrix" data-testid="coverage-matrix">
                  <div className="wb-matrix-row">
                    <span />
                    {BANDS.map((band) => (
                      <span key={band.id} className="wb-matrix-head">
                        {band.label}
                      </span>
                    ))}
                  </div>
                  {matrix.map((row) => (
                    <div key={row.domain} className="wb-matrix-row">
                      <span className="wb-matrix-label">{row.domain}</span>
                      {BANDS.map((band) => {
                        const count = row.bands[band.id] ?? 0;
                        const level = BANDS.findIndex((item) => item.id === band.id) + 1;
                        return (
                          <span
                            key={band.id}
                            className={`wb-matrix-cell is-${count === 0 ? 0 : level}`}
                            title={`${row.domain} · ${band.label}: ${count}`}
                          >
                            {count}
                          </span>
                        );
                      })}
                    </div>
                  ))}
                </div>
                </div>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.coverage.nextWork", { defaultValue: "Next work" })}
              meta={nextWork.isError ? t("workbench.coverage.nextUnavailable", { defaultValue: "Service unavailable" }) : t("workbench.coverage.nextMeta", { defaultValue: "Ranked" })}
            />
            <PanelBody flush>
              {nextRows.length === 0 ? (
                <Empty title={t("workbench.coverage.noNext", { defaultValue: "No ranked work" })}>
                  {t("workbench.coverage.noNextBody", {
                    defaultValue: "Every asset with a failing gate is already accounted for, or the ranking is unavailable.",
                  })}
                </Empty>
              ) : (
                <div className="wb-nextwork">
                  {nextRows.map((row, index) => (
                    <div key={`${row.assetId}:${index}`} className="wb-nextwork-row">
                      <span className="wb-nextwork-rank">{String(index + 1).padStart(2, "0")}</span>
                      <span style={{ minWidth: 0, flex: 1 }}>
                        <b>{row.name}</b>
                        <small style={{ display: "block" }}>
                          {row.nearestBlockingGate || row.domain} · {row.achieved || "unmeasured"}
                        </small>
                      </span>
                      <Chip tone={row.blocksDownstream > 0 ? "amber" : "ice"}>
                        {row.blocksDownstream > 0
                          ? t("workbench.coverage.downstream", {
                              defaultValue: "{{count}} downstream",
                              count: row.blocksDownstream,
                            })
                          : t("workbench.coverage.leaf", { defaultValue: "leaf" })}
                      </Chip>
                      <span className="wb-mono wb-small">{row.assetScore.toFixed(1)}</span>
                    </div>
                  ))}
                </div>
              )}
            </PanelBody>
          </Panel>
        </div>

        <aside className="wb-stack">
          <Panel>
            <PanelHead
              title={t("workbench.coverage.readiness", { defaultValue: "Definition of ready" })}
              meta={t("workbench.coverage.gates", { defaultValue: "4 gates" })}
            />
            <PanelBody flush>
              <div className="wb-readiness">
                {[
                  t("workbench.coverage.ready1", { defaultValue: "Source and manifest agree on identity." }),
                  t("workbench.coverage.ready2", { defaultValue: "Every declared state has a story and a capture." }),
                  t("workbench.coverage.ready3", { defaultValue: "Usage-contract checks pass at phone width." }),
                  t("workbench.coverage.ready4", { defaultValue: "Evidence is fresher than the release." }),
                ].map((line, index) => (
                  <div key={line} className="wb-readiness-row">
                    <b>{index + 1}</b>
                    <span>{line}</span>
                  </div>
                ))}
              </div>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.coverage.freshness", { defaultValue: "Evidence" })}
              meta={t("workbench.coverage.live", { defaultValue: "Live" })}
            />
            <PanelBody>
              {report?.maturity ? (
                <div className="wb-stack tight">
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">{t("workbench.coverage.pass", { defaultValue: "Passing" })}</span>
                    <b className="wb-mono">{formatNumber(report.maturity.passEvidence)}</b>
                  </div>
                  <Meter value={(report.maturity.passEvidence / Math.max(1, report.maturity.passEvidence + report.maturity.failEvidence)) * 100} tone="green" />
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">{t("workbench.coverage.fail", { defaultValue: "Failing" })}</span>
                    <b className="wb-mono">{formatNumber(report.maturity.failEvidence)}</b>
                  </div>
                  <div className="wb-row-between">
                    <span className="wb-small wb-muted">{t("workbench.coverage.unmeasured", { defaultValue: "Unmeasured" })}</span>
                    <b className="wb-mono">{formatNumber(report.maturity.unmeasuredEvidence)}</b>
                  </div>
                  <Link to={appRoutes.review} className="wb-btn ghost sm wide" style={{ marginTop: 8 }}>
                    {t("workbench.coverage.openReview", { defaultValue: "Open review" })}
                    <ArrowUpRight aria-hidden size={13} />
                  </Link>
                </div>
              ) : (
                <p className="wb-small wb-muted">
                  {t("workbench.coverage.noEvidence", { defaultValue: "No evidence summary is available." })}
                </p>
              )}
            </PanelBody>
          </Panel>
        </aside>
      </div>
    </div>
  );
}
