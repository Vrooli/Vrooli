import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../../lib/utils";

const modalContentVariants = cva(
  "relative flex max-h-[calc(100dvh-1rem)] w-full max-w-[calc(100vw-1rem)] flex-col overflow-hidden rounded-lg border border-border-default/70 bg-surface-elevated shadow-2xl sm:max-h-[90vh] sm:max-w-[calc(100vw-2rem)] sm:rounded-xl",
  {
    variants: {
      size: {
        sm: "max-w-lg",
        md: "max-w-2xl",
        lg: "max-w-3xl",
        xl: "max-w-5xl",
      },
    },
    defaultVariants: {
      size: "md",
    },
  }
);

interface ModalOverlayProps extends React.HTMLAttributes<HTMLDivElement> {
  onDismiss?: () => void;
}

export function ModalOverlay({ className, onDismiss, onClick, ...props }: ModalOverlayProps) {
  return (
    <div
      className={cn(
        "fixed inset-0 z-50 flex items-end justify-center bg-surface-base/70 p-2 pb-[max(env(safe-area-inset-bottom),0.5rem)] pt-[max(env(safe-area-inset-top),0.5rem)] backdrop-blur-sm sm:items-center sm:p-4",
        className
      )}
      onClick={(event) => {
        onClick?.(event);
        if (event.defaultPrevented) return;
        if (event.target === event.currentTarget) onDismiss?.();
      }}
      {...props}
    />
  );
}

export interface ModalContentProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof modalContentVariants> {}

export function ModalContent({ className, size, ...props }: ModalContentProps) {
  return <div className={cn(modalContentVariants({ size, className }))} {...props} />;
}
