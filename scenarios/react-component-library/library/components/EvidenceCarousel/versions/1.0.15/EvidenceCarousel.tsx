/**
 * @libraryId react-component-library:EvidenceCarousel
 * @displayName EvidenceCarousel
 * @description A compact evidence reference strip for captures and diagnostics.
 * @version 1.0.15
 * @tags ["visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/** @vrooliComponentSource react-component-library:EvidenceCarousel */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";
import { CheckCircle2, CircleAlert, FileText, Image, ScanSearch } from "lucide-react";

export interface EvidenceItem {
  id: string;
  kind: string;
  status: "available" | "missing" | "stale";
  reference?: string;
  label?: string;
}

export interface EvidenceCarouselProps {
  items?: EvidenceItem[];
  selectedId?: string;
  onSelect?: (item: EvidenceItem) => void;
  renderContent?: (item: EvidenceItem) => ReactNode;
  renderControls?: (item: EvidenceItem) => ReactNode;
}

const STATUS_LABELS: Record<EvidenceItem["status"], string> = {
  available: "Captured",
  missing: "Not captured",
  stale: "Stale capture",
};

const evidenceCarouselStyles = `
[data-rcl-evidence-carousel] { overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); }
.rcl-evidence-carousel__header { border-block-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface-muted); padding: var(--space-xs) var(--space-sm) 0; }
.rcl-evidence-carousel__summary { display: flex; align-items: center; justify-content: flex-end; gap: var(--space-xs); padding-block-end: var(--space-xs); }
.rcl-evidence-carousel__caption { color: var(--color-muted-foreground); font-size: var(--text-caption-size); line-height: var(--text-caption-line); }
.rcl-evidence-carousel__tabs { display: flex; gap: var(--space-2xs); overflow-x: auto; }
.rcl-evidence-carousel__tab { position: relative; display: flex; min-inline-size: max-content; align-items: center; gap: var(--space-2xs); border: 0; border-block-end: var(--border-medium) solid transparent; background: transparent; color: var(--color-muted-foreground); padding: var(--space-xs) var(--space-2xs); font: inherit; font-size: var(--text-caption-size); font-weight: 600; line-height: var(--text-caption-line); cursor: pointer; transition: border-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
.rcl-evidence-carousel__tab:hover { border-block-end-color: var(--color-border); color: var(--color-foreground); }
.rcl-evidence-carousel__tab:focus-visible { outline: var(--focus-ring-width) solid var(--focus-ring-color); outline-offset: calc(var(--focus-ring-width) * -1); }
.rcl-evidence-carousel__tab[data-selected="true"] { border-block-end-color: var(--color-primary); color: var(--color-primary); }
.rcl-evidence-carousel__kind-icon { inline-size: var(--icon-size-sm); block-size: var(--icon-size-sm); }
.rcl-evidence-carousel__status-icon { inline-size: var(--icon-size-xs); block-size: var(--icon-size-xs); color: var(--color-muted-foreground); }
.rcl-evidence-carousel__status-icon[data-status="available"] { color: var(--color-success); }
.rcl-evidence-carousel__status-icon[data-status="stale"] { color: var(--color-warning); }
.rcl-evidence-carousel__visually-hidden { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
.rcl-evidence-carousel__controls { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-xs) var(--space-sm); }
.rcl-evidence-carousel__content { min-block-size: var(--content-min-height); background: var(--color-background); }
.rcl-evidence-carousel__empty { display: flex; min-block-size: var(--content-min-height); align-items: center; justify-content: center; padding: var(--space-md); }
`;

function kindLabel(kind: string) {
  return kind
    .replace(/^bas-/, "")
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function iconFor(kind: string) {
  return kind === "screenshot" ? Image : kind === "accessibility-tree" ? ScanSearch : FileText;
}

function statusIcon(status: EvidenceItem["status"]) {
  return status === "available" ? CheckCircle2 : CircleAlert;
}

export const EvidenceCarousel = withClassName(function EvidenceCarousel({
  items = [],
  selectedId,
  onSelect,
  renderContent,
  renderControls,
}: EvidenceCarouselProps) {
  const strings = useStrings();
  const selected = items.find((item) => item.id === selectedId) ?? items[0];
  const selectedContent = selected && renderContent ? renderContent(selected) : null;

  return (
    <section
      aria-label={strings(
        "visualization.evidence-carousel.evidence-workspace",
        "Evidence workspace",
      )}
      data-rcl-asset="visualization.evidence-carousel"
      data-rcl-version="1.0.14"
      data-rcl-evidence-carousel
      data-rcl-stamp="source"
      data-testid="visualization-evidence-carousel"
      className="rcl-evidence-carousel"
    >
      <StyleSheet name="evidence-carousel-1-0-14" css={evidenceCarouselStyles} />
      <div className="rcl-evidence-carousel__header">
        <div className="rcl-evidence-carousel__summary">
          <span className="rcl-evidence-carousel__caption">
            {items.filter((item) => item.status === "available").length}/{items.length} captured
          </span>
        </div>
        <div
          role="tablist"
          aria-label={strings("visualization.evidence-carousel.evidence-types", "Evidence types")}
          className="rcl-evidence-carousel__tabs"
        >
          {items.map((item) => {
            const Icon = iconFor(item.id);
            const StatusIcon = statusIcon(item.status);
            const isSelected = item.id === selected?.id;
            const label = item.label ?? kindLabel(item.kind);
            return (
              <button
                data-testid="visualization.evidence-carousel"
                key={item.id}
                type="button"
                role="tab"
                aria-selected={isSelected}
                aria-controls={`evidence-panel-${item.id}`}
                aria-label={`${label}: ${STATUS_LABELS[item.status]}`}
                data-status={item.status}
                data-selected={isSelected ? "true" : "false"}
                onClick={() => onSelect?.(item)}
                className="rcl-evidence-carousel__tab"
              >
                <Icon className="rcl-evidence-carousel__kind-icon" aria-hidden />
                <span>{label}</span>
                <StatusIcon
                  className="rcl-evidence-carousel__status-icon"
                  data-status={item.status}
                  aria-hidden
                />
                <span className="rcl-evidence-carousel__visually-hidden">
                  {STATUS_LABELS[item.status]}
                </span>
              </button>
            );
          })}
        </div>
      </div>
      {selected ? (
        <div
          id={`evidence-panel-${selected.id}`}
          role="tabpanel"
          aria-label={selected.label ?? kindLabel(selected.kind)}
        >
          {renderControls?.(selected) ? (
            <div className="rcl-evidence-carousel__controls">{renderControls(selected)}</div>
          ) : null}
          <div className="rcl-evidence-carousel__content">{selectedContent}</div>
        </div>
      ) : (
        <div className="rcl-evidence-carousel__empty">
          <span className="rcl-evidence-carousel__caption">
            {strings(
              "visualization.evidence-carousel.no-evidence-captured",
              "No evidence captured.",
            )}
          </span>
        </div>
      )}
    </section>
  );
});
