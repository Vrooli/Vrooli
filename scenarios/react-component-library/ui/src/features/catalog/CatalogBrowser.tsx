import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Grid2X2, List, Network, Search } from "lucide-react";
import { type FormEvent, useDeferredValue, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { listCatalogAssets, type CatalogAsset } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { Input } from "../../components/ui/input";
import { Button } from "../../components/ui/button";
import { CreateComponentDialog } from "../components/CreateComponentDialog";
import { workflowsClient } from "../../api/workflows";
import { assetPath } from "../../routes";

type Presentation = "tree" | "list" | "cards";
type KindTab = "components" | "hooks";

interface Props {
  compact?: boolean;
  onNavigate?: () => void;
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

function AssetRow({ asset, presentation, selected, onNavigate }: { asset: CatalogAsset; presentation: Presentation; selected: boolean; onNavigate?: () => void }) {
  const { t } = useTranslation();
  const isHook = (asset.assetKind as unknown) === 2 || (asset.assetKind as unknown) === "ASSET_KIND_HOOK";
  const counts = adoptionCounts(asset);
  const content = (
    <>
      <span className="truncate font-medium">{asset.displayName || asset.libraryId}</span>
      <span className="flex gap-1 text-[11px] text-app-muted-foreground">
        <span className="rounded-pill bg-app-surface-muted px-1.5 py-0.5">{t("catalog.adoptions", { defaultValue: "{{count}} adoptions", count: counts.direct })}</span>
        {isHook && <span className="rounded-pill bg-app-surface-muted px-1.5 py-0.5">{t("catalog.effectiveAdoptions", { defaultValue: "{{count}} effective", count: counts.effective })}</span>}
      </span>
    </>
  );
  return (
    <Link
      to={assetPath(asset.id)}
      onClick={onNavigate}
      data-testid={selectors.catalog.asset}
      data-selected={selected || undefined}
      className={[
        presentation === "cards" ? "flex min-h-24 flex-col justify-between rounded-panel border p-3" : "flex items-center justify-between gap-2 rounded-control px-2 py-2",
        selected ? "border-app-primary bg-app-surface-muted text-app-foreground" : "border-app-border text-app-foreground hover:bg-app-surface-muted",
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
    mutationFn: () => workflowsClient.startWorkflow({
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
  return <div className="space-y-2">
    <div className="flex flex-wrap gap-2">
      <Button size="sm" onClick={() => setShowManual(true)}>{t("catalog.addManual", { defaultValue: "Add component" })}</Button>
      <Button size="sm" variant="secondary" onClick={() => setShowAssisted((open) => !open)}>{t("catalog.addAssisted", { defaultValue: "Assisted extraction" })}</Button>
    </div>
    {showManual && <CreateComponentDialog onClose={() => setShowManual(false)} />}
    {showAssisted && <form onSubmit={submit} className="grid gap-2 rounded-panel border border-app-border p-3 text-sm">
      <p className="text-app-muted-foreground">{t("catalog.assistedDescription", { defaultValue: "Queue a catalog-maintainer run. It will use direct React Component Library APIs for any catalog writes." })}</p>
      <label>{t("catalog.sourceScenario", { defaultValue: "Source scenario" })}<Input value={sourceScenario} onChange={(event) => setSourceScenario(event.target.value)} required className="mt-1" /></label>
      <label>{t("catalog.sourcePath", { defaultValue: "Source path" })}<Input value={sourcePath} onChange={(event) => setSourcePath(event.target.value)} required className="mt-1" placeholder={t(strings.catalog.sourcePathPlaceholder)} /></label>
      {assisted.error && <p role="alert" className="text-xs text-app-danger">{t("catalog.assistedError", { defaultValue: "Unable to queue assisted extraction." })}</p>}
      <div><Button size="sm" type="submit" disabled={assisted.isPending || !sourceScenario.trim() || !sourcePath.trim()}>{assisted.isPending ? t("catalog.assistedStarting", { defaultValue: "Starting…" }) : t("catalog.assistedStart", { defaultValue: "Start extraction" })}</Button></div>
    </form>}
  </div>;
}

export function CatalogBrowser({ compact = false, onNavigate }: Props) {
  const { t } = useTranslation();
  const { id: selectedID } = useParams<{ id: string }>();
  const [tab, setTab] = useState<KindTab>("components");
  const [presentation, setPresentation] = useState<Presentation>("tree");
  const [match, setMatch] = useState("");
  const deferredMatch = useDeferredValue(match);
  const query = useQuery({
    queryKey: ["catalog", tab, deferredMatch],
    queryFn: () => listCatalogAssets({ limit: 200, match: deferredMatch, assetKind: assetKindForTab[tab] }),
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

  return (
    <section data-testid={selectors.catalog.browser} className={compact ? "flex min-h-0 flex-1 flex-col gap-2" : "flex max-w-5xl flex-col gap-4"}>
      {!compact && <CatalogActions />}
      <div className="flex items-center gap-1" role="tablist" aria-label={t("catalog.kindTabs", { defaultValue: "Asset kind" })}>
        {(["components", "hooks"] as const).map((kind) => <button key={kind} type="button" role="tab" aria-selected={tab === kind} data-testid={kind === "components" ? selectors.catalog.componentsTab : selectors.catalog.hooksTab} onClick={() => setTab(kind)} className={tab === kind ? "rounded-control bg-app-surface-muted px-3 py-1.5 text-sm font-medium" : "rounded-control px-3 py-1.5 text-sm text-app-muted-foreground hover:bg-app-surface-muted"}>{kind === "components" ? t("catalog.components", { defaultValue: "Components" }) : t("catalog.hooks", { defaultValue: "Hooks" })}</button>)}
      </div>
      <label className="relative block"><span className="sr-only">{t("catalog.search", { defaultValue: "Search catalog" })}</span><Search aria-hidden className="absolute start-2 top-2.5 h-4 w-4 text-app-muted-foreground" /><Input data-testid={selectors.catalog.search} value={match} onChange={(event) => setMatch(event.target.value)} className="ps-8" placeholder={t("catalog.searchPlaceholder", { defaultValue: "Search assets" })} /></label>
      <div className="flex items-center gap-1" aria-label={t("catalog.presentation", { defaultValue: "Catalog presentation" })}>
        {([ ["tree", Network], ["list", List], ["cards", Grid2X2] ] as const).map(([mode, Icon]) => <button key={mode} type="button" data-testid={selectors.catalog.presentation} aria-pressed={presentation === mode} aria-label={t(`catalog.${mode}`, { defaultValue: mode })} onClick={() => setPresentation(mode)} className={presentation === mode ? "touch-target rounded-control bg-app-surface-muted p-2" : "touch-target rounded-control p-2 text-app-muted-foreground hover:bg-app-surface-muted"}><Icon aria-hidden className="h-4 w-4" /></button>)}
      </div>
      {query.isLoading && <p role="status" className="text-xs text-app-muted-foreground">{t("catalog.loading", { defaultValue: "Loading catalog…" })}</p>}
      {query.error && <p role="alert" className="text-xs text-app-danger">{t("catalog.error", { defaultValue: "The catalog could not be loaded." })}</p>}
      {!query.isLoading && !query.error && assets.length === 0 && <p className="rounded-control border border-dashed border-app-border p-3 text-sm text-app-muted-foreground">{t("catalog.empty", { defaultValue: "No matching assets." })}</p>}
      {presentation === "tree" ? <div className="space-y-3">{groups.map(([group, groupedAssets]) => <section key={group}><div className="mb-1 flex items-center justify-between px-1 text-xs font-medium uppercase text-app-muted-foreground"><span>{group}</span><span>{groupedAssets.reduce((total, asset) => total + adoptionCounts(asset).direct, 0)}</span></div><div className="space-y-1">{groupedAssets.map((asset) => <AssetRow key={asset.id} asset={asset} presentation="tree" selected={selectedID === asset.id} onNavigate={onNavigate} />)}</div></section>)}</div> : <div className={presentation === "cards" ? "grid gap-2 sm:grid-cols-2" : "space-y-1"}>{assets.map((asset) => <AssetRow key={asset.id} asset={asset} presentation={presentation} selected={selectedID === asset.id} onNavigate={onNavigate} />)}</div>}
    </section>
  );
}
