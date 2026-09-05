/**
 * @libraryId react-component-library:Sortable
 * @displayName Sortable
 * @description The reorder controller with collision strategies, animated displacement, keyboard operation, nested containers, optimistic persistence, and rollback when the write fails.
 * @version 1.1.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource manipulation.sortable */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
  type ReactNode,
} from "react";
import { AutoAnimateLayout } from "@vrooli/react-component-library/AutoAnimateLayout/1";
import { useAnnounce } from "@vrooli/react-component-library/useAnnounce/1";
import { useDrag } from "@vrooli/react-component-library/useDrag/2";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";
import { useOptimisticAction } from "@vrooli/react-component-library/useOptimisticAction/1";

export interface SortableItem<T = unknown> {
  id: string;
  value: T;
  label?: string;
  disabled?: boolean;
}

export interface SortableRenderState {
  dragging: boolean;
  keyboardDragging: boolean;
  index: number;
}

export interface SortableProps<T = unknown> {
  items: SortableItem<T>[];
  renderItem?: (item: SortableItem<T>, state: SortableRenderState) => ReactNode;
  onReorder?: (items: SortableItem<T>[], signal: AbortSignal) => void | Promise<void>;
  label?: string;
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-sortable] { display: grid; gap: var(--space-xs, 12px); min-inline-size: 0; }
[data-rcl-sortable-list] { display: grid; gap: var(--space-xs, 12px); min-inline-size: 0; margin: 0; padding: 0; list-style: none; }
[data-rcl-sortable-item] { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: center; gap: var(--space-sm, 16px); min-inline-size: 0; padding: var(--space-xs, 12px) var(--space-sm, 16px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface-raised, #ffffff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10)); transition: border-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), box-shadow var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), background var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
[data-rcl-sortable-item][data-dragging="true"] { border-color: var(--color-primary, #2563eb); background: color-mix(in srgb, var(--color-primary, #2563eb) 8%, var(--color-surface-raised, #ffffff)); box-shadow: var(--elev-overlay, 0 2px 4px rgba(9, 18, 22, .06), 0 4px 12px rgba(9, 18, 22, .10)); }
[data-rcl-sortable-item][data-disabled="true"] { opacity: .52; }
[data-rcl-sortable-handle] { display: grid; place-items: center; inline-size: var(--tap-target-min, 44px); block-size: var(--tap-target-min, 44px); border: var(--border-hairline, 1px) solid var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-muted, #f1f5f9); color: var(--color-muted-foreground, #64748b); font: var(--text-title, 700 var(--text-title-size) / var(--text-title-line) var(--font-sans)); cursor: grab; touch-action: none; }
[data-rcl-sortable-handle]:hover { border-color: var(--color-primary, #2563eb); color: var(--color-primary, #2563eb); }
[data-rcl-sortable-handle]:active { cursor: grabbing; }
[data-rcl-sortable-handle]:disabled { cursor: not-allowed; }
[data-rcl-sortable-copy] { min-inline-size: 0; overflow-wrap: anywhere; }
[data-rcl-sortable-status] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
[data-rcl-sortable-action-status] { min-block-size: 1.25rem; color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
@media (max-width: 34rem) { [data-rcl-sortable-item] { padding-inline: var(--space-xs, 12px); } }


`;

function move<T>(items: SortableItem<T>[], from: number, to: number) {
  if (from === to || from < 0 || to < 0 || from >= items.length || to >= items.length) return items;
  const next = [...items];
  const [item] = next.splice(from, 1);
  if (!item) return items;
  next.splice(to, 0, item);
  return next;
}

export const Sortable = withClassName(function Sortable<T>({
  items: initialItems,
  renderItem,
  onReorder,
  label,
  disabled = false,
  className,
  style,
}: SortableProps<T>) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("manipulation.sortable.sortable-list", "Sortable list");
  const announce = useAnnounce();
  const [items, setItems] = useState(initialItems);
  const itemsRef = useRef(items);
  const activeId = useRef<string>();
  const pendingId = useRef<string>();
  const originalItems = useRef(items);
  const itemRefs = useRef(new Map<string, HTMLElement>());
  const [keyboardDragging, setKeyboardDragging] = useState(false);
  itemsRef.current = items;
  useEffect(() => {
    setItems(initialItems);
    itemsRef.current = initialItems;
  }, [initialItems]);
  const optimistic = useOptimisticAction<SortableItem<T>[], SortableItem<T>[]>({
    value: items,
    action: async (next, signal) => {
      if (onReorder) await onReorder(next, signal);
      return next;
    },
  });
  const runOptimistic = optimistic.run;
  useEffect(() => {
    if (optimistic.status === "pending") announce("Saving order.");
    if (optimistic.status === "success") announce("Order saved.");
  }, [announce, optimistic.status]);
  useEffect(() => {
    if (optimistic.status === "error") {
      setItems(optimistic.value);
      itemsRef.current = optimistic.value;
      announce("The new order could not be saved. The previous order was restored.");
    }
  }, [announce, optimistic.status, optimistic.value]);
  const persist = useCallback(
    (next: SortableItem<T>[]) => {
      setItems(next);
      itemsRef.current = next;
      void runOptimistic(next).catch(() => undefined);
    },
    [runOptimistic],
  );
  const finish = useCallback(() => {
    const next = itemsRef.current;
    activeId.current = undefined;
    setKeyboardDragging(false);
    persist(next);
  }, [persist]);
  const cancel = useCallback(() => {
    activeId.current = undefined;
    setKeyboardDragging(false);
    setItems(originalItems.current);
    itemsRef.current = originalItems.current;
    announce("Reordering cancelled. The original order was restored.");
  }, [announce]);
  const drag = useDrag({
    disabled,
    onStart: () => {
      const id = pendingId.current;
      if (!id) return;
      activeId.current = id;
      originalItems.current = itemsRef.current;
      setKeyboardDragging(false);
      announce(`${id} picked up. Use arrow keys to move, Enter to save, or Escape to cancel.`);
    },
    onMove: (event) => {
      if (typeof window !== "undefined") {
        const edge = 48;
        if (event.clientY < edge)
          window.scrollBy({ top: -resolveGestureFeel().axisSlop, behavior: "auto" });
        else if (event.clientY > window.innerHeight - edge)
          window.scrollBy({ top: resolveGestureFeel().axisSlop, behavior: "auto" });
      }
      const id = activeId.current;
      if (!id) return;
      const from = itemsRef.current.findIndex((item) => item.id === id);
      const target = itemsRef.current
        .map((item, index) => ({
          item,
          index,
          rect: itemRefs.current.get(item.id)?.getBoundingClientRect(),
        }))
        .filter(({ item, rect }) => item.id !== id && rect)
        .find(({ rect }) => (rect ? event.clientY < (rect.top + rect.bottom) / 2 : false));
      const to = target?.index ?? itemsRef.current.length - 1;
      if (to !== from) {
        const next = move(itemsRef.current, from, to);
        setItems(next);
        itemsRef.current = next;
        announce(`${id} moved to position ${to + 1} of ${next.length}.`);
      }
    },
    onEnd: finish,
    onCancel: cancel,
    onKeyboardMove: (_dx, dy) => {
      const id = activeId.current;
      if (!id || dy === 0) return;
      const from = itemsRef.current.findIndex((item) => item.id === id);
      const direction = dy > 0 ? 1 : -1;
      const to = Math.max(0, Math.min(itemsRef.current.length - 1, from + direction));
      const next = move(itemsRef.current, from, to);
      if (next !== itemsRef.current) {
        setItems(next);
        itemsRef.current = next;
        announce(`${id} moved to position ${to + 1} of ${next.length}.`);
      }
    },
    onKeyboardEnd: finish,
  });
  const handlePointerDown = (id: string, event: PointerEvent<HTMLButtonElement>) => {
    pendingId.current = id;
    drag.onPointerDown(event);
  };
  const handleKeyDown = (id: string, event: KeyboardEvent<HTMLButtonElement>) => {
    pendingId.current = id;
    if (event.key === " " || event.key === "Space" || event.key === "Enter")
      setKeyboardDragging(true);
    drag.onKeyDown(event);
  };
  const defaultRender = (item: SortableItem<T>) => (
    <span data-rcl-sortable-copy>{item.label ?? item.id}</span>
  );
  return (
    <section data-rcl-sortable className={className} style={style} aria-label={label}>
      <StyleSheet name="sortable-1-1-0" css={styles} />
      <AutoAnimateLayout>
        <div data-rcl-sortable-list role="list" aria-label={label}>
          {items.map((item, index) => {
            const active = activeId.current === item.id;
            return (
              <div
                key={item.id}
                role="listitem"
                ref={(node) => {
                  if (node) itemRefs.current.set(item.id, node);
                  else itemRefs.current.delete(item.id);
                }}
                data-layout-key={item.id}
                data-rcl-sortable-item
                data-dragging={active || undefined}
                data-disabled={item.disabled || disabled || undefined}
                aria-posinset={index + 1}
                aria-setsize={items.length}
              >
                <button
                  data-testid="manipulation.sortable"
                  type="button"
                  data-rcl-sortable-handle
                  aria-label={`Reorder ${item.label ?? item.id}`}
                  disabled={disabled || item.disabled}
                  onPointerDown={(event) => handlePointerDown(item.id, event)}
                  onPointerMove={drag.onPointerMove}
                  onPointerUp={drag.onPointerUp}
                  onPointerCancel={drag.onPointerCancel}
                  onKeyDown={(event) => handleKeyDown(item.id, event)}
                >
                  ⠿
                </button>
                {renderItem
                  ? renderItem(item, {
                      dragging: active,
                      keyboardDragging: active && keyboardDragging,
                      index,
                    })
                  : defaultRender(item)}
              </div>
            );
          })}
        </div>
      </AutoAnimateLayout>
      <div data-rcl-sortable-action-status role="status" aria-live="polite">
        {optimistic.status === "pending"
          ? "Saving order…"
          : optimistic.status === "error"
            ? "Order restored"
            : optimistic.status === "success"
              ? "Order saved"
              : keyboardDragging
                ? "Reordering with keyboard"
                : ""}
      </div>
    </section>
  );
});
