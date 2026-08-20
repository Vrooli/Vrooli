// provider-free-exception: this suite intentionally verifies that useToast rejects a missing provider.
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ToastContainer } from './ToastContainer';
import { ToastProvider, useToast } from './ToastProvider';
import { StatusIndicator } from './StatusIndicator';
import { ConnectionStatusBanner } from './ConnectionStatusBanner';

function ToastHarness() {
  const { showToast, showApiError } = useToast();
  return <>
    <button type="button" onClick={() => { showToast('info', 'Info message', { autoDismissMs: 0 }); }}>info</button>
    <button type="button" onClick={() => { showToast('success', 'Success message'); }}>success</button>
    <button type="button" onClick={() => { showApiError({ error: 'Needs auth', detail: { recovery: 'authenticate', retryable: false, code: 'unauthorized', message: 'Needs auth' } }); }}>auth</button>
    <button type="button" onClick={() => { showApiError(new Error('ordinary')); }}>ordinary</button>
    <ToastContainer />
  </>;
}

describe('operator feedback surfaces', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('adds, deduplicates, retries, and dismisses toasts', () => {
    const retry = vi.fn();
    function RetryHarness() {
      const { showToast } = useToast();
      return <button type="button" onClick={() => { showToast('warning', 'Retry me', { retryFn: retry, autoDismissMs: 0 }); }}>retry</button>;
    }
    render(<ToastProvider><RetryHarness /><ToastHarness /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'info' }));
    fireEvent.click(screen.getByRole('button', { name: 'info' }));
    expect(screen.getAllByText('Info message')).toHaveLength(1);
    fireEvent.click(screen.getByRole('button', { name: 'auth' }));
    expect(screen.getByText('Needs auth. Please sign in and try again')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'ordinary' }));
    expect(screen.getAllByText('ordinary')).toHaveLength(2);
    fireEvent.click(screen.getByRole('button', { name: 'retry' }));
    fireEvent.click(screen.getByTitle('Retry'));
    expect(retry).toHaveBeenCalledOnce();
    const dismissButtons = screen.getAllByTitle('Dismiss');
    const lastDismiss = dismissButtons[dismissButtons.length - 1];
    if (!lastDismiss) throw new Error('toast dismiss button was not rendered');
    fireEvent.click(lastDismiss);
    expect(screen.queryByText('Retry me')).not.toBeInTheDocument();
  });

  it('auto dismisses timed toasts and rejects useToast outside a provider', () => {
    vi.useFakeTimers();
    render(<ToastProvider><ToastHarness /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'success' }));
    expect(screen.getByText('Success message')).toBeInTheDocument();
    act(() => { vi.advanceTimersByTime(5000); });
    expect(screen.queryByText('Success message')).not.toBeInTheDocument();
    expect(() => render(<ToastHarness />)).toThrow('useToast must be used within a ToastProvider');
    vi.useRealTimers();
  });

  it('shows status details, refreshes, toggles, and handles unavailable health', async () => {
    const toggle = vi.fn().mockResolvedValue(undefined);
    const refresh = vi.fn().mockResolvedValue(undefined);
    render(<StatusIndicator
      healthStatus={{ status: 'healthy', service: 'system-monitor', processor_active: true, maintenance_state: 'active', api_connectivity: { connected: true, latency_ms: 12 }, timestamp: 1770000000 }}
      healthError={null} onToggleMonitoring={toggle} onRefreshHealth={refresh} isLoading={false}
    />);
    fireEvent.click(screen.getByTitle('View status details'));
    expect(screen.getByRole('dialog', { name: 'System status details' })).toBeInTheDocument();
    expect(screen.getByText('Connected · 12ms')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Refresh status' }));
    await waitFor(() => { expect(refresh).toHaveBeenCalledOnce(); });
    fireEvent.click(screen.getByTitle('Pause monitoring'));
    await waitFor(() => { expect(toggle).toHaveBeenCalledOnce(); });
    fireEvent.mouseDown(document.body);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();

    render(<StatusIndicator healthStatus={null} healthError="health unavailable" onToggleMonitoring={toggle} onRefreshHealth={refresh} isLoading={false} />);
    const statusButtons = screen.getAllByTitle('View status details');
    const unavailableStatusButton = statusButtons[1];
    if (!unavailableStatusButton) throw new Error('unavailable status button was not rendered');
    fireEvent.click(unavailableStatusButton);
    expect(screen.getByText('Unavailable')).toBeInTheDocument();
  });

  it('renders stale connection age and calls manual refresh', () => {
    const refresh = vi.fn();
    render(<ConnectionStatusBanner isStale lastSuccessfulFetch={new Date(Date.now() - 65000)} onRefresh={refresh} />);
    expect(screen.getByText(/Data may be outdated/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Refresh Now' }));
    expect(refresh).toHaveBeenCalledOnce();
    const { container } = render(<ConnectionStatusBanner isStale={false} lastSuccessfulFetch={null} onRefresh={refresh} />);
    expect(container).toBeEmptyDOMElement();
  });
});
