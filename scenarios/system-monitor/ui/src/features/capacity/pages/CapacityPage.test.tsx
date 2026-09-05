import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { create } from '@bufbuild/protobuf';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { CapacityPage } from './CapacityPage';
import {
  GetCapacityOverviewResponseSchema,
  ReconcileCapacityResponseSchema,
  GetCapacityPolicyResponseSchema,
} from '../../../shared/api/proto-contracts';

vi.mock('../api', () => ({
  fetchCapacityOverview: vi.fn(),
  fetchCapacityReconciliation: vi.fn(),
  fetchCapacityPolicy: vi.fn(),
  setCapacityPolicy: vi.fn(),
}));

import {
  fetchCapacityOverview,
  fetchCapacityReconciliation,
  fetchCapacityPolicy,
  setCapacityPolicy,
} from '../api';

const GB = 1024 * 1024 * 1024;

const overview = create(GetCapacityOverviewResponseSchema, {
  success: true,
  sensingAvailable: true,
  warnings: [],
  gpus: [
    { index: 0, name: 'NVIDIA RTX', totalBytes: BigInt(16 * GB), usedBytes: BigInt(13 * GB), freeBytes: BigInt(3 * GB), claimedBytes: BigInt(8 * GB), memoryUtilizationPercent: 81 },
  ],
  claims: [
    { claimId: 'a', ownerKind: 'resource', ownerId: 'whisper', resourceKind: 'vram', gpuIndex: 0, amountBytes: BigInt(7 * GB), priorityTier: 'interactive', protected: true, status: 'granted', activityState: 'active' },
  ],
});

const reconciliation = create(ReconcileCapacityResponseSchema, {
  success: true,
  findings: [
    { class: 'unclaimed', ownerId: 'rogue-proc', pid: 1234, severity: 'warn', message: 'unclaimed GPU consumer "rogue-proc" holds no capacity claim' },
  ],
});

const policyLevers = create(GetCapacityPolicyResponseSchema, {
  success: true,
  levers: [
    { key: 'enforce', value: 'advisory' },
    { key: 'idle_grace', value: '5m' },
  ],
}).levers;

describe('CapacityPage', () => {
  beforeEach(() => {
    vi.mocked(fetchCapacityOverview).mockResolvedValue(overview);
    vi.mocked(fetchCapacityReconciliation).mockResolvedValue(reconciliation);
    vi.mocked(fetchCapacityPolicy).mockResolvedValue(policyLevers);
    vi.mocked(setCapacityPolicy).mockResolvedValue(policyLevers);
  });

  it('renders GPU contention, claims, findings and policy levers', async () => {
    // provider-free-exception: page uses local fetch adapters and no shared provider context.
    render(<CapacityPage />);

    await waitFor(() => expect(screen.getByText(/GPU 0 · NVIDIA RTX/)).toBeInTheDocument());
    // Claim row
    expect(screen.getByText(/resource\/whisper/)).toBeInTheDocument();
    // Unclaimed finding warning
    expect(screen.getByText(/unclaimed GPU consumer/i)).toBeInTheDocument();
    // Policy levers
    expect(screen.getByLabelText('enforce')).toBeInTheDocument();
    expect(screen.getByLabelText('idle_grace')).toBeInTheDocument();
  });

  it('surfaces sensing-unavailable findings honestly', async () => {
    vi.mocked(fetchCapacityOverview).mockResolvedValue(create(GetCapacityOverviewResponseSchema, {
      success: true,
      sensingAvailable: false,
      warnings: ['nvidia-smi binary not found'],
      gpus: [],
      claims: [],
    }));
    // provider-free-exception: page uses local fetch adapters and no shared provider context.
    render(<CapacityPage />);

    await waitFor(() =>
      expect(screen.getByText(/unclaimed consumers cannot be reconciled/i)).toBeInTheDocument(),
    );
    expect(screen.getByText(/nvidia-smi binary not found/i)).toBeInTheDocument();
    // The page must NOT claim the host has no GPU when it could not look. An
    // empty list under unavailable sensing is blindness, not a finding, and
    // this assertion previously required the page to state the opposite.
    expect(screen.queryByText(/No GPUs detected/i)).not.toBeInTheDocument();
    expect(screen.getByText(/GPU contention unreadable/i)).toBeInTheDocument();
    expect(screen.getByText(/No GPU probe answered on this host/i)).toBeInTheDocument();
  });

  it('separates a failed request from a sensing warning', async () => {
    // These were previously the same neutral card, so a request that failed
    // outright and a reading that came back with caveats looked identical.
    vi.mocked(fetchCapacityOverview).mockRejectedValue(new Error('capacity backend unreachable'));
    render(<CapacityPage />);

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(/Capacity request failed/i),
    );
    expect(screen.getByText(/capacity backend unreachable/i)).toBeInTheDocument();
  });

  it('shows sensing warnings as caution when sensing itself did work', async () => {
    vi.mocked(fetchCapacityOverview).mockResolvedValue(create(GetCapacityOverviewResponseSchema, {
      success: true,
      sensingAvailable: true,
      warnings: ['gpu 1 reported no serial'],
      gpus: [],
      claims: [],
    }));
    render(<CapacityPage />);

    await waitFor(() => expect(screen.getByText(/Sensing warnings/i)).toBeInTheDocument());
    expect(screen.getByText(/gpu 1 reported no serial/i)).toBeInTheDocument();
    // Sensing worked and returned nothing, so an empty list IS a finding here
    // and the honest message is the one that states it.
    expect(screen.getByText(/No GPUs detected on this host/i)).toBeInTheDocument();
    expect(screen.queryByText(/GPU contention unreadable/i)).not.toBeInTheDocument();
  });

  it('saves a policy lever change', async () => {
    const user = userEvent.setup();
    // provider-free-exception: page uses local fetch adapters and no shared provider context.
    render(<CapacityPage />);

    await waitFor(() => expect(screen.getByLabelText('enforce')).toBeInTheDocument());

    const select = screen.getByLabelText('enforce');
    await user.selectOptions(select, 'on');
    // The Save button for the enforce row becomes enabled once the draft diverges.
    const saveButtons = screen.getAllByRole('button', { name: /save/i });
    await user.click(saveButtons[0]);

    await waitFor(() => { expect(vi.mocked(setCapacityPolicy)).toHaveBeenCalledWith('enforce', 'on'); });
  });

  it('shows an error banner when the overview fetch fails', async () => {
    vi.mocked(fetchCapacityOverview).mockRejectedValue(new Error('boom'));
    // provider-free-exception: page uses local fetch adapters and no shared provider context.
    render(<CapacityPage />);
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(/boom|Failed to load/i));
  });
});
