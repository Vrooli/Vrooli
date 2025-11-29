import clsx from "clsx";
import { HTMLAttributes, PointerEvent, ReactNode, Ref, useEffect } from "react";
import "./ResponsiveDialog.css";
import { selectors } from "@constants/selectors";

type ResponsiveDialogSize = "default" | "wide" | "xl";

type ResponsiveDialogProps = {
  isOpen: boolean;
  children: ReactNode;
  onDismiss?: () => void;
  ariaLabel?: string;
  ariaLabelledBy?: string;
  size?: ResponsiveDialogSize;
  overlayClassName?: string;
  role?: "dialog" | "alertdialog";
  contentRef?: Ref<HTMLDivElement>;
} & HTMLAttributes<HTMLDivElement>;

const sizeClassMap: Record<ResponsiveDialogSize, string | null> = {
  default: null,
  wide: "responsive-dialog__content--wide",
  xl: "responsive-dialog__content--xl",
};

export default function ResponsiveDialog({
  isOpen,
  children,
  onDismiss,
  ariaLabel,
  ariaLabelledBy,
  size = "default",
  overlayClassName,
  role = "dialog",
  contentRef,
  className,
  ...contentProps
}: ResponsiveDialogProps) {
  // Handle Escape key
  useEffect(() => {
    if (!isOpen || !onDismiss) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onDismiss();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onDismiss]);

  console.log(`[DEBUG] ResponsiveDialog render - isOpen: ${isOpen}`);

  if (!isOpen) {
    console.log("[DEBUG] ResponsiveDialog returning null (not rendering)");
    return null;
  }

  const handleOverlayPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (!onDismiss) {
      return;
    }
    if (event.target !== event.currentTarget) {
      return;
    }
    onDismiss();
  };

  return (
    <div
      className={clsx("responsive-dialog__overlay", overlayClassName)}
      role="presentation"
      onPointerDown={handleOverlayPointerDown}
      data-testid={selectors.dialogs.responsive.overlay}
    >
      <div
        role={role}
        aria-modal="true"
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        ref={contentRef}
        className={clsx(
          "responsive-dialog__content",
          sizeClassMap[size],
          className,
        )}
        data-testid={selectors.dialogs.responsive.content}
        {...contentProps}
      >
        {children}
      </div>
    </div>
  );
}
