import type { ReactNode } from "react";
import { BottomSheet as LibraryBottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";

interface BottomSheetProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  height?: "auto" | "half" | "full";
}

/** Scenario content adapter; the library owns gestures, focus, scrolling and safe areas. */
export function BottomSheet({ isOpen, onClose, title, children, height = "auto" }: BottomSheetProps) {
  if (height === "full") {
    return <FullPageDrawer open={isOpen} onClose={onClose} title={title} closeLabel="Close"
      contentPadding="comfortable">{children}</FullPageDrawer>;
  }
  return <LibraryBottomSheet open={isOpen} onClose={onClose} title={title} closeLabel="Close"
    showCloseButton contentPadding="comfortable"
    panelClassName={height === "half" ? "max-h-[50%]" : undefined}>
    {children}
  </LibraryBottomSheet>;
}

interface BottomSheetActionProps {
  icon?: ReactNode;
  label: string;
  description?: string;
  onClick: () => void;
  variant?: "default" | "danger" | "success";
  disabled?: boolean;
  testId?: string;
}

export function BottomSheetAction({
  icon,
  label,
  description,
  onClick,
  variant = "default",
  disabled = false,
  testId,
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
      data-testid={testId}
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
