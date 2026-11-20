/**
 * The canonical public URL contract for the library workspace.
 *
 * Keep URL construction here rather than scattering string templates through
 * pages.  It makes deep links stable and gives the server and UI a small,
 * reviewable set of routes to protect.
 */
export const appRoutes = {
  /** Start: the search and browse front door. */
  start: "/",
  library: "/library",
  /** Legacy asset-catalog URL; redirects to `/library`. */
  components: "/components",
  /** Legacy asset-catalog alias; redirects to `/library`. */
  assetCatalog: "/catalog",
  asset: "/assets/:id",
  preview: "/assets/:id/preview",
  stats: "/stats",
  /** Legacy measurement URLs; each redirects to its Stats view. */
  coverage: "/coverage",
  capabilities: "/capabilities",
  activity: "/activity",
  review: "/review",
  settings: "/settings",
  create: "/new",
  design: "/design",
  designScenario: "/design/:scenario",
  designPage: "/design/:scenario/:page",
  legacySketch: "/sketch/:scenario/:page",
} as const;

export const statsViews = ["overview", "quality", "activity", "services"] as const;
export type StatsView = (typeof statsViews)[number];

const statsViewSet = new Set<string>(statsViews);

/** Read the Stats tab from `?view=`, defaulting to Overview. */
export function statsView(search: URLSearchParams): StatsView {
  const view = search.get("view");
  return view !== null && statsViewSet.has(view) ? (view as StatsView) : "overview";
}

/** Canonical Stats URL. Overview omits the parameter so `/stats` stays clean. */
export function statsPath(view: StatsView = "overview"): string {
  return view === "overview" ? appRoutes.stats : `${appRoutes.stats}?view=${view}`;
}

/** Workspace studio URL. Page/display/state default to the workspace's first. */
export function workspacePath(
  id: string,
  selection: { page?: string; display?: string; state?: string } = {},
): string {
  const search = new URLSearchParams({ workspace: id });
  if (selection.page) search.set("page", selection.page);
  if (selection.display) search.set("display", selection.display);
  if (selection.state) search.set("state", selection.state);
  return `${appRoutes.design}?${search.toString()}`;
}

/** Library URL carrying an optional search term for the asset index. */
export function libraryPath(query?: string): string {
  const term = query?.trim();
  return term ? `${appRoutes.library}?q=${encodeURIComponent(term)}` : appRoutes.library;
}

export type AssetInfoTab =
  | "preview"
  | "overview"
  | "files"
  | "versions"
  | "tests"
  | "experience"
  | "relationships";
export type AssetRouteState = {
  tab?: AssetInfoTab;
  story?: string;
  testReport?: string;
};

const assetTabs = new Set<AssetInfoTab>([
  "overview",
  "preview",
  "files",
  "versions",
  "tests",
  "experience",
  "relationships",
]);

export function assetPath(assetID: string, state: AssetRouteState = {}): string {
  const search = new URLSearchParams();
  if (state.tab && state.tab !== "preview") search.set("tab", state.tab);
  if (state.story) search.set("story", state.story);
  const serialized = search.toString();
  return `/assets/${encodeURIComponent(assetID)}${serialized ? `?${serialized}` : ""}`;
}

export function previewPath(assetID: string, story?: string): string {
  const search = new URLSearchParams();
  if (story) search.set("story", story);
  const serialized = search.toString();
  return `/assets/${encodeURIComponent(assetID)}/preview${serialized ? `?${serialized}` : ""}`;
}

export function assetInfoTab(search: URLSearchParams): AssetInfoTab {
  const tab = search.get("tab");
  return tab !== null && assetTabs.has(tab as AssetInfoTab) ? (tab as AssetInfoTab) : "preview";
}

export function assetStory(search: URLSearchParams): string | undefined {
  const story = search.get("story")?.trim();
  return story || undefined;
}

export function assetSearchForTab(tab: AssetInfoTab, reportID?: string, story?: string): string {
  const search = new URLSearchParams();
  if (reportID && tab === "overview") {
    search.set("tab", "tests");
    search.set("testReport", reportID);
  } else if (tab !== "preview") search.set("tab", tab);
  if (story) search.set("story", story);
  const serialized = search.toString();
  return serialized ? `?${serialized}` : "";
}

export function assetTestReportPath(assetID: string, reportID: string): string {
  return `${assetPath(assetID, { tab: "preview" })}${assetSearchForTab("overview", reportID)}`;
}

export function designPath(scenario?: string, page?: string): string {
  if (!scenario) return appRoutes.design;
  return `${appRoutes.design}/${encodeURIComponent(scenario)}${page ? `/${encodeURIComponent(page)}` : ""}`;
}
