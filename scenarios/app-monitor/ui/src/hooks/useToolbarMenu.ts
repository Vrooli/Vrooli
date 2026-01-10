import { useCallback, useEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import { useAnchoredPopover, type PopoverPlacement } from '@/components/popover/AnchoredPopover';

export type MenuId = string;

/**
 * Hook for managing a single toolbar dropdown menu with state, refs, and positioning.
 * Designed to work with mutually-exclusive menu systems.
 */
export interface UseToolbarMenuReturn {
  // State
  isOpen: boolean;

  // Refs
  menuRef: React.RefObject<HTMLDivElement>;
  buttonRef: React.RefObject<HTMLButtonElement>;
  popoverRef: React.RefObject<HTMLDivElement>;
  firstItemRef: React.RefObject<HTMLButtonElement>;

  // Positioning (from anchored popover)
  menuStyle: CSSProperties | undefined;
  placement: PopoverPlacement;

  // Actions
  open: () => void;
  close: () => void;
  toggle: () => void;
}

interface UseToolbarMenuOptions {
  id: MenuId;
  onOpenChange?: (id: MenuId, isOpen: boolean) => void;
  placement?: PopoverPlacement;
}

/**
 * Creates a toolbar menu instance with all necessary state, refs, and positioning logic.
 *
 * @example
 * const lifecycleMenu = useToolbarMenu({
 *   id: 'lifecycle',
 *   onOpenChange: handleMenuOpenChange
 * });
 */
export const useToolbarMenu = ({
  id,
  onOpenChange,
  placement,
}: UseToolbarMenuOptions): UseToolbarMenuReturn => {
  const [isOpen, setIsOpen] = useState(false);

  // Create refs
  const menuRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const firstItemRef = useRef<HTMLButtonElement>(null);

  const { style: menuStyle, placement: resolvedPlacement } = useAnchoredPopover({
    isOpen,
    anchorRef: buttonRef,
    popoverRef,
    placement,
  });

  // Actions
  const open = useCallback(() => {
    setIsOpen(true);
    onOpenChange?.(id, true);
  }, [id, onOpenChange]);

  const close = useCallback(() => {
    setIsOpen(false);
    onOpenChange?.(id, false);
  }, [id, onOpenChange]);

  const toggle = useCallback(() => {
    if (isOpen) {
      close();
    } else {
      open();
    }
  }, [close, isOpen, open]);

  return {
    isOpen,
    menuRef,
    buttonRef,
    popoverRef,
    firstItemRef,
    menuStyle,
    placement: resolvedPlacement,
    open,
    close,
    toggle,
  };
};

/**
 * Hook to coordinate multiple mutually-exclusive menus.
 * When one menu opens, all others close automatically.
 */
export const useMenuCoordinator = () => {
  const [openMenuId, setOpenMenuId] = useState<MenuId | null>(null);
  const closeHandlersRef = useRef(new Map<MenuId, () => void>());

  const registerMenu = useCallback((id: MenuId, closeHandler: () => void) => {
    closeHandlersRef.current.set(id, closeHandler);
    return () => {
      closeHandlersRef.current.delete(id);
    };
  }, []);

  const closeOthers = useCallback((id: MenuId) => {
    closeHandlersRef.current.forEach((handler, handlerId) => {
      if (handlerId !== id) {
        handler();
      }
    });
  }, []);

  const handleMenuOpenChange = useCallback((id: MenuId, isOpen: boolean) => {
    if (isOpen) {
      closeOthers(id);
      setOpenMenuId(id);
    } else if (openMenuId === id) {
      setOpenMenuId(null);
    }
  }, [closeOthers, openMenuId]);

  const closeAll = useCallback(() => {
    closeHandlersRef.current.forEach(handler => handler());
    setOpenMenuId(null);
  }, []);

  return {
    openMenuId,
    handleMenuOpenChange,
    closeAll,
    registerMenu,
  };
};

/**
 * Hook to automatically focus the first menu item when a menu opens.
 * Improves keyboard accessibility.
 */
export const useMenuAutoFocus = (
  isOpen: boolean,
  firstItemRef: React.RefObject<HTMLElement>,
) => {
  useEffect(() => {
    if (isOpen && firstItemRef.current) {
      // Use requestAnimationFrame to ensure DOM is ready
      requestAnimationFrame(() => {
        firstItemRef.current?.focus();
      });
    }
  }, [isOpen, firstItemRef]);
};

/**
 * Hook to handle clicking outside menus to close them.
 * Monitors pointer events and closes menus when clicking outside.
 */
export const useMenuOutsideClick = (
  refs: Array<React.RefObject<HTMLElement | null>>,
  closeCallback: () => void,
  enabled: boolean = true,
) => {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) {
        return;
      }

      // Check if click is inside any of the provided refs
      const isInsideAnyRef = refs.some(ref => {
        return ref.current?.contains(target);
      });

      if (!isInsideAnyRef) {
        closeCallback();
      }
    };

    document.addEventListener('pointerdown', handlePointerDown);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown);
    };
  }, [refs, closeCallback, enabled]);
};

export default useToolbarMenu;
