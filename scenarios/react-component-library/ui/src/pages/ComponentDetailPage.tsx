/**
 * ComponentDetailPage — full-width editor + preview for a component.
 *
 * Resolves the component id from the route param, fetches the
 * component record so the editor header shows the libraryId (instead
 * of the bare id), and hands the rest off to `<ComponentEditor />`.
 * Closing the editor returns the user to the components list.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";

import { adoptionsClient, RecommendationClass } from "../api/adoptions";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/ui/empty-state";
import { StatusBadge } from "../components/ui/status-badge";
import { componentsClient, getCatalogAsset, getComponentExperience, type CatalogAsset } from "../api/components";
import { strings } from "../consts/strings";
import { CreateAdoptionDialog } from "../features/adoptions/CreateAdoptionDialog";
import { ComponentEditor, type ComparisonSession } from "../features/components/ComponentEditor";
import { ComponentExperiencePanel } from "../features/components/ComponentExperiencePanel";
import { ComponentTestPanel } from "../features/components/ComponentTestPanel";
import { VersionsCard } from "../features/versions/VersionsCard";
import { useTranslation } from "../i18n";

type InfoTab = "overview" | "tests" | "versions" | "adoptions";

function DetailTabs({ active, onChange, versionCount, adoptionCount }: { active: InfoTab; onChange: (tab: InfoTab) => void; versionCount: number; adoptionCount: number }) {
  const { t } = useTranslation();
  const tabs: Array<{ id: InfoTab; label: string; count?: number }> = [
    { id: "overview", label: t("componentDetail.info.overview", { defaultValue: "Overview" }) },
    { id: "tests", label: t("componentDetail.info.tests", { defaultValue: "Tests" }) },
    { id: "versions", label: t("componentDetail.info.versions", { defaultValue: "Versions" }), count: versionCount },
    { id: "adoptions", label: t("componentDetail.info.adoptions", { defaultValue: "Adoptions" }), count: adoptionCount },
  ];

  return <div role="tablist" aria-label={t("componentDetail.info.tabs", { defaultValue: "Asset information" })} className="flex border-b border-app-border">
    {tabs.map((tab) => <Button key={tab.id} type="button" role="tab" variant="secondary" aria-selected={active === tab.id} onClick={() => onChange(tab.id)} className={`min-h-0 rounded-none border-0 px-2 py-2 text-xs font-medium ${active === tab.id ? "border-b-2 border-app-primary text-app-foreground" : "text-app-muted-foreground"}`}>
      {tab.label}{tab.count !== undefined && <span aria-hidden="true" className="ml-1.5 inline-flex min-w-4 items-center justify-center rounded-pill bg-app-surface-muted px-1 py-0.5 text-[10px] leading-none text-app-muted-foreground">{tab.count}</span>}
    </Button>)}
  </div>;
}

function statusTone(library: number, local: number): "success" | "warning" | "danger" | "info" | "neutral" {
  if (local === 2 || library === 2 || library === 3) return "warning";
  if (local === 3 || library === 4) return "danger";
  if (local === 4 || library === 5) return "info";
  return library === 1 && local === 1 ? "success" : "neutral";
}

function statusLabel(library: number, local: number) {
  const libraryLabel = ["Unspecified", "Current", "Behind", "Deprecated", "Missing", "Unknown"][library] ?? "Unknown";
  const localLabel = ["Unspecified", "Clean", "Modified", "Missing", "Unknown"][local] ?? "Unknown";
  return `${libraryLabel} / ${localLabel}`;
}

function isHook(asset: CatalogAsset) {
  return (asset.assetKind as unknown) === 2 || (asset.assetKind as unknown) === "ASSET_KIND_HOOK";
}

function HookWorkspace({ asset, onClose }: { asset: CatalogAsset; onClose: () => void }) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<InfoTab>("overview");
  const effective = useQuery({ queryKey: ["adoptions", "effective", asset.id], queryFn: () => adoptionsClient.listEffectiveAdoptions({ componentId: asset.id, limit: 100 }) });
  return <div data-testid="hook-detail-page" className="flex min-h-0 flex-1 flex-col"><ComponentEditor id={asset.id} libraryId={asset.libraryId || asset.id} onClose={onClose} renderable={false} metadataSlot={<aside data-testid="hook-workspace-details" className="space-y-3"><StatusBadge tone="info">{t("catalog.noPreview", { defaultValue: "No live preview — hooks are non-renderable." })}</StatusBadge><div className="flex flex-wrap gap-2 text-xs"><StatusBadge tone="neutral">{t("catalog.directAdoptions", { defaultValue: "{{count}} direct", count: asset.metrics?.directAdoptionCount ?? 0 })}</StatusBadge><StatusBadge tone="neutral">{t("catalog.effectiveAdoptions", { defaultValue: "{{count}} effective", count: asset.metrics?.effectiveAdoptionCount ?? 0 })}</StatusBadge></div><DetailTabs active={tab} onChange={setTab} versionCount={asset.metrics?.versionCount ?? 0} adoptionCount={asset.metrics?.effectiveAdoptionCount ?? 0} />{tab === "overview" && <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-2 text-xs"><dt className="text-app-muted-foreground">{t("catalog.kind", { defaultValue: "Kind" })}</dt><dd>{t("catalog.hook", { defaultValue: "Hook" })}</dd><dt className="text-app-muted-foreground">{t("catalog.source", { defaultValue: "Source" })}</dt><dd className="break-all font-mono">{asset.sourcePath || "—"}</dd></dl>}{tab === "tests" && <ComponentTestPanel componentId={asset.id} version={asset.latestVersion || asset.version} />}{tab === "versions" && <VersionsCard componentId={asset.id} onSelectVersion={() => undefined} onCompare={() => undefined} />}{tab === "adoptions" && <div data-testid="hook-effective-adoptions" className="space-y-2 text-xs">{effective.isLoading ? <p className="text-app-muted-foreground">{t("componentDetail.info.adoptionsLoading", { defaultValue: "Loading adoptions…" })}</p> : (effective.data?.adoptions ?? []).length === 0 ? <EmptyState className="p-2 text-xs" title={t("componentDetail.info.noAdoptions", { defaultValue: "No recorded usage." })} /> : <ul className="space-y-2">{(effective.data?.adoptions ?? []).map((entry) => <li key={`${entry.sourceAssetId}:${entry.parentAdoption?.id}`} className="rounded-control border border-app-border p-2"><p className="font-medium">{entry.mediated ? t("catalog.indirectUsage", { defaultValue: "Indirect usage" }) : t("catalog.directUsage", { defaultValue: "Direct usage" })}</p><p className="mt-1">{entry.parentAdoption?.scenario} · {entry.parentAdoption?.adoptedVersion}</p><p className="mt-1 font-mono text-app-muted-foreground">{entry.parentAdoption?.id}</p></li>)}</ul>}</div>}</aside>} /></div>;
}

export function ComponentDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id } = useParams<{ id: string }>();
  const [search, setSearch] = useSearchParams();
  const [selectedVersion, setSelectedVersion] = useState<string | undefined>();
  const [comparison, setComparison] = useState<ComparisonSession | null>(null);
  const tabParam = search.get("tab");
  const infoTab: InfoTab = tabParam === "tests" || tabParam === "versions" || tabParam === "adoptions" ? tabParam : "overview";
  const setInfoTab = (tab: InfoTab) => setSearch((current) => { const next = new URLSearchParams(current); if (tab === "overview") next.delete("tab"); else next.set("tab", tab); if (tab !== "tests") next.delete("testReport"); return next; }, { replace: true });
  const [selectedAdoptionID, setSelectedAdoptionID] = useState("");
  const [createTarget, setCreateTarget] = useState<{ componentId: string; scenario: string } | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["components", "get", id],
    queryFn: async () => {
      try {
        const byID = await componentsClient.getComponent({ id: id ?? "" });
        if (byID.component) return byID;
      } catch {
        // Dependency links use a stable library ID; fall through to that
        // canonical lookup when the route parameter is not a UUID.
      }
      return componentsClient.getComponentByLibraryId({ libraryId: id ?? "" });
    },
    enabled: !!id,
  });
  // The generated UI client may lag the catalog's newly generated asset-kind
  // field. Keep established component reads on that client, then enrich the
  // record with the RCL-owned catalog projection when it is available.
  const catalogAsset = useQuery({
    queryKey: ["catalog", "get", id],
    queryFn: () => getCatalogAsset(id ?? ""),
    enabled: Boolean(id) && Boolean(data?.component),
    retry: false,
  });
  const experienceQuery = useQuery({
    queryKey: ["components", "experience", id],
    queryFn: () => getComponentExperience(data?.component?.id ?? ""),
    enabled: Boolean(id) && Boolean(data?.component) && infoTab === "overview",
    retry: false,
  });

  const adoptionsQuery = useQuery({
    queryKey: ["adoptions", "component", id],
    queryFn: () => adoptionsClient.listAdoptions({ componentId: id ?? "", limit: 0 }),
    enabled: Boolean(id),
  });
  const suggestionsQuery = useQuery({
    queryKey: ["adoptions", "suggestions", "component", id],
    queryFn: () => adoptionsClient.suggestAdoptions({ componentId: id ?? "", limit: 8 }),
    enabled: Boolean(id) && infoTab === "adoptions",
  });
  const refreshMutation = useMutation({
    mutationFn: () => adoptionsClient.refreshAdoptions({ componentId: id ?? "" }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["adoptions", "component", id] }),
  });

  if (!id) {
    return (
      <p data-testid="component-detail-missing-id" className="text-sm text-app-danger">
        {t("componentDetail.missingId", { defaultValue: "No component id in URL." })}
      </p>
    );
  }

  if (isLoading) {
    return (
      <p data-testid="component-detail-loading" className="text-sm text-app-muted-foreground">
        {t("componentDetail.loading", { defaultValue: "Loading component…" })}
      </p>
    );
  }

  if (error || !data?.component) {
    return (
      <p data-testid="component-detail-error" className="text-sm text-app-danger">
        {t("componentDetail.notFound", { defaultValue: "Component not found." })}
      </p>
    );
  }

  const component = catalogAsset.data?.component ?? data.component;
  if (isHook(component)) {
    return <HookWorkspace asset={component} onClose={() => { void navigate("/"); }} />;
  }
  // Proto clients provide empty collections, while test and older persisted
  // projections can omit these optional-shaped values entirely.
  const headers = (component.headers as Record<string, string> | undefined) ?? {};
  const tags = (component.tags as string[] | undefined) ?? [];
  const designStyles = (component.designStyles as typeof component.designStyles | undefined) ?? [];
  const dependencies = (component.dependencies as Array<{ libraryId: string; version: string }> | undefined) ?? [];
  const adoptions = adoptionsQuery.data?.adoptions ?? [];
  const selectedAdoption = adoptions.find((adoption) => adoption.id === selectedAdoptionID) ?? adoptions[0];
  const suggestions = suggestionsQuery.data?.suggestions ?? [];

  return (
    <div data-testid="component-detail-page" className="flex min-h-0 flex-1 flex-col">
      <ComponentEditor
        id={component.id}
        libraryId={component.libraryId || component.id}
        onClose={() => {
          void navigate("/");
        }}
        selectedVersion={selectedVersion}
        comparison={comparison}
        onCloseComparison={() => setComparison(null)}
        metadataSlot={(
          <div className="space-y-4">
            <DetailTabs active={infoTab} onChange={setInfoTab} versionCount={component.metrics?.versionCount ?? 0} adoptionCount={component.metrics?.directAdoptionCount ?? adoptions.length} />
            {infoTab === "overview" && <>
              <ComponentExperiencePanel experience={experienceQuery.data} isLoading={experienceQuery.isLoading} isError={experienceQuery.isError} />
              <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
                <h3 className="font-medium">{t("componentDetail.info.identity", { defaultValue: "Identity" })}</h3>
                <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.libraryIdLabel)}</dt><dd className="break-all font-mono">{component.libraryId}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.slotLabel)}</dt><dd>{component.slot || "—"}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.categoryLabel)}</dt><dd>{component.category || headers.category || "—"}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.tagsLabel)}</dt><dd>{tags.join(", ") || "—"}</dd>
                </dl>
              </section>
              <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
                <h3 className="font-medium">{t("componentDetail.info.affinities", { defaultValue: "Design affinities" })}</h3>
                {designStyles.length === 0 ? <p className="mt-2 text-xs text-app-muted-foreground">{t("componentDetail.info.noAffinities", { defaultValue: "No design affinities declared." })}</p> : <ul className="mt-2 space-y-1 text-xs">{designStyles.map((affinity) => <li key={affinity.styleId}>{affinity.styleId}: {affinity.reason || affinity.affinity}</li>)}</ul>}
              </section>
              <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
                <h3 className="font-medium">{t("componentDetail.info.dependencies", { defaultValue: "Shared assets" })}</h3>
                {dependencies.length === 0 ? <p className="mt-2 text-xs text-app-muted-foreground">{t("componentDetail.info.noDependencies", { defaultValue: "No shared assets declared." })}</p> : <ul className="mt-2 space-y-1 text-xs">{dependencies.map((dependency) => <li key={`${dependency.libraryId}@${dependency.version}`}><Link to={`/assets/${encodeURIComponent(dependency.libraryId)}`} className="font-mono text-app-primary underline-offset-2 hover:underline">{dependency.libraryId}</Link><span className="text-app-muted-foreground"> · {dependency.version}</span></li>)}</ul>}
              </section>
            </>}
            {infoTab === "tests" && <ComponentTestPanel componentId={component.id} version={selectedVersion ?? component.latestVersion ?? component.version} />}
            {infoTab === "versions" && <VersionsCard componentId={component.id} selectedVersion={selectedVersion} onSelectVersion={setSelectedVersion} onCompare={setComparison} />}
            {infoTab === "adoptions" && <section data-testid="component-detail-adoptions" className="space-y-3 text-sm text-app-foreground">
              <div className="flex items-center justify-between"><h3 className="font-medium">{t("componentDetail.info.adoptions", { defaultValue: "Adoptions" })}</h3><Button size="sm" onClick={() => refreshMutation.mutate()} disabled={refreshMutation.isPending}>{refreshMutation.isPending ? t(strings.adoptions.refreshing) : t(strings.adoptions.refreshAction)}</Button></div>
              {adoptionsQuery.isLoading ? <p className="text-xs text-app-muted-foreground">{t("componentDetail.info.adoptionsLoading", { defaultValue: "Loading adoptions…" })}</p> : adoptions.length === 0 ? <EmptyState className="p-2 text-xs" title={t("componentDetail.info.noAdoptions", { defaultValue: "No scenarios have adopted this component yet." })} /> : <div className="space-y-2">{adoptions.map((adoption) => <Button key={adoption.id} type="button" variant="secondary" onClick={() => setSelectedAdoptionID(adoption.id)} className={`h-auto w-full rounded-control border p-2 text-left ${selectedAdoption?.id === adoption.id ? "border-app-primary" : "border-app-border"}`}><div className="flex items-center justify-between gap-2"><span className="font-medium">{adoption.scenario}</span><StatusBadge tone={statusTone(adoption.libraryVersionStatus, adoption.localStatus)}>{statusLabel(adoption.libraryVersionStatus, adoption.localStatus)}</StatusBadge></div><p className="mt-1 font-mono text-xs text-app-muted-foreground">{adoption.adoptedVersion} · {adoption.adoptedPath}</p></Button>)}</div>}
              {selectedAdoption && <ul data-testid="component-detail-adoption-file-tree" className="space-y-1 rounded-control bg-app-background p-2 font-mono text-xs text-app-muted-foreground">{(selectedAdoption.files.length > 0 ? selectedAdoption.files.map((file) => file.adoptedPath) : [selectedAdoption.adoptedPath]).map((path) => <li key={path}>{path}</li>)}</ul>}
              <div className="border-t border-app-border pt-3"><h3 className="font-medium">{t(strings.adoptions.suggestions.title)}</h3>{suggestionsQuery.isLoading ? <p className="mt-1 text-xs text-app-muted-foreground">{t("componentDetail.info.suggestionsLoading", { defaultValue: "Finding candidates…" })}</p> : <div className="mt-2 space-y-2">{suggestions.map((suggestion) => <div key={suggestion.scenario} className="rounded-control border border-app-border p-2"><p className="font-medium">{suggestion.scenario}</p><StatusBadge tone="neutral">{suggestion.classification === RecommendationClass.HEURISTIC ? t("adoptions.suggestions.heuristic", { defaultValue: "Heuristic candidate — review before adopting" }) : t("adoptions.suggestions.unavailable", { defaultValue: "Unavailable candidate" })}</StatusBadge><p className="mt-1 text-xs text-app-muted-foreground">{suggestion.reasons.join(" · ")}</p><Button size="sm" className="mt-2" onClick={() => setCreateTarget({ componentId: component.id, scenario: suggestion.scenario })}>{t(strings.adoptions.suggestions.adoptAction)}</Button></div>)}{suggestions.length === 0 && <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.adoptions.suggestions.empty)}</p>}</div>}</div>
            </section>}
          </div>
        )}
      />
      <CreateAdoptionDialog open={Boolean(createTarget)} initial={createTarget} onClose={() => setCreateTarget(null)} />
    </div>
  );
}
