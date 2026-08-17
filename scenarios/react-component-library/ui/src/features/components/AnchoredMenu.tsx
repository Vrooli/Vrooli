/** @vrooliComponentSource overlays.popover */
import {
  type CSSProperties,
  type ReactNode,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { ChevronDown } from "lucide-react";

import { cn } from "../../lib/utils";

interface AnchoredMenuProps {
  label: string;
  summary?: string;
  children: ReactNode;
  triggerTestId: string;
  panelTestId: string;
  icon?: ReactNode;
  className?: string;
  compactOnMobile?: boolean;
}

/**
 * A compact, dependency-free disclosure surface for preview controls.
 *
 * This is deliberately a dialog rather than an ARIA menu: its contents are
 * form fields, radios, sliders, and buttons rather than a command list. It
 * keeps the keyboard contract small and predictable: trigger to open, focus
 * enters the panel, Escape/outside click closes it, and focus returns to the
 * trigger. The panel is anchored so it stays useful inside a resizable pane.
 */
export function AnchoredMenu({
  label,
  summary,
  children,
  triggerTestId,
  panelTestId,
  icon,
  className,
  compactOnMobile = false,
}: AnchoredMenuProps) {
  const [open, setOpen] = useState(false);
  const id = useId();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const panelRef = useRef<HTMLDivElement | null>(null);
  const [panelStyle, setPanelStyle] = useState<CSSProperties>();

  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return undefined;

    const updatePosition = () => {
      const trigger = triggerRef.current?.getBoundingClientRect();
      if (!trigger) return;
      const viewportPadding = 12;
      const panelWidth = Math.min(320, window.innerWidth - viewportPadding * 2);
      const maxHeight = Math.min(384, window.innerHeight - viewportPadding * 2);
      const left = Math.max(
        viewportPadding,
        Math.min(trigger.left, window.innerWidth - panelWidth - viewportPadding),
      );
      const spaceBelow = window.innerHeight - trigger.bottom - viewportPadding;
      const top =
        spaceBelow >= Math.min(maxHeight, 320)
          ? trigger.bottom + 8
          : Math.max(viewportPadding, trigger.top - maxHeight - 8);
      setPanelStyle({ left, top, width: panelWidth, maxHeight });
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [open]);

  useEffect(() => {
    if (!open) return undefined;
    const onPointerDown = (event: PointerEvent) => {
      if (
        rootRef.current?.contains(event.target as Node) ||
        panelRef.current?.contains(event.target as Node)
      )
        return;
      setOpen(false);
      triggerRef.current?.focus();
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setOpen(false);
      triggerRef.current?.focus();
    };
    window.addEventListener("pointerdown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    const frame = window.requestAnimationFrame(() => {
      panelRef.current
        ?.querySelector<HTMLElement>(
          "button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled])",
        )
        ?.focus();
    });
    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("pointerdown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  return (
    <div ref={rootRef} className={cn("relative min-w-0", className)}>
      <button
        ref={triggerRef}
        type="button"
        data-testid={triggerTestId}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-controls={id}
        className={cn(
          "touch-target inline-flex h-11 min-h-11 max-w-full items-center justify-center gap-space-2xs rounded-control border border-app-border bg-app-surface px-space-xs text-xs font-medium text-app-foreground transition hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
          compactOnMobile && "max-sm:w-11 max-sm:px-0",
        )}
        onClick={() => setOpen((current) => !current)}
      >
        {icon}
        <span className={cn("truncate", compactOnMobile && "max-sm:hidden")}>{label}</span>
        {summary && (
          <span
            className={cn(
              "max-w-28 break-words text-app-muted-foreground",
              compactOnMobile && "max-sm:hidden",
            )}
          >
            {summary}
          </span>
        )}
        <ChevronDown
          aria-hidden
          className={cn(
            "h-3.5 w-3.5 shrink-0 transition-transform",
            compactOnMobile && "max-sm:hidden",
            open && "rotate-180",
          )}
        />
      </button>
      {open &&
        typeof document !== "undefined" &&
        createPortal(
          <div
            ref={panelRef}
            id={id}
            data-testid={panelTestId}
            role="dialog"
            aria-label={label}
            style={panelStyle}
            className="fixed z-[var(--layer-toast)] overflow-y-auto rounded-panel border border-app-border bg-app-surface p-space-xs text-app-foreground shadow-2xl"
          >
            {children}
          </div>,
          document.body,
        )}
    </div>
  );
}
