/**
 * @libraryId react-component-library:Tabs
 * @displayName Tabs
 * @description A keyboard-operable tab navigation surface with stable active state and responsive overflow.
 * @version 1.2.0
 * @tags ["navigation","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource navigation.tabs */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import {
  useCallback,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { motionTransition } from "@vrooli/react-component-library/VisualRecipes/1";
import { baseStyles } from "@vrooli/react-component-library/BaseStyles/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/** Whether the selected tab is owned by the caller or by the component. */
export type TabsMode = "controlled" | "uncontrolled";
export type TabsDensity = "comfortable" | "compact";
export type TabsVariant = "underline" | "segmented";

/** One tab. A bare string is shorthand for an item whose id and label are equal. */
export interface TabsItem {
  id: string;
  label: ReactNode;
  /**
   * A glyph rendered before the label. It is decorative: the label carries the
   * accessible name, so an icon-only tab still needs a label the caller has
   * hidden rather than omitted.
   */
  icon?: ReactNode;
  badge?: ReactNode;
}

/** Inputs to {@link Tabs}. */
export interface TabsProps {
  items?: Array<string | TabsItem>;
  active?: string;
  defaultActive?: string;
  onChange?: (item: string) => void;
  panels?: Record<string, ReactNode>;
  mode?: TabsMode;
  ariaLabel?: string;
  /**
   * Per-tab automation selector. Returning undefined for an item falls back to
   * the catalog-rooted default, so a caller may override only the tabs its
   * flows address.
   */
  itemTestId?: (item: string) => string | undefined;
  /** Comfortable is for primary navigation; compact is for dense toolbars. */
  density?: TabsDensity;
  /** Underline is the default navigation treatment; segmented groups the tabs as a control. */
  variant?: TabsVariant;
}

const styleSheet = `
[data-rcl-tabs] { position: relative; max-inline-size: 100%; }
[data-rcl-tab-scroller] { max-inline-size: 100%; overflow-x: auto; overscroll-behavior-inline: contain; scrollbar-width: none; -webkit-overflow-scrolling: touch; }
[data-rcl-tab-scroller]::-webkit-scrollbar { display: none; }
[data-rcl-tablist] { position: relative; display: flex; flex-wrap: nowrap; inline-size: max-content; min-inline-size: 100%; gap: var(--space-3xs); border-block-end: var(--border-hairline) solid var(--color-border); }
[data-rcl-tab] { position: relative; display: inline-flex; flex: 0 0 auto; align-items: center; gap: var(--space-2xs); min-block-size: var(--tap-target-min); padding-inline: var(--space-sm); border: 0; border-radius: var(--radius-control) var(--radius-control) 0 0; background: transparent; color: var(--color-muted-foreground); cursor: pointer; font: inherit; font-weight: 650; white-space: nowrap; scroll-margin-inline: var(--space-sm); transition: ${motionTransition(["color", "background-color"], "interaction")}; }
[data-rcl-tabs-density="compact"] [data-rcl-tab] { min-block-size: var(--control-size-sm); padding-inline: var(--space-xs); font-size: var(--text-label-size); font-weight: 600; }
[data-rcl-tabs-density="compact"] [data-rcl-tablist] { border-block-end-color: transparent; }
[data-rcl-tabs-density="compact"] [data-rcl-tab-indicator] { display: none; }
[data-rcl-tab]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-tab][aria-selected="true"] { color: var(--color-primary); }
[data-rcl-tab] > [data-rcl-tab-icon] { display: inline-flex; flex: 0 0 auto; align-items: center; inline-size: var(--icon-size-sm); block-size: var(--icon-size-sm); }
[data-rcl-tab] > [data-rcl-tab-icon] > svg { inline-size: 100%; block-size: 100%; }
[data-rcl-tab-badge] { display: inline-flex; min-inline-size: 1rem; align-items: center; justify-content: center; border-radius: var(--radius-pill); padding-inline: var(--space-3xs); padding-block: var(--space-3xs); background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); font-size: var(--text-caption-size); line-height: 1; }
[data-rcl-tab-indicator] { position: absolute; inset-block-end: 0; inline-size: 1px; block-size: var(--border-strong); border-radius: var(--radius-pill); background: var(--color-primary); pointer-events: none; transform-origin: left center; transition: ${motionTransition(["transform", "opacity"], "spring")}; will-change: transform, opacity; }
[data-rcl-tabs-variant="segmented"] [data-rcl-tablist] { inline-size: 100%; gap: 0; padding: var(--space-3xs); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-muted); }
[data-rcl-tabs-variant="segmented"] [data-rcl-tab] { flex: 1 1 0; justify-content: center; border-radius: var(--radius-control); }
[data-rcl-tabs-variant="segmented"] [data-rcl-tab][aria-selected="true"] { background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-raised); }
[data-rcl-tabs-variant="segmented"] [data-rcl-tab-indicator] { display: none; }
@media (max-width: 30rem) { [data-rcl-tab] { padding-inline: var(--space-xs); } }
`;

const normalizeItem = (item: string | TabsItem): TabsItem =>
  typeof item === "string" ? { id: item, label: item } : item;

/**
 * A horizontal tab strip that scrolls rather than wraps.
 *
 * Before 1.1.0 the list was `flex-wrap: wrap` inside an `overflow-x: auto`
 * container, which cannot both be true: wrapping removes the overflow the
 * scroll container exists to carry, so a strip with more tabs than fit stacked
 * into rows and grew the surface it sits in. The list is now `nowrap` with an
 * intrinsic inline size, the selected tab is scrolled back into view when it
 * changes, and `itemTestId` is honoured instead of accepted and discarded.
 */
export const Tabs = withClassName(function Tabs({
  items = [],
  active,
  defaultActive,
  onChange,
  panels,
  mode,
  ariaLabel = "Tabs",
  itemTestId,
  density = "comfortable",
  variant = "underline",
}: TabsProps) {
  useLibraryStyleSheet("base-styles-1.2.0", baseStyles);
  useLibraryStyleSheet("tabs-1.1.0", styleSheet);
  const normalizedItems = items.map(normalizeItem);
  const itemIDs = normalizedItems.map((item) => item.id);
  const [uncontrolledActive, setUncontrolledActive] = useState(defaultActive ?? itemIDs[0] ?? "");
  const resolvedMode: TabsMode = mode ?? (active === undefined ? "uncontrolled" : "controlled");
  const selectedItem =
    resolvedMode === "controlled" ? (active ?? itemIDs[0] ?? "") : uncontrolledActive;
  const selectedIndex = Math.max(0, itemIDs.indexOf(selectedItem));
  const resolvedItem = normalizedItems[selectedIndex]?.id ?? "";
  const scrollerRef = useRef<HTMLDivElement>(null);
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
        transform: `translateX(${tabRect.left - listRect.left}px) scaleX(${Math.max(tabRect.width, 1)})`,
      });
    };
    updateIndicator();
    const observer =
      typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(updateIndicator);
    if (observer && tablistRef.current) observer.observe(tablistRef.current);
    const scroller = scrollerRef.current;
    scroller?.addEventListener("scroll", updateIndicator, { passive: true });
    return () => {
      observer?.disconnect();
      scroller?.removeEventListener("scroll", updateIndicator);
    };
  }, [selectedIndex, normalizedItems.length]);

  // A strip that scrolls can hide the selected tab. Bring it back whenever the
  // selection changes, including when the caller changes it from elsewhere.
  useLayoutEffect(() => {
    const tab = tabRefs.current[selectedIndex];
    if (!tab || typeof tab.scrollIntoView !== "function") return;
    tab.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [selectedIndex]);

  const select = useCallback(
    (id: string) => {
      if (resolvedMode === "uncontrolled") setUncontrolledActive(id);
      onChange?.(id);
    },
    [onChange, resolvedMode],
  );

  const moveSelection = (index: number) => {
    const nextIndex = (index + normalizedItems.length) % normalizedItems.length;
    const next = normalizedItems[nextIndex]?.id;
    if (!next) return;
    select(next);
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
      moveSelection(normalizedItems.length - 1);
    }
  };

  return (
    <div
      data-rcl-tabs
      data-rcl-tabs-density={density}
      data-rcl-tabs-variant={variant}
      data-testid="navigation.tabs"
    >
      <div ref={scrollerRef} data-rcl-tab-scroller>
        <div
          ref={tablistRef}
          role="tablist"
          aria-label={ariaLabel}
          data-rcl-tab-list
          data-rcl-tablist
        >
          {normalizedItems.map((item, index) => {
            const selected = item.id === resolvedItem;
            return (
              <button
                data-testid={itemTestId?.(item.id) ?? `navigation.tabs.${item.id}`}
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
                data-rcl-tab-trigger
                data-rcl-tab
                onClick={() => select(item.id)}
                onKeyDown={handleKeyDown}
              >
                {item.icon === undefined ? null : (
                  <span aria-hidden="true" data-rcl-tab-icon>
                    {item.icon}
                  </span>
                )}
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
      </div>
      {panels && resolvedItem in panels && (
        <div
          id={`rcl-tab-panel-${selectedIndex}`}
          role="tabpanel"
          aria-labelledby={`rcl-tab-${selectedIndex}`}
          data-rcl-tab-panel
        >
          {panels[resolvedItem]}
        </div>
      )}
    </div>
  );
});
