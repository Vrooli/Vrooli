import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { EmptyState } from "../../components/EmptyState";
import { SeverityBadge } from "../../components/SeverityBadge";
import type { GraphLayout } from "./lib/graphAdapter";

export interface GraphAccessibleListProps {
  layout: GraphLayout;
  /** Outgoing import targets per package id (resolved label list). */
  importLabels: ReadonlyMap<string, readonly string[]>;
  className?: string;
}

/**
 * Text-alternative package list. Always present in the DOM so screen readers
 * always have a path to the graph data, even on desktop where the SVG canvas
 * is the primary surface. Mirrors every package node and its import edges.
 */
export function GraphAccessibleList({ layout, importLabels, className }: GraphAccessibleListProps) {
  const { t } = useTranslation();

  if (layout.nodes.length === 0) {
    return (
      <div
        data-testid={selectors.features.explorer.accessibleList.empty}
        className={cn(className)}
      >
        <EmptyState
          title={t(strings.explorer.emptyTitle)}
          description={t(strings.explorer.emptyDescription)}
        />
      </div>
    );
  }

  return (
    <section
      data-testid={selectors.features.explorer.accessibleList.root}
      aria-label={t(strings.explorer.accessibleList.title)}
      className={cn(
        "flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm",
        className,
      )}
    >
      <header className="flex flex-col gap-0.5">
        <h4 className="text-sm font-semibold">{t(strings.explorer.accessibleList.title)}</h4>
        <p className="text-xs text-app-muted-foreground">
          {t(strings.explorer.accessibleList.description)}
        </p>
      </header>
      <ul className="flex flex-col gap-1">
        {layout.nodes.map((node) => {
          const imports = importLabels.get(node.id) ?? [];
          return (
            <li
              key={node.id}
              data-testid={selectors.features.explorer.accessibleList.item({ id: node.id })}
            >
              <div className="flex w-full flex-col items-start gap-1 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-left text-sm">
                <span className="flex items-center gap-2 font-mono text-xs text-app-foreground">
                  {node.path}
                  {node.inCycle ? (
                    <SeverityBadge level="high" label={t(strings.explorer.legend.cycle)} />
                  ) : null}
                </span>
                <span className="text-xs text-app-muted-foreground">
                  {t(strings.explorer.accessibleList.importsLabel)}{" "}
                  {imports.length > 0 ? imports.join(", ") : "—"}
                </span>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
