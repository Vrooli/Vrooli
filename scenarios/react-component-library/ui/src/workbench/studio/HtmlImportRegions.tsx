import { useId, useMemo, useState, type CSSProperties } from "react";

import { useTranslation } from "../../i18n";
import type { HtmlRegion } from "./htmlImport";

/** Source tag shown on the whole-document row; data, not interface copy. */
const WHOLE_TAG = "body";

interface HtmlImportRegionsProps {
  regions: HtmlRegion[];
  layers: number;
  selectedRegion: string | null;
  onSelectRegion: (regionId: string | null) => void;
}

function depthStyle(depth: number): CSSProperties {
  return { "--hi-depth": Math.min(depth, 8) } as CSSProperties;
}

/** Searchable region list for choosing which part of a parsed page to import. */
export function HtmlImportRegions({
  regions,
  layers,
  selectedRegion,
  onSelectRegion,
}: HtmlImportRegionsProps) {
  const { t } = useTranslation();
  const [regionQuery, setRegionQuery] = useState("");
  const headingId = useId();
  const filteredRegions = useMemo(() => {
    const query = regionQuery.trim().toLocaleLowerCase();
    if (!query) return regions;
    return regions.filter((region) =>
      [region.sourceHint, region.label, region.tag].some(
        (value) => value?.toLocaleLowerCase().includes(query) ?? false,
      ),
    );
  }, [regionQuery, regions]);

  return (
    <div className="hi-regions" role="group" aria-labelledby={headingId}>
      <h3 id={headingId} className="hi-regions-title">
        {t("workbench.studio.htmlImportRegions", {
          defaultValue: "Choose a region to import",
        })}
      </h3>
      <label className="hi-field-label" htmlFor={`${headingId}-search`}>
        {t("workbench.studio.htmlImportRegionSearch", { defaultValue: "Search regions" })}
      </label>
      <input
        id={`${headingId}-search`}
        className="hi-region-search"
        type="search"
        value={regionQuery}
        onChange={(event) => setRegionQuery(event.target.value)}
        placeholder={t("workbench.studio.htmlImportRegionSearchPlaceholder", {
          defaultValue: "Name, text or element",
        })}
        data-testid="html-import-region-search"
      />
      <p
        className="hi-region-results"
        role="status"
        aria-live="polite"
        data-testid="html-import-region-results"
      >
        {t("workbench.studio.htmlImportRegionResults", {
          defaultValue: "Showing {{filtered}} of {{total}} regions",
          filtered: filteredRegions.length,
          total: regions.length,
        })}
      </p>
      <div className="hi-region-list">
        <button
          type="button"
          className="hi-region"
          aria-pressed={selectedRegion === null}
          onClick={() => onSelectRegion(null)}
          data-testid="html-import-region-whole"
          style={depthStyle(0)}
        >
          <span className="hi-region-tag">{WHOLE_TAG}</span>
          <span className="hi-region-label">
            {t("workbench.studio.htmlImportWholeDocument", {
              defaultValue: "Whole document",
            })}
          </span>
          <span className="hi-region-count">{layers}</span>
        </button>
        {filteredRegions.map((region) => (
          <button
            key={region.id}
            type="button"
            className="hi-region"
            aria-pressed={selectedRegion === region.id}
            onClick={() => onSelectRegion(region.id)}
            data-testid={`html-import-region-${region.id}`}
            style={depthStyle(region.depth)}
          >
            <span className="hi-region-tag">{region.tag}</span>
            <span className="hi-region-label">
              {region.sourceHint ? (
                <>
                  <strong>{region.sourceHint}</strong>
                  {region.label ? ` · ${region.label}` : ""}
                </>
              ) : (
                region.label || t("workbench.studio.htmlImportNoText", { defaultValue: "No text" })
              )}
            </span>
            <span className="hi-region-count">{region.layers}</span>
          </button>
        ))}
        {filteredRegions.length === 0 ? (
          <p className="hi-region-empty" role="status" data-testid="html-import-region-empty">
            {t("workbench.studio.htmlImportRegionNoResults", {
              defaultValue: "No regions match this search. Clear the search to see all regions.",
            })}
          </p>
        ) : null}
      </div>
    </div>
  );
}
