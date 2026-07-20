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
  settings: "/settings",
} as const;

export type AssetInfoTab = "overview" | "preview" | "files" | "tests" | "versions" | "adoptions";

const assetTabs = new Set<AssetInfoTab>(["overview", "preview", "files", "tests", "versions", "adoptions"]);

export function assetPath(assetID: string): string {
  return `/assets/${encodeURIComponent(assetID)}`;
}

export function assetInfoTab(search: URLSearchParams): AssetInfoTab {
  const tab = search.get("tab");
  return tab !== null && assetTabs.has(tab as AssetInfoTab) ? tab as AssetInfoTab : "overview";
}

export function assetSearchForTab(tab: AssetInfoTab, reportID?: string): string {
  const search = new URLSearchParams();
  if (tab !== "overview") search.set("tab", tab);
  if (tab === "tests" && reportID) search.set("testReport", reportID);
  const serialized = search.toString();
  return serialized ? `?${serialized}` : "";
}

export function assetTestReportPath(assetID: string, reportID: string): string {
  return `${assetPath(assetID)}${assetSearchForTab("tests", reportID)}`;
}
