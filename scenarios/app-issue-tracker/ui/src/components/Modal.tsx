import { useEffect, useRef, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import ResponsiveDialog from './dialog/ResponsiveDialog';

type ModalSize = 'compact' | 'default' | 'wide' | 'xl';

interface ModalProps {
  onClose: () => void;
  labelledBy?: string;
  panelClassName?: string;
  children: ReactNode;
  size?: ModalSize;
  ariaLabel?: string;
  role?: 'dialog' | 'alertdialog';
}

export function Modal({
  onClose,
  labelledBy,
  panelClassName,
  children,
  size = 'default',
  ariaLabel,
  role = 'dialog',
}: ModalProps) {
  const contentRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [onClose]);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const { style } = document.body;
    const previousOverflow = style.overflow;
    style.overflow = 'hidden';

    return () => {
      style.overflow = previousOverflow;
    };
  }, []);

  const modalContent = (
    <ResponsiveDialog
      isOpen
      onDismiss={onClose}
      ariaLabelledBy={labelledBy}
      ariaLabel={ariaLabel}
      overlayClassName="modal-backdrop"
      className={`modal-panel${panelClassName ? ` ${panelClassName}` : ''}`}
      contentRef={contentRef}
      size={size}
      role={role}
    >
      {children}
    </ResponsiveDialog>
  );

  return typeof document !== 'undefined'
    ? createPortal(modalContent, document.body)
    : modalContent;
}
