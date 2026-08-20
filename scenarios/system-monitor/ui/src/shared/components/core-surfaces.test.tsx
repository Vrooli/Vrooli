// provider-free-exception: this suite intentionally verifies that hooks reject missing providers.
import { act, fireEvent, render, screen } from '@testing-library/react';
import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { usePolling } from '../hooks/usePolling';
import { ThemeProvider, useTheme } from '../theme/ThemeProvider';
import { Terminal } from './Terminal';
import { ToastContainer } from './ToastContainer';
import { ToastProvider, useToast } from './ToastProvider';

describe('shared runtime surfaces', () => {
  beforeEach(() => {
    vi.useRealTimers();
    localStorage.clear();
  });

  it('polls only when enabled and backs off failed callbacks', async () => {
    vi.useFakeTimers();
    const callback = vi.fn<() => Promise<void>>().mockResolvedValue(undefined);
    const { unmount } = renderHook(() => { usePolling(callback, 100, true, { enabled: true, maxIntervalMs: 400 }); });
    await act(async () => { await vi.advanceTimersByTimeAsync(100); });
    expect(callback).toHaveBeenCalledOnce();
    callback.mockRejectedValueOnce(new Error('temporary failure'));
    await act(async () => { await vi.advanceTimersByTimeAsync(100); });
    await act(async () => { await vi.advanceTimersByTimeAsync(200); });
    expect(callback).toHaveBeenCalledTimes(3);
    unmount();

    const disabled = vi.fn();
    renderHook(() => { usePolling(disabled, 0, true); });
    renderHook(() => { usePolling(disabled, 100, false); });
    await act(async () => { await vi.advanceTimersByTimeAsync(500); });
    expect(disabled).not.toHaveBeenCalled();
  });

  it('supports theme persistence, toggling, OS preference changes, and guard errors', () => {
    function ThemeHarness() {
      const { theme, toggleTheme, setTheme } = useTheme();
      return <>
        <span>{theme}</span>
        <button type="button" onClick={toggleTheme}>toggle theme</button>
        <button type="button" onClick={() => { setTheme('light'); }}>set light</button>
      </>;
    }
    localStorage.setItem('system-monitor-theme', 'dark');
    render(<ThemeProvider><ThemeHarness /></ThemeProvider>);
    expect(screen.getByText('dark')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'toggle theme' }));
    expect(screen.getByText('light')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'set light' }));
    expect(document.documentElement).toHaveAttribute('data-theme', 'light');

    expect(() => render(<ThemeHarness />)).toThrow('useTheme must be used within ThemeProvider');
  });

  it('falls back to OS preference and honors preference change events', async () => {
    localStorage.setItem('system-monitor-theme', 'invalid');
    let handler: ((event: MediaQueryListEvent) => void) | undefined;
    Object.defineProperty(window, 'matchMedia', { configurable: true, value: () => ({
      matches: true,
      addEventListener: (_name: string, callback: (event: MediaQueryListEvent) => void) => { handler = callback; },
      removeEventListener: vi.fn(),
    }) });
    function ThemeValue() { return <span>{useTheme().theme}</span>; }
    render(<ThemeProvider><ThemeValue /></ThemeProvider>);
    expect(screen.getByText('light')).toBeInTheDocument();
    localStorage.removeItem('system-monitor-theme');
    await act(async () => { handler?.({ matches: true } as MediaQueryListEvent); await Promise.resolve(); });
    expect(screen.getByText('dark')).toBeInTheDocument();
    localStorage.setItem('system-monitor-theme', 'dark');
    act(() => { handler?.({ matches: true } as MediaQueryListEvent); });
    expect(screen.getByText('dark')).toBeInTheDocument();
    vi.unstubAllGlobals();
  });

  it('maps recovery metadata into actionable toast messages', () => {
    function Harness() {
      const { showApiError } = useToast();
      const errors = [
        { error: 'wait', detail: { recovery: 'wait', retryable: true, code: 'unavailable', message: 'wait', request_id: 'request-12345678' } },
        { error: 'input', detail: { recovery: 'fix_input', retryable: false, code: 'validation', message: 'input', field: 'limit' } },
        { error: 'input2', detail: { recovery: 'fix_input', retryable: false, code: 'validation', message: 'input2' } },
        { error: 'config', detail: { recovery: 'check_config', retryable: false, code: 'internal', message: 'config' } },
        { error: 'admin', detail: { recovery: 'contact_admin', retryable: false, code: 'forbidden', message: 'admin' } },
        { error: 'none', detail: { recovery: 'none', retryable: false, code: 'internal', message: 'none' } },
      ];
      return <>
        {errors.map((error, index) => <button key={error.error} type="button" onClick={() => { showApiError(error); }}>error-{index}</button>)}
        <button type="button" onClick={() => { showApiError('ordinary'); }}>ordinary</button>
        <ToastContainer />
      </>;
    }
    render(<ToastProvider><Harness /></ToastProvider>);
    const expected = [
      'ref: 12345678', 'Please correct the "limit" field', 'Please check your input',
      'Check your configuration', 'Contact your administrator', 'contact support', 'An unknown error occurred',
    ];
    for (let index = 0; index < expected.length; index++) {
      fireEvent.click(screen.getByRole('button', { name: index < 6 ? `error-${index}` : 'ordinary' }));
      expect(screen.getByText(new RegExp(expected[index] ?? ''))).toBeInTheDocument();
    }
  });

  it('shows terminal lines, appends periodic output, and closes', async () => {
    vi.useFakeTimers();
    const onClose = vi.fn();
    render(<Terminal isVisible onClose={onClose} />);
    expect(screen.getByText('[SYSTEM] System Monitor initialized')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Close system output' }));
    expect(onClose).toHaveBeenCalledOnce();
    await act(async () => { await vi.advanceTimersByTimeAsync(10000); });
    expect(screen.getAllByText(/\[(DEBUG|INFO)\]/).length).toBeGreaterThan(1);
  });
});
