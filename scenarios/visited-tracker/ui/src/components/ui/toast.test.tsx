import { act, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ToastProvider, useToast } from './toast';

function Actions() {
  const { showToast } = useToast();
  return <button onClick={() => { showToast('Saved', 'success'); showToast('Retry required', 'error'); }}>Notify</button>;
}

describe('notification lifecycle', () => {
  afterEach(() => vi.useRealTimers());

  it('dismisses one of simultaneous notifications without removing the other', () => {
    vi.useFakeTimers();
    render(<ToastProvider><Actions /></ToastProvider>);
    fireEvent.click(screen.getByText('Notify'));
    const saved = screen.getByText('Saved').closest('[role="alert"]') as HTMLElement;
    fireEvent.click(within(saved).getByRole('button', { name: 'Dismiss notification' }));
    expect(screen.queryByText('Saved')).not.toBeInTheDocument();
    expect(screen.getByText('Retry required')).toBeInTheDocument();
    expect(vi.getTimerCount()).toBe(1);
    act(() => vi.advanceTimersByTime(5000));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(vi.getTimerCount()).toBe(0);
  });

  it('releases outstanding timers when its provider unmounts', () => {
    vi.useFakeTimers();
    const view = render(<ToastProvider><Actions /></ToastProvider>);
    fireEvent.click(screen.getByText('Notify'));
    expect(vi.getTimerCount()).toBe(2);
    view.unmount();
    expect(vi.getTimerCount()).toBe(0);
  });
});
