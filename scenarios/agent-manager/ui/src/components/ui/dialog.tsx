import * as React from "react";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";

interface DialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
  contentClassName?: string;
  /** When true, dialog fills the screen on mobile and reverts to centered overlay on sm+ */
  fullScreenMobile?: boolean;
}

export function Dialog({ open, onOpenChange, children, contentClassName, fullScreenMobile }: DialogProps) {
  // Handle escape key
  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open) {
        onOpenChange(false);
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, onOpenChange]);

  // Prevent body scroll when dialog is open
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

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
        aria-hidden="true"
      />
      {/* Content */}
      <div
        className={cn(
          "relative z-50 w-full animate-in fade-in-0 zoom-in-95 duration-200",
          fullScreenMobile ? "mx-0 sm:mx-4" : "mx-4",
          contentClassName
        )}
      >
        {children}
      </div>
    </div>
  );
}

interface DialogContentProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  /** When true, fills the screen on mobile and reverts to standard dialog on sm+ */
  fullScreenMobile?: boolean;
}

export function DialogContent({
  className,
  children,
  fullScreenMobile,
  ...props
}: DialogContentProps) {
  return (
    <div
      className={cn(
        "relative bg-card shadow-xl flex flex-col",
        fullScreenMobile
          ? "h-[100dvh] max-h-none rounded-none border-0 sm:h-auto sm:max-h-[85vh] sm:rounded-lg sm:border sm:border-border"
          : "rounded-lg border border-border max-h-[85vh]",
        "max-w-lg mx-auto",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

interface DialogHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  onClose?: () => void;
}

export function DialogHeader({
  className,
  children,
  onClose,
  ...props
}: DialogHeaderProps) {
  return (
    <div
      className={cn(
        "flex items-start justify-between gap-4 border-b border-border p-4 sm:p-6",
        className
      )}
      {...props}
    >
      <div className="space-y-1.5">{children}</div>
      {onClose && (
        <button
          onClick={onClose}
          className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          aria-label="Close"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}

interface DialogTitleProps extends React.HTMLAttributes<HTMLHeadingElement> {
  children: React.ReactNode;
}

export function DialogTitle({
  className,
  children,
  ...props
}: DialogTitleProps) {
  return (
    <h2
      className={cn("text-lg font-semibold text-foreground", className)}
      {...props}
    >
      {children}
    </h2>
  );
}

interface DialogDescriptionProps
  extends React.HTMLAttributes<HTMLParagraphElement> {
  children: React.ReactNode;
}

export function DialogDescription({
  className,
  children,
  ...props
}: DialogDescriptionProps) {
  return (
    <p
      className={cn("text-sm text-muted-foreground", className)}
      {...props}
    >
      {children}
    </p>
  );
}

interface DialogBodyProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
}

export function DialogBody({
  className,
  children,
  ...props
}: DialogBodyProps) {
  return (
    <div className={cn("p-4 sm:p-6 overflow-y-auto flex-1 min-h-0", className)} {...props}>
      {children}
    </div>
  );
}

interface DialogFooterProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
}

export function DialogFooter({
  className,
  children,
  ...props
}: DialogFooterProps) {
  return (
    <div
      className={cn(
        "flex justify-end gap-2 border-t border-border p-4 sm:p-6",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}
