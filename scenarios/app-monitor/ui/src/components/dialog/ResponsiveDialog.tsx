import clsx from 'clsx';
import { CSSProperties, HTMLAttributes, MouseEvent as ReactMouseEvent, MutableRefObject, PointerEvent, ReactNode, Ref } from 'react';
import { useDraggablePosition } from '@/hooks/useDraggablePosition';
import './ResponsiveDialog.css';

type ResponsiveDialogSize = 'default' | 'wide' | 'xl';
type ResponsiveDialogMode = 'modal' | 'floating';

type ResponsiveDialogProps = {
  isOpen: boolean;
  children: ReactNode;
  onDismiss?: () => void;
  ariaLabel?: string;
  ariaLabelledBy?: string;
  size?: ResponsiveDialogSize;
  mode?: ResponsiveDialogMode;
  draggable?: boolean;
  dragHandleSelector?: string;
  floatingStorageKey?: string | null;
  floatingDefaultPosition?: { x: number; y: number } | (() => { x: number; y: number } | null);
  floatingMargin?: number;
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
  mode = 'modal',
  draggable = false,
  dragHandleSelector,
  floatingStorageKey = null,
  floatingDefaultPosition = { x: 24, y: 96 },
  floatingMargin = 12,
  overlayClassName,
  role = 'dialog',
  contentRef,
  className,
  onPointerDown,
  onPointerMove,
  onPointerUp,
  onPointerCancel,
  onClickCapture,
  style,
  ...contentProps
}: ResponsiveDialogProps) {
  const isFloating = mode === 'floating';
  const isDraggable = isFloating && draggable;
  const floating = useDraggablePosition({
    isActive: isFloating && isOpen,
    storageKey: floatingStorageKey,
    defaultPosition: floatingDefaultPosition,
    floatingMargin,
  });

  if (!isOpen) {
    return null;
  }

  const handleOverlayPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (isFloating) {
      return;
    }
    if (!onDismiss) {
      return;
    }
    if (event.target !== event.currentTarget) {
      return;
    }
    onDismiss();
  };

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (isDraggable) {
      const target = event.target as HTMLElement | null;
      const hasValidHandle = !dragHandleSelector || Boolean(target?.closest(dragHandleSelector));
      const onInteractiveControl = Boolean(target?.closest('button, a, input, textarea, select'));
      if (hasValidHandle && !onInteractiveControl) {
        floating.pointerHandlers.onPointerDown(event);
      }
    }
    onPointerDown?.(event);
  };

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (isDraggable) {
      floating.pointerHandlers.onPointerMove(event);
    }
    onPointerMove?.(event);
  };

  const handlePointerUp = (event: PointerEvent<HTMLDivElement>) => {
    if (isDraggable) {
      floating.pointerHandlers.onPointerUp(event);
    }
    onPointerUp?.(event);
  };

  const handlePointerCancel = (event: PointerEvent<HTMLDivElement>) => {
    if (isDraggable) {
      floating.pointerHandlers.onPointerCancel(event);
    }
    onPointerCancel?.(event);
  };

  const handleClickCapture = (event: ReactMouseEvent<HTMLDivElement>) => {
    if (isDraggable) {
      floating.handleClickCapture(event);
    }
    onClickCapture?.(event);
  };

  const mergedStyle: CSSProperties = {
    ...(style as CSSProperties | undefined),
    ...(isFloating && floating.floatingStyle ? floating.floatingStyle : {}),
  };

  return (
    <div
      className={clsx(
        'responsive-dialog__overlay',
        isFloating && 'responsive-dialog__overlay--floating',
        overlayClassName,
      )}
      role="presentation"
      onPointerDown={handleOverlayPointerDown}
    >
      <div
        {...contentProps}
        role={role}
        aria-modal={!isFloating}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        ref={(node) => {
          floating.elementRef.current = node;
          if (typeof contentRef === 'function') {
            contentRef(node);
            return;
          }
          if (contentRef && typeof contentRef === 'object') {
            (contentRef as MutableRefObject<HTMLDivElement | null>).current = node;
          }
        }}
        className={clsx(
          'responsive-dialog__content',
          sizeClassMap[size],
          isFloating && 'responsive-dialog__content--floating',
          isDraggable && 'responsive-dialog__content--draggable',
          isDraggable && floating.isDragging && 'responsive-dialog__content--dragging',
          className,
        )}
        style={mergedStyle}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerCancel}
        onClickCapture={handleClickCapture}
      >
        {children}
      </div>
    </div>
  );
}
