/**
 * @libraryId react-component-library:EvidenceCarousel
 * @displayName EvidenceCarousel
 * @description A compact evidence reference strip for captures and diagnostics.
 * @version 1.0.12
 * @tags ["visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:EvidenceCarousel */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
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
      aria-label={strings("visualization.evidence-carousel.evidence-workspace", "Evidence workspace")}
      data-rcl-asset="visualization.evidence-carousel"
      data-rcl-version="1.0.8"
      data-rcl-stamp="source"
      data-testid="visualization-evidence-carousel"
      className="overflow-hidden rounded-panel border border-app-border bg-app-surface"
    >
      <div className="border-b border-app-border bg-app-surface-muted px-space-sm pt-space-xs">
        <div className="flex items-center justify-end gap-space-xs pb-space-xs">
          <span className="text-xs text-app-muted-foreground">
            {items.filter((item) => item.status === "available").length}/{items.length} captured
          </span>
        </div>
        <div
          role="tablist"
          aria-label={strings("visualization.evidence-carousel.evidence-types", "Evidence types")}
          className="flex gap-space-2xs overflow-x-auto"
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
                onClick={() => onSelect?.(item)}
                className={`relative flex min-w-max items-center gap-space-2xs border-b-2 px-space-2xs py-space-xs text-xs font-medium transition focus:outline-none focus:ring-2 focus:ring-app-primary focus:ring-inset ${isSelected ? "border-app-primary text-app-primary" : "border-transparent text-app-muted-foreground hover:border-app-border hover:text-app-foreground"}`}
              >
                <Icon className="h-icon-sm w-icon-sm" aria-hidden />
                <span>{label}</span>
                <StatusIcon
                  className={`h-icon-xs w-icon-xs ${item.status === "available" ? "text-app-success" : item.status === "stale" ? "text-app-warning" : "text-app-muted-foreground"}`}
                  aria-hidden
                />
                <span className="sr-only">{STATUS_LABELS[item.status]}</span>
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
            <div className="flex flex-wrap items-center gap-space-xs border-b border-app-border px-space-sm py-space-xs">
              {renderControls(selected)}
            </div>
          ) : null}
          <div className="min-h-content bg-app-background">{selectedContent}</div>
        </div>
      ) : (
        <div className="flex min-h-content items-center justify-center p-space-md">
          <span className="text-xs text-app-muted-foreground">
            {strings("visualization.evidence-carousel.no-evidence-captured", "No evidence captured.")}
          </span>
        </div>
      )}
    </section>
  );
});
