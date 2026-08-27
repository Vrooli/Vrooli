/**
 * @libraryId react-component-library:ContextMenu
 * @displayName ContextMenu
 * @description Shared overlay presentation for ContextMenu.
 * @version 1.2.1
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
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  type RefObject,
} from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1.1.0";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.3.5";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { contextMenuStyles } from "./styles";
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
}
export function ContextMenu({
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
}: ContextMenuProps) {
  useLibraryStyleSheet("context-menu-1.2.1", contextMenuStyles);
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
    dismiss: { escape: true, backdrop: true, swipe: desktop ? false : "down" },
    scrollLock: !desktop,
  });
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
  const trigger =
    children && isValidElement(Children.only(children))
      ? cloneElement(Children.only(children) as ReactElement<Record<string, unknown>>, {
          ref: (element: HTMLElement | null) => {
            localAnchor.current = element;
          },
          "aria-haspopup": "menu",
          "aria-expanded": overlay.open,
          onContextMenu: (event: Event) => {
            event.preventDefault();
            overlay.setOpen(true);
          },
        })
      : null;
  if (!overlay.present) return trigger;
  const anchor = anchorRef?.current ?? localAnchor.current;
  const anchoredPosition = desktop && anchor ? anchor.getBoundingClientRect() : null;
  const position = desktop
    ? requestedPosition
      ? {
          top: Math.max(8, Math.min(requestedPosition.y, window.innerHeight - 320)),
          left: Math.max(8, Math.min(requestedPosition.x, window.innerWidth - 240)),
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
          data-rcl-context-menu
          data-presentation={desktop ? "menu" : "sheet"}
          data-state={overlay.state}
        >
          {!desktop ? (
            <button
              type="button"
              data-testid={`${testId}.backdrop`}
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
            style={
              position
                ? {
                    top: position.top,
                    left: position.left,
                  }
                : undefined
            }
            onKeyDown={onKeyDown}
          >
            {!desktop ? (
              <button
                {...overlay.grabberProps}
                data-testid={`${testId}.grabber`}
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
}
