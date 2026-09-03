/** @vrooliComponentSource data-display.data-table */
import { useQuery } from "@tanstack/react-query";
import { FileCode2, Grid2X2, List, Network, Search } from "lucide-react";
import { useDeferredValue, useMemo, useState } from "react";
import { Link, useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom";

import { listCatalogAssets, type CatalogAsset } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { TreeView, type TreeNode } from "@vrooli/react-component-library/TreeView/1";
import {
  ExperienceSurface,
  type ExperienceSurfaceState,
} from "@vrooli/react-component-library/ExperienceSurface/1";
import { AdoptedAssetShowcase } from "./AdoptedAssetShowcase";
import { assetInfoTab, assetPath } from "../../routes";

type Presentation = "tree" | "list" | "cards";
type KindTab = "components" | "hooks";

interface Props {
  compact?: boolean;
  onNavigate?: () => void;
  /** Only the primary workspace catalog is an authored readiness surface. */
  surfaceId?: string;
}

const assetKindForTab: Record<KindTab, 1 | 2> = {
  components: 1,
  hooks: 2,
};

function adoptionCounts(asset: CatalogAsset) {
  return {
    direct: asset.metrics?.directAdoptionCount ?? 0,
    effective: asset.metrics?.effectiveAdoptionCount ?? 0,
  };
}

function catalogGroups(assets: CatalogAsset[], other: string) {
  const domains = new Map<string, { order: number; rungs: Map<number, CatalogAsset[]> }>();
  for (const asset of assets) {
    const domain = asset.catalogDomain || asset.slot || asset.category || other;
    const rung = asset.catalogRung ?? -1;
    const group = domains.get(domain) ?? {
      order: asset.catalogDomainOrder || Number.MAX_SAFE_INTEGER,
      rungs: new Map<number, CatalogAsset[]>(),
    };
    group.rungs.set(rung, [...(group.rungs.get(rung) ?? []), asset]);
    domains.set(domain, group);
  }
  return [...domains.entries()]
    .sort(([, a], [, b]) => a.order - b.order)
    .map(
      ([domain, group]) => [domain, [...group.rungs.entries()].sort(([a], [b]) => a - b)] as const,
    );
}

/**
 * Adoption metrics in their prose form ("3 adoptions", "1 effective").
 *
 * Shared by the flat rows and the tree rows so the workspace catalog reports
 * the same counts in the same words no matter which presentation is active —
 * previously the tree branch rendered bare numerals only, so direct/effective
 * adoption counts were unreadable (and untestable) in the default view.
 */
function AssetMetricBadges({ asset }: { asset: CatalogAsset }) {
  const { t } = useTranslation();
  const counts = adoptionCounts(asset);
  const isHook =
    (asset.assetKind as unknown) === 2 || (asset.assetKind as unknown) === "ASSET_KIND_HOOK";
  return (
    <span className="flex shrink-0 flex-wrap justify-end gap-space-3xs text-[11px] text-app-muted-foreground">
      <span className="rounded-pill bg-app-surface-muted px-space-2xs py-space-3xs">
        {t("catalog.adoptions", { defaultValue: "{{count}} adoptions", count: counts.direct })}
      </span>
      {isHook && (
        <span className="rounded-pill bg-app-surface-muted px-space-2xs py-space-3xs">
          {t("catalog.effectiveAdoptions", {
            defaultValue: "{{count}} effective",
            count: counts.effective,
          })}
        </span>
      )}
    </span>
  );
}

function AssetRow({
  asset,
  presentation,
  selected,
  onNavigate,
  currentTab,
}: {
  asset: CatalogAsset;
  presentation: Presentation;
  selected: boolean;
  onNavigate?: () => void;
  currentTab?: ReturnType<typeof assetInfoTab>;
}) {
  const isTree = presentation === "tree";
  const content = (
    <>
      <span className="flex min-w-0 flex-1 items-center gap-space-2xs">
        <span
          aria-hidden
          className="flex h-control-compact w-control-compact shrink-0 items-center justify-center rounded-control bg-app-surface-muted text-app-primary"
        >
          <FileCode2 className="h-icon-sm w-icon-sm" />
        </span>
        <span className="min-w-0">
          <span className="block truncate font-medium">{asset.displayName || asset.libraryId}</span>
          {isTree && (
            <span className="block truncate text-[11px] text-app-muted-foreground">
              {asset.libraryId}
            </span>
          )}
        </span>
      </span>
      <AssetMetricBadges asset={asset} />
    </>
  );
  return (
    <Link
      to={assetPath(asset.id, currentTab ? { tab: currentTab } : {})}
      onClick={onNavigate}
      data-testid={selectors.catalog.asset}
      data-selected={selected || undefined}
      role={isTree ? "treeitem" : undefined}
      aria-level={isTree ? 2 : undefined}
      aria-selected={isTree ? selected : undefined}
      className={[
        presentation === "cards"
          ? "flex min-h-surface-short flex-col justify-between rounded-panel border p-space-xs"
          : isTree
            ? "touch-target flex items-center justify-between gap-space-xs rounded-control border border-transparent px-space-xs py-space-2xs"
            : "touch-target flex items-center justify-between gap-space-2xs rounded-control px-space-2xs py-space-2xs",
        selected
          ? "border-app-primary bg-app-surface-muted text-app-foreground shadow-sm"
          : "border-app-border text-app-foreground hover:bg-app-surface-muted",
      ].join(" ")}
    >
      {content}
    </Link>
  );
}

export function CatalogBrowser({ compact = false, onNavigate, surfaceId }: Props) {
  const { t } = useTranslation();
  const { id: selectedID } = useParams<{ id: string }>();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const [tab, setTab] = useState<KindTab>("components");
  const [presentation, setPresentation] = useState<Presentation>("tree");
  const [match, setMatch] = useState(() => searchParams.get("q") ?? "");
  const navigate = useNavigate();
  const deferredMatch = useDeferredValue(match);
  const query = useQuery({
    queryKey: ["catalog", tab, deferredMatch],
    queryFn: () =>
      listCatalogAssets({ limit: 200, match: deferredMatch, assetKind: assetKindForTab[tab] }),
    staleTime: 30_000,
  });
  const assets = useMemo(() => query.data?.components ?? [], [query.data]);
  const groups = useMemo(
    () => catalogGroups(assets, t("catalog.other", { defaultValue: "Other" })),
    [assets, t],
  );
  const treeNodes = useMemo<TreeNode[]>(
    () =>
      groups.map(([domain, rungGroups]) => {
        const domainAssets = rungGroups.flatMap(([, groupedAssets]) => groupedAssets);
        const adoptionTotal = domainAssets.reduce(
          (total, asset) => total + adoptionCounts(asset).direct,
          0,
        );
        return {
          id: `catalog-domain:${domain}`,
          label: (
            <span className="grid min-w-0 grid-cols-3 items-center gap-space-2xs">
              <span className="col-span-2 min-w-0 truncate text-label font-semibold uppercase tracking-wide">
                {domain}
              </span>
              <span
                data-testid="catalog-group-adoptions"
                className="rcl-tree-label-meta min-w-0 truncate text-end tabular-nums"
                title={`${domainAssets.length} assets, ${adoptionTotal} adoptions`}
              >
                {domainAssets.length} · {adoptionTotal}
              </span>
            </span>
          ),
          ariaLabel: `${domain}, ${domainAssets.length} assets, ${adoptionTotal} adoptions`,
          defaultExpanded: true,
          children: rungGroups.map(([rung, groupedAssets]) => ({
            id: `catalog-rung:${domain}:${rung}`,
            label: (
              <span className="text-label font-medium capitalize">
                {groupedAssets[0]?.catalogRungName || `Rung ${rung}`}
              </span>
            ),
            defaultExpanded: true,
            children: groupedAssets
              .sort((a, b) => a.id.localeCompare(b.id))
              .map((asset) => {
                const counts = adoptionCounts(asset);
                const isHook =
                  (asset.assetKind as unknown) === 2 ||
                  (asset.assetKind as unknown) === "ASSET_KIND_HOOK";
                return {
                  id: asset.id,
                  // A 2:1 grid rather than flex-1 + shrink-0. With shrink-0 the
                  // metadata kept its full width and the name — the only part
                  // that identifies the row — was starved to zero at this tree
                  // depth.
                  //
                  // In the narrow sidebar the metrics stay bare numerals: the
                  // prose form ("0 adoptions · 23 down") cannot fit a third of
                  // a sidebar row three levels deep, and truncating it yields
                  // "0 ad…", which costs the same space and carries no value.
                  // The full phrasing stays reachable through the row title and
                  // the aria-label. The full-width workspace catalog has the
                  // room, so it shows the same adoption badges the flat
                  // presentations do — direct for every asset, effective only
                  // for hooks.
                  //
                  // The test id rides the label because TreeView owns the row
                  // element itself; the label is the row's identity surface.
                  label: (
                    <span
                      data-testid={selectors.catalog.asset}
                      className="grid min-w-0 grid-cols-3 items-center gap-space-2xs"
                    >
                      <span className="rcl-tree-label-main col-span-2 min-w-0 truncate">
                        {asset.displayName || asset.libraryId}
                      </span>
                      {compact ? (
                        <span
                          className="rcl-tree-label-meta min-w-0 truncate text-end tabular-nums"
                          title={`${counts.direct} direct adoptions${isHook ? `, ${counts.effective} effective` : ""}${asset.transitiveDependentCount ? `, ${asset.transitiveDependentCount} downstream dependents` : ""}`}
                        >
                          {counts.direct}
                          {asset.transitiveDependentCount
                            ? ` · ${asset.transitiveDependentCount}↓`
                            : ""}
                        </span>
                      ) : (
                        <AssetMetricBadges asset={asset} />
                      )}
                    </span>
                  ),
                  ariaLabel: `${asset.displayName || asset.libraryId}, ${asset.libraryId}, ${counts.direct} direct adoptions${asset.transitiveDependentCount ? `, ${asset.transitiveDependentCount} downstream dependents` : ""}`,
                  icon: <FileCode2 aria-hidden className="h-icon-sm w-icon-sm" />,
                };
              }),
          })),
        };
      }),
    [compact, groups, t],
  );
  const assetsByID = useMemo(() => new Map(assets.map((asset) => [asset.id, asset])), [assets]);
  const currentTab = selectedID ? assetInfoTab(new URLSearchParams(location.search)) : undefined;
  const handleTreeSelect = (node: TreeNode) => {
    const asset = assetsByID.get(node.id);
    if (!asset) return;
    onNavigate?.();
    navigate(assetPath(asset.id, currentTab ? { tab: currentTab } : {}));
  };

  const readinessState: ExperienceSurfaceState = query.isLoading
    ? "loading"
    : query.error
      ? "error"
      : assets.length === 0
        ? "empty"
        : "ready";
  const content = (
    <section
      data-testid={selectors.catalog.browser}
      className={
        compact
          ? "flex min-h-0 w-full min-w-0 flex-1 flex-col gap-space-2xs"
          : "flex max-w-5xl flex-col gap-space-sm"
      }
    >
      {!compact && <AdoptedAssetShowcase />}
      <Tabs
        items={[
          {
            id: "components",
            label: t("catalog.components", { defaultValue: "Components" }),
          },
          { id: "hooks", label: t("catalog.hooks", { defaultValue: "Hooks" }) },
        ]}
        active={tab}
        onChange={(next) => setTab(next as KindTab)}
        ariaLabel={t("catalog.kindTabs", { defaultValue: "Asset kind" })}
        itemTestId={(item) =>
          item === "components" ? selectors.catalog.componentsTab : selectors.catalog.hooksTab
        }
      />
      <label className="relative block w-full min-w-0">
        <span className="sr-only">{t("catalog.search", { defaultValue: "Search catalog" })}</span>
        <Search
          aria-hidden
          className="absolute start-2 top-2.5 h-icon-sm w-icon-sm text-app-muted-foreground"
        />
        <Input
          data-testid={selectors.catalog.search}
          value={match}
          onChange={(event) => setMatch(event.target.value)}
          className="w-full ps-space-lg"
          placeholder={t("catalog.searchPlaceholder", { defaultValue: "Search assets" })}
        />
      </label>
      {!compact && (
        <div
          className="flex items-center gap-space-3xs"
          aria-label={t("catalog.presentation", { defaultValue: "Catalog presentation" })}
        >
          {(
            [
              ["tree", Network],
              ["list", List],
              ["cards", Grid2X2],
            ] as const
          ).map(([mode, Icon]) => (
            <button
              key={mode}
              type="button"
              data-testid={selectors.catalog.presentation}
              aria-pressed={presentation === mode}
              aria-label={t(`catalog.${mode}`, { defaultValue: mode })}
              onClick={() => setPresentation(mode)}
              className={
                presentation === mode
                  ? "touch-target rounded-control bg-app-surface-muted p-space-2xs"
                  : "touch-target rounded-control p-space-2xs text-app-muted-foreground hover:bg-app-surface-muted"
              }
            >
              <Icon aria-hidden className="h-icon-sm w-icon-sm" />
            </button>
          ))}
        </div>
      )}
      {query.isLoading && (
        <p role="status" className="text-xs text-app-muted-foreground">
          {t("catalog.loading", { defaultValue: "Loading catalog…" })}
        </p>
      )}
      {query.error && (
        <div
          role="alert"
          data-rcl-error-rpc="ListComponents"
          className="grid gap-space-2xs rounded-control border border-app-danger/30 bg-app-danger/5 p-space-xs text-xs text-app-danger"
        >
          <p>
            {t("catalog.error", {
              defaultValue: "ListComponents could not load the catalog.",
            })}
          </p>
          <Button type="button" size="sm" variant="secondary" onClick={() => void query.refetch()}>
            {t("common.retry", { defaultValue: "Retry" })}
          </Button>
        </div>
      )}
      {!query.isLoading && !query.error && assets.length === 0 && (
        <p className="rounded-control border border-dashed border-app-border p-space-xs text-sm text-app-muted-foreground">
          {t("catalog.empty", { defaultValue: "No matching assets." })}
        </p>
      )}
      {presentation === "tree" ? (
        <TreeView
          items={treeNodes}
          label={t("catalog.assetTree", { defaultValue: "Library assets" })}
          selectedId={selectedID}
          onSelect={handleTreeSelect}
        />
      ) : (
        <div
          className={
            presentation === "cards" ? "grid gap-space-2xs sm:grid-cols-2" : "space-y-space-3xs"
          }
        >
          {assets.map((asset) => (
            <AssetRow
              key={asset.id}
              asset={asset}
              presentation={presentation}
              selected={selectedID === asset.id}
              onNavigate={onNavigate}
              currentTab={
                selectedID ? assetInfoTab(new URLSearchParams(location.search)) : undefined
              }
            />
          ))}
        </div>
      )}
    </section>
  );
  if (!surfaceId) return content;
  return (
    <ExperienceSurface
      surfaceId={surfaceId}
      state={readinessState}
      statusMessage={
        query.isLoading
          ? t("catalog.loading", { defaultValue: "Loading catalog…" })
          : query.error
            ? t("catalog.error", { defaultValue: "ListComponents could not load the catalog." })
            : undefined
      }
    >
      {content}
    </ExperienceSurface>
  );
}
