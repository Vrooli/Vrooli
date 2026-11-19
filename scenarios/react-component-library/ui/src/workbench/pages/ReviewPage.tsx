import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { ArrowRight, ArrowUpRight, CircleAlert, Copy, Filter, Inbox } from "lucide-react";

import { useTranslation } from "../../i18n";
import {
  assetVersion,
  findingsFromCoverage,
  findingStats,
  formatRelative,
  useCatalog,
  useCoverage,
  type Finding,
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
  Skeleton,
  WbButton,
} from "../primitives";

export function ReviewPage() {
  const { t } = useTranslation();
  const coverage = useCoverage();
  const catalog = useCatalog();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const findings = useMemo(() => findingsFromCoverage(coverage.data), [coverage.data]);
  const stats = useMemo(() => findingStats(findings), [findings]);
  const selected = findings.find((finding) => finding.id === selectedId) ?? findings[0];

  const assetByCatalogId = useMemo(() => {
    const map = new Map<string, { id: string; version: string }>();
    for (const asset of catalog.data ?? []) {
      map.set(asset.catalogId || asset.libraryId, { id: asset.id, version: assetVersion(asset) });
    }
    return map;
  }, [catalog.data]);

  const unmeasured = useMemo(
    () =>
      (coverage.data?.rows ?? [])
        .filter((row) => row.failedGates.length === 0 && !row.achieved)
        .slice(0, 6),
    [coverage.data],
  );

  const copyEvidence = async (finding: Finding) => {
    const text = `${finding.assetName} · ${finding.gate} · ${finding.severity} · score ${finding.score.toFixed(2)} · blocks ${finding.blocksDownstream}`;
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  if (coverage.isLoading) {
    return (
      <div className="wb-page">
        <Skeleton height={40} width={340} />
        <div style={{ height: 18 }} />
        <Skeleton height={90} />
        <div style={{ height: 18 }} />
        <Skeleton height={420} />
      </div>
    );
  }

  return (
    <div className="wb-page" data-testid="review-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t("workbench.review.kicker", { defaultValue: "Library · Review" })}</p>
          <h1 className="wb-h1">{t("workbench.review.title", { defaultValue: "What needs a person" })}</h1>
          <p className="wb-lede">
            {t("workbench.review.lede", {
              defaultValue:
                "Findings that block a release, stale evidence, and captures waiting for a decision. Ordered by what will hurt most if ignored.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <span className="wb-mono" style={{ fontSize: 26, letterSpacing: "-0.03em" }}>
            {findings.length}
          </span>
          <span className="wb-muted wb-small">{t("workbench.review.open", { defaultValue: "open findings" })}</span>
          <WbButton variant="ghost" size="md">
            <Filter aria-hidden size={14} />
            {t("workbench.review.filter", { defaultValue: "Filter" })}
          </WbButton>
        </div>
      </div>

      {coverage.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
          <b>{t("workbench.review.unavailableTitle", { defaultValue: "The findings projection did not complete." })}</b>{" "}
          {t("workbench.review.unavailableBody", {
            defaultValue:
              "The coverage service is the only real owner of gate outcomes today. Review shows unavailable rather than an empty queue.",
          })}
        </Banner>
      ) : null}

      <div className="wb-kpis">
        <Kpi label={t("workbench.review.kpiOpen", { defaultValue: "Open findings" })} value={findings.length} detail={t("workbench.review.kpiOpenDetail", { defaultValue: "Across {{count}} assets", count: stats.affectedAssets })} />
        <Kpi label={t("workbench.review.kpiBlocking", { defaultValue: "Blocking" })} value={stats.blocking} tone="red" detail={t("workbench.review.kpiBlockingDetail", { defaultValue: "Blocks a release" })} />
        <Kpi label={t("workbench.review.kpiWarning", { defaultValue: "Needs a decision" })} value={stats.warning} tone="amber" detail={t("workbench.review.kpiWarningDetail", { defaultValue: "Advisory gate failures" })} />
        <Kpi label={t("workbench.review.kpiUnmeasured", { defaultValue: "Unmeasured" })} value={unmeasured.length} tone="signal" detail={t("workbench.review.kpiUnmeasuredDetail", { defaultValue: "No achieved rung" })} />
      </div>

      <div className="wb-review-layout">
        <Panel>
          <PanelHead
            title={t("workbench.review.queue", { defaultValue: "Review queue" })}
            meta={t("workbench.review.queueMeta", { defaultValue: "Sorted by blast radius" })}
          />
          <PanelBody flush>
            {findings.length === 0 && !coverage.isError ? (
              <Empty title={t("workbench.review.emptyTitle", { defaultValue: "Nothing needs a decision" })}>
                {t("workbench.review.emptyBody", {
                  defaultValue:
                    "No asset has a failing gate in the current report. The empty state names what was checked; it is not a substitute for a missing report.",
                })}
              </Empty>
            ) : (
              <div className="wb-board" style={{ padding: 12 }}>
                <BoardColumn
                  title={t("workbench.review.blocking", { defaultValue: "Blocking" })}
                  findings={findings.filter((finding) => finding.severity === "blocking").slice(0, 30)}
                  total={findings.filter((finding) => finding.severity === "blocking").length}
                  selectedId={selected?.id}
                  onSelect={setSelectedId}
                />
                <BoardColumn
                  title={t("workbench.review.decision", { defaultValue: "Needs a decision" })}
                  findings={findings.filter((finding) => finding.severity === "warning").slice(0, 30)}
                  total={findings.filter((finding) => finding.severity === "warning").length}
                  selectedId={selected?.id}
                  onSelect={setSelectedId}
                />
                <div className="wb-board-col">
                  <div className="wb-board-head">
                    <span>{t("workbench.review.unmeasuredColumn", { defaultValue: "Unmeasured" })}</span>
                    <span>{unmeasured.length}</span>
                  </div>
                  {unmeasured.length === 0 ? (
                    <p className="wb-muted-2 wb-small" style={{ padding: 6 }}>
                      {t("workbench.review.noUnmeasured", { defaultValue: "Every asset has an achieved rung." })}
                    </p>
                  ) : (
                    unmeasured.map((row) => (
                      <div key={row.assetId} className="wb-finding">
                        <b>{row.name}</b>
                        <small>
                          {row.domain} · {row.rungName || "no rung"}
                        </small>
                        <div className="wb-finding-meta">
                          <Chip tone="ice">{t("workbench.review.unmeasuredChip", { defaultValue: "unmeasured" })}</Chip>
                          <span className="wb-tag">{row.target || "no target"}</span>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
            {findings.length > 60 ? (
              <p className="wb-muted-2 wb-small" style={{ padding: "0 14px 12px" }}>
                {t("workbench.review.truncated", {
                  defaultValue:
                    "Showing the first 30 findings per column. {{count}} findings are in the current report.",
                  count: findings.length,
                })}
              </p>
            ) : null}
          </PanelBody>
        </Panel>

        <aside className="wb-selected">
          {selected ? (
            <>
              <div className="wb-row-between">
                <span className="wb-caption">{t("workbench.review.selected", { defaultValue: "Selected finding" })}</span>
                <span className="wb-tag">
                  {findings.findIndex((finding) => finding.id === selected.id) + 1} / {findings.length}
                </span>
              </div>
              <h2>{selected.title}</h2>
              <div className="wb-row" style={{ flexWrap: "wrap" }}>
                <Chip tone={selected.severity === "blocking" ? "red" : "amber"}>
                  <Dot tone={selected.severity === "blocking" ? "red" : "amber"} />
                  {selected.severity}
                </Chip>
                <Chip>{selected.gate}</Chip>
                <span className="wb-small wb-muted-2">
                  {formatRelative(selected.newestEvidence ? new Date(selected.newestEvidence) : undefined)}
                </span>
              </div>

              <div>
                <span className="wb-caption">{t("workbench.review.why", { defaultValue: "Why it matters" })}</span>
                <p className="wb-small wb-muted" style={{ marginTop: 4, lineHeight: 1.6 }}>
                  {t("workbench.review.whyBody", {
                    defaultValue:
                      "{{asset}} fails the {{gate}} gate. It blocks {{count}} downstream assets and currently sits at {{achieved}} against a {{target}} target.",
                    asset: selected.assetName,
                    gate: selected.gate,
                    count: selected.blocksDownstream,
                    achieved: selected.achieved || "no achieved rung",
                    target: selected.target || "declared",
                  })}
                  {selected.failedCount > 1
                    ? ` ${t("workbench.review.secondaryGates", {
                        defaultValue: "{{count}} additional gates also fail on this asset.",
                        count: selected.failedCount - 1,
                      })}`
                    : ""}
                </p>
              </div>

              <div>
                <span className="wb-caption">{t("workbench.review.evidence", { defaultValue: "Evidence" })}</span>
                <div className="wb-evidence" style={{ marginTop: 6 }}>
                  <div className="wb-evidence-row">
                    <Inbox aria-hidden size={13} />
                    <span>{t("workbench.review.coverageReport", { defaultValue: "{{gate}} · coverage report", gate: selected.gate })}</span>
                    <span className="wb-tag">{selected.severity}</span>
                  </div>
                  <div className="wb-evidence-row">
                    <ArrowUpRight aria-hidden size={13} />
                    <span>{selected.assetName} · {selected.achieved || "unmeasured"}</span>
                    <span className="wb-tag">{selected.rungName || "no rung"}</span>
                  </div>
                </div>
              </div>

              <div>
                <span className="wb-caption">{t("workbench.review.affected", { defaultValue: "Affected asset" })}</span>
                <div className="wb-row-between" style={{ marginTop: 4 }}>
                  <span className="wb-small">{selected.assetName}</span>
                  {assetByCatalogId.get(selected.assetId) ? (
                    <Link
                      to={`/assets/${encodeURIComponent(assetByCatalogId.get(selected.assetId)?.id ?? "")}`}
                      className="wb-btn ghost sm"
                    >
                      {t("workbench.review.openComponent", { defaultValue: "Open component" })}
                      <ArrowRight aria-hidden size={13} />
                    </Link>
                  ) : (
                    <span className="wb-small wb-muted-2">
                      {t("workbench.review.notIndexed", { defaultValue: "Not in the catalog index" })}
                    </span>
                  )}
                </div>
              </div>

              <div className="wb-stack tight">
                <WbButton variant="ghost" size="sm" onClick={() => void copyEvidence(selected)}>
                  <Copy aria-hidden size={13} />
                  {copied
                    ? t("workbench.review.copied", { defaultValue: "Copied" })
                    : t("workbench.review.copyEvidence", { defaultValue: "Copy evidence" })}
                </WbButton>
                <WbButton variant="ghost" size="sm" disabled>
                  {t("workbench.review.assign", { defaultValue: "Assign to agent" })}
                </WbButton>
                <p className="wb-muted-2 wb-small">
                  {t("workbench.review.authorityNote", {
                    defaultValue:
                      "Triage and assignment become available when the findings projection owns a disposition. The workbench does not fake a write.",
                  })}
                </p>
              </div>
            </>
          ) : (
            <Empty title={t("workbench.review.noSelection", { defaultValue: "No finding selected" })}>
              {t("workbench.review.noSelectionBody", {
                defaultValue: "Select a finding from the queue to see its evidence and affected asset.",
              })}
            </Empty>
          )}
        </aside>
      </div>

      <div style={{ height: 18 }} />
      <Panel>
        <PanelHead
          title={t("workbench.review.route", { defaultValue: "What the queue reads" })}
          meta={t("workbench.review.routeMeta", { defaultValue: "Real owners only" })}
        />
        <PanelBody>
          <div className="wb-grid-3">
            <div>
              <span className="wb-caption">{t("workbench.review.ownerGates", { defaultValue: "Gate outcomes" })}</span>
              <p className="wb-small wb-muted">
                {t("workbench.review.ownerGatesBody", {
                  defaultValue: "Failed gates and the nearest blocking gate come from the coverage report.",
                })}
              </p>
            </div>
            <div>
              <span className="wb-caption">{t("workbench.review.ownerEvidence", { defaultValue: "Evidence age" })}</span>
              <p className="wb-small wb-muted">
                {t("workbench.review.ownerEvidenceBody", {
                  defaultValue: "The newest evidence timestamp is shown; a missing capture stays unmeasured.",
                })}
              </p>
            </div>
            <div>
              <span className="wb-caption">{t("workbench.review.ownerBlast", { defaultValue: "Blast radius" })}</span>
              <p className="wb-small wb-muted">
                {t("workbench.review.ownerBlastBody", {
                  defaultValue: "Downstream impact is the catalog's computed blocks-downstream count.",
                })}
              </p>
            </div>
          </div>
        </PanelBody>
      </Panel>
    </div>
  );
}

function BoardColumn({
  title,
  findings,
  total,
  selectedId,
  onSelect,
}: {
  title: string;
  findings: Finding[];
  total: number;
  selectedId?: string;
  onSelect: (id: string) => void;
}) {
  return (
    <div className="wb-board-col">
      <div className="wb-board-head">
        <span>{title}</span>
        <span>{total}</span>
      </div>
      {findings.length === 0 ? (
        <p className="wb-muted-2 wb-small" style={{ padding: 6 }}>
          —
        </p>
      ) : (
        findings.map((finding) => (
          <button
            key={finding.id}
            type="button"
            className="wb-finding"
            aria-pressed={selectedId === finding.id}
            onClick={() => onSelect(finding.id)}
          >
            <b>{finding.title}</b>
            <small>
              {finding.assetName} · {finding.domain}
            </small>
            <div className="wb-finding-meta">
              <Chip tone={finding.severity === "blocking" ? "red" : "amber"}>{finding.severity}</Chip>
              <span className="wb-tag">{finding.gate}</span>
              {finding.failedCount > 1 ? (
                <span className="wb-tag">{`+${finding.failedCount - 1}`}</span>
              ) : null}
              <span className="wb-tag">
                {finding.blocksDownstream > 0
                  ? `${finding.blocksDownstream} downstream`
                  : "leaf"}
              </span>
            </div>
          </button>
        ))
      )}
    </div>
  );
}
