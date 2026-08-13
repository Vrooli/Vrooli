import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { create } from '@bufbuild/protobuf';
import { renderWithProviders } from "@vrooli/api-base/testing";
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
    renderWithProviders(<CapacityPage />);

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
    renderWithProviders(<CapacityPage />);

    await waitFor(() => expect(screen.getByText(/Capacity sensing is unavailable/i)).toBeInTheDocument());
    expect(screen.getByText(/nvidia-smi binary not found/i)).toBeInTheDocument();
    expect(screen.getByText(/No GPUs detected/i)).toBeInTheDocument();
  });

  it('saves a policy lever change', async () => {
    const user = userEvent.setup();
    renderWithProviders(<CapacityPage />);

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
    renderWithProviders(<CapacityPage />);
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(/boom|Failed to load/i));
  });
});
