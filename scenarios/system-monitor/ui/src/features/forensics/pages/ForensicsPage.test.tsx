import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { ForensicsPage } from './ForensicsPage';
import type { ForensicsSummary } from '../types';

vi.mock('../api', () => ({
  fetchForensicsSummary: vi.fn(),
}));

import { fetchForensicsSummary } from '../api';

const happy: ForensicsSummary = {
  generatedAt: '2026-05-07T10:00:00Z',
  pstore: {
    available: true,
    generatedAt: '2026-05-07T10:00:00Z',
    data: { path: '/sys/fs/pstore', entries: [] },
  },
  bootHistory: {
    available: true,
    generatedAt: '2026-05-07T10:00:00Z',
    data: {
      boots: [
        { index: 0, bootId: 'aaaaaaaa-bbbb', firstEntry: '2026-05-07T09:00:00Z', lastEntry: '2026-05-07T10:00:00Z', clean: true },
        { index: -1, bootId: 'cccccccc-dddd', firstEntry: '2026-05-06T09:00:00Z', lastEntry: '2026-05-06T22:00:00Z', clean: false, reason: 'no shutdown marker found in boot log' },
      ],
    },
  },
  mce: {
    available: true,
    generatedAt: '2026-05-07T10:00:00Z',
    data: { window: '1 hour ago', uncorrected: 0, corrected: 3 },
  },
  autoheal: { available: true, checks: [{ checkId: 'system-pstore-evidence', status: 'OK' }] },
};

const notProvisioned: ForensicsSummary = {
  generatedAt: '2026-05-07T10:00:00Z',
  pstore: { available: false, reason: 'pstore directory not present (kernel pstore not configured)', generatedAt: '2026-05-07T10:00:00Z' },
  bootHistory: { available: false, reason: 'journal reader not configured', generatedAt: '2026-05-07T10:00:00Z' },
  mce: { available: false, reason: 'ras-mc-ctl not installed', generatedAt: '2026-05-07T10:00:00Z' },
  autoheal: { available: false, reason: 'autoheal unreachable: connection refused' },
};

describe('ForensicsPage', () => {
  beforeEach(() => {
    vi.mocked(fetchForensicsSummary).mockReset();
  });

  it('renders all five panels with happy-path data', async () => {
    vi.mocked(fetchForensicsSummary).mockResolvedValue(happy);
    // provider-free-exception: page uses a local fetch adapter and no shared provider context.
    render(<ForensicsPage />);

    await waitFor(() => expect(screen.getByText(/Last Shutdown/i)).toBeInTheDocument());
    expect(screen.getByText(/Boot History/i)).toBeInTheDocument();
    expect(screen.getByText(/Machine Check Errors/i)).toBeInTheDocument();
    expect(screen.getByText(/Pstore Artifacts/i)).toBeInTheDocument();
    expect(screen.getByText(/Autoheal Checks/i)).toBeInTheDocument();
    expect(screen.getByText(/system-pstore-evidence/)).toBeInTheDocument();
    expect(screen.getAllByText(/Unclean shutdown/i).length).toBeGreaterThan(0);
  });

  it('renders not-provisioned reasons when envelopes report unavailable', async () => {
    vi.mocked(fetchForensicsSummary).mockResolvedValue(notProvisioned);
    // provider-free-exception: page uses a local fetch adapter and no shared provider context.
    render(<ForensicsPage />);

    await waitFor(() =>
      expect(screen.getByText(/pstore directory not present/i)).toBeInTheDocument(),
    );
    expect(screen.getAllByText(/journal reader not configured/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/ras-mc-ctl not installed/i)).toBeInTheDocument();
    expect(screen.getByText(/autoheal unreachable/i)).toBeInTheDocument();
  });
});
