/** @vrooliComponentSource overlays.popover */
import { type ReactNode, useEffect, useId, useRef, useState } from "react";
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
}: AnchoredMenuProps) {
  const [open, setOpen] = useState(false);
  const id = useId();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const panelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return undefined;
    const onPointerDown = (event: PointerEvent) => {
      if (rootRef.current?.contains(event.target as Node)) return;
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
        className="touch-target inline-flex h-11 min-h-11 max-w-full items-center justify-center gap-space-2xs rounded-control border border-app-border bg-app-surface px-space-xs text-xs font-medium text-app-foreground transition hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
        onClick={() => setOpen((current) => !current)}
      >
        {icon}
        <span className="truncate">{label}</span>
        {summary && (
          <span className="max-w-28 break-words text-app-muted-foreground">{summary}</span>
        )}
        <ChevronDown
          aria-hidden
          className={cn("h-3.5 w-3.5 shrink-0 transition-transform", open && "rotate-180")}
        />
      </button>
      {open && (
        <div
          ref={panelRef}
          id={id}
          data-testid={panelTestId}
          role="dialog"
          aria-label={label}
          className="absolute left-0 top-full z-40 mt-space-3xs max-h-96 w-80 overflow-y-auto rounded-md border border-app-border bg-app-surface p-space-xs text-app-foreground shadow-xl"
        >
          {children}
        </div>
      )}
    </div>
  );
}
