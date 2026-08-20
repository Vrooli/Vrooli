import { act, fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { AlertPanel } from './AlertPanel';
import { ConnectionStatusBanner } from './ConnectionStatusBanner';
import { Terminal } from './Terminal';
import { NotProvisionedCard } from '../../features/forensics/components/NotProvisionedCard';
import { GpuContentionCard } from '../../features/capacity/components/GpuContentionCard';
import { RefreshControl } from '../../features/logs/components/RefreshControl';
import { LogRow } from '../../features/logs/components/LogRow';
import { useTimeRange } from '../time/TimeRangeContext';
import { DetailRow } from './DetailRow';
import { Header } from './Header';

describe('additional operator surfaces', () => {
  it('renders alert, unavailable, GPU contention, refresh, and log variants', () => {
    const refresh = vi.fn();
    const toggle = vi.fn();
    const { rerender } = render(<AlertPanel alerts={[]} />);
    expect(screen.getByText('NO ACTIVE ALERTS')).toBeInTheDocument();
    rerender(<AlertPanel alerts={[{ id: 'a', severity: 'critical', category: 'CPU', message: 'hot', timestamp: '2026-01-01T00:00:00Z' } as never]} />);
    expect(screen.getByText('[CRITICAL] CPU')).toBeInTheDocument();
    expect(screen.getByText('hot')).toBeInTheDocument();

    rerender(<NotProvisionedCard title="Pstore" />);
    expect(screen.getByText('Not provisioned on this host.')).toBeInTheDocument();
    rerender(<NotProvisionedCard title="Pstore" reason="disabled by policy" />);
    expect(screen.getByText('disabled by policy')).toBeInTheDocument();

    rerender(<GpuContentionCard gpu={{ index: 0, name: '', totalBytes: 0, usedBytes: 0, claimedBytes: 0, freeBytes: 0 } as never} />);
    expect(screen.getByText('GPU 0 · unknown')).toBeInTheDocument();
    rerender(<GpuContentionCard gpu={{ index: 1, name: 'A10', totalBytes: 1000, usedBytes: 500, claimedBytes: 1200, freeBytes: 500 } as never} />);
    expect(screen.getByText(/GPU 1 · A10/)).toBeInTheDocument();

    rerender(<RefreshControl paused={false} atTop={false} isLoading={false} onRefresh={refresh} onTogglePause={toggle} />);
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    fireEvent.click(screen.getByRole('button', { name: 'Pause' }));
    expect(screen.getByText('(auto-paused)')).toBeInTheDocument();
    rerender(<RefreshControl paused={false} atTop isLoading onRefresh={refresh} onTogglePause={toggle} />);
    expect(screen.getByRole('button', { name: 'Refreshing…' })).toBeDisabled();

    rerender(<LogRow entry={{ timestamp: '', priority: 7, unit: '', userUnit: 'user', identifier: 'id', message: '', raw: 'raw' } as never} />);
    expect(screen.getByText('user')).toBeInTheDocument();
    expect(screen.getByText('raw')).toBeInTheDocument();
  });

  it('supports time-range changes and terminal output visibility', async () => {
    function Harness() {
      const { range, paused, setRange, setPaused } = useTimeRange();
      return <><span>{range.key}</span><button onClick={() => { setRange('24h'); }}>range</button><button onClick={() => { setRange('missing'); }}>missing</button><button onClick={() => { setPaused(!paused); }}>pause</button></>;
    }
    render(<Harness />);
    expect(screen.getByText('1h')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'range' }));
    expect(screen.getByText('24h')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'missing' }));
    fireEvent.click(screen.getByRole('button', { name: 'pause' }));

    vi.useFakeTimers();
    const onClose = vi.fn();
    render(<Terminal isVisible={false} onClose={onClose} />);
    expect(screen.queryByText('System Output')).not.toBeInTheDocument();
    render(<Terminal isVisible onClose={onClose} />);
    await act(async () => { await vi.advanceTimersByTimeAsync(10000); });
    expect(screen.getAllByText(/\[(DEBUG|INFO)\]/).length).toBeGreaterThan(0);
    vi.useRealTimers();
  });

  it('renders connection age branches', () => {
    const refresh = vi.fn();
    const { rerender } = render(<ConnectionStatusBanner isStale lastSuccessfulFetch={new Date()} onRefresh={refresh} />);
    expect(screen.getByText(/just now/)).toBeInTheDocument();
    rerender(<ConnectionStatusBanner isStale lastSuccessfulFetch={new Date(Date.now() - 3_600_000)} onRefresh={refresh} />);
    expect(screen.getByText(/1 hour ago/)).toBeInTheDocument();
    rerender(<ConnectionStatusBanner isStale lastSuccessfulFetch={new Date(Date.now() - 3_720_000)} onRefresh={refresh} />);
    expect(screen.getByText(/1 hour ago/)).toBeInTheDocument();
    rerender(<ConnectionStatusBanner isStale lastSuccessfulFetch={new Date(Date.now() - 10_000)} onRefresh={refresh} />);
    expect(screen.getByText(/10 seconds ago/)).toBeInTheDocument();
    rerender(<ConnectionStatusBanner isStale lastSuccessfulFetch={new Date(Date.now() - 120_000)} onRefresh={refresh} />);
    expect(screen.getByText(/2 minutes ago/)).toBeInTheDocument();
  });

  it('renders detail row styling variants', () => {
    render(<DetailRow label="Normal" value="42" />);
    expect(screen.getByText('42')).toHaveClass('detail-row-value');
    render(<DetailRow label="Muted" value="—" muted className="extra" valueColor="red" />);
    expect(screen.getByText('—')).toHaveClass('detail-row-value-sm');
    expect(screen.getByText('—')).toHaveStyle({ color: 'rgb(255, 0, 0)' });
  });

  it('exercises header controls and shared range state', () => {
    const callbacks = { stop: vi.fn().mockResolvedValue(undefined), terminal: vi.fn(), settings: vi.fn(), toggle: vi.fn().mockResolvedValue(undefined), refresh: vi.fn().mockResolvedValue(undefined) };
    render(<Header
      unreadErrorCount={2} agents={[]} stoppingAgentIds={new Set()} agentErrors={{}}
      onStopAgent={callbacks.stop} onToggleTerminal={callbacks.terminal} onOpenSettings={callbacks.settings}
      healthStatus={null} healthError="offline" onToggleMonitoring={callbacks.toggle} onRefreshHealth={callbacks.refresh} isLoadingHealth={false}
    />);
    fireEvent.change(screen.getByLabelText('Shared time range'), { target: { value: '24h' } });
    fireEvent.click(screen.getByLabelText('Pause live updates'));
    fireEvent.click(screen.getByTitle(/Switch to light mode/));
    fireEvent.click(screen.getByTitle('Open system settings'));
    fireEvent.click(screen.getByTitle('Toggle system output'));
    expect(callbacks.settings).toHaveBeenCalledOnce();
    expect(callbacks.terminal).toHaveBeenCalledOnce();
    expect(screen.getByText('2')).toBeInTheDocument();
  });
});
