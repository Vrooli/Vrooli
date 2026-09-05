/**
 * @libraryId react-component-library:ResponsivePanel
 * @displayName ResponsivePanel
 * @version 1.1.5
 * @tags ["overlay","responsive","token-bound"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
  type RefObject,
} from "react";
import { useControllableState } from "@vrooli/react-component-library/useControllableState/1";
import { useFocusTrap } from "@vrooli/react-component-library/useFocusTrap/1";
import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1";
import { useScrollLock } from "@vrooli/react-component-library/useScrollLock/2";
import { layerManager } from "@vrooli/react-component-library/LayerManager/2";
export const responsivePanelStyles = `
[data-rcl-responsive-panel-root] { position: relative; min-block-size: 0; min-inline-size: 0; }
[data-rcl-responsive-panel] { position: relative; display: flex; min-block-size: 0; min-inline-size: 0; flex-direction: column; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-raised); }
[data-rcl-responsive-panel][data-mobile="false"] { block-size: 100%; inline-size: var(--rcl-responsive-panel-width, var(--sidebar-width)); }
[data-rcl-responsive-panel][data-mobile="true"] { position: fixed; inset-block-start: env(safe-area-inset-top); inset-block-end: 0; inset-inline-start: 0; z-index: var(--layer-modal); inline-size: min(var(--rcl-responsive-panel-width, var(--sidebar-width)), calc(100vw - (var(--space-md) * 2))); max-inline-size: 100%; border-block-start: 0; border-inline-start: 0; border-block-end: 0; border-radius: 0 var(--radius-sheet) var(--radius-sheet) 0; box-shadow: var(--elev-modal); animation: rcl-responsive-panel-enter var(--dur-moderate) var(--ease-enter) both; }
[data-rcl-responsive-panel-backdrop] { position: fixed; inset: 0; z-index: calc(var(--layer-modal) - 1); border: 0; background: color-mix(in srgb, var(--color-shell) var(--opacity-scrim), transparent); cursor: default; }
[data-rcl-responsive-panel-header] { display: flex; min-block-size: var(--tap-target-min); min-inline-size: 0; flex-shrink: 0; align-items: flex-start; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm); padding-block-start: max(var(--space-sm), env(safe-area-inset-top)); }
[data-rcl-responsive-panel-heading] { min-inline-size: 0; flex: 1; }
[data-rcl-responsive-panel-heading] h2 { overflow-wrap: anywhere; margin: 0; color: var(--color-foreground); font-size: var(--text-heading-size); font-weight: 700; line-height: var(--text-heading-line); letter-spacing: -0.01em; }
[data-rcl-responsive-panel-heading] p { overflow-wrap: anywhere; margin: var(--space-3xs) 0 0; color: var(--color-muted-foreground); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-responsive-panel-close] { display: inline-grid; flex: 0 0 auto; place-items: center; inline-size: var(--tap-target-min); block-size: var(--tap-target-min); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; font: 700 var(--text-heading-size)/1 var(--font-sans); }
[data-rcl-responsive-panel-close]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-responsive-panel-content] { min-block-size: 0; min-inline-size: 0; flex: 1; overflow: auto; overscroll-behavior: contain; padding: var(--space-md); }
[data-rcl-responsive-panel-resize] { position: absolute; inset-block: 0; inset-inline-end: calc(var(--space-3xs) * -1); z-index: var(--layer-sticky); inline-size: var(--space-xs); block-size: 100%; appearance: none; border: 0; background: transparent; padding: 0; cursor: col-resize; touch-action: none; writing-mode: vertical-lr; }
[data-rcl-responsive-panel-resize]::-webkit-slider-runnable-track { inline-size: var(--space-3xs); block-size: 100%; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]::-webkit-slider-thumb { inline-size: var(--space-3xs); block-size: var(--space-lg); appearance: none; border: 0; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]::-moz-range-track { inline-size: var(--space-3xs); block-size: 100%; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]::-moz-range-thumb { inline-size: var(--space-3xs); block-size: var(--space-lg); border: 0; border-radius: var(--radius-pill); background: transparent; }
[data-rcl-responsive-panel-resize]:hover { background: color-mix(in srgb, var(--color-primary) 20%, transparent); }
@keyframes rcl-responsive-panel-enter { from { opacity: 0; transform: translateX(-100%); } to { opacity: 1; transform: translateX(0); } }
`;
export interface ResponsivePanelProps {
  children?: ReactNode;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
  title?: ReactNode;
  description?: ReactNode;
  ariaLabel?: string;
  closeLabel?: string;
  width?: number;
  defaultWidth?: number;
  minWidth?: number;
  maxWidth?: number;
  widthStorageKey?: string;
  onWidthChange?: (width: number) => void;
  resizable?: boolean;
  restoreFocus?: boolean;
  triggerRef?: RefObject<HTMLElement | null>;
  initialFocusRef?: RefObject<HTMLElement | null>;
  className?: string;
  panelClassName?: string;
  contentClassName?: string;
}

const readTokenPixels = (name: string, fallback: number) => {
  if (typeof document === "undefined") return fallback;
  const raw = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  const value = Number.parseFloat(raw);
  if (raw.endsWith("rem")) {
    const rootSize = Number.parseFloat(getComputedStyle(document.documentElement).fontSize);
    return Number.isFinite(rootSize) ? value * rootSize : fallback;
  }
  return Number.isFinite(value) ? value : fallback;
};

const readStoredWidth = (key: string | undefined) => {
  if (!key || typeof window === "undefined") return undefined;
  try {
    const stored = Number.parseFloat(window.localStorage.getItem(key) ?? "");
    return Number.isFinite(stored) ? stored : undefined;
  } catch {
    return undefined;
  }
};

export const ResponsivePanel = withClassName(function ResponsivePanel({
  children,
  open: controlledOpen,
  defaultOpen = false,
  onOpenChange,
  onClose,
  title,
  description,
  ariaLabel = "Responsive panel",
  closeLabel = "Close panel",
  width: controlledWidth,
  defaultWidth,
  minWidth,
  maxWidth,
  widthStorageKey = "rcl-responsive-panel-width",
  onWidthChange,
  resizable = true,
  restoreFocus = true,
  triggerRef,
  initialFocusRef,
  className,
  panelClassName,
  contentClassName,
}: ResponsivePanelProps) {
  const strings = useStrings();
  const [open, setOpen] = useControllableState({
    value: controlledOpen,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });
  const [uncontrolledWidth, setUncontrolledWidth] = useState<number | undefined>(
    () => readStoredWidth(widthStorageKey) ?? defaultWidth,
  );
  const isMobile = useMediaQuery("(max-width: 47.999rem)");
  const panelRef = useRef<HTMLElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const wasOpenRef = useRef(false);
  const wasMobileModalRef = useRef(false);
  const id = useId().replace(/:/g, "");
  const titleId = title ? `rcl-responsive-panel-${id}-title` : undefined;
  const descriptionId = description ? `rcl-responsive-panel-${id}-description` : undefined;
  const layerId = useRef(`responsive-panel-${id}`);

  const width = controlledWidth ?? uncontrolledWidth;
  const minimumWidth = minWidth ?? readTokenPixels("--sidebar-min-width", 260);
  const maximumWidth = maxWidth ?? readTokenPixels("--sidebar-max-width", 480);
  const currentWidth = width ?? readTokenPixels("--sidebar-width", 320);

  const setWidth = useCallback(
    (nextWidth: number) => {
      const next = Math.round(Math.min(maximumWidth, Math.max(minimumWidth, nextWidth)));
      if (controlledWidth === undefined) setUncontrolledWidth(next);
      onWidthChange?.(next);
    },
    [controlledWidth, maximumWidth, minimumWidth, onWidthChange],
  );

  const close = useCallback(() => {
    setOpen(false);
    onClose?.();
  }, [onClose, setOpen]);

  useFocusTrap(open && isMobile, panelRef);
  useScrollLock(open && isMobile);

  useEffect(() => {
    if (controlledWidth !== undefined || !widthStorageKey || width === undefined) return;
    try {
      window.localStorage.setItem(widthStorageKey, String(width));
    } catch {
      // Storage is an enhancement; private browsing and embedded previews may reject it.
    }
  }, [controlledWidth, width, widthStorageKey]);

  useEffect(() => {
    if (!open || !isMobile) return;
    const removeLayer = layerManager.push({
      id: layerId.current,
      kind: "responsive-panel",
      dismiss: close,
    });
    const onKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape" && layerManager.top()?.id === layerId.current) {
        event.preventDefault();
        close();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => {
      removeLayer();
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [close, isMobile, open]);

  useEffect(() => {
    if (open && !wasOpenRef.current) {
      previousFocusRef.current = document.activeElement as HTMLElement | null;
      wasOpenRef.current = true;
    }
    if (!open && wasOpenRef.current) {
      wasOpenRef.current = false;
      if (restoreFocus) {
        (triggerRef?.current ?? previousFocusRef.current)?.focus();
      }
    }
  }, [open, restoreFocus, triggerRef]);

  useEffect(() => {
    if (!open || !isMobile || wasMobileModalRef.current) return;
    wasMobileModalRef.current = true;
    const frame = window.requestAnimationFrame(() => {
      (initialFocusRef?.current ?? closeRef.current ?? panelRef.current)?.focus();
    });
    return () => window.cancelAnimationFrame(frame);
  }, [initialFocusRef, isMobile, open]);

  useEffect(() => {
    if (isMobile) return;
    wasMobileModalRef.current = false;
  }, [isMobile]);

  if (!open) return null;

  const panelStyle = {
    "--rcl-responsive-panel-width": width ? `${width}px` : undefined,
  } as CSSProperties;

  return (
    <>
      <StyleSheet name="responsivepanel-1-1-4-1" css={responsivePanelStyles} />
      <div
        data-rcl-responsive-panel-root
        data-mobile={isMobile ? "true" : "false"}
        data-open="true"
        className={className}
      >
        {isMobile && (
          <button
            data-testid="overlays.responsive-panel"
            type="button"
            data-rcl-responsive-panel-backdrop
            aria-label={closeLabel}
            onClick={close}
          />
        )}
        <section
          ref={panelRef}
          data-rcl-responsive-panel
          data-mobile={isMobile ? "true" : "false"}
          data-open="true"
          role={isMobile ? "dialog" : "complementary"}
          aria-modal={isMobile ? "true" : undefined}
          aria-label={title ? undefined : ariaLabel}
          aria-labelledby={titleId}
          aria-describedby={descriptionId}
          style={panelStyle}
          className={panelClassName}
        >
          <header data-rcl-responsive-panel-header>
            <div data-rcl-responsive-panel-heading>
              {title ? <h2 id={titleId}>{title}</h2> : null}
              {description ? <p id={descriptionId}>{description}</p> : null}
            </div>
            <button
              data-testid="overlays.responsive-panel"
              ref={closeRef}
              type="button"
              data-rcl-responsive-panel-close
              aria-label={closeLabel}
              onClick={close}
            >
              <span aria-hidden="true">×</span>
            </button>
          </header>
          <div data-rcl-responsive-panel-content className={contentClassName}>
            {children ?? (
              <p>{strings("overlays.responsive-panel.panel-content", "Panel content")}</p>
            )}
          </div>
          {resizable && !isMobile ? (
            <input
              data-testid="overlays.responsive-panel"
              type="range"
              data-rcl-responsive-panel-resize
              aria-label={strings("overlays.responsive-panel.resize-panel", "Resize panel")}
              aria-orientation="vertical"
              min={minimumWidth}
              max={maximumWidth}
              step={1}
              value={currentWidth}
              onChange={(event) => setWidth(event.currentTarget.valueAsNumber)}
            />
          ) : null}
        </section>
      </div>
    </>
  );
});
