import { useEffect, useCallback, useRef, type ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";
import { Button } from "./button";

interface DialogProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
  className?: string;
}

export function Dialog({ open, onClose, children, className }: DialogProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  const handleEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        onClose();
      }
    },
    [onClose]
  );

  const handleOverlayClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === overlayRef.current) {
        onClose();
      }
    },
    [onClose]
  );

  useEffect(() => {
    if (open) {
      // Use capture phase to intercept Escape before other handlers
      document.addEventListener("keydown", handleEscape, { capture: true });
      document.body.style.overflow = "hidden";
    }
    return () => {
      document.removeEventListener("keydown", handleEscape, { capture: true });
      document.body.style.overflow = "";
    };
  }, [open, handleEscape]);

  if (!open) return null;

  return (
    <div
      ref={overlayRef}
      onClick={handleOverlayClick}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-in fade-in-0 duration-150"
      data-testid="dialog-overlay"
    >
      <div
        className={cn(
          "relative flex flex-col bg-slate-900 border border-white/10 rounded-xl shadow-2xl w-full max-w-md mx-4 max-h-[90vh] animate-in zoom-in-95 duration-150",
          className
        )}
        data-testid="dialog-content"
      >
        {children}
      </div>
    </div>
  );
}

interface DialogHeaderProps {
  children: ReactNode;
  onClose?: () => void;
}

export function DialogHeader({ children, onClose }: DialogHeaderProps) {
  return (
    <div className="flex-shrink-0 flex items-center justify-between p-4 border-b border-white/10">
      <h2 className="text-lg font-semibold text-white">{children}</h2>
      {onClose && (
        <Button
          variant="ghost"
          size="icon"
          onClick={onClose}
          className="text-slate-400 hover:text-white"
          data-testid="dialog-close-button"
        >
          <X className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
}

interface DialogBodyProps {
  children: ReactNode;
  className?: string;
  onKeyDown?: (e: React.KeyboardEvent) => void;
}

export function DialogBody({ children, className, onKeyDown }: DialogBodyProps) {
  return (
    <div className={cn("flex-1 overflow-y-auto p-4", className)} onKeyDown={onKeyDown}>
      {children}
    </div>
  );
}

export function DialogFooter({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("flex-shrink-0 flex items-center justify-end gap-2 p-4 border-t border-white/10", className)}>
      {children}
    </div>
  );
}
