import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, ArrowRight, Boxes, CheckCircle2, Clock3, Palette, RefreshCw } from "lucide-react";
import { Link, useLocation } from "react-router-dom";

import { adoptionsClient, LibraryVersionStatus, LocalStatus } from "../api/adoptions";
import { listCatalogAssets, type CatalogAsset } from "../api/components";
import { useTranslation } from "../i18n";
import { assetInfoTab, assetPath } from "../routes";

function countAdoptionIssues(adoptions: Awaited<ReturnType<typeof adoptionsClient.listAdoptions>>["adoptions"]) {
  return adoptions.filter((adoption) =>
    adoption.libraryVersionStatus !== LibraryVersionStatus.UNSPECIFIED &&
    adoption.libraryVersionStatus !== LibraryVersionStatus.CURRENT ||
    adoption.localStatus === LocalStatus.MODIFIED || adoption.localStatus === LocalStatus.MISSING,
  ).length;
}

function Metric({ label, value, tone = "default" }: { label: string; value: number; tone?: "default" | "warning" }) {
  return <div className="rounded-panel border border-app-border bg-app-surface p-3">
    <dt className="text-xs text-app-muted-foreground">{label}</dt>
    <dd className={tone === "warning" ? "mt-1 text-2xl font-semibold text-app-danger" : "mt-1 text-2xl font-semibold text-app-foreground"}>{value}</dd>
  </div>;
}

function AssetLink({ asset, tab }: { asset: CatalogAsset; tab?: ReturnType<typeof assetInfoTab> }) {
  return <Link to={assetPath(asset.id, tab ? { tab } : {})} className="flex items-center justify-between gap-3 rounded-control px-2 py-2 text-sm hover:bg-app-surface-muted">
    <span className="truncate font-medium">{asset.displayName || asset.libraryId}</span>
    <span className="shrink-0 text-xs text-app-muted-foreground">v{asset.version || "draft"}</span>
  </Link>;
}

export function DashboardPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const currentTab = location.pathname.startsWith("/assets/") ? assetInfoTab(new URLSearchParams(location.search)) : undefined;
  const assets = useQuery({ queryKey: ["dashboard", "assets"], queryFn: () => listCatalogAssets({ limit: 200, assetKind: 1 }), staleTime: 30_000 });
  const adoptions = useQuery({ queryKey: ["dashboard", "adoptions"], queryFn: () => adoptionsClient.listAdoptions({ limit: 500 }), staleTime: 30_000 });
  const allAssets = assets.data?.components ?? [];
  const allAdoptions = adoptions.data?.adoptions ?? [];
  const scenarios = new Map<string, number>();
  for (const adoption of allAdoptions) scenarios.set(adoption.scenario, (scenarios.get(adoption.scenario) ?? 0) + 1);
  const issueCount = countAdoptionIssues(allAdoptions);
  const recentlyEvolved = [...allAssets].sort((a, b) => (b.updatedAt?.seconds ?? 0n) > (a.updatedAt?.seconds ?? 0n) ? 1 : -1).slice(0, 4);

  return <div data-testid="dashboard-page" className="mx-auto flex w-full max-w-6xl flex-col gap-6">
    <section aria-labelledby="dashboard-welcome">
      <p className="text-sm text-app-muted-foreground">{t("dashboard.greeting", { defaultValue: "Good morning" })}</p>
      <h2 id="dashboard-welcome" className="mt-1 text-2xl font-semibold tracking-tight">{t("dashboard.review", { defaultValue: "Review attention items" })}</h2>
    </section>

    <dl className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <Metric label={t("dashboard.assets", { defaultValue: "Assets" })} value={allAssets.length} />
      <Metric label={t("dashboard.adoptionCurrency", { defaultValue: "Adoption currency" })} value={allAdoptions.length - issueCount} />
      <Metric label={t("dashboard.needsReview", { defaultValue: "Need review" })} value={issueCount} tone={issueCount > 0 ? "warning" : "default"} />
    </dl>

    <div className="grid gap-4 lg:grid-cols-2">
      <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="attention-heading">
        <div className="flex items-center gap-2"><AlertTriangle aria-hidden className="h-4 w-4 text-app-warning" /><h2 id="attention-heading" className="font-semibold">{t("dashboard.attention", { defaultValue: "Needs your attention" })}</h2></div>
        {adoptions.isLoading ? <p className="mt-3 text-sm text-app-muted-foreground">{t("dashboard.loading", { defaultValue: "Loading dashboard…" })}</p> : issueCount > 0 ? <p className="mt-3 text-sm text-app-muted-foreground">{t("dashboard.attentionDetail", { defaultValue: "{{count}} adoption(s) have version or local-change drift.", count: issueCount })}</p> : <p className="mt-3 flex items-center gap-2 text-sm text-app-muted-foreground"><CheckCircle2 aria-hidden className="h-4 w-4 text-app-success" />{t("dashboard.clear", { defaultValue: "No adoption drift needs review." })}</p>}
      </section>
      <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="evolved-heading">
        <div className="flex items-center gap-2"><Clock3 aria-hidden className="h-4 w-4 text-app-muted-foreground" /><h2 id="evolved-heading" className="font-semibold">{t("dashboard.recentlyEvolved", { defaultValue: "Recently evolved" })}</h2></div>
        <div className="mt-2 divide-y divide-app-border">{recentlyEvolved.length ? recentlyEvolved.map((asset) => <AssetLink key={asset.id} asset={asset} tab={currentTab} />) : <p className="py-2 text-sm text-app-muted-foreground">{t("dashboard.noAssets", { defaultValue: "No assets yet." })}</p>}</div>
      </section>
      <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="adoption-heading">
        <div className="flex items-center gap-2"><Boxes aria-hidden className="h-4 w-4 text-app-muted-foreground" /><h2 id="adoption-heading" className="font-semibold">{t("dashboard.adoptionHealth", { defaultValue: "Adoption health by scenario" })}</h2></div>
        <div className="mt-3 space-y-2">{scenarios.size ? [...scenarios.entries()].sort(([a], [b]) => a.localeCompare(b)).slice(0, 5).map(([scenario, count]) => <div key={scenario} className="flex items-center justify-between text-sm"><span>{scenario}</span><span className="text-app-muted-foreground">{count}</span></div>) : <p className="text-sm text-app-muted-foreground">{t("dashboard.noAdoptions", { defaultValue: "No adoptions recorded yet." })}</p>}</div>
      </section>
      <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="moves-heading">
        <div className="flex items-center gap-2"><RefreshCw aria-hidden className="h-4 w-4 text-app-muted-foreground" /><h2 id="moves-heading" className="font-semibold">{t("dashboard.nextMoves", { defaultValue: "Suggested next moves" })}</h2></div>
        <div className="mt-3 space-y-2 text-sm"><Link to="/catalog" className="flex items-center justify-between rounded-control px-2 py-2 hover:bg-app-surface-muted"><span>{t("dashboard.browseAssets", { defaultValue: "Browse and refine assets" })}</span><ArrowRight aria-hidden className="h-4 w-4" /></Link><Link to="/settings" className="flex items-center justify-between rounded-control px-2 py-2 hover:bg-app-surface-muted"><span>{t("dashboard.reviewPreferences", { defaultValue: "Review workspace preferences" })}</span><Palette aria-hidden className="h-4 w-4" /></Link></div>
      </section>
    </div>
  </div>;
}
