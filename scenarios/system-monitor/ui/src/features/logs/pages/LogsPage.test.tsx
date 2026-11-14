import { describe, expect, it, vi, beforeEach } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { render } from '@testing-library/react';
import { useState } from 'react';
import { TimeRangeProvider, useTimeRange } from '../../../shared/time/TimeRangeContext';
import { clearLogsRouteSnapshot, LogsPage } from './LogsPage';

vi.mock('../api', () => ({
  fetchLogs: vi.fn(),
  fetchUnits: vi.fn(),
  fetchBoots: vi.fn(),
}));
vi.mock('../../../shared/components/JsonEvidenceReader', () => ({
  JsonEvidenceReader: ({ value, onClose }: { value: string; onClose: () => void }) => <section aria-label="Journal metadata"><pre>{value}</pre><button onClick={onClose}>Back</button></section>,
}));

// react-window relies on a measurable container; jsdom reports 0 width/height
// which makes FixedSizeList render no rows. Mock it to a plain list so we can
// assert "fetch fired" without battling layout primitives.
vi.mock('react-window', () => ({
  VariableSizeList: ({
    itemCount,
    children,
    itemData,
  }: {
    itemCount: number;
    itemData: unknown;
    children: (props: { index: number; style: React.CSSProperties; data: unknown }) => React.ReactNode;
  }) => (
    <div data-testid="virtual-list">
      {Array.from({ length: itemCount }, (_, i) => children({ index: i, style: {}, data: itemData }))}
    </div>
  ),
}));

import { fetchBoots, fetchLogs, fetchUnits } from '../api';

describe('LogsPage', () => {
  beforeEach(() => {
    clearLogsRouteSnapshot();
    vi.mocked(fetchLogs).mockReset();
    vi.mocked(fetchUnits).mockReset();
    vi.mocked(fetchBoots).mockReset();

    vi.mocked(fetchUnits).mockResolvedValue({
      available: true,
      units: ['nginx.service', 'sshd.service'],
      generatedAt: '2026-05-07T10:00:00Z',
    });
    vi.mocked(fetchBoots).mockResolvedValue({
      available: true,
      boots: [
        { index: 0, bootId: 'aaaaaaaa-bbbb', firstEntry: '2026-05-07T09:00:00Z', lastEntry: '2026-05-07T10:00:00Z' },
      ],
      generatedAt: '2026-05-07T10:00:00Z',
    });
    vi.mocked(fetchLogs).mockResolvedValue({
      available: true,
      entries: [
        { timestamp: '2026-05-07T10:00:00Z', realtime: 0, priority: 6, message: 'first' },
        { timestamp: '2026-05-07T10:00:01Z', realtime: 1, priority: 4, message: 'second' },
      ],
      generatedAt: '2026-05-07T10:00:00Z',
    });
  });

  it('renders entries from initial fetch and fires on filter change', async () => {
    // provider-free-exception: page uses local fetch adapters and no shared provider context.
    render(<LogsPage />);

    await waitFor(() => { expect(fetchLogs).toHaveBeenCalledTimes(1); });
    expect(screen.getByText('first')).toBeInTheDocument();
    expect(screen.getByText('second')).toBeInTheDocument();

    // Toggle the kernel-only checkbox — should fire a second fetch.
    const kernelCheckbox = screen.getByLabelText(/Kernel only/i);
    fireEvent.click(kernelCheckbox);

    await waitFor(() => { expect(fetchLogs).toHaveBeenCalledTimes(2); });
    const lastCall = vi.mocked(fetchLogs).mock.calls[1]?.[0];
    expect(lastCall?.filters.kernel).toBe(true);
  });

  it('surfaces reason when journal reader unavailable', async () => {
    vi.mocked(fetchLogs).mockResolvedValueOnce({
      available: false,
      reason: 'journal reader not configured',
      generatedAt: '2026-05-07T10:00:00Z',
    });
    // provider-free-exception: page uses local fetch adapters and no shared provider context.
    render(<LogsPage />);
    await waitFor(() =>
      expect(screen.getByText(/journal reader not configured/i)).toBeInTheDocument(),
    );
  });

  it('keeps a malformed-timestamp entry readable and preserves its reported value', async () => {
    vi.mocked(fetchLogs).mockResolvedValueOnce({ available: true, generatedAt: '2026-05-07T10:00:00Z', entries: [
      { timestamp: 'not-a-time', realtime: 1, priority: 4, cursor: 'bad-time', unit: 'systemd', message: 'complete evidence' },
    ] });
    render(<LogsPage />);
    fireEvent.click(await screen.findByRole('button', { name: /Read entry time unrecognized/ }));
    const detail = screen.getByRole('article', { name: 'Selected log entry' });
    expect(detail).toHaveTextContent('Time unrecognized');
    expect(detail).toHaveTextContent('not-a-time');
    expect(detail).toHaveTextContent('complete evidence');
    expect(detail).toHaveTextContent('bad-time');
  });

  it('opens the complete phone reader and restores the list on browser Back', async () => {
    const original = window.matchMedia;
    window.matchMedia = (query) => ({ matches: query.includes('48rem'), media: query, onchange: null, addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn() });
    vi.mocked(fetchLogs).mockResolvedValueOnce({
      available: true, generatedAt: '2026-05-07T10:00:00Z',
      entries: [{ timestamp: '2026-05-07T10:00:00Z', realtime: 1, priority: 4, cursor: 'entry-1', hostname: 'host-a', pid: 42, message: 'Complete message\nsecond line' }],
    });
    render(<LogsPage />);
    fireEvent.click(await screen.findByRole('button', { name: /Read entry/ }));
    expect(screen.getByRole('article', { name: 'Selected log entry' })).toHaveTextContent('Complete message');
    expect(screen.getByText('host-a')).toBeInTheDocument();
    expect(screen.getByText('entry-1')).toBeInTheDocument();
    fireEvent.popState(window);
    await waitFor(() => { expect(screen.queryByRole('article', { name: 'Selected log entry' })).not.toBeInTheDocument(); });
    expect(screen.getByRole('button', { name: /Read entry/ })).toBeInTheDocument();
    window.matchMedia = original;
  });

  it('opens raw metadata from the selected record and returns to the reader', async () => {
    vi.mocked(fetchLogs).mockResolvedValueOnce({ available: true, generatedAt: '2026-05-07T10:00:00Z', entries: [
      { timestamp: '2026-05-07T10:00:00Z', realtime: 1, priority: 4, cursor: 'entry-1', message: 'message', raw: '{"EXTRA":"preserved"}' },
    ] });
    render(<LogsPage />);
    fireEvent.click(await screen.findByRole('button', { name: /Read entry/ }));
    fireEvent.click(screen.getByRole('button', { name: 'Raw metadata' }));
    expect(await screen.findByLabelText('Journal metadata')).toHaveTextContent('preserved');
    fireEvent.popState(window);
    await waitFor(() => { expect(screen.queryByLabelText('Journal metadata')).not.toBeInTheDocument(); });
    expect(screen.getByRole('article', { name: 'Selected log entry' })).toBeInTheDocument();
  });

  it('copies the complete message and reports clipboard failure honestly', async () => {
    const previous = Object.getOwnPropertyDescriptor(navigator, 'clipboard');
    const writeText = vi.fn().mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error('denied'));
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    try {
      render(<LogsPage />);
      fireEvent.click((await screen.findAllByRole('button', { name: /Read entry/ }))[0]);
      fireEvent.click(screen.getByRole('button', { name: 'Copy message' }));
      await screen.findByText('Message copied');
      expect(writeText).toHaveBeenCalledWith('first');
      fireEvent.click(screen.getByRole('button', { name: 'Copy message' }));
      await screen.findByText('Could not copy message');
    } finally {
      if (previous) Object.defineProperty(navigator, 'clipboard', previous);
      else Reflect.deleteProperty(navigator, 'clipboard');
    }
  });

  it('does not attribute a completed copy to a different selected entry', async () => {
    const previous = Object.getOwnPropertyDescriptor(navigator, 'clipboard');
    let finish!: () => void;
    const writeText = vi.fn(() => new Promise<void>(resolve => { finish = resolve; }));
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    try {
      render(<LogsPage />);
      const rows = await screen.findAllByRole('button', { name: /Read entry/ });
      fireEvent.click(rows[0]);
      fireEvent.click(screen.getByRole('button', { name: 'Copy message' }));
      fireEvent.click(rows[1]);
      await act(async () => { finish(); await Promise.resolve(); });
      expect(writeText).toHaveBeenCalledWith('first');
      expect(screen.getByRole('article', { name: 'Selected log entry' })).toHaveTextContent('second');
      expect(screen.queryByText('Message copied')).not.toBeInTheDocument();
    } finally {
      if (previous) Object.defineProperty(navigator, 'clipboard', previous);
      else Reflect.deleteProperty(navigator, 'clipboard');
    }
  });

  it('replaces the previous copy result while a new clipboard write is pending', async () => {
    const previous = Object.getOwnPropertyDescriptor(navigator, 'clipboard');
    let finish!: () => void;
    const writeText = vi.fn().mockResolvedValueOnce(undefined).mockImplementationOnce(() => new Promise<void>(resolve => { finish = resolve; }));
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    try {
      render(<LogsPage />);
      fireEvent.click((await screen.findAllByRole('button', { name: /Read entry/ }))[0]);
      fireEvent.click(screen.getByRole('button', { name: 'Copy message' }));
      await screen.findByText('Message copied');
      fireEvent.click(screen.getByRole('button', { name: 'Copy message' }));
      expect(screen.queryByText('Message copied')).not.toBeInTheDocument();
      expect(screen.getByText('Copying message…')).toBeInTheDocument();
      await act(async () => { finish(); await Promise.resolve(); });
      expect(screen.getByText('Message copied')).toBeInTheDocument();
    } finally {
      if (previous) Object.defineProperty(navigator, 'clipboard', previous);
      else Reflect.deleteProperty(navigator, 'clipboard');
    }
  });

  it('holds automatic updates while a desktop entry is selected', async () => {
    render(<LogsPage />);
    fireEvent.click((await screen.findAllByRole('button', { name: /Read entry/ }))[0]);
    expect(screen.getByText(/Log updates paused/)).toHaveTextContent('Host collection continues');
    expect(screen.getByRole('button', { name: 'Return to latest' })).toBeInTheDocument();
  });

  it('keeps the view paused after page scrolling until return to latest', async () => {
    render(<LogsPage />);
    await screen.findByText('first');
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 20 });
    fireEvent.scroll(window);
    expect(screen.queryByText(/Log updates paused/)).not.toBeInTheDocument();
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 25 });
    fireEvent.scroll(window);
    expect(screen.getByText(/Log updates paused/)).toBeInTheDocument();
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 0 });
    fireEvent.scroll(window);
    expect(screen.getByRole('button', { name: 'Return to latest' })).toBeInTheDocument();
  });

  it('refreshes on global resume at the live edge but respects an explicit Logs pause', async () => {
    const GlobalPause = () => {
      const { paused, setPaused } = useTimeRange();
      return <button onClick={() => { setPaused(!paused); }}>Toggle global pause</button>;
    };
    render(<TimeRangeProvider><GlobalPause /><LogsPage /></TimeRangeProvider>);
    await screen.findByText('first');
    fireEvent.click(screen.getByRole('button', { name: 'Toggle global pause' }));
    fireEvent.click(screen.getByRole('button', { name: 'Toggle global pause' }));
    await waitFor(() => { expect(fetchLogs).toHaveBeenCalledTimes(2); });
    fireEvent.click(screen.getByRole('button', { name: 'Pause log updates' }));
    fireEvent.click(screen.getByRole('button', { name: 'Toggle global pause' }));
    fireEvent.click(screen.getByRole('button', { name: 'Toggle global pause' }));
    expect(fetchLogs).toHaveBeenCalledTimes(2);
    expect(screen.getByRole('button', { name: 'Resume log updates' })).toBeInTheDocument();
  });

  it('does not fetch on global resume when reading away from the live edge', async () => {
    const GlobalPause = () => {
      const { paused, setPaused } = useTimeRange();
      return <button onClick={() => { setPaused(!paused); }}>Toggle global pause</button>;
    };
    render(<TimeRangeProvider><GlobalPause /><LogsPage /></TimeRangeProvider>);
    await screen.findByText('first');
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 100 });
    fireEvent.scroll(window);
    fireEvent.click(screen.getByRole('button', { name: 'Toggle global pause' }));
    fireEvent.click(screen.getByRole('button', { name: 'Toggle global pause' }));
    expect(fetchLogs).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('button', { name: 'Return to latest' })).toBeInTheDocument();
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 0 });
  });

  it('fetches immediately when resuming at the live edge and reports the pending fetch', async () => {
    render(<LogsPage />);
    await screen.findByText('first');
    let finish!: (value: Awaited<ReturnType<typeof fetchLogs>>) => void;
    vi.mocked(fetchLogs).mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    fireEvent.click(screen.getByRole('button', { name: 'Pause log updates' }));
    const resume = screen.getByRole('button', { name: 'Resume log updates' });
    resume.focus();
    fireEvent.click(resume);
    expect(fetchLogs).toHaveBeenCalledTimes(2);
    expect(screen.getByRole('status', { name: '' })).toHaveTextContent('Fetching latest log entries');
    expect(screen.getByRole('button', { name: 'Pause log updates' })).toHaveFocus();
    finish({ available: true, entries: [], generatedAt: '2026-05-07T10:01:00Z' });
    await waitFor(() => expect(screen.queryByText('Fetching latest log entries…')).not.toBeInTheDocument());
  });

  it('does not fetch on resume while the reader is away from the live edge', async () => {
    render(<LogsPage />);
    await screen.findByText('first');
    fireEvent.click(screen.getByRole('button', { name: 'Pause log updates' }));
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 100 });
    fireEvent.scroll(window);
    fireEvent.click(screen.getByRole('button', { name: 'Resume log updates' }));
    expect(fetchLogs).toHaveBeenCalledTimes(1);
    expect(screen.getByText(/Log updates paused/)).toHaveTextContent('Host collection continues');
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 0 });
  });

  it('retains the query, selection and pause when the route is remounted', async () => {
    const first = render(<LogsPage />);
    fireEvent.click((await screen.findAllByRole('button', { name: /Read entry/ }))[0]);
    fireEvent.click(screen.getByRole('button', { name: 'Pause log updates' }));
    const grep = screen.getByLabelText(/Grep/i);
    fireEvent.change(grep, { target: { value: 'dbus' } });
    fireEvent.keyDown(grep, { key: 'Enter', code: 'Enter' });
    await waitFor(() => { expect(vi.mocked(fetchLogs).mock.calls.at(-1)?.[0].filters.grep).toBe('dbus'); });
    await waitFor(() => expect(screen.getByRole('button', { name: 'Refresh' })).toBeEnabled());
    first.unmount();
    const calls = vi.mocked(fetchLogs).mock.calls.length;
    render(<LogsPage />);
    expect(screen.getByLabelText(/Grep/i)).toHaveValue('dbus');
    expect(screen.getByRole('button', { name: 'Resume log updates' })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: 'Selected log entry' })).toHaveTextContent('first');
    expect(fetchLogs).toHaveBeenCalledTimes(calls);
  });

  it('retains explicit pause when the shared range changes while the route is away', async () => {
    const Navigation = () => {
      const [visible, setVisible] = useState(true);
      const { setRange } = useTimeRange();
      return <><button onClick={() => { setVisible(!visible); }}>Switch route</button><button onClick={() => { setRange('6h'); }}>Change range</button>{visible && <LogsPage />}</>;
    };
    render(<TimeRangeProvider><Navigation /></TimeRangeProvider>);
    await screen.findByText('first');
    fireEvent.click(screen.getByRole('button', { name: 'Pause log updates' }));
    fireEvent.click(screen.getByRole('button', { name: 'Switch route' }));
    fireEvent.click(screen.getByRole('button', { name: 'Change range' }));
    fireEvent.click(screen.getByRole('button', { name: 'Switch route' }));
    expect(screen.getByRole('button', { name: 'Resume log updates' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText(/shared time range changed while away/i)).toBeInTheDocument();
    await waitFor(() => { expect(vi.mocked(fetchLogs).mock.calls.at(-1)?.[0].filters.since).toBe('6h ago'); });
  });

  it('retains the loaded older page and cursor across a route remount', async () => {
    vi.mocked(fetchLogs).mockResolvedValueOnce({ available: true, generatedAt: '2026-05-07T10:00:00Z', nextCursor: 'older-cursor', entries: [{ timestamp: '2026-05-07T10:00:00Z', realtime: 1, priority: 4, cursor: 'entry-a', message: 'newer' }] });
    vi.mocked(fetchLogs).mockResolvedValueOnce({ available: true, generatedAt: '2026-05-07T10:00:00Z', nextCursor: 'next-cursor', entries: [{ timestamp: '2026-05-07T09:00:00Z', realtime: 2, priority: 4, cursor: 'entry-b', message: 'older' }] });
    const first = render(<LogsPage />);
    await screen.findByText('newer');
    fireEvent.click(screen.getByRole('button', { name: /Older/ }));
    await screen.findByText('older');
    fireEvent.click(screen.getByRole('button', { name: /Read entry/ }));
    first.unmount();
    render(<LogsPage />);
    expect(screen.getByText(/page 2/)).toBeInTheDocument();
    expect(screen.getByRole('article', { name: 'Selected log entry' })).toHaveTextContent('entry-b');
    expect(fetchLogs).toHaveBeenCalledTimes(2);
  });

  it('restarts an interrupted query on remount and ignores the old response', async () => {
    const first = render(<LogsPage />);
    await screen.findByText('first');
    let finish!: (value: Awaited<ReturnType<typeof fetchLogs>>) => void;
    vi.mocked(fetchLogs).mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    const grep = screen.getByLabelText(/Grep/i);
    fireEvent.change(grep, { target: { value: 'dbus' } });
    fireEvent.keyDown(grep, { key: 'Enter', code: 'Enter' });
    await waitFor(() => { expect(fetchLogs).toHaveBeenCalledTimes(2); });
    first.unmount();
    render(<LogsPage />);
    await waitFor(() => { expect(fetchLogs).toHaveBeenCalledTimes(3); });
    await act(async () => { finish({ available: true, generatedAt: '2026-05-07T10:00:00Z', entries: [{ timestamp: '2026-05-07T10:00:00Z', realtime: 0, priority: 4, message: 'stale result' }] }); await Promise.resolve(); });
    expect(screen.queryByText('stale result')).not.toBeInTheDocument();
    expect(screen.getByLabelText(/Grep/i)).toHaveValue('dbus');
  });
});
