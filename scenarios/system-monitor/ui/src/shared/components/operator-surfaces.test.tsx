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
    // Queried by accessible name rather than by title: the button now states
    // the health it is reporting, which is the text alternative for a dot that
    // otherwise carries state in colour alone.
    fireEvent.click(screen.getByRole('button', { name: /System status: .*View status details/i }));
    expect(screen.getByRole('dialog', { name: 'System status details' })).toBeInTheDocument();
    expect(screen.getByText('Connected · 12ms')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Refresh status' }));
    await waitFor(() => { expect(refresh).toHaveBeenCalledOnce(); });
    fireEvent.click(screen.getByTitle('Pause monitoring'));
    await waitFor(() => { expect(toggle).toHaveBeenCalledOnce(); });
    fireEvent.mouseDown(document.body);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();

    render(<StatusIndicator healthStatus={null} healthError="health unavailable" onToggleMonitoring={toggle} onRefreshHealth={refresh} isLoading={false} />);
    const statusButtons = screen.getAllByRole('button', { name: /System status: .*View status details/i });
    const unavailableStatusButton = statusButtons[1];
    if (!unavailableStatusButton) throw new Error('unavailable status button was not rendered');
    fireEvent.click(unavailableStatusButton);
    expect(screen.getByText('Unavailable')).toBeInTheDocument();
  });

  it('names the health it reports in every state, not just when healthy', () => {
    // The dot carries health in colour alone and is aria-hidden, so this
    // accessible name is the ONLY text alternative for the state. Each branch
    // must produce a distinct one.
    const noop = vi.fn().mockResolvedValue(undefined);

    const loading = render(<StatusIndicator
      healthStatus={null} healthError={null}
      onToggleMonitoring={noop} onRefreshHealth={noop} isLoading
    />);
    expect(screen.getByRole('button', { name: /System status: loading/i })).toBeInTheDocument();
    loading.unmount();

    const offline = render(<StatusIndicator
      healthStatus={{ status: 'unhealthy' }} healthError={null}
      onToggleMonitoring={noop} onRefreshHealth={noop} isLoading={false}
    />);
    expect(screen.getByRole('button', { name: /System status: offline/i })).toBeInTheDocument();
    offline.unmount();

    const errored = render(<StatusIndicator
      healthStatus={null} healthError="health unavailable"
      onToggleMonitoring={noop} onRefreshHealth={noop} isLoading={false}
    />);
    expect(screen.getByRole('button', { name: /System status: error/i })).toBeInTheDocument();
    errored.unmount();
  });

  it('states the monitoring toggle as a pressed state and an action, not a bare word', () => {
    // The visible label is the STATE while the control does the OPPOSITE, so
    // the accessible name has to carry both or the button is unusable
    // non-visually.
    const noop = vi.fn().mockResolvedValue(undefined);

    const active = render(<StatusIndicator
      healthStatus={{ status: 'healthy', processor_active: true }} healthError={null}
      onToggleMonitoring={noop} onRefreshHealth={noop} isLoading={false}
    />);
    const pressed = screen.getByRole('button', { name: /Monitoring active\. Pause monitoring/i });
    expect(pressed).toHaveAttribute('aria-pressed', 'true');
    active.unmount();

    render(<StatusIndicator
      healthStatus={{ status: 'healthy', processor_active: false, maintenance_state: 'paused' }} healthError={null}
      onToggleMonitoring={noop} onRefreshHealth={noop} isLoading={false}
    />);
    const unpressed = screen.getByRole('button', { name: /Monitoring inactive\. Activate monitoring/i });
    expect(unpressed).toHaveAttribute('aria-pressed', 'false');
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
