/** @vrooliComponentSource data-display.data-table */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Grid2X2, List, Network, Search } from "lucide-react";
import { type FormEvent, useDeferredValue, useMemo, useState } from "react";
import { Link, useLocation, useParams, useSearchParams } from "react-router-dom";

import { listCatalogAssets, type CatalogAsset } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { Input } from "../../components/Input";
import { Button } from "../../components/Button";
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
  const content = (
    <>
      <span className="truncate font-medium">{asset.displayName || asset.libraryId}</span>
      <span className="flex gap-space-3xs text-[11px] text-app-muted-foreground">
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
      className={[
        presentation === "cards"
          ? "flex min-h-24 flex-col justify-between rounded-panel border p-space-xs"
          : "touch-target flex items-center justify-between gap-space-2xs rounded-control px-space-2xs py-space-2xs",
        selected
          ? "border-app-primary bg-app-surface-muted text-app-foreground"
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
          <label>
            {t("catalog.sourceScenario", { defaultValue: "Source scenario" })}
            <Input
              value={sourceScenario}
              onChange={(event) => setSourceScenario(event.target.value)}
              required
              className="mt-space-3xs"
            />
          </label>
          <label>
            {t("catalog.sourcePath", { defaultValue: "Source path" })}
            <Input
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
          ? "flex min-h-0 flex-1 flex-col gap-space-2xs"
          : "flex max-w-5xl flex-col gap-space-sm"
      }
    >
      {!compact && <CatalogActions />}
      <div
        className="flex items-center gap-space-3xs"
        role="tablist"
        aria-label={t("catalog.kindTabs", { defaultValue: "Asset kind" })}
      >
        {(["components", "hooks"] as const).map((kind) => (
          <button
            key={kind}
            type="button"
            role="tab"
            aria-selected={tab === kind}
            data-testid={
              kind === "components" ? selectors.catalog.componentsTab : selectors.catalog.hooksTab
            }
            onClick={() => setTab(kind)}
            className={
              tab === kind
                ? "touch-target rounded-control bg-app-surface-muted px-space-xs py-space-2xs text-sm font-medium"
                : "touch-target rounded-control px-space-xs py-space-2xs text-sm text-app-muted-foreground hover:bg-app-surface-muted"
            }
          >
            {kind === "components"
              ? t("catalog.components", { defaultValue: "Components" })
              : t("catalog.hooks", { defaultValue: "Hooks" })}
          </button>
        ))}
      </div>
      <label className="relative block">
        <span className="sr-only">{t("catalog.search", { defaultValue: "Search catalog" })}</span>
        <Search
          aria-hidden
          className="absolute start-2 top-2.5 h-4 w-4 text-app-muted-foreground"
        />
        <Input
          data-testid={selectors.catalog.search}
          value={match}
          onChange={(event) => setMatch(event.target.value)}
          className="ps-space-lg"
          placeholder={t("catalog.searchPlaceholder", { defaultValue: "Search assets" })}
        />
      </label>
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
        <div className="space-y-space-xs">
          {groups.map(([group, groupedAssets]) => (
            <section key={group}>
              <div className="mb-space-3xs flex items-center justify-between px-space-3xs text-xs font-medium uppercase text-app-muted-foreground">
                <span>{group}</span>
                <span>
                  {groupedAssets.reduce((total, asset) => total + adoptionCounts(asset).direct, 0)}
                </span>
              </div>
              <div className="space-y-space-3xs">
                {groupedAssets.map((asset) => (
                  <AssetRow
                    key={asset.id}
                    asset={asset}
                    presentation="tree"
                    selected={selectedID === asset.id}
                    onNavigate={onNavigate}
                    currentTab={
                      selectedID ? assetInfoTab(new URLSearchParams(location.search)) : undefined
                    }
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
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
