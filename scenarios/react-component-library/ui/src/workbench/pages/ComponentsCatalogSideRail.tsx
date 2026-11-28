import { useTranslation } from "../../i18n";
import type { ComponentsCatalogViewProps as Props } from "./ComponentsCatalogView";
import { Link } from "react-router-dom";
import { CircleAlert } from "lucide-react";
import { statsPath } from "../../routes";
import { assetHasDraft, formatNumber } from "../data";
import { Banner, Meter, Panel, PanelBody, PanelHead, Ring } from "../primitives";

type SideRailProps = Pick<
  Props,
  | "coverage"
  | "assets"
  | "rows"
  | "domains"
  | "collections"
  | "maxCollection"
  | "verifiedRatio"
  | "reviewCount"
  | "setDomain"
>;

export function CatalogSideRail({
  coverage,
  assets,
  rows,
  domains,
  collections,
  maxCollection,
  verifiedRatio,
  reviewCount,
  setDomain,
}: SideRailProps) {
  const { t } = useTranslation();
  return (
    <aside
      className="wb-side-rail"
      aria-label={t("workbench.components.filtersAria", { defaultValue: "Catalog filters" })}
    >
      <Panel>
        <PanelHead
          title={t("workbench.components.collections", { defaultValue: "Collections" })}
          meta={`${domains.length}`}
        />
        <PanelBody>
          <div className="wb-stack tight">
            {collections.map((entry) => (
              <button
                key={entry.name}
                type="button"
                className="wb-collection-row"
                onClick={() => setDomain(entry.name)}
              >
                <span className="wb-dot" style={{ background: "var(--wb-signal)" }} aria-hidden />
                <span className="wb-name">{entry.name}</span>
                <span className="wb-collection-bar">
                  <span style={{ width: `${(entry.count / maxCollection) * 100}%` }} />
                </span>
                <span className="wb-count">{entry.count}</span>
              </button>
            ))}
          </div>
        </PanelBody>
      </Panel>

      <Panel>
        <PanelHead
          title={t("workbench.components.coverage", { defaultValue: "Coverage" })}
          meta={t("workbench.components.live", { defaultValue: "Live" })}
        />
        <PanelBody>
          {coverage.isError ? (
            <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
              {t("workbench.components.coverageUnavailable", {
                defaultValue: "Coverage service did not complete. No ratio is shown.",
              })}
            </Banner>
          ) : (
            <div className="wb-health">
              <Ring
                value={verifiedRatio * 100}
                size={66}
                tone={verifiedRatio > 0.7 ? "green" : "amber"}
              />
              <div className="wb-health-copy">
                <b>
                  {t("workbench.components.verifiedAssets", {
                    defaultValue: "{{count}} verified",
                    count: assets.filter((asset) => {
                      const row = rows.get(asset.catalogId) ?? rows.get(asset.libraryId);
                      return Boolean(row && row.failedGates.length === 0 && row.achieved);
                    }).length,
                  })}
                </b>
                <span>
                  {t("workbench.components.coverageBody", {
                    defaultValue: "{{count}} need attention",
                    count: reviewCount ?? 0,
                  })}
                </span>
              </div>
            </div>
          )}
          <Link
            to={statsPath("quality")}
            className="wb-btn ghost sm wide"
            style={{ marginTop: 12 }}
          >
            {t("workbench.components.openCoverage", { defaultValue: "Open coverage" })}
          </Link>
        </PanelBody>
      </Panel>

      <Panel>
        <PanelHead title={t("workbench.components.health", { defaultValue: "Library pulse" })} />
        <PanelBody>
          <div className="wb-stack tight">
            <div className="wb-row-between">
              <span className="wb-small wb-muted">
                {t("workbench.components.published", { defaultValue: "Published" })}
              </span>
              <b className="wb-mono">{formatNumber(assets.length)}</b>
            </div>
            <Meter value={(assets.length / Math.max(1, assets.length)) * 100} tone="green" />
            <div className="wb-row-between">
              <span className="wb-small wb-muted">
                {t("workbench.components.draftsLabel", { defaultValue: "Drafts" })}
              </span>
              <b className="wb-mono">{assets.filter(assetHasDraft).length}</b>
            </div>
            <div className="wb-row-between">
              <span className="wb-small wb-muted">
                {t("workbench.components.pages", { defaultValue: "Design pages" })}
              </span>
              <b className="wb-mono">—</b>
            </div>
          </div>
        </PanelBody>
      </Panel>
    </aside>
  );
}
