import { useTranslation } from "../../i18n";
import type { ComponentsCatalogViewProps as Props } from "./ComponentsCatalogView";
import { Filter, Grid2X2, List, Network, Plus } from "lucide-react";
import { assetDomain, assetHasDraft } from "../data";
import { SearchBox, Segmented, Select, Tabs, WbButton } from "../primitives";

type ToolbarProps = Pick<
  Props,
  | "catalog"
  | "assets"
  | "grouped"
  | "filtered"
  | "reviewCount"
  | "query"
  | "setQuery"
  | "group"
  | "setGroup"
  | "view"
  | "setView"
  | "status"
  | "setStatus"
  | "domain"
  | "setDomain"
  | "domains"
  | "sort"
  | "setSort"
  | "onCreate"
>;

export function CatalogToolbar({
  catalog,
  assets,
  grouped,
  filtered,
  reviewCount,
  query,
  setQuery,
  group,
  setGroup,
  view,
  setView,
  status,
  setStatus,
  domain,
  setDomain,
  domains,
  sort,
  setSort,
  onCreate,
}: ToolbarProps) {
  const { t } = useTranslation();
  return (
    <>
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">
            {t("workbench.components.kicker", { defaultValue: "Library · Assets" })}
          </p>
          <h1 className="wb-h1">{t("workbench.components.title", { defaultValue: "Library" })}</h1>
          <p className="wb-lede">
            {catalog.isLoading
              ? t("workbench.components.loadingLede", {
                  defaultValue: "Loading the published library…",
                })
              : t("workbench.components.lede", {
                  defaultValue:
                    "{{count}} published assets across {{domains}} domains. Search the surface, inspect the evidence, or open a component into a real page.",
                  count: assets.length,
                  domains: new Set(assets.map(assetDomain)).size,
                })}
          </p>
        </div>
        <div className="wb-actions">
          <WbButton variant="ghost" size="md">
            <Filter aria-hidden size={14} />
            {t("workbench.components.filters", { defaultValue: "Filters" })}
          </WbButton>
          <WbButton variant="primary" size="md" onClick={onCreate}>
            <Plus aria-hidden size={14} />
            {t("workbench.components.new", { defaultValue: "New asset" })}
          </WbButton>
        </div>
      </div>

      <Tabs
        label={t("workbench.components.kinds", { defaultValue: "Asset kinds" })}
        value={group}
        onChange={setGroup}
        items={[
          {
            id: "components",
            label: t("workbench.components.kindComponents", { defaultValue: "Components" }),
            count: grouped.components,
          },
          {
            id: "hooks",
            label: t("workbench.components.kindHooks", { defaultValue: "Hooks" }),
            count: grouped.hooks,
          },
          {
            id: "support",
            label: t("workbench.components.kindSupport", { defaultValue: "Support assets" }),
            count: grouped.support,
          },
        ]}
      />

      <div style={{ height: 12 }} />

      <div className="wb-filterbar">
        <SearchBox
          value={query}
          onChange={setQuery}
          label={t("workbench.components.search", { defaultValue: "Filter assets" })}
          placeholder={t("workbench.components.searchPlaceholder", {
            defaultValue: "Filter assets, ids, domains",
          })}
          shortcut={t("workbench.components.searchKey", { defaultValue: "/" })}
        />
        <div className="wb-row" style={{ flexWrap: "wrap" }}>
          <button
            type="button"
            onClick={() => setStatus("all")}
            className="wb-chip"
            aria-pressed={status === "all"}
          >
            {t("workbench.components.all", { defaultValue: "All" })} {filtered.length}
          </button>
          <button
            type="button"
            onClick={() => setStatus("review")}
            className="wb-chip"
            aria-pressed={status === "review"}
          >
            {t("workbench.components.review", { defaultValue: "Needs review" })}{" "}
            {reviewCount ?? "—"}
          </button>
          <button
            type="button"
            onClick={() => setStatus("verified")}
            className="wb-chip"
            aria-pressed={status === "verified"}
          >
            {t("workbench.components.verified", { defaultValue: "Verified" })}
          </button>
          <button
            type="button"
            onClick={() => setStatus("drafts")}
            className="wb-chip"
            aria-pressed={status === "drafts"}
          >
            {t("workbench.components.drafts", { defaultValue: "Drafts" })}{" "}
            {assets.filter(assetHasDraft).length}
          </button>
        </div>
        <Select
          aria-label={t("workbench.components.domain", { defaultValue: "Filter by domain" })}
          value={domain}
          onChange={(event) => setDomain(event.target.value)}
        >
          <option value="all">
            {t("workbench.components.allDomains", { defaultValue: "Domain: All" })}
          </option>
          {domains.map(([name, count]) => (
            <option key={name} value={name}>
              {name} ({count})
            </option>
          ))}
        </Select>
        <Select
          aria-label={t("workbench.components.sort", { defaultValue: "Sort assets" })}
          value={sort}
          onChange={(event) => setSort(event.target.value as typeof sort)}
        >
          <option value="attention">
            {t("workbench.components.sortAttention", { defaultValue: "Sort: Attention" })}
          </option>
          <option value="recent">
            {t("workbench.components.sortRecent", { defaultValue: "Sort: Recent" })}
          </option>
          <option value="adoptions">
            {t("workbench.components.sortAdoptions", { defaultValue: "Sort: Adoptions" })}
          </option>
          <option value="name">
            {t("workbench.components.sortName", { defaultValue: "Sort: Name" })}
          </option>
        </Select>
        <Segmented
          label={t("workbench.components.view", { defaultValue: "Result view" })}
          value={view}
          onChange={setView}
          items={[
            {
              id: "gallery",
              label: <Grid2X2 aria-hidden size={14} />,
              ariaLabel: t("workbench.components.viewGallery", { defaultValue: "Gallery" }),
            },
            {
              id: "list",
              label: <List aria-hidden size={14} />,
              ariaLabel: t("workbench.components.viewList", { defaultValue: "List" }),
            },
            {
              id: "tree",
              label: <Network aria-hidden size={14} />,
              ariaLabel: t("workbench.components.viewTree", { defaultValue: "Tree" }),
            },
          ]}
        />
      </div>
    </>
  );
}
