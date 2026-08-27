/** @vrooliComponentSource navigation.page
 *
 * ComponentDetailPage — full-width editor + preview for a component.
 *
 * Resolves the component id from the route param, fetches the
 * component record so the editor header shows the libraryId (instead
 * of the bare id), and hands the rest off to `<ComponentEditor />`.
 * Closing the editor returns the user to the components list.
 */
import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";

import { Tabs } from "@vrooli/react-component-library/Tabs/1.0.0";
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
import { ComponentExperiencePanel } from "../features/components/ComponentExperiencePanel";
import { VersionsCard } from "../features/versions/VersionsCard";
import { VersionCleanupPanel } from "../features/versions/VersionCleanupPanel";
import { RelationshipsPanel } from "../features/catalog/RelationshipsPanel";
import { versionsClient } from "../api/versions";
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
  renderable = true,
}: {
  active: InfoTab;
  onChange: (tab: InfoTab) => void;
  versionCount: number;
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
    {
      id: "versions",
      label: t("componentDetail.info.versions", { defaultValue: "Versions" }),
      count: versionCount,
    },
    {
      id: "experience",
      label: t("componentDetail.info.experience", { defaultValue: "Experience" }),
    },
    { id: "relationships", label: "Relationships" },
  ];

  return (
    <div className="min-w-0" data-testid="component-detail-tabs-scroll">
      <Tabs
        items={tabs.map(({ id, label, count }) => ({
          id,
          label,
          ...(count !== undefined ? { badge: count } : {}),
        }))}
        active={active}
        onChange={(next) => onChange(next as InfoTab)}
        ariaLabel={t("componentDetail.info.tabs", { defaultValue: "Asset information" })}
        itemTestId={(item) =>
          item === "overview"
            ? selectors.assets.hookOverviewTab
            : item === "files"
              ? selectors.assets.hookFilesTab
              : item === "preview"
                ? selectors.assets.componentPreviewTab
                : undefined
        }
      />
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
              <dl className="grid grid-cols-[auto_1fr] gap-x-space-xs gap-y-space-2xs text-xs">
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
            {tab === "versions" && (
              <div className="space-y-space-md">
                <VersionCleanupPanel componentId={asset.id} compact />
                <VersionsCard
                  componentId={asset.id}
                  onSelectVersion={() => undefined}
                  onCompare={() => undefined}
                />
              </div>
            )}
            {tab === "relationships" && <RelationshipsPanel assetId={asset.id} />}
          </aside>
        }
      />
    </div>
  );
}

export function ComponentDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [search, setSearch] = useSearchParams();
  const [selectedVersion, setSelectedVersion] = useState<string | undefined>();
  const [comparison, setComparison] = useState<ComparisonSession | null>(null);
  const infoTab = assetInfoTab(search);
  const selectedStory = assetStory(search);
  const requestedPreviewView = search.get("view");
  const setInfoTab = (tab: InfoTab) =>
    setSearch(assetSearchForTab(tab, undefined, selectedStory), { replace: true });
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
    enabled:
      Boolean(id) && Boolean(data?.component) && (infoTab === "overview" || infoTab === "experience"),
    retry: false,
  });
  const sourceContentQuery = useQuery({
    queryKey: ["components", "content", id, "overview"],
    queryFn: () => componentsClient.getComponentContent({ id: data?.component?.id ?? id ?? "" }),
    enabled: Boolean(data?.component) && infoTab === "overview",
    retry: false,
  });
  const indexedVersionsQuery = useQuery({
    queryKey: ["versions", "source-status", id],
    queryFn: () =>
      versionsClient.listVersions({ componentId: data?.component?.id ?? id ?? "", limit: 0 }),
    enabled: Boolean(data?.component),
    retry: false,
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
  const indexedRelease = (indexedVersionsQuery.data?.versions ?? []).find(
    (version) => version.version === (component.latestVersion || component.version),
  );
  const sourceDrifted = Boolean(
    sourceContentQuery.data?.sha256 &&
      indexedRelease?.contentSha256 &&
      sourceContentQuery.data.sha256 !== indexedRelease.contentSha256,
  );
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
        stageMode={
          requestedPreviewView === "focus"
            ? true
            : requestedPreviewView === "canvas"
              ? false
              : /navigation|pattern|sidebar|shell|pageframe|page-template|bottomnav/i.test(
                  component.libraryId || component.id,
                )
        }
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
          />
        }
        comparison={comparison}
        onCloseComparison={() => setComparison(null)}
        metadataSlot={
          <div className="space-y-space-sm">
            <section
              data-testid="component-source-status"
              className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs text-sm text-app-foreground"
            >
              <h3 className="font-medium">Source and index</h3>
              <dl className="mt-space-2xs grid grid-cols-[auto_1fr] gap-x-space-xs gap-y-space-3xs text-xs">
                <dt className="text-app-muted-foreground">Indexed version</dt>
                <dd className="font-mono">{component.latestVersion || component.version || "—"}</dd>
                <dt className="text-app-muted-foreground">Source path</dt>
                <dd className="break-all font-mono">{component.sourcePath || "—"}</dd>
              </dl>
              {sourceDrifted ? (
                <StatusBadge tone="warning" className="mt-space-2xs">
                  Source changed since indexing — run components refresh.
                </StatusBadge>
              ) : (
                <StatusBadge tone="success" className="mt-space-2xs">
                  Source matches indexed release.
                </StatusBadge>
              )}
            </section>
            {infoTab === "experience" && (
              <ComponentExperiencePanel
                experience={experienceQuery.data}
                isLoading={experienceQuery.isLoading}
                isError={experienceQuery.isError}
              />
            )}
            {infoTab === "overview" && (
              <>
                <section className="rounded-lg border border-app-border bg-app-surface-muted p-space-xs text-sm text-app-foreground">
                  <h3 className="font-medium">
                    {t("componentDetail.info.identity", { defaultValue: "Identity" })}
                  </h3>
                  <dl className="mt-space-2xs grid grid-cols-[auto_1fr] gap-x-space-xs gap-y-space-3xs text-xs">
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
            {infoTab === "versions" && (
              <div className="space-y-space-md">
                <VersionCleanupPanel componentId={component.id} compact />
                <VersionsCard
                  componentId={component.id}
                  selectedVersion={selectedVersion}
                  onSelectVersion={setSelectedVersion}
                  onCompare={setComparison}
                />
              </div>
            )}
            {infoTab === "relationships" && (
              <RelationshipsPanel assetId={loadedAsset?.catalogId ?? component.id} />
            )}
          </div>
        }
      />
    </div>
  );
}
