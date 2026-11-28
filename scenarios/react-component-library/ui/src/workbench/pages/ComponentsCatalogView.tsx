import type { CatalogAsset, CoverageRow, AssetGroup } from "../data";
import { CatalogToolbar } from "./ComponentsCatalogToolbar";
import { CatalogResults } from "./ComponentsCatalogResults";
import { CatalogResultPanel } from "./ComponentsCatalogResultPanel";
import { CatalogSideRail } from "./ComponentsCatalogSideRail";

type View = "gallery" | "list" | "tree";
type StatusFilter = "all" | "review" | "verified" | "drafts";
type Sort = "attention" | "name" | "recent" | "adoptions";

export interface ComponentsCatalogViewProps {
  catalog: { isLoading: boolean; isError: boolean };
  coverage: { isError: boolean };
  assets: CatalogAsset[];
  rows: Map<string, CoverageRow>;
  grouped: Record<AssetGroup, number>;
  domains: [string, number][];
  filtered: CatalogAsset[];
  visible: CatalogAsset[];
  collections: { name: string; count: number }[];
  maxCollection: number;
  verifiedRatio: number;
  reviewCount: number | null;
  query: string;
  setQuery: (query: string) => void;
  group: AssetGroup;
  setGroup: (group: AssetGroup) => void;
  view: View;
  setView: (view: View) => void;
  status: StatusFilter;
  setStatus: (status: StatusFilter) => void;
  domain: string;
  setDomain: (domain: string) => void;
  sort: Sort;
  setSort: (sort: Sort) => void;
  setLimit: (update: (limit: number) => number) => void;
  onCreate: () => void;
}

export function ComponentsCatalogView(props: ComponentsCatalogViewProps) {
  return (
    <div
      className="wb-page"
      data-experience-surface="catalog-results"
      data-testid="components-page"
    >
      <CatalogToolbar {...props} />
      <CatalogResults {...props} />
      <div className="wb-catalog-layout">
        <CatalogResultPanel {...props} />
        <CatalogSideRail {...props} />
      </div>
    </div>
  );
}
