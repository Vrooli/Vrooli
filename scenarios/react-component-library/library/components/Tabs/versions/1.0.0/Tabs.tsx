/** @vrooliComponentSource navigation.tabs */
import {
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { motionTransition } from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";

export type TabsMode = "controlled" | "uncontrolled";

export interface TabsItem {
  id: string;
  label: ReactNode;
  badge?: ReactNode;
}

export interface TabsProps {
  items?: Array<string | TabsItem>;
  active?: string;
  defaultActive?: string;
  onChange?: (item: string) => void;
  panels?: Record<string, ReactNode>;
  mode?: TabsMode;
  ariaLabel?: string;
  itemTestId?: (item: string) => string | undefined;
}

const styleSheet = `
[data-rcl-tabs] { position: relative; max-width: 100%; overflow-x: auto; scrollbar-width: none; }
[data-rcl-tabs]::-webkit-scrollbar { display: none; }
[data-rcl-tablist] { position: relative; display: inline-flex; min-width: 100%; gap: var(--space-3xs); border-bottom: var(--border-hairline) solid var(--color-border); }
[data-rcl-tab] { position: relative; min-height: var(--tap-target-min); padding-inline: var(--space-sm); border: 0; border-radius: var(--radius-control) var(--radius-control) 0 0; background: transparent; color: var(--color-muted-foreground); cursor: pointer; font: inherit; font-weight: 650; white-space: nowrap; transition: ${motionTransition(["color", "background-color"], "interaction")}; }
[data-rcl-tab]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-tab]:focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: calc(var(--space-3xs) * -1); }
[data-rcl-tab][aria-selected="true"] { color: var(--color-primary); }
[data-rcl-tab-badge] { display: inline-flex; min-inline-size: 1rem; align-items: center; justify-content: center; border-radius: var(--radius-pill); padding-inline: var(--space-3xs); padding-block: var(--space-3xs); background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); font-size: 0.6875rem; line-height: 1; }
[data-rcl-tab-indicator] { position: absolute; inset-block-end: 0; inline-size: 1px; block-size: var(--border-strong); border-radius: var(--radius-pill); background: var(--color-primary); pointer-events: none; transform-origin: left center; transition: ${motionTransition(["transform", "opacity"], "spring")}; will-change: transform, opacity; }
@media (prefers-reduced-motion: reduce) { [data-rcl-tab], [data-rcl-tab-indicator] { transition: none; } }
@media (max-width: 480px) { [data-rcl-tab] { padding-inline: var(--space-xs); } }
`;

export function Tabs({
  items = [],
  active,
  defaultActive,
  onChange,
  panels,
  mode,
  ariaLabel = "Tabs",
  itemTestId,
}: TabsProps) {
  const normalizedItems = items.map((item) =>
    typeof item === "string" ? { id: item, label: item } : item,
  );
  const itemIDs = normalizedItems.map((item) => item.id);
  const [uncontrolledActive, setUncontrolledActive] = useState(
    defaultActive ?? itemIDs[0] ?? "",
  );
  const resolvedMode: TabsMode =
    mode ?? (active === undefined ? "uncontrolled" : "controlled");
  const selectedItem =
    resolvedMode === "controlled"
      ? (active ?? itemIDs[0] ?? "")
      : uncontrolledActive;
  const selectedIndex = Math.max(0, itemIDs.indexOf(selectedItem));
  const resolvedItem = normalizedItems[selectedIndex]?.id ?? "";
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
        transform: `translateX(${tabRect.left - listRect.left + list.scrollLeft}px) scaleX(${Math.max(tabRect.width, 1)})`,
      });
    };
    updateIndicator();
    const observer =
      typeof ResizeObserver === "undefined"
        ? undefined
        : new ResizeObserver(updateIndicator);
    if (observer && tablistRef.current) observer.observe(tablistRef.current);
    return () => observer?.disconnect();
  }, [selectedIndex, normalizedItems.length]);

  const moveSelection = (index: number) => {
    const nextIndex = (index + normalizedItems.length) % normalizedItems.length;
    const next = normalizedItems[nextIndex]?.id;
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
      <div data-rcl-tabs role="tablist" aria-label={ariaLabel}>
        <div ref={tablistRef} data-rcl-tab-list data-rcl-tablist>
          {normalizedItems.map((item, index) => {
            const selected = item.id === resolvedItem;
            return (
              <button
                key={item.id}
                ref={(node) => {
                  tabRefs.current[index] = node;
                }}
                id={`rcl-tab-${index}`}
                type="button"
                role="tab"
                aria-selected={selected}
                aria-controls={panels ? `rcl-tab-panel-${index}` : undefined}
                tabIndex={selected ? 0 : -1}
                data-index={index}
                data-testid={itemTestId?.(item.id)}
                data-rcl-tab-trigger
                data-rcl-tab
                onClick={() => {
                  if (resolvedMode === "uncontrolled")
                    setUncontrolledActive(item.id);
                  onChange?.(item.id);
                }}
                onKeyDown={handleKeyDown}
              >
                {item.label}
                {item.badge !== undefined && (
                  <span aria-hidden="true" data-rcl-tab-badge>
                    {item.badge}
                  </span>
                )}
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
