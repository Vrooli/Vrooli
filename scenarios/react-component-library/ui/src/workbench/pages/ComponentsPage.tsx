import "../components.css";
import { useDeferredValue, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useSearchParams } from "react-router-dom";
import { appRoutes } from "../../routes";
import type { CoverageRow } from "../data";
import {
  assetAdoptions,
  assetDomain,
  assetGroup,
  assetHasDraft,
  useCatalog,
  useCoverage,
  type AssetGroup,
} from "../data";
import { getReviewSummary } from "../../api/review";
import { ComponentsCatalogView } from "./ComponentsCatalogView";

function coverageIndex(rows: CoverageRow[] | undefined): Map<string, CoverageRow> {
  const index = new Map<string, CoverageRow>();
  for (const row of rows ?? []) {
    index.set(row.assetId, row);
    index.set(row.name, row);
  }
  return index;
}

type View = "gallery" | "list" | "tree";
type StatusFilter = "all" | "review" | "verified" | "drafts";

export function ComponentsPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const catalog = useCatalog();
  const coverage = useCoverage();
  const [query, setQuery] = useState(searchParams.get("q") ?? "");
  const deferredQuery = useDeferredValue(query);
  const requestedGroup = searchParams.get("group");
  const initialGroup: AssetGroup =
    requestedGroup === "hooks" || requestedGroup === "support" ? requestedGroup : "components";
  const [group, setGroup] = useState<AssetGroup>(initialGroup);
  const [view, setView] = useState<View>("gallery");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [domain, setDomain] = useState(searchParams.get("domain") ?? "all");
  const [sort, setSort] = useState<"attention" | "name" | "recent" | "adoptions">("attention");
  const [limit, setLimit] = useState(60);

  const rows = useMemo(() => coverageIndex(coverage.data?.rows), [coverage.data]);
  // The queue count is a server aggregate. Listing findings here capped the
  // number at the page size and under-reported a 13.9k backlog as "400".
  const review = useQuery({
    queryKey: ["review", "summary"],
    queryFn: getReviewSummary,
    retry: 0,
  });
  const reviewCount = review.isError ? null : (review.data?.open ?? null);

  const assets = useMemo(() => catalog.data ?? [], [catalog.data]);
  const grouped = useMemo(() => {
    const counts: Record<AssetGroup, number> = { components: 0, hooks: 0, support: 0 };
    for (const asset of assets) counts[assetGroup(asset)] += 1;
    return counts;
  }, [assets]);

  const domains = useMemo(() => {
    const counts = new Map<string, number>();
    for (const asset of assets) {
      if (assetGroup(asset) !== group) continue;
      const key = assetDomain(asset);
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    return [...counts.entries()].sort(([left], [right]) => left.localeCompare(right));
  }, [assets, group]);

  const filtered = useMemo(() => {
    const term = deferredQuery.trim().toLowerCase();
    const result = assets.filter((asset) => {
      if (assetGroup(asset) !== group) return false;
      if (domain !== "all" && assetDomain(asset) !== domain) return false;
      const row = rows.get(asset.catalogId) ?? rows.get(asset.libraryId);
      if (status === "drafts" && !assetHasDraft(asset)) return false;
      if (status === "review" && !(row?.failedGates ?? []).length) return false;
      if (status === "verified") {
        if (!row || row.failedGates.length > 0 || !row.achieved) return false;
      }
      if (!term) return true;
      return (
        asset.displayName.toLowerCase().includes(term) ||
        asset.libraryId.toLowerCase().includes(term) ||
        asset.catalogId.toLowerCase().includes(term) ||
        asset.description.toLowerCase().includes(term)
      );
    });
    result.sort((left, right) => {
      if (sort === "name") return left.displayName.localeCompare(right.displayName);
      if (sort === "adoptions") return assetAdoptions(right) - assetAdoptions(left);
      if (sort === "recent") {
        const leftTime = left.updatedAt as { toDate?: () => Date } | undefined;
        const rightTime = right.updatedAt as { toDate?: () => Date } | undefined;
        return (rightTime?.toDate?.().getTime() ?? 0) - (leftTime?.toDate?.().getTime() ?? 0);
      }
      const leftRow = rows.get(left.catalogId) ?? rows.get(left.libraryId);
      const rightRow = rows.get(right.catalogId) ?? rows.get(right.libraryId);
      const leftFail = (leftRow?.failedGates ?? []).length;
      const rightFail = (rightRow?.failedGates ?? []).length;
      return (
        rightFail - leftFail || (rightRow?.blocksDownstream ?? 0) - (leftRow?.blocksDownstream ?? 0)
      );
    });
    return result;
  }, [assets, deferredQuery, domain, group, rows, sort, status]);

  const visible = useMemo(() => filtered.slice(0, limit), [filtered, limit]);

  const collections = useMemo(
    () =>
      [...domains]
        .map(([name, count]) => ({ name, count }))
        .sort((left, right) => right.count - left.count)
        .slice(0, 7),
    [domains],
  );
  const maxCollection = Math.max(1, ...collections.map((entry) => entry.count));

  const verifiedRatio = useMemo(() => {
    const eligible = assets.filter((asset) => assetGroup(asset) === "components");
    if (eligible.length === 0) return 0;
    const verified = eligible.filter((asset) => {
      const row = rows.get(asset.catalogId) ?? rows.get(asset.libraryId);
      return Boolean(row && row.failedGates.length === 0 && row.achieved);
    }).length;
    return verified / eligible.length;
  }, [assets, rows]);

  return (
    <ComponentsCatalogView
      catalog={catalog}
      coverage={coverage}
      assets={assets}
      rows={rows}
      grouped={grouped}
      domains={domains}
      filtered={filtered}
      visible={visible}
      collections={collections}
      maxCollection={maxCollection}
      verifiedRatio={verifiedRatio}
      reviewCount={reviewCount}
      query={query}
      setQuery={setQuery}
      group={group}
      setGroup={setGroup}
      view={view}
      setView={setView}
      status={status}
      setStatus={setStatus}
      domain={domain}
      setDomain={setDomain}
      sort={sort}
      setSort={setSort}
      setLimit={setLimit}
      onCreate={() => void navigate(appRoutes.create)}
    />
  );
}
