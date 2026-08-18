/**
 * The canonical public URL contract for the library workspace.
 *
 * Keep URL construction here rather than scattering string templates through
 * pages.  It makes deep links stable and gives the server and UI a small,
 * reviewable set of routes to protect.
 */
export const appRoutes = {
  catalog: "/",
  assetCatalog: "/catalog",
  asset: "/assets/:id",
  preview: "/assets/:id/preview",
  coverage: "/coverage",
  capabilities: "/capabilities",
  settings: "/settings",
} as const;

export type AssetInfoTab =
  | "preview"
  | "overview"
  | "files"
  | "tests"
  | "versions"
  | "progression"
  | "adoptions"
  | "relationships";
export type AssetRouteState = {
  tab?: AssetInfoTab;
  story?: string;
  testReport?: string;
  view?: "focus" | "canvas";
};

const assetTabs = new Set<AssetInfoTab>([
  "overview",
  "preview",
  "files",
  "tests",
  "versions",
  "progression",
  "adoptions",
  "relationships",
]);

export function assetPath(assetID: string, state: AssetRouteState = {}): string {
  const search = new URLSearchParams();
  if (state.tab && state.tab !== "preview") search.set("tab", state.tab);
  if (state.story) search.set("story", state.story);
  if (state.testReport && state.tab === "tests") search.set("testReport", state.testReport);
  if (state.view) search.set("view", state.view);
  const serialized = search.toString();
  return `/assets/${encodeURIComponent(assetID)}${serialized ? `?${serialized}` : ""}`;
}

export function previewPath(assetID: string, story?: string, view?: "focus" | "canvas"): string {
  const search = new URLSearchParams();
  if (story) search.set("story", story);
  if (view) search.set("view", view);
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
  if (tab !== "preview") search.set("tab", tab);
  if (story) search.set("story", story);
  if (tab === "tests" && reportID) search.set("testReport", reportID);
  const serialized = search.toString();
  return serialized ? `?${serialized}` : "";
}

export function assetTestReportPath(assetID: string, reportID: string): string {
  return `${assetPath(assetID)}${assetSearchForTab("tests", reportID)}`;
}
