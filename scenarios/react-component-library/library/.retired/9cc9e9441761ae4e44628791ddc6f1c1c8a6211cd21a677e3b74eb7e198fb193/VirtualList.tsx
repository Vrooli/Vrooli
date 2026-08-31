/**
 * @libraryId react-component-library:VirtualList
 * @displayName VirtualList
 * @description A measured, accessible large-collection renderer with overscan, focus preservation, sticky rows, and scroll restoration.
 * @version 1.0.2
 * @tags ["data-display","virtualization","performance","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource data-display.virtual-list */
import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";

export interface VirtualListProps<T> {
  items?: T[];
  renderItem: (item: T, index: number) => ReactNode;
  getItemKey?: (item: T, index: number) => string;
  estimateItemHeight?: number;
  overscan?: number;
  height?: number | string;
  label?: string;
  title?: ReactNode;
  description?: ReactNode;
  empty?: ReactNode;
  initialScrollTop?: number;
  onScrollPositionChange?: (scrollTop: number) => void;
  stickyIndices?: number[];
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-virtual-list] { min-inline-size: 0; overflow: hidden; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, .75rem); background: var(--color-surface, #fff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08)); }
  [data-rcl-virtual-list-header] { display: grid; gap: var(--space-3xs, .25rem); padding: var(--space-md, 1rem) var(--space-lg, 1.5rem); border-block-end: 1px solid var(--color-border, #cbd5e1); background: color-mix(in srgb, var(--color-primary, #2563eb) 4%, var(--color-surface-raised, #fff)); }
  [data-rcl-virtual-list-title] { font: var(--text-subtitle, 650 1rem/1.35 system-ui, sans-serif); }
  [data-rcl-virtual-list-description], [data-rcl-virtual-list-status] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 500 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-virtual-list-viewport] { position: relative; overflow: auto; overscroll-behavior: contain; scrollbar-color: var(--color-border-strong, #94a3b8) transparent; }
  [data-rcl-virtual-list-content] { position: relative; min-inline-size: 100%; }
  [data-rcl-virtual-list-sticky-layer] { position: sticky; inset-block-start: 0; z-index: 4; block-size: 0; pointer-events: none; overflow: visible; }
  [data-rcl-virtual-list-sticky-row] { box-sizing: border-box; min-inline-size: 100%; padding: var(--space-md, 1rem) var(--space-lg, 1.5rem); border-block-end: 1px solid var(--color-border, #cbd5e1); background: var(--color-surface, #fff); box-shadow: 0 5px 12px rgb(15 23 42 / .12); overflow-wrap: anywhere; }
  [data-rcl-virtual-list-row] { position: absolute; inset-inline: 0; display: block; box-sizing: border-box; min-inline-size: 0; padding: var(--space-md, 1rem) var(--space-lg, 1.5rem); border-block-end: 1px solid var(--color-border, #cbd5e1); background: var(--color-surface, #fff); overflow-wrap: anywhere; }
  [data-rcl-virtual-list-row]:focus-within { z-index: 2; outline: 2px solid var(--color-focus, #2563eb); outline-offset: -2px; }
  [data-rcl-virtual-list-row][data-sticky="true"] { z-index: 3; box-shadow: 0 5px 12px rgb(15 23 42 / .12); }
  [data-rcl-virtual-list-empty] { display: grid; min-block-size: 10rem; place-items: center; padding: var(--space-lg, 1.5rem); color: var(--color-muted-foreground, #64748b); text-align: center; font: var(--text-body, 400 .875rem/1.375rem system-ui, sans-serif); }
  @media (max-width: 30rem) { [data-rcl-virtual-list-header], [data-rcl-virtual-list-row] { padding-inline: var(--space-md, 1rem); } }
`;

function lowerBound(values: number[], target: number) {
  let low = 0;
  let high = values.length - 1;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    if ((values[middle] ?? 0) < target) low = middle + 1;
    else high = middle;
  }
  return low;
}

export const VirtualList = withClassName(function VirtualList<T>({
  items = [],
  renderItem,
  getItemKey = (_item, index) => `virtual-item-${index}`,
  estimateItemHeight = 72,
  overscan = 4,
  height = 360,
  label = resolveStrings(
    "data-display.virtual-list.virtual-list",
    "Virtual list",
  ),
  title,
  description,
  empty = "Nothing here yet.",
  initialScrollTop = 0,
  onScrollPositionChange,
  stickyIndices = [],
  className,
  style,
}: VirtualListProps<T>) {
  const [scrollTop, setScrollTop] = useState(initialScrollTop);
  const [viewportHeight, setViewportHeight] = useState(360);
  const [measurements, setMeasurements] = useState<Record<number, number>>({});
  const viewportRef = useRef<HTMLDivElement>(null);
  const frame = useRef<number>();

  const offsets = useMemo(() => {
    const result = [0];
    items.forEach((_, index) => {
      result.push(
        (result[index] ?? 0) + (measurements[index] ?? estimateItemHeight),
      );
    });
    return result;
  }, [estimateItemHeight, items, measurements]);
  const totalHeight = offsets[items.length] ?? 0;
  const firstVisible = items.length
    ? Math.min(items.length - 1, lowerBound(offsets, scrollTop))
    : 0;
  const lastVisible = items.length
    ? Math.min(
        items.length - 1,
        lowerBound(offsets, scrollTop + viewportHeight),
      )
    : 0;
  const start = Math.max(0, firstVisible - overscan);
  const end = Math.min(items.length - 1, lastVisible + overscan);
  const activeStickyIndex = [...stickyIndices]
    .filter((index) => index >= 0 && index < items.length)
    .filter((index) => (offsets[index] ?? 0) < scrollTop)
    .sort((a, b) => a - b)
    .at(-1);
  const indexes = useMemo(() => {
    const visible = new Set<number>();
    for (let index = start; index <= end; index += 1) visible.add(index);
    stickyIndices.forEach((index) => {
      if (index >= 0 && index < items.length) visible.add(index);
    });
    return [...visible].sort((a, b) => a - b);
  }, [end, items.length, start, stickyIndices]);

  useLayoutEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    viewport.scrollTop = initialScrollTop;
    const resize = () => setViewportHeight(viewport.clientHeight || 360);
    resize();
    if (typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(resize);
    observer.observe(viewport);
    return () => observer.disconnect();
  }, [initialScrollTop]);

  useEffect(
    () => () => {
      if (frame.current !== undefined) cancelAnimationFrame(frame.current);
    },
    [],
  );

  const measure = (index: number, node: HTMLLIElement | null) => {
    if (!node || typeof ResizeObserver === "undefined") return;
    const update = (next: number) => {
      if (next > 0 && next !== measurements[index])
        setMeasurements((current) => ({ ...current, [index]: next }));
    };
    update(Math.ceil(node.getBoundingClientRect().height));
    const observer = new ResizeObserver(([entry]) => {
      const next = Math.ceil(entry?.contentRect.height ?? 0);
      update(next);
    });
    observer.observe(node);
  };

  const onScroll = () => {
    const viewport = viewportRef.current;
    if (!viewport || frame.current !== undefined) return;
    frame.current = requestAnimationFrame(() => {
      frame.current = undefined;
      const next = viewport.scrollTop;
      setScrollTop(next);
      onScrollPositionChange?.(next);
    });
  };

  return (
    <>
      <style
        data-rcl-virtual-list-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <section
        data-testid="data-display.virtual-list"
        className={className}
        style={style}
        data-rcl-virtual-list
        aria-label={label}
      >
        {(title || description) && (
          <header data-rcl-virtual-list-header>
            {title && <strong data-rcl-virtual-list-title>{title}</strong>}
            {description && (
              <span data-rcl-virtual-list-description>{description}</span>
            )}
          </header>
        )}
        {items.length === 0 ? (
          <div data-rcl-virtual-list-empty role="status">
            {empty}
          </div>
        ) : (
          <>
            <div
              ref={viewportRef}
              data-rcl-virtual-list-viewport
              style={{ height }}
              onScroll={onScroll}
            >
              <div data-rcl-virtual-list-sticky-layer aria-hidden="true">
                {activeStickyIndex !== undefined &&
                  items[activeStickyIndex] !== undefined && (
                    <div data-rcl-virtual-list-sticky-row>
                      {renderItem(items[activeStickyIndex], activeStickyIndex)}
                    </div>
                  )}
              </div>
              <ul
                data-rcl-virtual-list-content
                style={{
                  height: totalHeight,
                  margin: 0,
                  padding: 0,
                  listStyle: "none",
                }}
                aria-label={label}
              >
                {indexes.map((index) => {
                  const item = items[index];
                  if (item === undefined) return null;
                  const top = offsets[index] ?? 0;
                  return (
                    <li
                      key={getItemKey(item, index)}
                      ref={(node) => measure(index, node)}
                      data-rcl-virtual-list-row
                      data-sticky={stickyIndices.includes(index) || undefined}
                      aria-posinset={index + 1}
                      aria-setsize={items.length}
                      style={{
                        transform: `translateY(${top}px)`,
                        minHeight: measurements[index] ?? estimateItemHeight,
                      }}
                    >
                      {renderItem(item, index)}
                    </li>
                  );
                })}
              </ul>
            </div>
            <div data-rcl-virtual-list-status role="status" aria-live="polite">
              Showing {start + 1}–{Math.min(items.length, end + 1)} of{" "}
              {items.length} items
            </div>
          </>
        )}
      </section>
    </>
  );
});
