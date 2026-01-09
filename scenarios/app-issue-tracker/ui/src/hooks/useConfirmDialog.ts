import { useMemo } from 'react';

/**
 * Hook to manage confirmation dialogs with touch device bypass support.
 * On touch devices (coarse pointers), confirmations are automatically bypassed.
 */
export function useConfirmDialog() {
  const shouldBypassConfirm = useMemo(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return false;
    }
    return window.matchMedia('(pointer: coarse)').matches;
  }, []);

  const confirm = (message: string): boolean => {
    if (shouldBypassConfirm) {
      return true;
    }
    if (typeof window !== 'undefined' && typeof window.confirm === 'function') {
      return window.confirm(message);
    }
    return true;
  };

  return { confirm, shouldBypassConfirm };
}
