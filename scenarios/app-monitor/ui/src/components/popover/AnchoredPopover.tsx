import { createPortal } from 'react-dom';
import type { CSSProperties, ReactNode, RefObject } from 'react';
import type { PopoverPlacement } from './anchoredPopoverUtils';

export type AnchoredPopoverProps = {
  isOpen: boolean;
  portalHost: HTMLElement | null;
  popoverRef: RefObject<HTMLDivElement>;
  style?: CSSProperties;
  placement?: PopoverPlacement;
  className?: string;
  role?: string;
  children: ReactNode;
};

export const AnchoredPopover = ({
  isOpen,
  portalHost,
  popoverRef,
  style,
  placement,
  className,
  role,
  children,
}: AnchoredPopoverProps) => {
  if (!portalHost || !isOpen || !style) {
    return null;
  }

  return createPortal(
    <div
      className={className}
      role={role}
      ref={popoverRef}
      style={style}
      data-placement={placement}
    >
      {children}
    </div>,
    portalHost,
  );
};
