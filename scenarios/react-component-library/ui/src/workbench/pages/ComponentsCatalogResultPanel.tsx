import { useTranslation } from "../../i18n";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.8.0";
import type { ComponentsCatalogViewProps as Props } from "./ComponentsCatalogView";
import { SlidersHorizontal } from "lucide-react";
import { assetDomain } from "../data";
import { Panel, PanelBody, PanelHead, Skeleton, WbButton } from "../primitives";
import { AssetCard, AssetListRow } from "./ComponentsCatalogAssets";

type ResultPanelProps = Pick<
  Props,
  "catalog" | "group" | "filtered" | "sort" | "visible" | "rows" | "domains" | "view" | "setLimit"
>;

export function CatalogResultPanel({
  catalog,
  group,
  filtered,
  sort,
  visible,
  rows,
  domains,
  view,
  setLimit,
}: ResultPanelProps) {
  const { t } = useTranslation();
  return (
    <Panel>
      <PanelHead
        title={
          group === "components"
            ? t("workbench.components.allAssets", { defaultValue: "All assets" })
            : group === "hooks"
              ? t("workbench.components.hookAssets", { defaultValue: "Hooks" })
              : t("workbench.components.supportAssets", { defaultValue: "Support assets" })
        }
        meta={t("workbench.components.resultMeta", {
          count: filtered.length,
          sort,
        })}
        actions={
          <span className="wb-muted-2 wb-small" aria-hidden>
            <SlidersHorizontal size={13} />
          </span>
        }
      />
      <PanelBody flush>
        {catalog.isLoading ? (
          <div className="wb-gallery">
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} height={210} />
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            density="compact"
            className="wb-catalog-empty"
            title={t("workbench.components.emptyTitle", {
              defaultValue: "No assets match these filters",
            })}
            description={t("workbench.components.emptyBody", {
              defaultValue:
                "Clear the search or choose another domain. The library itself is not empty.",
            })}
          />
        ) : view === "gallery" ? (
          <div className="wb-gallery">
            {visible.map((asset) => (
              <AssetCard
                key={asset.id}
                asset={asset}
                row={rows.get(asset.catalogId) ?? rows.get(asset.libraryId)}
              />
            ))}
          </div>
        ) : view === "list" ? (
          <div className="wb-list">
            {visible.map((asset) => (
              <AssetListRow
                key={asset.id}
                asset={asset}
                row={rows.get(asset.catalogId) ?? rows.get(asset.libraryId)}
              />
            ))}
          </div>
        ) : (
          <div className="wb-tree">
            {domains.map(([name, count]) => (
              <div key={name}>
                <div className="wb-tree-row is-branch">
                  <span>{name}</span>
                  <span className="wb-count">{count}</span>
                </div>
                {visible
                  .filter((asset) => assetDomain(asset) === name)
                  .map((asset) => (
                    <AssetListRow
                      key={asset.id}
                      asset={asset}
                      row={rows.get(asset.catalogId) ?? rows.get(asset.libraryId)}
                      tree
                    />
                  ))}
              </div>
            ))}
          </div>
        )}
        {filtered.length > visible.length ? (
          <div style={{ padding: 14, borderTop: "1px solid var(--wb-line)", textAlign: "center" }}>
            <WbButton variant="ghost" size="sm" onClick={() => setLimit((value) => value + 60)}>
              {t("workbench.components.loadMore", {
                defaultValue: "Show more · {{shown}} of {{total}}",
                shown: visible.length,
                total: filtered.length,
              })}
            </WbButton>
          </div>
        ) : null}
      </PanelBody>
    </Panel>
  );
}
