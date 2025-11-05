import clsx from 'clsx';
import { HTMLAttributes, PointerEvent, ReactNode, Ref } from 'react';
import './ResponsiveDialog.css';

type ResponsiveDialogSize = 'default' | 'wide' | 'xl';

type ResponsiveDialogProps = {
  isOpen: boolean;
  children: ReactNode;
  onDismiss?: () => void;
  ariaLabel?: string;
  ariaLabelledBy?: string;
  size?: ResponsiveDialogSize;
  overlayClassName?: string;
  role?: 'dialog' | 'alertdialog';
  contentRef?: Ref<HTMLDivElement>;
} & HTMLAttributes<HTMLDivElement>;

const sizeClassMap: Record<ResponsiveDialogSize, string | null> = {
  default: null,
  wide: 'responsive-dialog__content--wide',
  xl: 'responsive-dialog__content--xl',
};

export default function ResponsiveDialog({
  isOpen,
  children,
  onDismiss,
  ariaLabel,
  ariaLabelledBy,
  size = 'default',
  overlayClassName,
  role = 'dialog',
  contentRef,
  className,
  ...contentProps
}: ResponsiveDialogProps) {
  if (!isOpen) {
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

  // Extract data-testid from contentProps if provided, otherwise use default
  const { 'data-testid': dataTestId, ...otherContentProps } = contentProps as Record<string, unknown>;
  const testId = (dataTestId as string | undefined) || 'responsive-dialog-content';

  return (
    <div
      className={clsx('responsive-dialog__overlay', overlayClassName)}
      role="presentation"
      onPointerDown={handleOverlayPointerDown}
      data-testid="responsive-dialog-overlay"
    >
      <div
        role={role}
        aria-modal="true"
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        ref={contentRef}
        className={clsx(
          'responsive-dialog__content',
          sizeClassMap[size],
          className,
        )}
        data-testid={testId}
        {...otherContentProps}
      >
        {children}
      </div>
    </div>
  );
}
