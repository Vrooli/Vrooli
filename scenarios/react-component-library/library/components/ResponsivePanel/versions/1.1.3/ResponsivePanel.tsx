/**
 * @libraryId react-component-library:ResponsivePanel
 * @displayName ResponsivePanel
 * @description One responsive layered panel contract that preserves open state while moving between desktop and mobile presentations.
 * @version 1.1.3
 * @tags ["overlay","responsive","token-bound"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
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
import { useControllableState } from "../../../../hooks/useControllableState/versions/1.0.0/useControllableState";
import { useFocusTrap } from "../../../../hooks/useFocusTrap/versions/1.0.0/useFocusTrap";
import { useMediaQuery } from "../../../../hooks/useMediaQuery/versions/1.0.0/useMediaQuery";
import { useScrollLock } from "../../../../hooks/useScrollLock/versions/1.0.0/useScrollLock";
import { layerManager } from "../../../../services/LayerManager/versions/1.0.0/LayerManager";
import { responsivePanelStyles } from "./styles";

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

export function ResponsivePanel({
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
      <style
        data-rcl-responsive-panel-styles
        dangerouslySetInnerHTML={{ __html: responsivePanelStyles }}
      />
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
            {children ?? <p>{translate("overlays.responsive-panel.text.1", "Panel content")}</p>}
          </div>
          {resizable && !isMobile ? (
            <input
              data-testid="overlays.responsive-panel"
              type="range"
              data-rcl-responsive-panel-resize
              aria-label={translate("overlays.responsive-panel.aria-label.1", "Resize panel")}
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
}
