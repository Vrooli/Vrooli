/**
 * @libraryId react-component-library:VirtualList
 * @displayName VirtualList
 * @description The large-collection renderer with dynamic measurement, overscan, preserved focus, sticky items, scroll restoration, and accessible item-count semantics.
 * @version 1.2.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
/** @vrooliComponentSource data-display.virtual-list */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  useEffect,
  useImperativeHandle,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
  type Ref,
} from "react";

export interface VirtualListPosition {
  scrollTop: number;
  scrollHeight: number;
  viewportHeight: number;
  atStart: boolean;
  atEnd: boolean;
}
export interface VirtualListHandle {
  scrollToOffset: (offset: number, behavior?: ScrollBehavior) => void;
  scrollToEnd: (behavior?: ScrollBehavior) => void;
  getScrollPosition: () => VirtualListPosition | null;
}
export interface VirtualListProps<T> {
  items?: T[];
  renderItem: (item: T, index: number) => ReactNode;
  /** Stable, unique keys are required when items can be inserted or reordered. */
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
  onViewportChange?: (position: VirtualListPosition) => void;
  controllerRef?: Ref<VirtualListHandle>;
  /** Follow growth only while the reader is at the end. Start at the end unless initialScrollTop is supplied. */
  followEnd?: boolean;
  endThreshold?: number;
  stickyIndices?: number[];
  className?: string;
  style?: CSSProperties;
}

const styles = `
  /* CollectionList composes VirtualList as a full-width row shell. Keep the
     virtualization layers block-sized so scenario card content never falls
     back to its intrinsic width and wraps one character per line. */
  .rcl-collection-list__virtual[data-rcl-virtual-list],
  .rcl-collection-list__virtual [data-rcl-virtual-list-content],
  .rcl-collection-list__virtual [data-rcl-virtual-list-row],
  .rcl-collection-list__virtual [data-rcl-collection-key] { inline-size: 100%; }
  [data-rcl-virtual-list] { min-inline-size: 0; overflow: hidden; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface, #ffffff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10)); }
  [data-rcl-virtual-list-header] { display: grid; gap: var(--space-3xs, 4px); padding: var(--space-md, 24px) var(--space-lg, 32px); border-block-end: 1px solid var(--color-border, #cbd5e1); background: color-mix(in srgb, var(--color-primary, #2563eb) 4%, var(--color-surface-raised, #ffffff)); }
  [data-rcl-virtual-list-title] { font: var(--text-subtitle, 600 var(--text-subheading-size) / var(--text-subheading-line) var(--font-sans)); }
  [data-rcl-virtual-list-description], [data-rcl-virtual-list-status] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-virtual-list-viewport] { position: relative; overflow: auto; overflow-anchor: none; overscroll-behavior: contain; scrollbar-color: var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))) transparent; }
  [data-rcl-virtual-list-content] { position: relative; min-inline-size: 100%; }
  [data-rcl-virtual-list-sticky-layer] { position: sticky; inset-block-start: 0; z-index: 4; block-size: 0; pointer-events: none; overflow: visible; }
  [data-rcl-virtual-list-sticky-row] { box-sizing: border-box; min-inline-size: 100%; padding: var(--space-md, 24px) var(--space-lg, 32px); border-block-end: 1px solid var(--color-border, #cbd5e1); background: var(--color-surface, #ffffff); box-shadow: 0 5px 12px rgb(15 23 42 / .12); overflow-wrap: anywhere; }
  [data-rcl-virtual-list-row] { position: absolute; inset-inline: 0; display: block; box-sizing: border-box; min-inline-size: 0; padding: var(--space-md, 24px) var(--space-lg, 32px); border-block-end: 1px solid var(--color-border, #cbd5e1); background: var(--color-surface, #ffffff); overflow-wrap: anywhere; }
  [data-rcl-virtual-list-row]:focus-within { z-index: 2; outline: 2px solid var(--color-focus, #2563eb); outline-offset: -2px; }
  .rcl-collection-list__virtual [data-rcl-virtual-list-row] { inline-size: 100%; }
  [data-rcl-virtual-list-row][data-sticky="true"] { z-index: 3; box-shadow: 0 5px 12px rgb(15 23 42 / .12); }
  [data-rcl-virtual-list-empty] { display: grid; min-block-size: 10rem; place-items: center; padding: var(--space-lg, 32px); color: var(--color-muted-foreground, #64748b); text-align: center; font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
  @media (max-width: 30rem) { [data-rcl-virtual-list-header], [data-rcl-virtual-list-row] { padding-inline: var(--space-md, 24px); } }
`;

// Find the row containing the offset, including a partially visible row.
function indexAtOffset(offsets: number[], target: number) {
  let low = 0;
  let high = Math.max(0, offsets.length - 2);
  while (low < high) {
    const middle = Math.ceil((low + high) / 2);
    if ((offsets[middle] ?? 0) <= target) low = middle;
    else high = middle - 1;
  }
  return low;
}
const defaultKey = (_item: unknown, index: number) => `virtual-item-${index}`;
const noItems: never[] = [];
const noSticky: number[] = [];
type Layout = { keys: string[]; offsets: number[]; viewportHeight: number };
function positionOf(viewport: HTMLDivElement, threshold: number): VirtualListPosition {
  return {
    scrollTop: viewport.scrollTop,
    scrollHeight: viewport.scrollHeight,
    viewportHeight: viewport.clientHeight,
    atStart: viewport.scrollTop <= 1,
    atEnd: viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop <= threshold,
  };
}

export const VirtualList = withClassName(function VirtualList<T>({
  items = noItems,
  renderItem,
  getItemKey = defaultKey,
  estimateItemHeight = 72,
  overscan = 4,
  height = 360,
  label,
  title,
  description,
  empty,
  initialScrollTop,
  onScrollPositionChange,
  onViewportChange,
  controllerRef,
  followEnd = false,
  endThreshold = 24,
  stickyIndices = noSticky,
  className,
  style,
}: VirtualListProps<T>) {
  const strings = useStrings();
  label = label ?? strings("data-display.virtual-list.virtual-list", "Virtual list");
  empty = empty ?? strings("data-display.virtual-list.empty", "Nothing here yet.");
  const estimate = Number.isFinite(estimateItemHeight) ? Math.max(1, estimateItemHeight) : 72;
  const extra = Number.isFinite(overscan) ? Math.max(0, Math.floor(overscan)) : 4;
  const threshold = Number.isFinite(endThreshold) ? Math.max(0, endThreshold) : 24;
  const [scrollTop, setScrollTop] = useState(initialScrollTop ?? 0);
  const [viewportHeight, setViewportHeight] = useState(360);
  const [measurements, setMeasurements] = useState<Record<string, number>>({});
  const [focusedKey, setFocusedKey] = useState<string>();
  const viewportRef = useRef<HTMLDivElement>(null);
  const layoutRef = useRef<Layout | null>(null);
  const positionRef = useRef(initialScrollTop ?? 0);
  const initialRef = useRef(initialScrollTop);
  const frame = useRef<number>();
  const rowObservers = useRef(new Map<string, ResizeObserver>());
  const rowRefs = useRef(new Map<string, (node: HTMLLIElement | null) => void>());
  const measuredHeights = useRef(new Map<string, number>());
  const callbacks = useRef({ onScrollPositionChange, onViewportChange });
  callbacks.current = { onScrollPositionChange, onViewportChange };
  const lastReported = useRef("");
  const nextKeys = useMemo(() => items.map(getItemKey), [items, getItemKey]);
  const stableKeys = useRef(nextKeys);
  if (
    stableKeys.current.length !== nextKeys.length ||
    nextKeys.some((key, index) => key !== stableKeys.current[index])
  )
    stableKeys.current = nextKeys;
  const keys = stableKeys.current;
  const keyIndexes = useMemo(() => new Map(keys.map((key, index) => [key, index])), [keys]);
  if (keyIndexes.size !== keys.length) throw new Error("VirtualList item keys must be unique");
  const offsets = useMemo(() => {
    const result = [0];
    keys.forEach((key, index) =>
      result.push(
        (result[index] ?? 0) +
          (Object.prototype.hasOwnProperty.call(measurements, key)
            ? (measurements[key] ?? estimate)
            : estimate),
      ),
    );
    return result;
  }, [keys, measurements, estimate]);
  const totalHeight = offsets[items.length] ?? 0;
  const firstVisible = indexAtOffset(offsets, scrollTop);
  const lastVisible = indexAtOffset(offsets, scrollTop + viewportHeight);
  const start = Math.max(0, firstVisible - extra);
  const end = Math.min(items.length - 1, lastVisible + extra);
  const activeStickyIndex = [...stickyIndices]
    .filter((index) => index >= 0 && index < items.length && (offsets[index] ?? 0) < scrollTop)
    .sort((a, b) => a - b)
    .at(-1);
  const indexes = useMemo(() => {
    const visible = new Set<number>();
    for (let index = start; index <= end; index += 1) visible.add(index);
    stickyIndices.forEach((index) => {
      if (index >= 0 && index < items.length) visible.add(index);
    });
    const focused = focusedKey === undefined ? undefined : keyIndexes.get(focusedKey);
    if (focused !== undefined) visible.add(focused);
    return [...visible].sort((a, b) => a - b);
  }, [start, end, stickyIndices, items.length, focusedKey, keyIndexes]);

  const reportPosition = () => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const position = positionOf(viewport, threshold);
    const identity = JSON.stringify(position);
    if (identity === lastReported.current) return;
    lastReported.current = identity;
    callbacks.current.onViewportChange?.(position);
  };
  useImperativeHandle(
    controllerRef,
    () => ({
      scrollToOffset(offset, behavior = "auto") {
        const viewport = viewportRef.current;
        if (!viewport || !Number.isFinite(offset)) return;
        const motion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches
          ? "auto"
          : behavior;
        viewport.scrollTo({ top: Math.max(0, offset), behavior: motion });
        positionRef.current = viewport.scrollTop;
        setScrollTop(viewport.scrollTop);
        reportPosition();
      },
      scrollToEnd(behavior = "auto") {
        const viewport = viewportRef.current;
        if (!viewport) return;
        const motion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches
          ? "auto"
          : behavior;
        viewport.scrollTo({ top: viewport.scrollHeight, behavior: motion });
        positionRef.current = viewport.scrollTop;
        setScrollTop(viewport.scrollTop);
        reportPosition();
      },
      getScrollPosition: () =>
        viewportRef.current ? positionOf(viewportRef.current, threshold) : null,
    }),
    [threshold],
  );

  useLayoutEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const resize = () => setViewportHeight(viewport.clientHeight);
    resize();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(resize);
    observer.observe(viewport);
    return () => observer.disconnect();
  }, []);

  useLayoutEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const previous = layoutRef.current;
    let next = positionRef.current;
    if (!previous || initialRef.current !== initialScrollTop) {
      next = initialScrollTop ?? (followEnd ? viewport.scrollHeight : 0);
    } else if (
      followEnd &&
      (previous.offsets[previous.keys.length] ?? 0) -
        previous.viewportHeight -
        positionRef.current <=
        threshold
    ) {
      next = viewport.scrollHeight;
    } else if (previous.keys.length) {
      const oldIndex = indexAtOffset(previous.offsets, positionRef.current);
      // Prefer the same visible item; if removed, preserve the nearest surviving
      // successor (or predecessor) at its former viewport offset.
      const survives = (index: number) => {
        const key = previous.keys[index];
        return key !== undefined && keyIndexes.has(key);
      };
      let anchor = oldIndex;
      while (anchor < previous.keys.length && !survives(anchor)) anchor += 1;
      if (anchor === previous.keys.length) {
        anchor = oldIndex - 1;
        while (anchor >= 0 && !survives(anchor)) anchor -= 1;
      }
      if (anchor >= 0 && anchor < previous.keys.length) {
        const key = previous.keys[anchor];
        const newIndex = key === undefined ? undefined : keyIndexes.get(key);
        const newOffset = newIndex === undefined ? undefined : offsets[newIndex];
        const oldOffset = previous.offsets[anchor];
        if (newOffset !== undefined && oldOffset !== undefined)
          next = newOffset + positionRef.current - oldOffset;
      }
    }
    viewport.scrollTop = Number.isFinite(next) ? Math.max(0, next) : 0;
    positionRef.current = viewport.scrollTop;
    if (scrollTop !== viewport.scrollTop) setScrollTop(viewport.scrollTop);
    layoutRef.current = { keys, offsets, viewportHeight };
    initialRef.current = initialScrollTop;
    reportPosition();
  }, [keys, keyIndexes, offsets, viewportHeight, initialScrollTop, followEnd, threshold]);

  useEffect(() => {
    for (const key of rowRefs.current.keys()) {
      if (keyIndexes.has(key)) continue;
      rowObservers.current.get(key)?.disconnect();
      rowObservers.current.delete(key);
      rowRefs.current.delete(key);
      measuredHeights.current.delete(key);
    }
    setMeasurements((current) => {
      if (Object.keys(current).every((key) => keyIndexes.has(key))) return current;
      return Object.fromEntries(Object.entries(current).filter(([key]) => keyIndexes.has(key)));
    });
    if (focusedKey !== undefined && !keyIndexes.has(focusedKey)) setFocusedKey(undefined);
  }, [keyIndexes, focusedKey]);
  useEffect(
    () => () => {
      if (frame.current !== undefined) cancelAnimationFrame(frame.current);
      rowObservers.current.forEach((observer) => observer.disconnect());
      rowObservers.current.clear();
    },
    [],
  );

  const rowRef = (key: string) => {
    let callback = rowRefs.current.get(key);
    if (!callback) {
      callback = (node) => {
        rowObservers.current.get(key)?.disconnect();
        rowObservers.current.delete(key);
        if (!node) return;
        const update = () => {
          // Both first paint and resize use the border box. Mixing contentRect
          // with this measurement repeatedly adds/removes row padding and borders.
          const next = Math.ceil(node.getBoundingClientRect().height);
          if (next <= 0 || measuredHeights.current.get(key) === next) return;
          measuredHeights.current.set(key, next);
          setMeasurements((current) =>
            current[key] === next ? current : { ...current, [key]: next },
          );
        };
        update();
        if (typeof ResizeObserver !== "undefined") {
          const observer = new ResizeObserver(update);
          observer.observe(node, { box: "border-box" });
          rowObservers.current.set(key, observer);
        }
      };
      rowRefs.current.set(key, callback);
    }
    return callback;
  };
  const onScroll = () => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    positionRef.current = viewport.scrollTop;
    if (frame.current !== undefined) return;
    frame.current = requestAnimationFrame(() => {
      frame.current = undefined;
      setScrollTop(viewport.scrollTop);
      callbacks.current.onScrollPositionChange?.(viewport.scrollTop);
      reportPosition();
    });
  };
  const range = strings(
    "data-display.virtual-list.range",
    "Showing {first}–{last} of {count} items",
  )
    .replace("{first}", String(items.length ? firstVisible + 1 : 0))
    .replace("{last}", String(items.length ? lastVisible + 1 : 0))
    .replace("{count}", String(items.length));
  return (
    <>
      <StyleSheet libraryId="react-component-library:VirtualList" version="1.2.2" css={styles} />
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
            {description && <span data-rcl-virtual-list-description>{description}</span>}
          </header>
        )}
        <div
          ref={viewportRef}
          data-rcl-virtual-list-viewport
          style={{ height }}
          onScroll={onScroll}
          tabIndex={0}
          role="region"
          aria-label={label}
          onFocusCapture={(event) => {
            const row =
              event.target instanceof Element
                ? event.target.closest<HTMLElement>("[data-rcl-virtual-list-key]")
                : null;
            setFocusedKey(row?.dataset.rclVirtualListKey);
          }}
          onBlurCapture={(event) => {
            if (
              !(event.relatedTarget instanceof Node) ||
              !event.currentTarget.contains(event.relatedTarget)
            )
              setFocusedKey(undefined);
          }}
        >
          {items.length === 0 ? (
            <div data-rcl-virtual-list-empty role="status">
              {empty}
            </div>
          ) : (
            <>
              <div
                data-rcl-virtual-list-sticky-layer
                aria-hidden="true"
                ref={(node) => node?.setAttribute("inert", "")}
              >
                {activeStickyIndex !== undefined && items[activeStickyIndex] !== undefined && (
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
                  const key = keys[index];
                  if (item === undefined || key === undefined) return null;
                  return (
                    <li
                      key={key}
                      ref={rowRef(key)}
                      data-rcl-virtual-list-row
                      data-rcl-virtual-list-key={key}
                      data-sticky={stickyIndices.includes(index) || undefined}
                      aria-posinset={index + 1}
                      aria-setsize={items.length}
                      style={{ transform: `translateY(${offsets[index]}px)` }}
                    >
                      {renderItem(item, index)}
                    </li>
                  );
                })}
              </ul>
            </>
          )}
        </div>
        <div data-rcl-virtual-list-status role="status" aria-live="polite">
          {range}
        </div>
      </section>
    </>
  );
});
