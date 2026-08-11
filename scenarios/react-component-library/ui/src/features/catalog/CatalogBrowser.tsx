/** @vrooliComponentSource data-display.data-table */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  FileCode2,
  Grid2X2,
  List,
  Network,
  Search,
} from "lucide-react";
import { type FormEvent, useDeferredValue, useMemo, useState } from "react";
import { Link, useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom";

import { listCatalogAssets, type CatalogAsset } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { Input } from "../../components/Input";
import { Button } from "../../components/Button";
import { Tabs } from "../../components/Tabs";
import { TreeView, type TreeNode } from "../../components/TreeView";
import {
  ExperienceSurface,
  type ExperienceSurfaceState,
} from "../../components/ExperienceSurface/versions/1.0.0/ExperienceSurface";
import { CreateComponentDialog } from "../components/CreateComponentDialog";
import { workflowsClient } from "../../api/workflows";
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
  const { t } = useTranslation();
  const isHook =
    (asset.assetKind as unknown) === 2 || (asset.assetKind as unknown) === "ASSET_KIND_HOOK";
  const counts = adoptionCounts(asset);
  const isTree = presentation === "tree";
  const content = (
    <>
      <span className="flex min-w-0 flex-1 items-center gap-space-2xs">
        <span
          aria-hidden
          className="flex h-7 w-7 shrink-0 items-center justify-center rounded-control bg-app-surface-muted text-app-primary"
        >
          <FileCode2 className="h-4 w-4" />
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
          ? "flex min-h-24 flex-col justify-between rounded-panel border p-space-xs"
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

function CatalogActions() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [showManual, setShowManual] = useState(false);
  const [showAssisted, setShowAssisted] = useState(false);
  const [sourceScenario, setSourceScenario] = useState("");
  const [sourcePath, setSourcePath] = useState("");
  const assisted = useMutation({
    mutationFn: () =>
      workflowsClient.startWorkflow({
        kind: 1,
        sourceScenario,
        sourcePath,
        idempotencyKey: `catalog-extract:${sourceScenario}:${sourcePath}`,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["workflows"] });
      setShowAssisted(false);
    },
  });
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    assisted.mutate();
  };
  return (
    <div className="space-y-space-2xs">
      <div className="flex flex-wrap gap-space-2xs">
        <Button size="sm" onClick={() => setShowManual(true)}>
          {t("catalog.addManual", { defaultValue: "Add component" })}
        </Button>
        <Button size="sm" variant="secondary" onClick={() => setShowAssisted((open) => !open)}>
          {t("catalog.addAssisted", { defaultValue: "Assisted extraction" })}
        </Button>
      </div>
      {showManual && <CreateComponentDialog onClose={() => setShowManual(false)} />}
      {showAssisted && (
        <form
          onSubmit={submit}
          className="grid gap-space-2xs rounded-panel border border-app-border p-space-xs text-sm"
        >
          <p className="text-app-muted-foreground">
            {t("catalog.assistedDescription", {
              defaultValue:
                "Queue a catalog-maintainer run. It will use direct React Component Library APIs for any catalog writes.",
            })}
          </p>
          <label htmlFor="catalog-source-scenario">
            {t("catalog.sourceScenario", { defaultValue: "Source scenario" })}
            <Input
              id="catalog-source-scenario"
              aria-label={t("catalog.sourceScenario", { defaultValue: "Source scenario" })}
              value={sourceScenario}
              onChange={(event) => setSourceScenario(event.target.value)}
              required
              className="mt-space-3xs"
            />
          </label>
          <label htmlFor="catalog-source-path">
            {t("catalog.sourcePath", { defaultValue: "Source path" })}
            <Input
              id="catalog-source-path"
              aria-label={t("catalog.sourcePath", { defaultValue: "Source path" })}
              value={sourcePath}
              onChange={(event) => setSourcePath(event.target.value)}
              required
              className="mt-space-3xs"
              placeholder={t(strings.catalog.sourcePathPlaceholder)}
            />
          </label>
          {assisted.error && (
            <p role="alert" className="text-xs text-app-danger">
              {t("catalog.assistedError", { defaultValue: "Unable to queue assisted extraction." })}
            </p>
          )}
          <div>
            <Button
              size="sm"
              type="submit"
              disabled={assisted.isPending || !sourceScenario.trim() || !sourcePath.trim()}
            >
              {assisted.isPending
                ? t("catalog.assistedStarting", { defaultValue: "Starting…" })
                : t("catalog.assistedStart", { defaultValue: "Start extraction" })}
            </Button>
          </div>
        </form>
      )}
    </div>
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
  const groups = useMemo(() => {
    const grouped = new Map<string, CatalogAsset[]>();
    for (const asset of assets) {
      const key = asset.slot || asset.category || t("catalog.other", { defaultValue: "Other" });
      grouped.set(key, [...(grouped.get(key) ?? []), asset]);
    }
    return [...grouped.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [assets, t]);
  const treeNodes = useMemo<TreeNode[]>(
    () =>
      groups.map(([group, groupedAssets]) => {
        const adoptionTotal = groupedAssets.reduce(
          (total, asset) => total + adoptionCounts(asset).direct,
          0,
        );
        return {
          id: `catalog-group:${group}`,
          label: (
            <span className="flex min-w-0 items-center gap-space-2xs">
              <span className="min-w-0 flex-1 truncate text-xs font-semibold uppercase tracking-wide">
                {group}
              </span>
              <span data-testid="catalog-group-adoptions" className="rcl-tree-label-meta shrink-0">
                {groupedAssets.length} assets · {adoptionTotal} adoptions
              </span>
            </span>
          ),
          ariaLabel: `${group}, ${groupedAssets.length} assets, ${adoptionTotal} adoptions`,
          defaultExpanded: true,
          children: groupedAssets.map((asset) => {
            const counts = adoptionCounts(asset);
            const isHook =
              (asset.assetKind as unknown) === 2 ||
              (asset.assetKind as unknown) === "ASSET_KIND_HOOK";
            return {
              id: asset.id,
              label: (
                <span className="flex min-w-0 items-center gap-space-2xs">
                  <span
                    data-testid={selectors.catalog.asset}
                    className="rcl-tree-label-main min-w-0 flex-1 truncate"
                  >
                    {asset.displayName || asset.libraryId}
                  </span>
                  <span className="rcl-tree-label-meta shrink-0">
                    <span>
                      {t("catalog.adoptions", {
                        defaultValue: "{{count}} adoptions",
                        count: counts.direct,
                      })}
                    </span>
                    {isHook && (
                      <span>
                        {t("catalog.effectiveAdoptions", {
                          defaultValue: "{{count}} effective",
                          count: counts.effective,
                        })}
                      </span>
                    )}
                  </span>
                </span>
              ),
              ariaLabel: `${asset.displayName || asset.libraryId}, ${asset.libraryId}, ${counts.direct} direct adoptions${isHook ? `, ${counts.effective} effective adoptions` : ""}`,
              icon: <FileCode2 aria-hidden className="h-4 w-4" />,
            };
          }),
        };
      }),
    [groups, t],
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
      {!compact && <CatalogActions />}
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
          className="absolute start-2 top-2.5 h-4 w-4 text-app-muted-foreground"
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
              <Icon aria-hidden className="h-4 w-4" />
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
        <p role="alert" className="text-xs text-app-danger">
          {t("catalog.error", { defaultValue: "The catalog could not be loaded." })}
        </p>
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
        query.isLoading ? t("catalog.loading", { defaultValue: "Loading catalog…" }) : undefined
      }
    >
      {content}
    </ExperienceSurface>
  );
}
