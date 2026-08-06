/** @vrooliComponentSource overlays.inspector-panel */
import { Button } from "../../components/Button";
import { StatusBadge } from "../../components/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { UseComponentInspectorReturn } from "../../hooks/useComponentInspector";

interface InspectorPanelProps {
  inspector: UseComponentInspectorReturn;
  specimenLabel?: string;
}

/**
 * InspectorPanel — small surface beneath the live preview that lets
 * the user pick an element from the iframe, then shows the selected
 * element's selector / tag / rect / text plus an ancestor breadcrumb.
 *
 * Surface for req 06 (IS-001..003). Selection state is owned by
 * `useComponentInspector`; the panel is a pure renderer.
 */
export function InspectorPanel({ inspector, specimenLabel }: InspectorPanelProps) {
  const { t } = useTranslation();
  const { active, selected, startInspect, stopInspect, lastReason } = inspector;

  const statusKey = active
    ? strings.components.inspector.statusActive
    : lastReason === "complete"
      ? strings.components.inspector.statusReady
      : strings.components.inspector.statusIdle;

  return (
    <section
      data-testid={selectors.components.inspector.panel}
      aria-label={t(strings.components.inspector.title)}
      className="mt-space-xs rounded-lg border border-app-border bg-app-surface p-space-xs backdrop-blur-sm"
    >
      <header className="flex items-center justify-between gap-space-xs">
        <div className="min-w-0">
          <h3 className="text-xs font-medium text-app-muted-foreground">
            {t(strings.components.inspector.title)}
          </h3>
          {specimenLabel && <p className="truncate text-xs text-app-muted-foreground">{specimenLabel}</p>}
        </div>
        <div className="flex items-center gap-space-2xs">
          <StatusBadge
            data-testid={selectors.components.inspector.statusBadge}
            tone={active ? "warning" : "neutral"}
          >
            {t(statusKey)}
          </StatusBadge>
          <Button
            data-testid={selectors.components.inspector.toggleButton}
            onClick={() => (active ? stopInspect() : startInspect())}
            className="h-7 px-space-xs text-xs"
            variant={active ? "secondary" : "primary"}
          >
            {active
              ? t(strings.components.inspector.toggleStop)
              : t(strings.components.inspector.toggleStart)}
          </Button>
        </div>
      </header>

      {!selected && (
        <p
          data-testid={selectors.components.inspector.empty}
          className="mt-space-2xs text-xs text-app-muted-foreground"
        >
          {t(strings.components.inspector.empty)}
        </p>
      )}

      {selected && (
        <div className="mt-space-2xs space-y-space-2xs text-xs text-app-foreground">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <span data-testid={selectors.components.inspector.selectedTag} className="font-mono text-app-foreground">
              &lt;{selected.meta.tag}&gt;
            </span>
            <span
              data-testid={selectors.components.inspector.selectedSelector}
              className="font-mono text-app-muted-foreground"
            >
              {selected.meta.selector}
            </span>
            {selected.documentRect && (
              <span
                data-testid={selectors.components.inspector.selectedRect}
                className="text-app-muted-foreground"
              >
                {t(strings.components.inspector.rectLabel, {
                  w: Math.round(selected.documentRect.width),
                  h: Math.round(selected.documentRect.height),
                  x: Math.round(selected.documentRect.x),
                  y: Math.round(selected.documentRect.y),
                })}
              </span>
            )}
          </div>
          {selected.meta.text && (
            <p
              data-testid={selectors.components.inspector.selectedText}
              className="rounded border border-app-border bg-app-surface-muted px-space-2xs py-space-3xs font-mono text-[0.7rem] text-app-muted-foreground"
            >
              {selected.meta.text}
            </p>
          )}
          {selected.ancestors.length > 0 && (
            <div
              data-testid={selectors.components.inspector.breadcrumb}
              role="navigation"
              aria-label={t(strings.components.inspector.breadcrumbLabel)}
              className="flex flex-wrap items-center gap-space-3xs text-[0.7rem] text-app-muted-foreground"
            >
              {selected.ancestors
                .slice()
                .reverse()
                .map((a) => (
                  <span
                    key={`${a.depth}-${a.selector}`}
                    data-testid={selectors.components.inspector.breadcrumbItem}
                    className="rounded bg-app-surface-muted px-space-2xs py-space-3xs font-mono"
                  >
                    {a.selector || a.tag}
                  </span>
                ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
}
