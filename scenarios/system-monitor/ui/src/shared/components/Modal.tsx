import { useCallback, useRef } from 'react';
import type { ReactNode } from 'react';
import { X } from 'lucide-react';
import { useEscapeKey } from '../hooks/useEscapeKey';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: ReactNode;
  /** Additional className for the modal-content container */
  className?: string;
  /** aria-label for the dialog */
  ariaLabel?: string;
}

export const Modal = ({ isOpen, onClose, children, className, ariaLabel }: ModalProps) => {
  const contentRef = useRef<HTMLDivElement>(null);

  useEscapeKey(onClose, isOpen);

  const handleOverlayClick = useCallback((e: React.MouseEvent) => {
    if (contentRef.current && !contentRef.current.contains(e.target as Node)) {
      onClose();
    }
  }, [onClose]);

  if (!isOpen) return null;

  return (
    <div
      className="modal-overlay"
      onClick={handleOverlayClick}
      role="dialog"
      aria-modal="true"
      aria-label={ariaLabel}
    >
      <div
        ref={contentRef}
        className={`modal-content ${className ?? ''}`}
      >
        {children}
      </div>
    </div>
  );
};

interface ModalHeaderProps {
  children: ReactNode;
  onClose: () => void;
}

export const ModalHeader = ({ children, onClose }: ModalHeaderProps) => (
  <div className="modal-header">
    <div style={{ flex: 1 }}>{children}</div>
    <button className="modal-close" onClick={onClose} type="button" aria-label="Close">
      <X size={20} />
    </button>
  </div>
);
