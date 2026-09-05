import { describe, expect, it, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { ToastContext, type ToastContextValue } from './ToastContext';
import { useToast } from './useToast';

vi.unmock('./useToast');

const toastContext: ToastContextValue = {
  toasts: [], addToast: () => 'toast-1', removeToast: () => undefined,
  success: () => 'success-1', error: () => 'error-1', warning: () => 'warning-1', info: () => 'info-1',
};

describe('useToast', () => {
  it('returns the complete toast provider contract', () => {
    const wrapper = ({ children }: { children: ReactNode }) => <ToastContext.Provider value={toastContext}>{children}</ToastContext.Provider>;
    const { result } = renderHook(() => useToast(), { wrapper });
    expect(result.current).toBe(toastContext);
    expect(result.current.success('Saved')).toBe('success-1');
  });

  it('fails closed when used outside a toast provider', () => {
    expect(() => renderHook(() => useToast())).toThrow('useToast must be used within a ToastProvider');
  });
});
