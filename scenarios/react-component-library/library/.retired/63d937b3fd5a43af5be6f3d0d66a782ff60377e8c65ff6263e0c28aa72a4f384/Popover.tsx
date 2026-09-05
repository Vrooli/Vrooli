/**
 * @libraryId react-component-library:Popover
 * @displayName Popover
 * @description The collision-aware anchored surface with focus management, pointer-safe interaction corridors, arrow positioning, nested layering, and a responsive fallback presentation.
 * @version 1.2.4
 * @tags ["overlay","anchored","accessible","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

/** @vrooliComponentSource overlays.popover */
import {
  Children,
  cloneElement,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  isValidElement,
  type ButtonHTMLAttributes,
  type HTMLAttributes,
  type KeyboardEvent,
  type MouseEvent as ReactMouseEvent,
  type MutableRefObject,
  type ReactElement,
  type ReactNode,
} from "react";
import { Presence } from "@vrooli/react-component-library/Presence/1.0.1";
import { Surface } from "@vrooli/react-component-library/Surface/1.0.1";
import { useControllableState } from "@vrooli/react-component-library/useControllableState/1.0.0";
import { useOutsideInteraction } from "@vrooli/react-component-library/useOutsideInteraction/1.0.0";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.1.1";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";

export type PopoverPlacement =
  | "top"
  | "top-start"
  | "top-end"
  | "bottom"
  | "bottom-start"
  | "bottom-end"
  | "right-start"
  | "right-end"
  | "left-start"
  | "left-end";
export type PopoverMode = "controlled" | "uncontrolled";

interface PopoverContextValue {
  contentId: string;
  triggerId: string;
  contentRef: MutableRefObject<HTMLDivElement | null>;
  triggerRef: MutableRefObject<HTMLElement | null>;
  open: boolean;
  setOpen: (next: boolean) => void;
  restoreFocus: boolean;
  placement: PopoverPlacement;
  responsive: "auto" | "none";
}

const PopoverContext = createContext<PopoverContextValue | null>(null);

const styles = `
[data-rcl-popover-content] { --rcl-popover-top: 0px; --rcl-popover-left: 0px; --rcl-popover-arrow-left: 50%; position: fixed; inset-block-start: var(--rcl-popover-top); inset-inline-start: var(--rcl-popover-left); z-index: var(--layer-popover, 200); inline-size: min(calc(100vw - (var(--space-lg) * 2)), 24rem); max-block-size: min(32rem, calc(100vh - (var(--space-lg) * 2))); overflow: auto; overscroll-behavior: contain; transform-origin: var(--rcl-popover-arrow-left) top; background: linear-gradient(var(--color-surface-raised, #ffffff), var(--color-surface-raised, #ffffff)), var(--color-background, #f8fafc); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); box-shadow: var(--elev-floating, 0 4px 12px rgba(9, 18, 22, 0.12)); color: var(--color-foreground, #0f172a); pointer-events: auto; }
[data-rcl-popover-content] :is(button, a[href], input, select, textarea, [role="button"]) { min-block-size: var(--tap-target-min, 44px); min-inline-size: var(--tap-target-min, 44px); }
[data-rcl-popover-content][data-placement^="top"] { transform-origin: var(--rcl-popover-arrow-left) bottom; }
[data-rcl-popover-arrow] { position: absolute; inline-size: var(--space-sm); block-size: var(--space-sm); background: inherit; border-block-start: inherit; border-inline-start: inherit; transform: translateX(-50%) rotate(45deg); }
[data-rcl-popover-content][data-placement^="bottom"] [data-rcl-popover-arrow] { inset-block-start: calc(var(--space-sm) * -.5); inset-inline-start: var(--rcl-popover-arrow-left); }
[data-rcl-popover-content][data-placement^="top"] [data-rcl-popover-arrow] { inset-block-end: calc(var(--space-sm) * -.5); inset-inline-start: var(--rcl-popover-arrow-left); transform: translateX(-50%) rotate(225deg); }
[data-rcl-popover-content][data-placement$="-start"] { transform-origin: var(--space-lg) top; }
[data-rcl-popover-content][data-placement$="-end"] { transform-origin: calc(100% - var(--space-lg)) top; }
[data-rcl-popover-content][data-placement^="right"] [data-rcl-popover-arrow], [data-rcl-popover-content][data-placement^="left"] [data-rcl-popover-arrow] { display: none; }
[data-rcl-popover-trigger]:not([data-rcl-popover-trigger-as-child]) { min-block-size: var(--tap-target-min, 44px); padding-inline: var(--space-md); border: 1px solid var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface, #ffffff); color: var(--color-foreground, #0f172a); cursor: pointer; font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); transition: background-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), border-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), box-shadow var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
[data-rcl-popover-trigger]:not([data-rcl-popover-trigger-as-child]):hover { background: var(--color-surface-raised, #ffffff); border-color: var(--color-primary, #2563eb); }
[data-rcl-popover-trigger]:focus-visible { outline: var(--border-strong, 2px) solid var(--color-focus, #2563eb); outline-offset: var(--space-3xs, 4px); box-shadow: 0 0 0 var(--space-3xs, 4px) color-mix(in srgb, var(--color-focus, #2563eb) 18%, transparent); }
@media (prefers-reduced-motion: reduce) { [data-rcl-popover-content] { transition: none; } }
[data-rcl-popover-content] :is(button, a[href], input, select, textarea, [role="button"]) { min-block-size: var(--tap-target-min, 44px); min-inline-size: var(--tap-target-min, 44px); }
@media (max-width: 38rem) {
  [data-rcl-popover-content][data-responsive="sheet"] { inset-block-start: auto; inset-block-end: calc(var(--space-sm) + env(safe-area-inset-bottom)); inset-inline: var(--space-sm); inline-size: auto; max-block-size: min(70vh, 32rem); transform-origin: bottom center; }
  [data-rcl-popover-content][data-responsive="sheet"] [data-rcl-popover-arrow] { display: none; }
}
`;

export interface PopoverProps {
  children: ReactNode;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  placement?: PopoverPlacement;
  restoreFocus?: boolean;
  responsive?: "auto" | "none";
}

export const Popover = withClassName(function Popover({
  children,
  defaultOpen = false,
  onOpenChange,
  open: controlledOpen,
  placement = "bottom-start",
  responsive = "auto",
  restoreFocus = true,
}: PopoverProps) {
  const [open, setOpen] = useControllableState({
    value: controlledOpen,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });
  const id = useId().replace(/:/g, "");
  const triggerRef = useRef<HTMLElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const wasOpen = useRef(open);
  const context = useMemo<PopoverContextValue>(
    () => ({
      contentId: `popover-content-${id}`,
      triggerId: `popover-trigger-${id}`,
      contentRef,
      triggerRef,
      open,
      setOpen,
      restoreFocus,
      placement,
      responsive,
    }),
    [id, open, placement, responsive, restoreFocus, setOpen],
  );

  useEffect(() => {
    if (wasOpen.current && !open && restoreFocus) triggerRef.current?.focus();
    wasOpen.current = open;
  }, [open, restoreFocus]);

  return (
    <PopoverContext.Provider value={context}>
      <style
        data-rcl-popover-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <PopoverPositioner placement={placement}>
        <div data-rcl-popover data-placement={placement} data-open={open}>
          {children}
        </div>
      </PopoverPositioner>
    </PopoverContext.Provider>
  );
});

function usePopoverContext() {
  const value = useContext(PopoverContext);
  if (!value) throw new Error("Popover parts must be used inside Popover");
  return value;
}

export function usePopover() {
  return usePopoverContext();
}

function PopoverPositioner({
  children,
  placement,
}: {
  children: ReactNode;
  placement: PopoverPlacement;
}) {
  const context = usePopoverContext();
  useOverlaySurface({
    open: context.open,
    onOpenChange: context.setOpen,
    modal: false,
    kind: "popover",
    returnFocusRef: context.restoreFocus ? context.triggerRef : undefined,
    scrollLock: false,
  });
  const excludeRefs = useMemo(() => [context.triggerRef], [context.triggerRef]);
  useOutsideInteraction({
    active: context.open,
    surfaceRef: context.contentRef,
    excludeRefs,
    dismissOnEscape: false,
    onPointerDownOutside: () => context.setOpen(false),
    onFocusOutside: () => context.setOpen(false),
  });
  useLayoutEffect(() => {
    if (!context.open || typeof window === "undefined") return;
    let frameId: number | undefined;
    let attempts = 0;
    const position = () => {
      const trigger = context.triggerRef.current;
      const content = context.contentRef.current;
      if (!trigger || !content) return;
      const triggerRect = trigger.getBoundingClientRect();
      const contentRect = content.getBoundingClientRect();
      const margin = readTokenPixels("--space-sm", 8);
      const arrowInset = readTokenPixels("--space-lg", 24);
      let nextPlacement = placement;
      let top = triggerRect.bottom + margin;
      let left = triggerRect.left + (triggerRect.width - contentRect.width) / 2;
      if (placement.startsWith("top"))
        top = triggerRect.top - contentRect.height - margin;
      if (placement.startsWith("right")) {
        top = triggerRect.top;
        left = triggerRect.right + margin;
      }
      if (placement.startsWith("left")) {
        top = triggerRect.top;
        left = triggerRect.left - contentRect.width - margin;
      }
      if (top < margin && placement.startsWith("top")) {
        top = triggerRect.bottom + margin;
        nextPlacement = placement.replace("top", "bottom") as PopoverPlacement;
      } else if (
        top + contentRect.height > window.innerHeight - margin &&
        placement.startsWith("bottom")
      ) {
        top = triggerRect.top - contentRect.height - margin;
        nextPlacement = placement.replace("bottom", "top") as PopoverPlacement;
      }
      if (
        placement.startsWith("right") &&
        left + contentRect.width > window.innerWidth - margin
      ) {
        left = triggerRect.left - contentRect.width - margin;
        nextPlacement = placement.replace("right", "left") as PopoverPlacement;
      } else if (placement.startsWith("left") && left < margin) {
        left = triggerRect.right + margin;
        nextPlacement = placement.replace("left", "right") as PopoverPlacement;
      } else if (placement.endsWith("-start"))
        left =
          placement.startsWith("right") || placement.startsWith("left")
            ? left
            : triggerRect.left;
      if (
        placement.endsWith("-end") &&
        !placement.startsWith("right") &&
        !placement.startsWith("left")
      )
        left = triggerRect.right - contentRect.width;
      left = Math.min(
        Math.max(margin, left),
        window.innerWidth - contentRect.width - margin,
      );
      top = Math.min(
        Math.max(margin, top),
        window.innerHeight - contentRect.height - margin,
      );
      content.style.setProperty("--rcl-popover-top", `${top}px`);
      content.style.setProperty("--rcl-popover-left", `${left}px`);
      const measured = content.getBoundingClientRect();
      const adjustedLeft = left + left - measured.left;
      const adjustedTop = top + top - measured.top;
      const arrowLeft = Math.min(
        Math.max(arrowInset, triggerRect.left + triggerRect.width / 2 - left),
        Math.max(arrowInset, contentRect.width - arrowInset),
      );
      content.style.setProperty("--rcl-popover-top", `${adjustedTop}px`);
      content.style.setProperty("--rcl-popover-left", `${adjustedLeft}px`);
      content.style.setProperty("--rcl-popover-arrow-left", `${arrowLeft}px`);
      content.dataset.placement = nextPlacement;
    };
    const positionWhenReady = () => {
      attempts += 1;
      if (context.triggerRef.current && context.contentRef.current) {
        position();
        return;
      }
      if (attempts < 12)
        frameId = window.requestAnimationFrame(positionWhenReady);
    };
    frameId = window.requestAnimationFrame(positionWhenReady);
    window.addEventListener("resize", position);
    window.addEventListener("scroll", position, true);
    return () => {
      if (frameId !== undefined) window.cancelAnimationFrame(frameId);
      window.removeEventListener("resize", position);
      window.removeEventListener("scroll", position, true);
    };
  }, [context.contentRef, context.open, context.triggerRef, placement]);
  return children;
}

function readTokenPixels(name: string, fallback: number) {
  if (typeof document === "undefined") return fallback;
  const root = getComputedStyle(document.documentElement);
  const raw = root.getPropertyValue(name).trim();
  const value = Number.parseFloat(raw);
  if (raw.endsWith("rem")) {
    const rootSize = Number.parseFloat(root.fontSize);
    return Number.isFinite(rootSize) ? value * rootSize : fallback;
  }
  return Number.isFinite(value) ? value : fallback;
}

export interface PopoverTriggerProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> {
  children: ReactNode;
  /** Compose an existing Button, Link, or identity surface without adding a nested button. */
  asChild?: boolean;
}

export const PopoverTrigger = withClassName(function PopoverTrigger({
  children,
  asChild = false,
  onClick,
  ...props
}: PopoverTriggerProps) {
  const context = usePopoverContext();
  const sharedProps = {
    ...props,
    id: context.triggerId,
    type: props.type ?? "button",
    "data-rcl-popover-trigger": true,
    "aria-haspopup": props["aria-haspopup"] ?? "dialog",
    "aria-expanded": context.open,
    "aria-controls": context.open ? context.contentId : undefined,
  };
  if (asChild) {
    const child = Children.only(children);
    if (!isValidElement(child))
      throw new Error("PopoverTrigger asChild requires one element child");
    const childProps = child.props as {
      onClick?: (event: ReactMouseEvent<HTMLElement>) => void;
      ref?: (element: HTMLElement | null) => void;
    };
    const childClick = childProps.onClick;
    return cloneElement(child as ReactElement<Record<string, unknown>>, {
      ...sharedProps,
      "data-rcl-popover-trigger-as-child": true,
      ref: mergeRefs(context.triggerRef, childProps.ref),
      onClick: (event: ReactMouseEvent<HTMLElement>) => {
        childClick?.(event);
        onClick?.(
          event as unknown as Parameters<NonNullable<typeof onClick>>[0],
        );
        if (!event.defaultPrevented) context.setOpen(!context.open);
      },
    });
  }
  return (
    <button
      data-testid="overlays.popover"
      {...sharedProps}
      ref={(element) => {
        context.triggerRef.current = element;
      }}
      onClick={(event) => {
        onClick?.(event);
        if (!event.defaultPrevented) context.setOpen(!context.open);
      }}
    >
      {children}
    </button>
  );
});

function mergeRefs<T>(
  ...refs: Array<
    ((value: T | null) => void) | MutableRefObject<T | null> | undefined
  >
) {
  return (value: T | null) => {
    refs.forEach((ref) => {
      if (!ref) return;
      if (typeof ref === "function") ref(value);
      else ref.current = value;
    });
  };
}

export interface PopoverContentProps
  extends Omit<HTMLAttributes<HTMLDivElement>, "children"> {
  children: ReactNode;
  placement?: PopoverPlacement;
  responsive?: "auto" | "none";
  initialFocus?: "first" | "content" | "none";
}

export const PopoverContent = withClassName(function PopoverContent({
  children,
  placement,
  responsive,
  initialFocus = "first",
  onKeyDown: onConsumerKeyDown,
  ...props
}: PopoverContentProps) {
  const context = usePopoverContext();
  useLayoutEffect(() => {
    if (
      !context.open ||
      initialFocus === "none" ||
      typeof window === "undefined"
    )
      return;
    const frame = window.requestAnimationFrame(() => {
      const content = context.contentRef.current;
      if (!content) return;
      if (initialFocus === "content") content.focus();
      else
        content
          .querySelector<HTMLElement>(
            "button, [href], input, select, textarea, [tabindex]:not([tabindex='-1'])",
          )
          ?.focus();
    });
    return () => window.cancelAnimationFrame(frame);
  }, [context.contentRef, context.open, initialFocus]);
  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>) => {
      if (event.key === "Tab" && context.open) {
        const focusable = Array.from(
          context.contentRef.current?.querySelectorAll<HTMLElement>(
            "button, [href], input, select, textarea, [tabindex]:not([tabindex='-1'])",
          ) ?? [],
        );
        if (focusable.length > 0) {
          if (event.shiftKey && document.activeElement === focusable[0]) {
            event.preventDefault();
            focusable.at(-1)?.focus();
          } else if (
            !event.shiftKey &&
            document.activeElement === focusable.at(-1)
          ) {
            event.preventDefault();
            focusable[0]?.focus();
          }
        }
      }
      onConsumerKeyDown?.(event);
    },
    [context.contentRef, context.open, onConsumerKeyDown],
  );
  return (
    <Portal>
      <Presence
        present={context.open}
        duration="quick"
        initial={false}
        style={{ position: "fixed", inset: 0, pointerEvents: "none" }}
      >
        <Surface
          {...props}
          ref={(element) => {
            context.contentRef.current = element;
          }}
          id={context.contentId}
          elevation="floating"
          role={props.role ?? "dialog"}
          tabIndex={initialFocus === "content" ? -1 : undefined}
          aria-modal="false"
          data-rcl-popover-content
          data-placement={placement ?? context.placement}
          data-responsive={
            (responsive ?? context.responsive) === "none" ? "none" : "sheet"
          }
          onKeyDown={onKeyDown}
        >
          <span data-rcl-popover-arrow aria-hidden="true" />
          {children}
        </Surface>
      </Presence>
    </Portal>
  );
});

export const PopoverParts = {
  Trigger: PopoverTrigger,
  Content: PopoverContent,
};
