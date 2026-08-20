/**
 * @libraryId react-component-library:Tooltip
 * @displayName Tooltip
 * @description A concise, accessible explanation attached to a focusable trigger.
 * @version 1.0.1
 * @tags ["overlay","accessible","interaction","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import {
  Children,
  cloneElement,
  createContext,
  createElement,
  isValidElement,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useRef,
  type FocusEvent,
  type HTMLAttributes,
  type KeyboardEvent,
  type MutableRefObject,
  type PointerEvent,
  type ReactElement,
  type ReactNode,
} from "react";
import { Presence } from "../../../../primitives/Presence/versions/1.0.0/Presence";
import { useControllableState } from "../../../../hooks/useControllableState/versions/1.0.0/useControllableState";

const styles = `
[data-rcl-tooltip] { position: relative; display: inline-flex; max-inline-size: 100%; vertical-align: middle; }
[data-rcl-tooltip-trigger] { max-inline-size: 100%; }
[data-rcl-tooltip-content] { position: absolute; inset-block-end: calc(100% + var(--space-2xs, .5rem)); inset-inline-start: 50%; z-index: var(--layer-tooltip, 900); inline-size: max-content; max-inline-size: min(18rem, calc(100vw - (var(--space-lg, 1.5rem) * 2))); padding: var(--space-2xs, .5rem) var(--space-xs, .75rem); border: 1px solid color-mix(in srgb, var(--color-border-strong, #64748b) 72%, transparent); border-radius: var(--radius-control, .625rem); background: var(--color-foreground, #0f172a); color: var(--color-surface, #fff); box-shadow: var(--elev-floating, 0 18px 48px rgb(15 23 42 / 18%)); font: var(--text-caption, 500 .75rem/1.35 system-ui, sans-serif); overflow-wrap: anywhere; pointer-events: none; transform: translateX(-50%); }
[data-rcl-tooltip-content]::after { position: absolute; inset-block-end: calc(var(--space-2xs, .5rem) * -1); inset-inline-start: 50%; inline-size: var(--space-2xs, .5rem); block-size: var(--space-2xs, .5rem); border-inline-end: inherit; border-block-end: inherit; background: inherit; content: ""; transform: translateX(-50%) rotate(45deg); }
[data-rcl-tooltip][data-placement="bottom"] [data-rcl-tooltip-content] { inset-block-start: calc(100% + var(--space-2xs, .5rem)); inset-block-end: auto; transform: translateX(-50%); }
[data-rcl-tooltip][data-placement="bottom"] [data-rcl-tooltip-content]::after { inset-block-start: calc(var(--space-2xs, .5rem) * -1); inset-block-end: auto; border-inline-end: 0; border-block-end: 0; border-inline-start: inherit; border-block-start: inherit; }
[data-rcl-tooltip][data-placement="start"] [data-rcl-tooltip-content] { inset-block-start: 50%; inset-inline-end: calc(100% + var(--space-2xs, .5rem)); inset-block-end: auto; inset-inline-start: auto; transform: translateY(-50%); }
[data-rcl-tooltip][data-placement="start"] [data-rcl-tooltip-content]::after { inset-block-start: 50%; inset-block-end: auto; inset-inline-end: calc(var(--space-2xs, .5rem) * -1); inset-inline-start: auto; border-inline-start: 0; border-block-start: inherit; border-inline-end: inherit; border-block-end: inherit; transform: translateY(-50%) rotate(-45deg); }
[data-rcl-tooltip][data-placement="end"] [data-rcl-tooltip-content] { inset-block-start: 50%; inset-inline-start: calc(100% + var(--space-2xs, .5rem)); inset-block-end: auto; transform: translateY(-50%); }
[data-rcl-tooltip][data-placement="end"] [data-rcl-tooltip-content]::after { inset-block-start: 50%; inset-block-end: auto; inset-inline-start: calc(var(--space-2xs, .5rem) * -1); inset-block-end: auto; transform: translateY(-50%) rotate(135deg); }
[data-rcl-tooltip] :focus-visible { outline: var(--border-strong, 2px) solid var(--color-focus, #2563eb); outline-offset: var(--space-3xs, .25rem); }
@media (prefers-reduced-motion: reduce) { [data-rcl-tooltip-content] { transition: none; } }
@media (forced-colors: active) { [data-rcl-tooltip-content] { border-color: CanvasText; background: CanvasText; color: Canvas; } [data-rcl-tooltip-content]::after { border-color: CanvasText; background: CanvasText; } }
`;

export type TooltipPlacement = "top" | "bottom" | "start" | "end";

interface TooltipContextValue {
  contentId: string;
  triggerId: string;
  triggerRef: MutableRefObject<HTMLElement | null>;
  open: boolean;
  setOpen: (next: boolean) => void;
  scheduleOpen: () => void;
  scheduleClose: () => void;
  cancelTimers: () => void;
  placement: TooltipPlacement;
}

const TooltipContext = createContext<TooltipContextValue | null>(null);

export interface TooltipProps {
  children: ReactNode;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  delay?: number;
  closeDelay?: number;
  placement?: TooltipPlacement;
}

export function Tooltip({
  children,
  open: controlledOpen,
  defaultOpen = false,
  onOpenChange,
  delay = 280,
  closeDelay = 100,
  placement = "top",
}: TooltipProps) {
  const [open, setOpen] = useControllableState({
    value: controlledOpen,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });
  const id = useId().replace(/:/g, "");
  const triggerRef = useRef<HTMLElement>(null);
  const timers = useRef<{ open?: ReturnType<typeof setTimeout>; close?: ReturnType<typeof setTimeout> }>({});
  const cancelTimers = useCallback(() => {
    if (timers.current.open) clearTimeout(timers.current.open);
    if (timers.current.close) clearTimeout(timers.current.close);
    timers.current = {};
  }, []);
  const scheduleOpen = useCallback(() => {
    cancelTimers();
    if (open) return;
    timers.current.open = setTimeout(() => setOpen(true), Math.max(0, delay));
  }, [cancelTimers, delay, open, setOpen]);
  const scheduleClose = useCallback(() => {
    cancelTimers();
    if (!open) return;
    timers.current.close = setTimeout(() => setOpen(false), Math.max(0, closeDelay));
  }, [cancelTimers, closeDelay, open, setOpen]);
  useEffect(() => cancelTimers, [cancelTimers]);
  const context = useMemo<TooltipContextValue>(
    () => ({
      contentId: `tooltip-content-${id}`,
      triggerId: `tooltip-trigger-${id}`,
      triggerRef,
      open,
      setOpen,
      scheduleOpen,
      scheduleClose,
      cancelTimers,
      placement,
    }),
    [cancelTimers, id, open, placement, scheduleClose, scheduleOpen, setOpen],
  );
  return (
    <TooltipContext.Provider value={context}>
      <span data-rcl-tooltip data-placement={placement}>
        <style data-rcl-tooltip-styles dangerouslySetInnerHTML={{ __html: styles }} />
        {children}
      </span>
    </TooltipContext.Provider>
  );
}

function useTooltipContext() {
  const value = useContext(TooltipContext);
  if (!value) throw new Error("Tooltip parts must be used inside Tooltip");
  return value;
}

export function useTooltip() {
  return useTooltipContext();
}

function mergeRefs<T>(
  ...refs: Array<((value: T | null) => void) | MutableRefObject<T | null> | undefined>
) {
  return (value: T | null) => {
    refs.forEach((ref) => {
      if (!ref) return;
      if (typeof ref === "function") ref(value);
      else ref.current = value;
    });
  };
}

function composeEvent<T extends { defaultPrevented: boolean }>(
  first: ((event: T) => void) | undefined,
  second: (event: T) => void,
) {
  return (event: T) => {
    first?.(event);
    if (!event.defaultPrevented) second(event);
  };
}

export interface TooltipTriggerProps extends HTMLAttributes<HTMLElement> {
  children: ReactNode;
  asChild?: boolean;
}

export function TooltipTrigger({
  children,
  asChild = false,
  ...props
}: TooltipTriggerProps) {
  const context = useTooltipContext();
  const triggerProps = {
    ...props,
    id: context.triggerId,
    ref: mergeRefs<HTMLElement>(
      context.triggerRef,
      (props as { ref?: (value: HTMLElement | null) => void }).ref,
    ),
    "data-rcl-tooltip-trigger": true,
    "aria-describedby": context.open ? context.contentId : undefined,
    onPointerEnter: composeEvent<PointerEvent<HTMLElement>>(
      props.onPointerEnter,
      () => context.scheduleOpen(),
    ),
    onPointerLeave: composeEvent<PointerEvent<HTMLElement>>(
      props.onPointerLeave,
      () => context.scheduleClose(),
    ),
    onFocus: composeEvent<FocusEvent<HTMLElement>>(props.onFocus, () => {
      context.cancelTimers();
      context.setOpen(true);
    }),
    onBlur: composeEvent<FocusEvent<HTMLElement>>(props.onBlur, () =>
      context.scheduleClose(),
    ),
    onKeyDown: composeEvent<KeyboardEvent<HTMLElement>>(props.onKeyDown, (event) => {
      if (event.key === "Escape") {
        context.cancelTimers();
        context.setOpen(false);
      }
    }),
  };
  if (asChild) {
    const child = Children.only(children);
    if (!isValidElement(child))
      throw new Error("TooltipTrigger asChild requires one element child");
    return cloneElement(child as ReactElement<Record<string, unknown>>, triggerProps);
  }
  return createElement("button", { ...triggerProps, type: "button" }, children);
}

export interface TooltipContentProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
}

export function TooltipContent({ children, ...props }: TooltipContentProps) {
  const context = useTooltipContext();
  return (
    <Presence present={context.open} duration="quick" initial={false} as="span">
      <span {...props} id={context.contentId} role="tooltip" data-rcl-tooltip-content>
        {children}
      </span>
    </Presence>
  );
}

export const TooltipParts = { Trigger: TooltipTrigger, Content: TooltipContent };

export default Tooltip;