import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { render } from '@testing-library/react';
import { LogsPage } from './LogsPage';

vi.mock('../api', () => ({
  fetchLogs: vi.fn(),
  fetchUnits: vi.fn(),
  fetchBoots: vi.fn(),
}));

// react-window relies on a measurable container; jsdom reports 0 width/height
// which makes FixedSizeList render no rows. Mock it to a plain list so we can
// assert "fetch fired" without battling layout primitives.
vi.mock('react-window', () => ({
  VariableSizeList: ({
    itemCount,
    children,
  }: {
    itemCount: number;
    children: (props: { index: number; style: React.CSSProperties }) => React.ReactNode;
  }) => (
    <div data-testid="virtual-list">
      {Array.from({ length: itemCount }, (_, i) => children({ index: i, style: {} }))}
    </div>
  ),
}));

import { fetchBoots, fetchLogs, fetchUnits } from '../api';

describe('LogsPage', () => {
  beforeEach(() => {
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
});
