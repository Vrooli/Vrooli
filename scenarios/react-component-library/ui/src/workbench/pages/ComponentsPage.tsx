import { useDeferredValue, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  Boxes,
  CircleAlert,
  Filter,
  Grid2X2,
  List,
  Network,
  Plus,
  SlidersHorizontal,
} from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import type { CatalogAsset, CoverageRow } from "../data";
import {
  assetAdoptions,
  assetDomain,
  assetGroup,
  assetHasDraft,
  assetVersion,
  findingsFromCoverage,
  formatNumber,
  formatRelative,
  useCatalog,
  useCoverage,
  timestampToDate,
  type AssetGroup,
} from "../data";
import {
  Banner,
  Chip,
  Dot,
  Empty,
  Meter,
  Panel,
  PanelBody,
  PanelHead,
  Ring,
  SearchBox,
  Segmented,
  Select,
  Skeleton,
  Tabs,
  WbButton,
} from "../primitives";

type View = "gallery" | "list" | "tree";
type StatusFilter = "all" | "review" | "verified" | "drafts";

function previewURL(asset: CatalogAsset): string {
  const version = asset.latestVersion || asset.version || asset.draftVersion || "";
  return `/preview/${encodeURIComponent(asset.libraryId)}/preview.png?version=${encodeURIComponent(version)}`;
}

function coverageIndex(rows: CoverageRow[] | undefined): Map<string, CoverageRow> {
  const index = new Map<string, CoverageRow>();
  for (const row of rows ?? []) {
    index.set(row.assetId, row);
    index.set(row.name, row);
  }
  return index;
}

function assetStatus(asset: CatalogAsset, row: CoverageRow | undefined): {
  label: string;
  tone: "green" | "amber" | "red" | "ice";
} {
  if (assetHasDraft(asset)) return { label: "Draft", tone: "amber" };
  if (!row) return { label: "Unmeasured", tone: "ice" };
  if (row.failedGates.length > 0) {
    return {
      label: "Needs review",
      tone: row.blocksDownstream > 0 ? "red" : "amber",
    };
  }
  if (row.achieved) return { label: "Verified", tone: "green" };
  return { label: "Indexed", tone: "ice" };
}

export function ComponentsPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const catalog = useCatalog();
  const coverage = useCoverage();
  const [query, setQuery] = useState(searchParams.get("q") ?? "");
  const deferredQuery = useDeferredValue(query);
  const [group, setGroup] = useState<AssetGroup>("components");
  const [view, setView] = useState<View>("gallery");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [domain, setDomain] = useState("all");
  const [sort, setSort] = useState<"attention" | "name" | "recent" | "adoptions">("attention");
  const [limit, setLimit] = useState(60);

  const rows = useMemo(() => coverageIndex(coverage.data?.rows), [coverage.data]);
  const findings = useMemo(() => findingsFromCoverage(coverage.data), [coverage.data]);
  const reviewAssets = useMemo(() => new Set(findings.map((finding) => finding.assetId)), [findings]);

  const assets = useMemo(() => catalog.data ?? [], [catalog.data]);
  const grouped = useMemo(() => {
    const counts: Record<AssetGroup, number> = { components: 0, hooks: 0, support: 0 };
    for (const asset of assets) counts[assetGroup(asset)] += 1;
    return counts;
  }, [assets]);

  const domains = useMemo(() => {
    const counts = new Map<string, number>();
    for (const asset of assets) {
      if (assetGroup(asset) !== group) continue;
      const key = assetDomain(asset);
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    return [...counts.entries()].sort(([left], [right]) => left.localeCompare(right));
  }, [assets, group]);

  const filtered = useMemo(() => {
    const term = deferredQuery.trim().toLowerCase();
    const result = assets.filter((asset) => {
      if (assetGroup(asset) !== group) return false;
      if (domain !== "all" && assetDomain(asset) !== domain) return false;
      const row = rows.get(asset.catalogId) ?? rows.get(asset.libraryId);
      if (status === "drafts" && !assetHasDraft(asset)) return false;
      if (status === "review" && !reviewAssets.has(asset.catalogId) && !(row?.failedGates ?? []).length) {
        return false;
      }
      if (status === "verified") {
        if (!row || row.failedGates.length > 0 || !row.achieved) return false;
      }
      if (!term) return true;
      return (
        asset.displayName.toLowerCase().includes(term) ||
        asset.libraryId.toLowerCase().includes(term) ||
        asset.catalogId.toLowerCase().includes(term) ||
        asset.description.toLowerCase().includes(term)
      );
    });
    result.sort((left, right) => {
      if (sort === "name") return left.displayName.localeCompare(right.displayName);
      if (sort === "adoptions") return assetAdoptions(right) - assetAdoptions(left);
      if (sort === "recent") {
        const leftTime = left.updatedAt as { toDate?: () => Date } | undefined;
        const rightTime = right.updatedAt as { toDate?: () => Date } | undefined;
        return (rightTime?.toDate?.().getTime() ?? 0) - (leftTime?.toDate?.().getTime() ?? 0);
      }
      const leftRow = rows.get(left.catalogId) ?? rows.get(left.libraryId);
      const rightRow = rows.get(right.catalogId) ?? rows.get(right.libraryId);
      const leftFail = (leftRow?.failedGates ?? []).length;
      const rightFail = (rightRow?.failedGates ?? []).length;
      return rightFail - leftFail || (rightRow?.blocksDownstream ?? 0) - (leftRow?.blocksDownstream ?? 0);
    });
    return result;
  }, [assets, deferredQuery, domain, group, reviewAssets, rows, sort, status]);

  const visible = useMemo(() => filtered.slice(0, limit), [filtered, limit]);

  const collections = useMemo(
    () =>
      [...domains]
        .map(([name, count]) => ({ name, count }))
        .sort((left, right) => right.count - left.count)
        .slice(0, 7),
    [domains],
  );
  const maxCollection = Math.max(1, ...collections.map((entry) => entry.count));

  const verifiedRatio = useMemo(() => {
    const eligible = assets.filter((asset) => assetGroup(asset) === "components");
    if (eligible.length === 0) return 0;
    const verified = eligible.filter((asset) => {
      const row = rows.get(asset.catalogId) ?? rows.get(asset.libraryId);
      return Boolean(row && row.failedGates.length === 0 && row.achieved);
    }).length;
    return verified / eligible.length;
  }, [assets, rows]);

  return (
    <div className="wb-page" data-experience-surface="catalog-results" data-testid="components-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t("workbench.components.kicker", { defaultValue: "Library · Components" })}</p>
          <h1 className="wb-h1">{t("workbench.components.title", { defaultValue: "Components" })}</h1>
          <p className="wb-lede">
            {t("workbench.components.lede", {
              defaultValue:
                "{{count}} published assets across {{domains}} domains. Search the surface, inspect the evidence, or open a component into a real page.",
              count: assets.length,
              domains: new Set(assets.map(assetDomain)).size,
            })}
          </p>
        </div>
        <div className="wb-actions">
          <WbButton variant="ghost" size="md">
            <Filter aria-hidden size={14} />
            {t("workbench.components.filters", { defaultValue: "Filters" })}
          </WbButton>
          <WbButton variant="primary" size="md" onClick={() => void navigate(appRoutes.create)}>
            <Plus aria-hidden size={14} />
            {t("workbench.components.new", { defaultValue: "New asset" })}
          </WbButton>
        </div>
      </div>

      <Tabs
        label={t("workbench.components.kinds", { defaultValue: "Asset kinds" })}
        value={group}
        onChange={setGroup}
        items={[
          {
            id: "components",
            label: t("workbench.components.kindComponents", { defaultValue: "Components" }),
            count: grouped.components,
          },
          {
            id: "hooks",
            label: t("workbench.components.kindHooks", { defaultValue: "Hooks" }),
            count: grouped.hooks,
          },
          {
            id: "support",
            label: t("workbench.components.kindSupport", { defaultValue: "Support assets" }),
            count: grouped.support,
          },
        ]}
      />

      <div style={{ height: 12 }} />

      <div className="wb-filterbar">
        <SearchBox
          value={query}
          onChange={setQuery}
          label={t("workbench.components.search", { defaultValue: "Search assets" })}
          placeholder={t("workbench.components.searchPlaceholder", {
            defaultValue: "Search assets, ids, props, evidence",
          })}
          shortcut={t("workbench.components.searchKey", { defaultValue: "/" })}
        />
        <div className="wb-row" style={{ flexWrap: "wrap" }}>
          <button type="button" onClick={() => setStatus("all")} className="wb-chip" aria-pressed={status === "all"}>
            {t("workbench.components.all", { defaultValue: "All" })} {filtered.length}
          </button>
          <button type="button" onClick={() => setStatus("review")} className="wb-chip" aria-pressed={status === "review"}>
            {t("workbench.components.review", { defaultValue: "Needs review" })} {findings.length}
          </button>
          <button type="button" onClick={() => setStatus("verified")} className="wb-chip" aria-pressed={status === "verified"}>
            {t("workbench.components.verified", { defaultValue: "Verified" })}
          </button>
          <button type="button" onClick={() => setStatus("drafts")} className="wb-chip" aria-pressed={status === "drafts"}>
            {t("workbench.components.drafts", { defaultValue: "Drafts" })}{" "}
            {assets.filter(assetHasDraft).length}
          </button>
        </div>
        <Select
          aria-label={t("workbench.components.domain", { defaultValue: "Filter by domain" })}
          value={domain}
          onChange={(event) => setDomain(event.target.value)}
        >
          <option value="all">{t("workbench.components.allDomains", { defaultValue: "Domain: All" })}</option>
          {domains.map(([name, count]) => (
            <option key={name} value={name}>
              {name} ({count})
            </option>
          ))}
        </Select>
        <Select
          aria-label={t("workbench.components.sort", { defaultValue: "Sort assets" })}
          value={sort}
          onChange={(event) => setSort(event.target.value as typeof sort)}
        >
          <option value="attention">{t("workbench.components.sortAttention", { defaultValue: "Sort: Attention" })}</option>
          <option value="recent">{t("workbench.components.sortRecent", { defaultValue: "Sort: Recent" })}</option>
          <option value="adoptions">{t("workbench.components.sortAdoptions", { defaultValue: "Sort: Adoptions" })}</option>
          <option value="name">{t("workbench.components.sortName", { defaultValue: "Sort: Name" })}</option>
        </Select>
        <Segmented
          label={t("workbench.components.view", { defaultValue: "Result view" })}
          value={view}
          onChange={setView}
          items={[
            { id: "gallery", label: <Grid2X2 aria-hidden size={14} /> },
            { id: "list", label: <List aria-hidden size={14} /> },
            { id: "tree", label: <Network aria-hidden size={14} /> },
          ]}
        />
      </div>

      {catalog.isError ? (
        <div style={{ marginTop: 14 }}>
          <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
            <b>{t("workbench.components.errorTitle", { defaultValue: "The catalog could not be read." })}</b>{" "}
            {t("workbench.components.errorBody", {
              defaultValue: "The API did not answer. Nothing is shown as an empty library.",
            })}
          </Banner>
        </div>
      ) : null}

      <div className="wb-catalog-layout">
        <Panel>
          <PanelHead
            title={
              group === "components"
                ? t("workbench.components.allAssets", { defaultValue: "All assets" })
                : group === "hooks"
                  ? t("workbench.components.hookAssets", { defaultValue: "Hooks" })
                  : t("workbench.components.supportAssets", { defaultValue: "Support assets" })
            }
            meta={t("workbench.components.resultMeta", {
              defaultValue: "{{count}} results · sorted by {{sort}}",
              count: filtered.length,
              sort,
            })}
            actions={
              <span className="wb-muted-2 wb-small" aria-hidden>
                <SlidersHorizontal size={13} />
              </span>
            }
          />
          <PanelBody flush>
            {catalog.isLoading ? (
              <div className="wb-gallery">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} height={210} />
                ))}
              </div>
            ) : filtered.length === 0 ? (
              <Empty title={t("workbench.components.emptyTitle", { defaultValue: "No assets match these filters" })}>
                {t("workbench.components.emptyBody", {
                  defaultValue: "Clear the search or choose another domain. The library itself is not empty.",
                })}
              </Empty>
            ) : view === "gallery" ? (
              <div className="wb-gallery">
                {visible.map((asset) => (
                  <AssetCard
                    key={asset.id}
                    asset={asset}
                    row={rows.get(asset.catalogId) ?? rows.get(asset.libraryId)}
                  />
                ))}
              </div>
            ) : view === "list" ? (
              <div className="wb-list">
                {visible.map((asset) => (
                  <AssetListRow
                    key={asset.id}
                    asset={asset}
                    row={rows.get(asset.catalogId) ?? rows.get(asset.libraryId)}
                  />
                ))}
              </div>
            ) : (
              <div className="wb-tree">
                {domains.map(([name, count]) => (
                  <div key={name}>
                    <div className="wb-tree-row is-branch">
                      <span>{name}</span>
                      <span className="wb-count">{count}</span>
                    </div>
                    {visible
                      .filter((asset) => assetDomain(asset) === name)
                      .map((asset) => (
                        <AssetListRow
                          key={asset.id}
                          asset={asset}
                          row={rows.get(asset.catalogId) ?? rows.get(asset.libraryId)}
                          tree
                        />
                      ))}
                  </div>
                ))}
              </div>
            )}
            {filtered.length > visible.length ? (
              <div style={{ padding: 14, borderTop: "1px solid var(--wb-line)", textAlign: "center" }}>
                <WbButton variant="ghost" size="sm" onClick={() => setLimit((value) => value + 60)}>
                  {t("workbench.components.loadMore", {
                    defaultValue: "Show more · {{shown}} of {{total}}",
                    shown: visible.length,
                    total: filtered.length,
                  })}
                </WbButton>
              </div>
            ) : null}
          </PanelBody>
        </Panel>

        <aside className="wb-side-rail">
          <Panel>
            <PanelHead title={t("workbench.components.collections", { defaultValue: "Collections" })} meta={`${domains.length}`} />
            <PanelBody>
              <div className="wb-stack tight">
                {collections.map((entry) => (
                  <button
                    key={entry.name}
                    type="button"
                    className="wb-collection-row"
                    onClick={() => setDomain(entry.name)}
                  >
                    <span className="wb-dot" style={{ background: "var(--wb-signal)" }} aria-hidden />
                    <span className="wb-name">{entry.name}</span>
                    <span className="wb-collection-bar">
                      <span style={{ width: `${(entry.count / maxCollection) * 100}%` }} />
                    </span>
                    <span className="wb-count">{entry.count}</span>
                  </button>
                ))}
              </div>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.components.coverage", { defaultValue: "Coverage" })}
              meta={t("workbench.components.live", { defaultValue: "Live" })}
            />
            <PanelBody>
              {coverage.isError ? (
                <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                  {t("workbench.components.coverageUnavailable", {
                    defaultValue: "Coverage service did not complete. No ratio is shown.",
                  })}
                </Banner>
              ) : (
                <div className="wb-health">
                  <Ring value={verifiedRatio * 100} size={66} tone={verifiedRatio > 0.7 ? "green" : "amber"} />
                  <div className="wb-health-copy">
                    <b>
                      {t("workbench.components.verifiedAssets", {
                        defaultValue: "{{count}} verified",
                        count: assets.filter((asset) => {
                          const row = rows.get(asset.catalogId) ?? rows.get(asset.libraryId);
                          return Boolean(row && row.failedGates.length === 0 && row.achieved);
                        }).length,
                      })}
                    </b>
                    <span>
                      {t("workbench.components.coverageBody", {
                        defaultValue: "{{count}} need attention",
                        count: findings.length,
                      })}
                    </span>
                  </div>
                </div>
              )}
              <Link to={appRoutes.coverage} className="wb-btn ghost sm wide" style={{ marginTop: 12 }}>
                {t("workbench.components.openCoverage", { defaultValue: "Open coverage" })}
              </Link>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead title={t("workbench.components.health", { defaultValue: "Library pulse" })} />
            <PanelBody>
              <div className="wb-stack tight">
                <div className="wb-row-between">
                  <span className="wb-small wb-muted">{t("workbench.components.published", { defaultValue: "Published" })}</span>
                  <b className="wb-mono">{formatNumber(assets.length)}</b>
                </div>
                <Meter value={(assets.length / Math.max(1, assets.length)) * 100} tone="green" />
                <div className="wb-row-between">
                  <span className="wb-small wb-muted">{t("workbench.components.draftsLabel", { defaultValue: "Drafts" })}</span>
                  <b className="wb-mono">{assets.filter(assetHasDraft).length}</b>
                </div>
                <div className="wb-row-between">
                  <span className="wb-small wb-muted">{t("workbench.components.pages", { defaultValue: "Design pages" })}</span>
                  <b className="wb-mono">—</b>
                </div>
              </div>
            </PanelBody>
          </Panel>
        </aside>
      </div>
    </div>
  );
}

function AssetCard({ asset, row }: { asset: CatalogAsset; row?: CoverageRow }) {
  const { t } = useTranslation();
  const [failed, setFailed] = useState(false);
  const status = assetStatus(asset, row);
  return (
    <Link to={`/assets/${encodeURIComponent(asset.id)}`} className="wb-asset-card">
      <span className="wb-asset-thumb">
        {failed ? (
          <span className="wb-asset-thumb-state">
            {t("workbench.components.previewPending", { defaultValue: "Preview pending" })}
          </span>
        ) : (
          <img
            src={previewURL(asset)}
            alt=""
            loading="lazy"
            onError={() => setFailed(true)}
          />
        )}
        <span className="wb-asset-thumb-corner">
          {assetHasDraft(asset)
            ? t("workbench.components.draft", { defaultValue: "draft" })
            : asset.catalogKind || t("workbench.components.asset", { defaultValue: "asset" })}
        </span>
      </span>
      <span className="wb-asset-body">
        <b>{asset.displayName}</b>
        <small>
          {asset.catalogId || asset.libraryId} · {assetVersion(asset)}
        </small>
      </span>
      <span className="wb-asset-foot">
        <Chip tone={status.tone} title={row?.failedGates.length ? `${row.failedGates.length} failed gates` : undefined}>
          <Dot tone={status.tone} />
          {status.label}
        </Chip>
        <span>{formatRelative(assetTime(asset))}</span>
      </span>
    </Link>
  );
}

function AssetListRow({
  asset,
  row,
  tree,
}: {
  asset: CatalogAsset;
  row?: CoverageRow;
  tree?: boolean;
}) {
  const status = assetStatus(asset, row);
  return (
    <Link
      to={`/assets/${encodeURIComponent(asset.id)}`}
      className={tree ? "wb-tree-row" : "wb-list-row"}
    >
      <Boxes aria-hidden size={14} style={{ color: "var(--wb-muted-2)", flex: "none" }} />
      <span style={{ minWidth: 0, flex: 1 }}>
        <b>{asset.displayName}</b>
        <small style={{ display: "block" }}>
          {asset.catalogId || asset.libraryId} · {assetVersion(asset)}
        </small>
      </span>
      <Chip tone={status.tone}>{status.label}</Chip>
      <span className="wb-mono wb-small wb-muted-2">{formatNumber(assetAdoptions(asset))}</span>
    </Link>
  );
}

function assetTime(asset: CatalogAsset): Date | undefined {
  return timestampToDate(asset.updatedAt);
}
