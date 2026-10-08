import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ReportsPanel } from './ReportsPanel';

const mocks = vi.hoisted(() => ({ protoFetch: vi.fn() }));
vi.mock('../../../shared/api/apiFetch', () => ({
  protoFetch: mocks.protoFetch,
  extractErrorMessage: (error: unknown, fallback: string) => error instanceof Error ? error.message : fallback,
}));

const report = {
  id: 'daily-1', type: 'daily', generatedAt: timestampFromDate(new Date('2026-01-01T00:00:00Z')),
  summary: { avgCpuUsage: 20, avgMemoryUsage: 30, totalAlerts: 2, uptimePercentage: 99.9 },
  recommendations: ['Reduce disk churn', 'Review process growth'],
};

describe('ReportsPanel', () => {
  beforeEach(() => mocks.protoFetch.mockReset());

  it('renders generated reports and supports daily generation and refresh', async () => {
    mocks.protoFetch
      .mockResolvedValueOnce({ reports: [report] })
      .mockResolvedValueOnce({})
      .mockResolvedValueOnce({ reports: [report] })
      .mockResolvedValueOnce({ reports: [report] });
    render(<ReportsPanel />);
    expect(await screen.findByText('daily Report')).toBeInTheDocument();
    expect(screen.getByText('Reduce disk churn')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /GENERATE DAILY/ }));
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledWith('/reports/generate', expect.anything(), expect.objectContaining({ method: 'POST' })); });
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledTimes(3); });
    fireEvent.click(screen.getByRole('button', { name: /^REFRESH$/ }));
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledTimes(4); });
  });

  it('shows empty and error states with retry behavior', async () => {
    mocks.protoFetch.mockResolvedValueOnce({ reports: [] });
    render(<ReportsPanel />);
    expect(await screen.findByText('NO REPORTS AVAILABLE')).toBeInTheDocument();

    mocks.protoFetch.mockRejectedValueOnce(new Error('reports unavailable'));
    fireEvent.click(screen.getByRole('button', { name: /^REFRESH$/ }));
    expect(await screen.findByText('FAILED TO LOAD REPORTS')).toBeInTheDocument();
    mocks.protoFetch.mockResolvedValueOnce({ reports: [] });
    fireEvent.click(screen.getByRole('button', { name: /RETRY/ }));
    await waitFor(() => expect(screen.getByText('NO REPORTS AVAILABLE')).toBeInTheDocument());
  });
});
