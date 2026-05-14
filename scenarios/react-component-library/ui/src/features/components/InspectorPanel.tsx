import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { UseComponentInspectorReturn } from "../../hooks/useComponentInspector";

interface InspectorPanelProps {
  inspector: UseComponentInspectorReturn;
}

/**
 * InspectorPanel — small surface beneath the live preview that lets
 * the user pick an element from the iframe, then shows the selected
 * element's selector / tag / rect / text plus an ancestor breadcrumb.
 *
 * Surface for req 06 (IS-001..003). Selection state is owned by
 * `useComponentInspector`; the panel is a pure renderer.
 */
export function InspectorPanel({ inspector }: InspectorPanelProps) {
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
      className="mt-3 rounded-lg border border-white/10 bg-black/30 p-3 backdrop-blur-sm"
    >
      <header className="flex items-center justify-between gap-3">
        <h3 className="text-xs font-medium text-slate-300">
          {t(strings.components.inspector.title)}
        </h3>
        <div className="flex items-center gap-2">
          <span
            data-testid={selectors.components.inspector.statusBadge}
            className={
              active
                ? "rounded-full bg-amber-500/20 px-2 py-0.5 text-xs text-amber-200"
                : "rounded-full bg-slate-500/20 px-2 py-0.5 text-xs text-slate-300"
            }
          >
            {t(statusKey)}
          </span>
          <Button
            data-testid={selectors.components.inspector.toggleButton}
            onClick={() => (active ? stopInspect() : startInspect())}
            className="h-7 px-3 text-xs"
            variant={active ? "outline" : "default"}
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
          className="mt-2 text-xs text-slate-500"
        >
          {t(strings.components.inspector.empty)}
        </p>
      )}

      {selected && (
        <div className="mt-2 space-y-2 text-xs text-slate-200">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <span data-testid={selectors.components.inspector.selectedTag} className="font-mono text-slate-100">
              &lt;{selected.meta.tag}&gt;
            </span>
            <span
              data-testid={selectors.components.inspector.selectedSelector}
              className="font-mono text-slate-400"
            >
              {selected.meta.selector}
            </span>
            {selected.documentRect && (
              <span
                data-testid={selectors.components.inspector.selectedRect}
                className="text-slate-500"
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
              className="rounded border border-white/5 bg-black/40 px-2 py-1 font-mono text-[0.7rem] text-slate-300"
            >
              {selected.meta.text}
            </p>
          )}
          {selected.ancestors.length > 0 && (
            <nav
              data-testid={selectors.components.inspector.breadcrumb}
              aria-label={t(strings.components.inspector.breadcrumbLabel)}
              className="flex flex-wrap items-center gap-1 text-[0.7rem] text-slate-400"
            >
              {selected.ancestors
                .slice()
                .reverse()
                .map((a) => (
                  <span
                    key={`${a.depth}-${a.selector}`}
                    data-testid={selectors.components.inspector.breadcrumbItem}
                    className="rounded bg-white/5 px-1.5 py-0.5 font-mono"
                  >
                    {a.selector || a.tag}
                  </span>
                ))}
            </nav>
          )}
        </div>
      )}
    </section>
  );
}
