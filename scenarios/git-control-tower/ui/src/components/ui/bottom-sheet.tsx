import { useEffect, useRef, useCallback, type ReactNode } from "react";
import { X } from "lucide-react";

interface BottomSheetProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  /** Height preset: "auto" fits content, "half" is 50vh, "full" is 100vh minus safe area */
  height?: "auto" | "half" | "full";
}

export function BottomSheet({
  isOpen,
  onClose,
  title,
  children,
  height = "auto"
}: BottomSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const startYRef = useRef<number>(0);
  const currentYRef = useRef<number>(0);

  // Handle escape key
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  // Prevent body scroll when open
  useEffect(() => {
    if (!isOpen) return;

    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [isOpen]);

  // Handle touch gestures for swipe-to-dismiss
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    startYRef.current = e.touches[0].clientY;
    currentYRef.current = 0;
  }, []);

  const handleTouchMove = useCallback((e: React.TouchEvent) => {
    const deltaY = e.touches[0].clientY - startYRef.current;
    // Only allow dragging down
    if (deltaY > 0 && sheetRef.current) {
      currentYRef.current = deltaY;
      sheetRef.current.style.transform = `translateY(${deltaY}px)`;
    }
  }, []);

  const handleTouchEnd = useCallback(() => {
    if (!sheetRef.current) return;

    // If dragged more than 100px down, close the sheet
    if (currentYRef.current > 100) {
      onClose();
    } else {
      // Snap back to original position
      sheetRef.current.style.transform = "translateY(0)";
    }
    currentYRef.current = 0;
  }, [onClose]);

  if (!isOpen) return null;

  const heightClass =
    height === "full"
      ? "h-[calc(100vh-env(safe-area-inset-top))]"
      : height === "half"
        ? "max-h-[50vh]"
        : "max-h-[85vh]";

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-200"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Sheet */}
      <div
        ref={sheetRef}
        className={`fixed inset-x-0 bottom-0 z-50 flex flex-col rounded-t-2xl border-t border-slate-800 bg-slate-950 shadow-xl animate-in slide-in-from-bottom duration-300 ${heightClass}`}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        {/* Handle bar for dragging */}
        <div className="flex justify-center pt-3 pb-2">
          <div className="h-1.5 w-12 rounded-full bg-slate-700" />
        </div>

        {/* Header */}
        {title && (
          <div className="flex items-center justify-between px-4 pb-3 border-b border-slate-800">
            <h2 className="text-sm font-semibold text-slate-100">{title}</h2>
            <button
              type="button"
              onClick={onClose}
              className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700"
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto overscroll-contain px-4 py-4 pb-[env(safe-area-inset-bottom)]">
          {children}
        </div>
      </div>
    </>
  );
}

interface BottomSheetActionProps {
  icon?: ReactNode;
  label: string;
  description?: string;
  onClick: () => void;
  variant?: "default" | "danger" | "success";
  disabled?: boolean;
}

export function BottomSheetAction({
  icon,
  label,
  description,
  onClick,
  variant = "default",
  disabled = false
}: BottomSheetActionProps) {
  const variantClasses = {
    default: "text-slate-200 hover:bg-slate-800/60 active:bg-slate-700",
    danger: "text-red-300 hover:bg-red-950/40 active:bg-red-900/40",
    success: "text-emerald-300 hover:bg-emerald-950/40 active:bg-emerald-900/40"
  };

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`w-full flex items-center gap-4 px-4 py-4 rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${variantClasses[variant]}`}
    >
      {icon && (
        <span className="flex-shrink-0 h-10 w-10 flex items-center justify-center rounded-full bg-slate-800/60">
          {icon}
        </span>
      )}
      <div className="flex-1 text-left">
        <div className="text-sm font-medium">{label}</div>
        {description && (
          <div className="text-xs text-slate-500 mt-0.5">{description}</div>
        )}
      </div>
    </button>
  );
}
