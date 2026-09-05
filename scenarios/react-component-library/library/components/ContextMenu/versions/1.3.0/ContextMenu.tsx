/**
 * @libraryId react-component-library:ContextMenu
 * @displayName ContextMenu
 * @version 1.3.0
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import {
  Children,
  cloneElement,
  isValidElement,
  useCallback,
  useEffect,
  useRef,
  useState,
  forwardRef,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  type RefObject,
} from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useLongPress, type LongPressOrigin } from "@vrooli/react-component-library/useLongPress/1";
export const contextMenuStyles = `
[data-rcl-context-menu] { position: fixed; inset: 0; z-index: var(--layer-menu, 610); pointer-events: none; }
[data-rcl-context-menu][data-presentation="sheet"] { display: grid; align-items: end; }

.rcl-context-menu__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, color-mix(in srgb, var(--color-shell) 52%, transparent)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-context-menu][data-state="closed"] .rcl-context-menu__backdrop { opacity: 0; }

.rcl-context-menu__surface { position: relative; display: grid; gap: var(--space-3xs); align-content: start; inline-size: 100%; max-block-size: calc(100% - var(--rcl-safe-top, 0px) - var(--overlay-drawer-top-gap, 32px)); overflow: auto; overscroll-behavior: contain; padding: var(--space-2xs); padding-block-end: calc(var(--space-2xs) + var(--rcl-safe-bottom, 0px)); border: var(--border-hairline) solid var(--color-border); border-block-end: 0; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: transform var(--dur-quick) var(--ease-standard); animation: rcl-context-menu-enter-sheet var(--dur-moderate) var(--ease-enter); }
.rcl-context-menu__surface[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-context-menu][data-state="closed"] .rcl-context-menu__surface { transform: translateY(100%); animation: none; }
@keyframes rcl-context-menu-enter-sheet { from { transform: translateY(100%); } }

.rcl-context-menu__handle { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-context-menu__handle[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-context-menu__handle > span { display: block; inline-size: var(--overlay-grabber-inline, 36px); block-size: var(--overlay-grabber-block, 4px); border-radius: var(--radius-pill); background: var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))); }

.rcl-context-menu__surface h2 { margin: var(--space-sm) var(--space-xs) var(--space-3xs); font: var(--text-heading); }
[data-rcl-context-menu-item-wrap][data-separator] { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-start: var(--space-3xs); margin-block-start: var(--space-3xs); }
[data-rcl-context-menu-item-wrap] > button[role="menuitem"], [data-rcl-context-menu-item-wrap] > button[role="menuitemcheckbox"] { display: flex; align-items: center; inline-size: 100%; gap: var(--space-sm); min-block-size: var(--tap-target-min); padding: var(--space-xs) var(--space-sm); border: 0; border-radius: var(--radius-control); background: transparent; color: inherit; text-align: start; }
[data-rcl-context-menu-item-wrap] > button:hover, [data-rcl-context-menu-item-wrap] > button[data-active] { background: var(--color-surface-muted); }
[data-rcl-context-menu-item-wrap] > button[data-destructive] { color: var(--color-danger); }
[data-rcl-context-menu-item-icon] { display: inline-flex; flex: 0 0 auto; } [data-rcl-context-menu-item-wrap] kbd { margin-inline-start: auto; }
.rcl-context-menu__surface [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }

@media (min-width: 48rem) {
  .rcl-context-menu__surface { position: absolute; inset: auto; inline-size: auto; min-inline-size: 14rem; max-inline-size: 24rem; max-block-size: calc(100% - (var(--space-lg) * 2)); padding-block-end: var(--space-2xs); border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); box-shadow: var(--elev-overlay); transform: translateX(var(--overlay-menu-align, 0px)); animation-name: rcl-context-menu-enter-menu; }
  [data-rcl-context-menu][data-state="closed"] .rcl-context-menu__surface { transform: translateX(var(--overlay-menu-align, 0px)) translateY(var(--space-3xs)); opacity: 0; }
  .rcl-context-menu__surface[data-placement="bottom-end"] { transform: translateX(-100%); }
  [data-rcl-context-menu][data-state="closed"] .rcl-context-menu__surface[data-placement="bottom-end"] { transform: translateX(-100%) translateY(var(--space-3xs)); }
  .rcl-context-menu__surface h2 { margin: var(--space-xs); font: var(--text-label); }
}
@keyframes rcl-context-menu-enter-menu { from { opacity: 0; } }
`;
export interface ContextMenuItem {
  id: string;
  label: string;
  disabled?: boolean;
  onSelect: () => void | Promise<void>;
  shortcut?: string;
  icon?: ReactNode;
  testId?: string;
  separatorBefore?: boolean;
  destructive?: boolean;
  pressed?: boolean;
  state?: string;
  closeOnSelect?: boolean;
}
export interface ContextMenuProps {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  anchorRef?: RefObject<HTMLElement | null>;
  position?: { x: number; y: number };
  placement?: "bottom-start" | "bottom-end";
  title: string;
  items: ContextMenuItem[];
  children?: ReactNode;
  closeLabel: string;
  testId?: string;
  triggers?: readonly ContextMenuTrigger[];
  onOpenAt?: (origin: LongPressOrigin) => void;
}
type TriggerProps = React.HTMLAttributes<HTMLElement> & {
  ref?: (element: HTMLElement | null) => void;
};
export type ContextMenuTrigger = "contextmenu" | "long-press" | "anchor";
export const ContextMenu = forwardRef<HTMLDivElement, ContextMenuProps>(function ContextMenu({
  open,
  defaultOpen = false,
  onOpenChange,
  anchorRef,
  position: requestedPosition,
  placement = "bottom-start",
  title,
  items,
  children,
  closeLabel,
  testId = "overlays.context-menu",
  triggers = ["contextmenu", "long-press"],
  onOpenAt,
}: ContextMenuProps, forwardedRef) {
  useLibraryStyleSheet("context-menu-1.2.2", contextMenuStyles);
  const desktop = useBreakpoint("md");
  const localAnchor = useRef<HTMLElement | null>(null);
  const [active, setActive] = useState(0);
  const overlay = useOverlaySurface({
    open,
    defaultOpen,
    onOpenChange,
    modal: !desktop,
    kind: "menu",
    // The sheet presentation is dismissed by dragging it back down; the
    // anchored menu has no edge to drag toward, so it asks for no gesture.
    dismiss: { escape: true, backdrop: true, swipe: desktop ? false : "bottom" },
    scrollLock: !desktop,
  });
  const openAt = useCallback((origin: LongPressOrigin) => {
    setPointerPosition({ x: origin.x, y: origin.y });
    onOpenAt?.(origin);
    overlay.setOpen(true);
  }, [onOpenAt, overlay]);
  const [pointerPosition, setPointerPosition] = useState<{ x: number; y: number } | null>(null);
  const longPress = useLongPress({ onLongPress: openAt, disabled: !triggers.includes("long-press") });
  useEffect(() => {
    if (overlay.open)
      setActive(
        Math.max(
          0,
          items.findIndex((item) => !item.disabled),
        ),
      );
  }, [items, overlay.open]);
  const move = useCallback(
    (delta: number) => {
      setActive((current) => {
        let next = current;
        do next = (next + delta + items.length) % items.length;
        while (items[next]?.disabled && next !== current);
        return next;
      });
    },
    [items],
  );
  const onKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      move(1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      move(-1);
    } else if (event.key === "Home") {
      event.preventDefault();
      setActive(0);
    } else if (event.key === "End") {
      event.preventDefault();
      setActive(items.length - 1);
    } else if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      const item = items[active];
      if (item && !item.disabled) {
        void item.onSelect();
        overlay.close();
      }
    } else if (event.key.length === 1) {
      const index = items.findIndex(
        (item) =>
          !item.disabled &&
          item.label.toLocaleLowerCase().startsWith(event.key.toLocaleLowerCase()),
      );
      if (index >= 0) setActive(index);
    }
  };
  const trigger = children && isValidElement(Children.only(children))
    ? (() => {
        const child = Children.only(children) as ReactElement<TriggerProps>;
        const childProps = child.props;
        return cloneElement(child, {
          ref: (element: HTMLElement | null) => { localAnchor.current = element; },
          "aria-haspopup": "menu", "aria-expanded": overlay.open,
          onContextMenu: (event: React.MouseEvent<HTMLElement>) => { event.preventDefault(); childProps.onContextMenu?.(event); if (triggers.includes("contextmenu")) openAt({ x: event.clientX, y: event.clientY, pointerType: "mouse" }); },
          onPointerDown: (event: React.PointerEvent<HTMLElement>) => { childProps.onPointerDown?.(event); longPress.longPressProps.onPointerDown?.(event); },
          onPointerMove: (event: React.PointerEvent<HTMLElement>) => { childProps.onPointerMove?.(event); longPress.longPressProps.onPointerMove?.(event); },
          onPointerUp: (event: React.PointerEvent<HTMLElement>) => { childProps.onPointerUp?.(event); longPress.longPressProps.onPointerUp?.(event); },
          onPointerCancel: (event: React.PointerEvent<HTMLElement>) => { childProps.onPointerCancel?.(event); longPress.longPressProps.onPointerCancel?.(event); },
          onClick: (event: React.MouseEvent<HTMLElement>) => { longPress.longPressProps.onClick?.(event); if (!event.defaultPrevented) childProps.onClick?.(event); },
        });
      })()
    : null;
  if (!overlay.present) return trigger;
  const anchor = anchorRef?.current ?? localAnchor.current;
  const anchoredPosition = desktop && anchor ? anchor.getBoundingClientRect() : null;
  const viewportBounds =
    typeof window !== "undefined" ? { height: window.innerHeight, width: window.innerWidth } : null;
  const origin = pointerPosition ?? requestedPosition;
  const position = desktop
    ? origin && viewportBounds
      ? {
          top: Math.max(8, Math.min(origin.y, viewportBounds.height - 320)),
          left: Math.max(8, Math.min(origin.x, viewportBounds.width - 240)),
        }
      : anchoredPosition
        ? { top: anchoredPosition.bottom + 8, left: anchoredPosition.left }
        : null
    : null;
  return (
    <>
      {trigger}
      <Portal>
        <div
          ref={forwardedRef}
          {...overlay.rootProps}
          data-rcl-context-menu
          data-presentation={desktop ? "menu" : "sheet"}
          data-state={overlay.state}
        >
          {!desktop ? (
            <button
              type="button"
              data-testid="overlays.context-menu"
              aria-label={closeLabel}
              className="rcl-context-menu__backdrop"
              {...overlay.backdropProps}
            />
          ) : null}
          <div
            ref={(node) => {
              overlay.surfaceRef.current = node;
            }}
            data-testid={testId}
            role="menu"
            aria-label={title}
            tabIndex={-1}
            data-state={overlay.state}
            className="rcl-context-menu__surface"
            data-placement={placement}
            style={{
              ...(position ? { top: position.top, left: position.left } : {}),
            }}
            onKeyDown={onKeyDown}
          >
            {!desktop ? (
              <button
                {...overlay.grabberProps}
                data-testid="overlays.context-menu"
                aria-label={closeLabel}
                className="rcl-context-menu__handle"
              >
                <span aria-hidden />
              </button>
            ) : null}
            {!desktop ? <h2>{title}</h2> : null}
            {items.map((item, index) => (
              <div
                key={item.id}
                data-rcl-context-menu-item-wrap
                data-separator={item.separatorBefore || undefined}
              >
                <button
                  type="button"
                  role={item.pressed === undefined ? "menuitem" : "menuitemcheckbox"}
                  data-testid={item.testId}
                  data-rcl-selector="overlays.context-menu"
                  data-rcl-context-menu-item-id={item.id}
                  disabled={item.disabled}
                  aria-checked={item.pressed}
                  tabIndex={index === active ? 0 : -1}
                  data-active={index === active || undefined}
                  data-destructive={item.destructive || undefined}
                  data-state={item.state}
                  onFocus={() => setActive(index)}
                  onClick={() => {
                    void item.onSelect();
                    if (item.closeOnSelect !== false) overlay.close();
                  }}
                >
                  {item.icon ? <span data-rcl-context-menu-item-icon>{item.icon}</span> : null}
                  <span>{item.label}</span>
                  {item.shortcut ? <kbd>{item.shortcut}</kbd> : null}
                </button>
              </div>
            ))}
          </div>
        </div>
      </Portal>
    </>
  );
});
