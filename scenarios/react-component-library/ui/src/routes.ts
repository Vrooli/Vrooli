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
  /** Read-only composition target used exclusively by BAS workspace captures. */
  workspaceCapture: "/workspace-capture/:id",
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

export function statsPath(view: StatsView = "overview"): string {
  return view === "overview" ? appRoutes.stats : `${appRoutes.stats}?view=${view}`;
}

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

/** Library URL filtered by its real catalog domain and asset group. */
export function libraryDomainPath(domain: string, group: string, query?: string): string {
  const search = new URLSearchParams({ domain, group });
  if (query?.trim()) search.set("q", query.trim());
  return `${appRoutes.library}?${search.toString()}`;
}

export function designPath(scenario?: string, page?: string): string {
  if (!scenario) return appRoutes.design;
  return `${appRoutes.design}/${encodeURIComponent(scenario)}${page ? `/${encodeURIComponent(page)}` : ""}`;
}
