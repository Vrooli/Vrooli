import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../../lib/utils";

const modalContentVariants = cva(
  "relative flex max-h-[90vh] w-full flex-col overflow-hidden rounded-xl border border-border-default/70 bg-surface-elevated shadow-2xl",
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
        "fixed inset-0 z-50 flex items-center justify-center bg-surface-base/70 p-4 backdrop-blur-sm",
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
