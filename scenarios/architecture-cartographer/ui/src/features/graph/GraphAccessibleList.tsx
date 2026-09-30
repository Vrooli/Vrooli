import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { EmptyState } from "../../components/EmptyState";
import { SeverityBadge } from "../../components/SeverityBadge";
import type { GraphLayout } from "./lib/graphAdapter";
import type { SeverityLevel } from "../../components/SeverityBadge";

const SEVERITY_LABEL_KEY = {
  info: strings.shared.severity.info,
  low: strings.shared.severity.low,
  medium: strings.shared.severity.medium,
  high: strings.shared.severity.high,
  critical: strings.shared.severity.critical,
} as const satisfies Record<SeverityLevel, string>;

export interface GraphAccessibleListProps {
  layout: GraphLayout;
  /**
   * Additional class names. The list is always in the DOM; callers control
   * responsive visibility (e.g. `sr-only md:not-sr-only` to hide on mobile
   * and show on desktop, or the inverse). The default behavior is visible.
   */
  className?: string;
}

/**
 * Text-alternative node list. Always present in the DOM so screen readers
 * always have a path to the graph data, even on desktop where the SVG
 * canvas is the primary surface.
 */
export function GraphAccessibleList({ layout, className }: GraphAccessibleListProps) {
  const { t } = useTranslation();

  if (layout.nodes.length === 0) {
    return (
      <div
        data-testid={selectors.features.graph.accessibleList.empty}
        className={cn(className)}
      >
        <EmptyState
          title={t(strings.pages.targetGraph.emptyTitle)}
          description={t(strings.pages.targetGraph.emptyDescription)}
        />
      </div>
    );
  }

  return (
    <section
      data-testid={selectors.features.graph.accessibleList.root}
      aria-label={t(strings.features.graph.accessibleList.title)}
      className={cn(
        "flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm",
        className,
      )}
    >
      <header className="flex flex-col gap-0.5">
        <h4 className="text-sm font-semibold">
          {t(strings.features.graph.accessibleList.title)}
        </h4>
        <p className="text-xs text-app-muted-foreground">
          {t(strings.features.graph.accessibleList.description)}
        </p>
      </header>
      <ul className="flex flex-col gap-1">
        {layout.nodes.map((node) => (
          <li
            key={node.id}
            data-testid={selectors.features.graph.accessibleList.item({ id: node.id })}
          >
            <button
              type="button"
              className="flex w-full flex-col items-start gap-1 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-left text-sm hover:bg-app-surface focus:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
            >
              <span className="font-mono text-xs text-app-foreground">{node.path}</span>
              <span className="flex flex-wrap gap-2 text-xs text-app-muted-foreground">
                <span>
                  {t(strings.features.graph.accessibleList.domainLabel)}{" "}
                  {node.domain.length > 0
                    ? node.domain
                    : t(strings.features.graph.accessibleList.noDomain)}
                </span>
                <span>
                  {t(strings.features.graph.accessibleList.conflictLabel)}{" "}
                  {node.conflictSeverity ? (
                    <SeverityBadge
                      level={node.conflictSeverity}
                      label={t(SEVERITY_LABEL_KEY[node.conflictSeverity])}
                    />
                  ) : (
                    <span>{t(strings.features.graph.accessibleList.noConflict)}</span>
                  )}
                </span>
              </span>
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}
