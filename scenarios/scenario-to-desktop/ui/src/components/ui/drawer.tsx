import {
  forwardRef,
  useEffect,
  useRef,
  useState,
  type HTMLAttributes,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";
import { useEscapeDismiss } from "../../hooks/useEscapeDismiss";

interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  side?: "right" | "left";
  /** Extra classes for the sliding panel (e.g. to override max-width). */
  panelClassName?: string;
  children: ReactNode;
}

export function Drawer({
  open,
  onClose,
  title,
  description,
  side = "right",
  panelClassName,
  children,
}: DrawerProps) {
  const [mounted, setMounted] = useState(false);
  const [visible, setVisible] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open) {
      setMounted(true);
      // Trigger transition on next frame
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          setVisible(true);
        });
      });
    } else {
      setVisible(false);
    }
  }, [open]);

  useEscapeDismiss(open, onClose);

  const handleTransitionEnd = () => {
    if (!visible) setMounted(false);
  };

  if (!mounted) return null;

  const translateClass =
    side === "right"
      ? visible
        ? "translate-x-0"
        : "translate-x-full"
      : visible
        ? "translate-x-0"
        : "-translate-x-full";

  return createPortal(
    <div className="fixed inset-0 z-[99999]" aria-modal="true" role="dialog">
      {/* Backdrop */}
      <div
        className={cn(
          "absolute inset-0 bg-black/60 backdrop-blur-sm transition-opacity duration-300",
          visible ? "opacity-100" : "opacity-0",
        )}
        onClick={onClose}
      />
      {/* Panel */}
      <div
        ref={panelRef}
        className={cn(
          "absolute inset-y-0 flex flex-col bg-slate-950 border-slate-800 shadow-2xl transition-transform duration-300 ease-in-out",
          // Mobile: full screen. Desktop: side panel.
          panelClassName ?? "w-full md:max-w-lg",
          side === "right" ? "right-0 border-l" : "left-0 border-r",
          translateClass,
        )}
        onTransitionEnd={handleTransitionEnd}
      >
        {(title || description) && (
          <DrawerHeader>
            <div className="flex items-start justify-between gap-3">
              <div className="space-y-1 min-w-0">
                {title && (
                  <h2 className="text-lg font-semibold text-slate-50 truncate">
                    {title}
                  </h2>
                )}
                {description && (
                  <p className="text-sm text-slate-400">{description}</p>
                )}
              </div>
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg p-1.5 text-slate-400 transition hover:bg-slate-800 hover:text-slate-200"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </DrawerHeader>
        )}
        {children}
      </div>
    </div>,
    document.body,
  );
}

type DrawerSectionProps = HTMLAttributes<HTMLDivElement>;

export const DrawerHeader = forwardRef<HTMLDivElement, DrawerSectionProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("border-b border-slate-800 px-5 py-4", className)}
      {...props}
    />
  ),
);
DrawerHeader.displayName = "DrawerHeader";

export const DrawerBody = forwardRef<HTMLDivElement, DrawerSectionProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("flex-1 overflow-y-auto px-5 py-4", className)}
      {...props}
    />
  ),
);
DrawerBody.displayName = "DrawerBody";

export const DrawerFooter = forwardRef<HTMLDivElement, DrawerSectionProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("border-t border-slate-800 px-5 py-4", className)}
      {...props}
    />
  ),
);
DrawerFooter.displayName = "DrawerFooter";
