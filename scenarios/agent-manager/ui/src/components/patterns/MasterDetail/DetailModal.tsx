import * as React from "react";
import { X, ChevronLeft } from "lucide-react";
import { cn } from "../../../lib/utils";

interface DetailModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

export function DetailModal({ open, onClose, title, children }: DetailModalProps) {
  const [isClosing, setIsClosing] = React.useState(false);
  const [shouldRender, setShouldRender] = React.useState(open);

  React.useEffect(() => {
    if (open) {
      setShouldRender(true);
      setIsClosing(false);
    }
  }, [open]);

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open && !isClosing) {
        handleClose();
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, isClosing]);

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

  const handleClose = () => {
    setIsClosing(true);
    setTimeout(() => {
      setShouldRender(false);
      setIsClosing(false);
      onClose();
    }, 150);
  };

  const handleAnimationEnd = () => {
    if (isClosing) {
      setShouldRender(false);
      setIsClosing(false);
    }
  };

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
        onAnimationEnd={handleAnimationEnd}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="flex items-center gap-2">
            <button
              onClick={handleClose}
              className="flex items-center gap-1 text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Back to list"
            >
              <ChevronLeft className="h-5 w-5" />
              <span className="text-sm">Back</span>
            </button>
          </div>
          <h2 className="font-semibold text-lg truncate px-4 flex-1 text-center">
            {title}
          </h2>
          <button
            onClick={handleClose}
            className="rounded-sm p-1 opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        {/* Body */}
        <div className="flex-1 overflow-y-auto">
          {children}
        </div>
      </div>
    </div>
  );
}
