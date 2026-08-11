/** @vrooliComponentSource navigation.page
 *
 * ComponentDetailPage — full-width editor + preview for a component.
 *
 * Resolves the component id from the route param, fetches the
 * component record so the editor header shows the libraryId (instead
 * of the bare id), and hands the rest off to `<ComponentEditor />`.
 * Closing the editor returns the user to the components list.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";

import { adoptionsClient } from "../api/adoptions";
import { Button } from "../components/Button";
import { EmptyState } from "../components/EmptyState";
import { StatusBadge } from "../components/StatusBadge";
import {
  componentsClient,
  getCatalogAsset,
  getComponentExperience,
  type CatalogAsset,
} from "../api/components";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ComponentEditor, type ComparisonSession } from "../features/components/ComponentEditor";
import { AdoptionsCard } from "../features/adoptions/AdoptionsCard";
import { ComponentExperiencePanel } from "../features/components/ComponentExperiencePanel";
import { ComponentTestPanel } from "../features/components/ComponentTestPanel";
import { VersionsCard } from "../features/versions/VersionsCard";
import { useTranslation } from "../i18n";
import {
  assetInfoTab,
  assetPath,
  assetSearchForTab,
  assetStory,
  type AssetInfoTab,
} from "../routes";

const assetNavigationStorageKey = (assetID: string) => `rcl.asset-navigation.${assetID}`;
type StoredAssetNavigation = { tab?: InfoTab; story?: string };

type InfoTab = AssetInfoTab;

function DetailTabs({
  active,
  onChange,
  versionCount,
  adoptionCount,
  renderable = true,
}: {
  active: InfoTab;
  onChange: (tab: InfoTab) => void;
  versionCount: number;
  adoptionCount: number;
  renderable?: boolean;
}) {
  const { t } = useTranslation();
  const tabs: Array<{ id: InfoTab; label: string; count?: number }> = [
    ...(renderable
      ? [
          {
            id: "preview" as const,
            label: t("components.editor.previewMode", { defaultValue: "Preview" }),
          },
        ]
      : []),
    { id: "overview", label: t("componentDetail.info.overview", { defaultValue: "Overview" }) },
    { id: "files", label: t("components.editor.files", { defaultValue: "Files" }) },
    { id: "tests", label: t("componentDetail.info.tests", { defaultValue: "Tests" }) },
    {
      id: "versions",
      label: t("componentDetail.info.versions", { defaultValue: "Versions" }),
      count: versionCount,
    },
    {
      id: "adoptions",
      label: t("componentDetail.info.adoptions", { defaultValue: "Adoptions" }),
      count: adoptionCount,
    },
  ];

  return (
    <div className="min-w-0 overflow-x-auto" data-testid="component-detail-tabs-scroll">
      <div
        role="tablist"
        aria-label={t("componentDetail.info.tabs", { defaultValue: "Asset information" })}
        className="flex min-w-max items-stretch gap-space-2xs"
      >
        {tabs.map((tab) => {
          const selected = active === tab.id;
          return (
            <button
              key={tab.id}
              data-testid={
                tab.id === "overview"
                  ? selectors.assets.hookOverviewTab
                  : tab.id === "files"
                    ? selectors.assets.hookFilesTab
                    : tab.id === "preview"
                      ? selectors.assets.componentPreviewTab
                      : undefined
              }
              type="button"
              role="tab"
              aria-selected={selected}
              onClick={() => onChange(tab.id)}
              className={`relative flex min-h-11 shrink-0 items-center gap-space-3xs border-b-2 border-transparent px-space-xs py-space-2xs text-xs font-semibold transition-colors ${selected ? "border-app-primary text-app-foreground" : "text-app-muted-foreground hover:border-app-border hover:text-app-foreground"}`}
            >
              {tab.label}
              {tab.count !== undefined && (
                <span
                  aria-hidden="true"
                  className={`inline-flex min-w-4 items-center justify-center rounded-pill px-space-3xs py-space-3xs text-[10px] leading-none ${selected ? "bg-app-primary/10 text-app-primary" : "bg-app-surface-muted text-app-muted-foreground"}`}
                >
                  {tab.count}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function paneForTab(tab: InfoTab): "details" | "files" | "preview" {
  if (tab === "files") return "files";
  if (tab === "preview") return "preview";
  return "details";
}

function tabForPane(pane: "details" | "files" | "preview", current: InfoTab): InfoTab {
  if (pane === "files") return "files";
  if (pane === "preview") return "preview";
  return current === "files" || current === "preview" ? "overview" : current;
}

function statusTone(
  library: number,
  local: number,
): "success" | "warning" | "danger" | "info" | "neutral" {
  if (local === 2 || library === 2 || library === 3) return "warning";
  if (local === 3 || library === 4) return "danger";
  if (local === 4 || library === 5) return "info";
  return library === 1 && local === 1 ? "success" : "neutral";
}

function statusLabel(library: number, local: number) {
  const libraryLabel =
    ["Unspecified", "Current", "Behind", "Deprecated", "Missing", "Unknown"][library] ?? "Unknown";
  const localLabel = ["Unspecified", "Clean", "Modified", "Missing", "Unknown"][local] ?? "Unknown";
  return `${libraryLabel} / ${localLabel}`;
}

function isHook(asset: CatalogAsset) {
  return (asset.assetKind as unknown) === 2 || (asset.assetKind as unknown) === "ASSET_KIND_HOOK";
}

function HookWorkspace({
  asset,
  onClose,
  tab,
  onTabChange,
  selectedStory,
  onSelectedStoryChange,
}: {
  asset: CatalogAsset;
  onClose: () => void;
  tab: InfoTab;
  onTabChange: (tab: InfoTab) => void;
  selectedStory?: string;
  onSelectedStoryChange: (story: string) => void;
}) {
  const { t } = useTranslation();
  const effective = useQuery({
    queryKey: ["adoptions", "effective", asset.id],
    queryFn: () => adoptionsClient.listEffectiveAdoptions({ componentId: asset.id, limit: 100 }),
  });
  return (
    <div data-testid="hook-detail-page" className="flex min-h-0 flex-1 flex-col">
      <ComponentEditor
        id={asset.id}
        libraryId={asset.libraryId || asset.id}
        latestVersion={asset.latestVersion || asset.version}
        onClose={onClose}
        renderable={false}
        activePane={paneForTab(tab)}
        onActivePaneChange={(pane) => onTabChange(tabForPane(pane, tab))}
        selectedStory={selectedStory}
        onSelectedStoryChange={onSelectedStoryChange}
        navigationSlot={
          <DetailTabs
            active={tab}
            onChange={onTabChange}
            versionCount={asset.metrics?.versionCount ?? 0}
            adoptionCount={asset.metrics?.effectiveAdoptionCount ?? 0}
            renderable={false}
          />
        }
        metadataSlot={
          <aside data-testid="hook-workspace-details" className="space-y-space-xs">
            <StatusBadge tone="info">
              {t("catalog.hookFixturePreview", {
                defaultValue: "Live fixture preview — declared inputs and adapters only.",
              })}
            </StatusBadge>
            <div className="flex flex-wrap gap-space-2xs text-xs">
              <StatusBadge tone="neutral">
                {t("catalog.directAdoptions", {
                  defaultValue: "{{count}} direct",
                  count: asset.metrics?.directAdoptionCount ?? 0,
                })}
              </StatusBadge>
              <StatusBadge tone="neutral">
                {t("catalog.effectiveAdoptions", {
                  defaultValue: "{{count}} effective",
                  count: asset.metrics?.effectiveAdoptionCount ?? 0,
                })}
              </StatusBadge>
            </div>
            {tab === "overview" && (
              <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-2 text-xs">
                <dt className="text-app-muted-foreground">
                  {t("catalog.kind", { defaultValue: "Kind" })}
                </dt>
                <dd>{t("catalog.hook", { defaultValue: "Hook" })}</dd>
                <dt className="text-app-muted-foreground">
                  {t("catalog.source", { defaultValue: "Source" })}
                </dt>
                <dd className="break-all font-mono">{asset.sourcePath || "—"}</dd>
              </dl>
            )}
            {tab === "tests" && (
              <ComponentTestPanel
                componentId={asset.id}
                version={asset.latestVersion || asset.version}
              />
            )}
            {tab === "versions" && (
              <VersionsCard
                componentId={asset.id}
                onSelectVersion={() => undefined}
                onCompare={() => undefined}
              />
            )}
            {tab === "adoptions" && (
              <div data-testid="hook-effective-adoptions" className="space-y-space-2xs text-xs">
                {effective.isLoading ? (
                  <p className="text-app-muted-foreground">
                    {t("componentDetail.info.adoptionsLoading", {
                      defaultValue: "Loading adoptions…",
                    })}
                  </p>
                ) : (effective.data?.adoptions ?? []).length === 0 ? (
                  <EmptyState
                    className="p-space-2xs text-xs"
                    title={t("componentDetail.info.noAdoptions", {
                      defaultValue: "No recorded usage.",
                    })}
                  />
                ) : (
                  <ul className="space-y-space-2xs">
                    {(effective.data?.adoptions ?? []).map((entry) => (
                      <li
                        key={`${entry.sourceAssetId}:${entry.parentAdoption?.id}`}
                        className="rounded-control border border-app-border p-space-2xs"
                      >
                        <p className="font-medium">
                          {entry.mediated
                            ? t("catalog.indirectUsage", { defaultValue: "Indirect usage" })
                            : t("catalog.directUsage", { defaultValue: "Direct usage" })}
                        </p>
                        <p className="mt-space-3xs">
                          {entry.parentAdoption?.scenario} · {entry.parentAdoption?.adoptedVersion}
                        </p>
                        <p className="mt-space-3xs font-mono text-app-muted-foreground">
                          {entry.parentAdoption?.id}
                        </p>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </aside>
        }
      />
    </div>
  );
}

export function ComponentDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id } = useParams<{ id: string }>();
  const [search, setSearch] = useSearchParams();
  const [selectedVersion, setSelectedVersion] = useState<string | undefined>();
  const [comparison, setComparison] = useState<ComparisonSession | null>(null);
  const infoTab = assetInfoTab(search);
  const selectedStory = assetStory(search);
  const setInfoTab = (tab: InfoTab) =>
    setSearch(assetSearchForTab(tab, undefined, selectedStory), { replace: true });
  const [selectedAdoptionID, setSelectedAdoptionID] = useState("");
  const [previewExperienceState, setPreviewExperienceState] = useState<
    "loading" | "partial" | "ready" | "error"
  >("loading");

  useEffect(() => {
    if (!id || search.has("tab") || search.has("story")) return;
    try {
      const saved = window.localStorage.getItem(assetNavigationStorageKey(id));
      const navigation = saved ? (JSON.parse(saved) as StoredAssetNavigation) : undefined;
      if (navigation?.tab || navigation?.story)
        setSearch(assetSearchForTab(navigation.tab ?? "preview", undefined, navigation.story), {
          replace: true,
        });
    } catch {
      // Storage may be unavailable in private browsing; URL/default remain authoritative.
    }
  }, [id, search, setSearch]);

  useEffect(() => {
    if (!id) return;
    try {
      window.localStorage.setItem(
        assetNavigationStorageKey(id),
        JSON.stringify({ tab: infoTab, story: selectedStory }),
      );
    } catch {
      // Navigation continuity degrades safely to the URL when storage is unavailable.
    }
  }, [id, infoTab, selectedStory]);

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
      try {
        const byLibrary = await componentsClient.getComponentByLibraryId({ libraryId: id ?? "" });
        if (byLibrary.component) return byLibrary;
      } catch {
        // Hand-typed and externally shared URLs use the bare slug; resolve it
        // against the catalog before declaring the asset missing.
      }
      const list = await componentsClient.listComponents({});
      const match = (list.components ?? []).find((component) => component.slug === id);
      if (match) return componentsClient.getComponent({ id: match.id });
      // Resolving to "no component" (instead of throwing) renders not-found
      // immediately rather than after the query client's retry backoff.
      return { component: undefined };
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
  const sourceContentQuery = useQuery({
    queryKey: ["components", "content", id, "overview"],
    queryFn: () => componentsClient.getComponentContent({ id: data?.component?.id ?? id ?? "" }),
    enabled: Boolean(data?.component) && infoTab === "overview",
    retry: false,
  });

  const adoptionsQuery = useQuery({
    queryKey: ["adoptions", "component", id],
    queryFn: () => adoptionsClient.listAdoptions({ componentId: id ?? "", limit: 0 }),
    enabled: Boolean(id),
  });
  const refreshMutation = useMutation({
    mutationFn: () => adoptionsClient.refreshAdoptions({ componentId: id ?? "" }),
    onSuccess: () =>
      void queryClient.invalidateQueries({ queryKey: ["adoptions", "component", id] }),
  });

  const loadedAsset = catalogAsset.data?.component ?? data?.component;
  useEffect(() => {
    if (loadedAsset && isHook(loadedAsset) && infoTab === "preview") {
      setInfoTab("overview");
    }
  }, [loadedAsset, infoTab]);

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

  const component = loadedAsset ?? data.component;
  if (isHook(component)) {
    // Hooks are catalog assets but have no browser-renderable preview. The
    // effect above repairs stale URLs and saved preferences; this fallback
    // keeps the first render useful while it does so.
    const hookTab = infoTab === "preview" ? "overview" : infoTab;
    return (
      <HookWorkspace
        asset={component}
        onClose={() => {
          void navigate("/");
        }}
        tab={hookTab}
        onTabChange={setInfoTab}
        selectedStory={selectedStory}
        onSelectedStoryChange={(story) =>
          setSearch(assetSearchForTab(hookTab, undefined, story), { replace: true })
        }
      />
    );
  }
  // Proto clients provide empty collections, while test and older persisted
  // projections can omit these optional-shaped values entirely.
  const headers = (component.headers as Record<string, string> | undefined) ?? {};
  const tags = (component.tags as string[] | undefined) ?? [];
  const designStyles = (component.designStyles as typeof component.designStyles | undefined) ?? [];
  const dependencies =
    (component.dependencies as Array<{ libraryId: string; version: string }> | undefined) ?? [];
  const adoptions = adoptionsQuery.data?.adoptions ?? [];
  const selectedAdoption =
    adoptions.find((adoption) => adoption.id === selectedAdoptionID) ?? adoptions[0];

  return (
    <div
      data-testid="component-detail-page"
      data-preview-state={previewExperienceState}
      className="flex min-h-0 min-w-0 w-full flex-1 flex-col"
    >
      <ComponentEditor
        id={component.id}
        libraryId={component.libraryId || component.id}
        latestVersion={component.latestVersion || component.version}
        stageMode={/navigation|pattern|sidebar|shell|pageframe|page-template|bottomnav/i.test(
          component.libraryId || component.id,
        )}
        onClose={() => {
          void navigate("/");
        }}
        selectedVersion={selectedVersion}
        activePane={paneForTab(infoTab)}
        onActivePaneChange={(pane) => setInfoTab(tabForPane(pane, infoTab))}
        selectedStory={selectedStory}
        onSelectedStoryChange={(story) =>
          setSearch(assetSearchForTab(infoTab, undefined, story), { replace: true })
        }
        onPreviewExperienceStateChange={setPreviewExperienceState}
        navigationSlot={
          <DetailTabs
            active={infoTab}
            onChange={setInfoTab}
            versionCount={component.metrics?.versionCount ?? 0}
            adoptionCount={component.metrics?.directAdoptionCount ?? adoptions.length}
          />
        }
        comparison={comparison}
        onCloseComparison={() => setComparison(null)}
        metadataSlot={
          <div className="space-y-space-sm">
            {infoTab === "overview" && (
              <>
                <ComponentExperiencePanel
                  experience={experienceQuery.data}
                  isLoading={experienceQuery.isLoading}
                  isError={experienceQuery.isError}
                />
                <section className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs text-sm text-app-foreground">
                  <h3 className="font-medium">
                    {t("componentDetail.info.identity", { defaultValue: "Identity" })}
                  </h3>
                  <dl className="mt-space-2xs grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                    <dt className="text-app-muted-foreground">
                      {t(strings.components.editor.libraryIdLabel)}
                    </dt>
                    <dd className="break-all font-mono">{component.libraryId}</dd>
                    <dt className="text-app-muted-foreground">
                      {t(strings.components.editor.slotLabel)}
                    </dt>
                    <dd>{component.slot || "—"}</dd>
                    <dt className="text-app-muted-foreground">
                      {t(strings.components.editor.categoryLabel)}
                    </dt>
                    <dd>{component.category || headers.category || "—"}</dd>
                    <dt className="text-app-muted-foreground">
                      {t(strings.components.editor.tagsLabel)}
                    </dt>
                    <dd>{tags.join(", ") || "—"}</dd>
                    <dt className="text-app-muted-foreground">
                      {t("componentDetail.info.sourceHash", { defaultValue: "Source hash" })}
                    </dt>
                    <dd className="break-all font-mono">
                      {sourceContentQuery.data?.sha256 || "—"}
                    </dd>
                  </dl>
                </section>
                <section className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs text-sm text-app-foreground">
                  <h3 className="font-medium">
                    {t("componentDetail.info.affinities", { defaultValue: "Design affinities" })}
                  </h3>
                  {designStyles.length === 0 ? (
                    <p className="mt-space-2xs text-xs text-app-muted-foreground">
                      {t("componentDetail.info.noAffinities", {
                        defaultValue: "No design affinities declared.",
                      })}
                    </p>
                  ) : (
                    <ul className="mt-space-2xs space-y-space-3xs text-xs">
                      {designStyles.map((affinity) => (
                        <li key={affinity.styleId}>
                          {affinity.styleId}: {affinity.reason || affinity.affinity}
                        </li>
                      ))}
                    </ul>
                  )}
                </section>
                <section className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs text-sm text-app-foreground">
                  <h3 className="font-medium">
                    {t("componentDetail.info.dependencies", { defaultValue: "Shared assets" })}
                  </h3>
                  {dependencies.length === 0 ? (
                    <p className="mt-space-2xs text-xs text-app-muted-foreground">
                      {t("componentDetail.info.noDependencies", {
                        defaultValue: "No shared assets declared.",
                      })}
                    </p>
                  ) : (
                    <ul className="mt-space-2xs space-y-space-3xs text-xs">
                      {dependencies.map((dependency) => (
                        <li key={`${dependency.libraryId}@${dependency.version}`}>
                          <Link
                            to={assetPath(dependency.libraryId, { tab: infoTab })}
                            className="font-mono text-app-primary underline-offset-2 hover:underline"
                          >
                            {dependency.libraryId}
                          </Link>
                          <span className="text-app-muted-foreground"> · {dependency.version}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </section>
              </>
            )}
            {infoTab === "tests" && (
              <ComponentTestPanel
                componentId={component.id}
                version={selectedVersion ?? component.latestVersion ?? component.version}
              />
            )}
            {infoTab === "versions" && (
              <VersionsCard
                componentId={component.id}
                selectedVersion={selectedVersion}
                onSelectVersion={setSelectedVersion}
                onCompare={setComparison}
              />
            )}
            {infoTab === "adoptions" && (
              <section
                data-testid="component-detail-adoptions"
                className="space-y-space-xs text-sm text-app-foreground"
              >
                <div className="flex items-center justify-between">
                  <h3 className="font-medium">
                    {t("componentDetail.info.adoptions", { defaultValue: "Adoptions" })}
                  </h3>
                  <Button
                    size="sm"
                    onClick={() => refreshMutation.mutate()}
                    disabled={refreshMutation.isPending}
                  >
                    {refreshMutation.isPending
                      ? t(strings.adoptions.refreshing)
                      : t(strings.adoptions.refreshAction)}
                  </Button>
                </div>
                {adoptionsQuery.isLoading ? (
                  <p className="text-xs text-app-muted-foreground">
                    {t("componentDetail.info.adoptionsLoading", {
                      defaultValue: "Loading adoptions…",
                    })}
                  </p>
                ) : adoptions.length === 0 ? (
                  <EmptyState
                    className="p-space-2xs text-xs"
                    title={t("componentDetail.info.noAdoptions", {
                      defaultValue: "No scenarios have adopted this component yet.",
                    })}
                  />
                ) : (
                  <div className="space-y-space-2xs">
                    {adoptions.map((adoption) => (
                      <Button
                        key={adoption.id}
                        type="button"
                        variant="secondary"
                        onClick={() => setSelectedAdoptionID(adoption.id)}
                        className={`h-auto w-full rounded-control border p-space-2xs text-left ${selectedAdoption?.id === adoption.id ? "border-app-primary" : "border-app-border"}`}
                      >
                        <div className="flex items-center justify-between gap-space-2xs">
                          <span className="font-medium">{adoption.scenario}</span>
                          <StatusBadge
                            tone={statusTone(adoption.libraryVersionStatus, adoption.localStatus)}
                          >
                            {statusLabel(adoption.libraryVersionStatus, adoption.localStatus)}
                          </StatusBadge>
                        </div>
                        <p className="mt-space-3xs font-mono text-xs text-app-muted-foreground">
                          {adoption.adoptedVersion} · {adoption.adoptedPath}
                        </p>
                      </Button>
                    ))}
                  </div>
                )}
                {selectedAdoption && (
                  <ul
                    data-testid="component-detail-adoption-file-tree"
                    className="space-y-space-3xs rounded-control bg-app-background p-space-2xs font-mono text-xs text-app-muted-foreground"
                  >
                    {(selectedAdoption.files.length > 0
                      ? selectedAdoption.files.map((file) => file.adoptedPath)
                      : [selectedAdoption.adoptedPath]
                    ).map((path) => (
                      <li key={path}>{path}</li>
                    ))}
                  </ul>
                )}
                <AdoptionsCard componentId={component.id} suggestionsOnly />
              </section>
            )}
          </div>
        }
      />
    </div>
  );
}
