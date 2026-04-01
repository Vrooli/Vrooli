import * as React from "react";
import { X } from "lucide-react";
import { cn } from "../../../lib/utils";

interface DetailModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  /** Content rendered before the title (e.g. status indicator) */
  headerLeft?: React.ReactNode;
  /** Content rendered after the title, before the close button (e.g. action buttons) */
  headerRight?: React.ReactNode;
  children: React.ReactNode;
}

export function DetailModal({ open, onClose, title, headerLeft, headerRight, children }: DetailModalProps) {
  const [isClosing, setIsClosing] = React.useState(false);
  const [shouldRender, setShouldRender] = React.useState(open);
  const closeTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  // Use a ref for the close guard so rapid taps can't bypass it via stale closure
  const isClosingRef = React.useRef(false);

  React.useEffect(() => {
    if (open) {
      setShouldRender(true);
      setIsClosing(false);
      isClosingRef.current = false;
    } else {
      // When parent sets open=false directly, clean up
      if (closeTimerRef.current) {
        clearTimeout(closeTimerRef.current);
        closeTimerRef.current = null;
      }
      setShouldRender(false);
      setIsClosing(false);
      isClosingRef.current = false;
    }
  }, [open]);

  React.useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  const handleClose = React.useCallback(() => {
    if (isClosingRef.current) return; // Guard against double-close (ref avoids stale closure)
    isClosingRef.current = true;
    setIsClosing(true);
    closeTimerRef.current = setTimeout(() => {
      closeTimerRef.current = null;
      setShouldRender(false);
      setIsClosing(false);
      isClosingRef.current = false;
      onClose();
    }, 150);
  }, [onClose]);

  // Escape key handler
  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open && !isClosingRef.current) {
        handleClose();
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, handleClose]);

  if (!shouldRender) return null;

  return (
    <div className="fixed inset-0 z-50 lg:hidden">
      {/* Backdrop */}
      <div
        className={cn(
          "fixed inset-0 bg-black/60 backdrop-blur-sm",
          isClosing ? "animate-fade-out" : "animate-fade-in"
        )}
        onClick={handleClose}
        aria-hidden="true"
      />
      {/* Content - full screen slide up */}
      <div
        className={cn(
          "fixed inset-x-0 bottom-0 top-0 z-50 flex flex-col bg-background",
          isClosing ? "animate-slide-down" : "animate-slide-up"
        )}
      >
        {/* Header */}
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          {headerLeft}
          <h2 className="font-semibold text-base truncate flex-1 min-w-0">
            {title}
          </h2>
          <div className="flex items-center gap-1 shrink-0">
            {headerRight}
            <button
              onClick={handleClose}
              className="rounded-sm p-1 opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>
        {/* Body */}
        <div className="flex-1 overflow-y-auto">
          {children}
        </div>
      </div>
    </div>
  );
}
