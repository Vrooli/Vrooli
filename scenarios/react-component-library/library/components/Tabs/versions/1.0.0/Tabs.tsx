/** @vrooliComponentSource navigation.tabs */
import {
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type ReactNode,
} from "react";

export type TabsMode = "controlled" | "uncontrolled";

export interface TabsProps {
  items?: string[];
  active?: string;
  defaultActive?: string;
  onChange?: (item: string) => void;
  panels?: Record<string, ReactNode>;
  mode?: TabsMode;
  ariaLabel?: string;
}

const styleSheet = `
[data-rcl-tabs] { position: relative; max-width: 100%; overflow-x: auto; scrollbar-width: none; }
[data-rcl-tabs]::-webkit-scrollbar { display: none; }
[data-rcl-tablist] { display: inline-flex; min-width: 100%; gap: var(--space-3xs); border-bottom: var(--border-hairline) solid var(--color-border); }
[data-rcl-tab] { position: relative; min-height: var(--tap-target-min); border: 0; border-radius: var(--radius-control) var(--radius-control) 0 0; background: transparent; color: var(--color-muted-foreground); cursor: pointer; font: inherit; font-weight: 650; white-space: nowrap; transition: color var(--dur-quick) var(--ease-standard), background var(--dur-quick) var(--ease-standard); }
[data-rcl-tab]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-tab]:focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: calc(var(--space-3xs) * -1); }
[data-rcl-tab][aria-selected="true"] { color: var(--color-primary); }
[data-rcl-tab-indicator] { position: absolute; inset-block-end: 0; block-size: var(--border-strong); border-radius: var(--radius-pill); background: var(--color-primary); pointer-events: none; transition: transform var(--dur-moderate) var(--ease-standard), width var(--dur-moderate) var(--ease-standard); }
@media (prefers-reduced-motion: reduce) { [data-rcl-tab], [data-rcl-tab-indicator] { transition: none; } }
`;

export function Tabs({
  items = [],
  active,
  defaultActive,
  onChange,
  panels,
  mode,
  ariaLabel = "Tabs",
}: TabsProps) {
  const [uncontrolledActive, setUncontrolledActive] = useState(
    defaultActive ?? items[0] ?? "",
  );
  const resolvedMode: TabsMode = mode ?? (active === undefined ? "uncontrolled" : "controlled");
  const selectedItem = resolvedMode === "controlled"
    ? active ?? items[0] ?? ""
    : uncontrolledActive;
  const selectedIndex = Math.max(0, items.indexOf(selectedItem));
  const resolvedItem = items[selectedIndex] ?? "";
  const tablistRef = useRef<HTMLDivElement>(null);
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const [indicator, setIndicator] = useState<CSSProperties>({ opacity: 0 });

  useLayoutEffect(() => {
    const updateIndicator = () => {
      const list = tablistRef.current;
      const tab = tabRefs.current[selectedIndex];
      if (!list || !tab) return;
      const listRect = list.getBoundingClientRect();
      const tabRect = tab.getBoundingClientRect();
      setIndicator({
        opacity: 1,
        width: tabRect.width,
        transform: `translateX(${tabRect.left - listRect.left + list.scrollLeft}px)`,
      });
    };
    updateIndicator();
    const observer =
      typeof ResizeObserver === "undefined"
        ? undefined
        : new ResizeObserver(updateIndicator);
    if (observer && tablistRef.current) observer.observe(tablistRef.current);
    return () => observer?.disconnect();
  }, [selectedIndex, items.length]);

  const moveSelection = (index: number) => {
    const nextIndex = (index + items.length) % items.length;
    const next = items[nextIndex];
    if (!next) return;
    if (resolvedMode === "uncontrolled") setUncontrolledActive(next);
    onChange?.(next);
    tabRefs.current[nextIndex]?.focus();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    const index = Number(event.currentTarget.dataset.index ?? selectedIndex);
    if (event.key === "ArrowRight" || event.key === "ArrowDown") {
      event.preventDefault();
      moveSelection(index + 1);
    } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      event.preventDefault();
      moveSelection(index - 1);
    } else if (event.key === "Home") {
      event.preventDefault();
      moveSelection(0);
    } else if (event.key === "End") {
      event.preventDefault();
      moveSelection(items.length - 1);
    }
  };

  return (
    <>
      <style
        data-rcl-tabs-styles
        dangerouslySetInnerHTML={{ __html: styleSheet }}
      />
      <div data-rcl-tabs>
        <div
          ref={tablistRef}
          role="tablist"
          aria-label={ariaLabel}
          data-rcl-tab-list
          data-rcl-tablist
        >
          {items.map((item, index) => {
            const selected = item === resolvedItem;
            return (
              <button
                key={item}
                ref={(node) => {
                  tabRefs.current[index] = node;
                }}
                id={`rcl-tab-${index}`}
                type="button"
                role="tab"
                aria-selected={selected}
                aria-controls={`rcl-tab-panel-${index}`}
                tabIndex={selected ? 0 : -1}
                data-index={index}
                data-rcl-tab-trigger
                data-rcl-tab
                onClick={() => {
                  if (resolvedMode === "uncontrolled") setUncontrolledActive(item);
                  onChange?.(item);
                }}
                onKeyDown={handleKeyDown}
                style={{ paddingInline: "var(--space-sm)" }}
              >
                {item}
              </button>
            );
          })}
          <span aria-hidden="true" data-rcl-tab-indicator style={indicator} />
        </div>
        {panels && resolvedItem in panels && (
          <div
            id={`rcl-tab-panel-${selectedIndex}`}
            role="tabpanel"
            aria-labelledby={`rcl-tab-${selectedIndex}`}
            data-rcl-tab-panel
            style={{ paddingBlock: "var(--space-md)" }}
          >
            {panels[resolvedItem]}
          </div>
        )}
      </div>
    </>
  );
}
