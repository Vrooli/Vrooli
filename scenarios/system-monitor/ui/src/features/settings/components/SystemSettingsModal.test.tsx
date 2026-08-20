import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SystemSettingsModal } from './SystemSettingsModal';

const mocks = vi.hoisted(() => ({ protoFetch: vi.fn() }));
vi.mock('../../../shared/api/apiFetch', () => ({ protoFetch: mocks.protoFetch }));
vi.mock('../../../shared/api/proto-contracts', () => ({
  parseGetSettingsResponse: vi.fn(),
  parseUpdateSettingsResponse: vi.fn(),
  parseResetSettingsResponse: vi.fn(),
  SystemSettingsSchema: {},
  create: (_schema: unknown, value: Record<string, unknown>) => ({ ...value }),
  toJsonString: (_schema: unknown, value: unknown) => JSON.stringify(value),
}));

const settings = {
  active: true, metricCollectionInterval: 15, anomalyDetectionInterval: 35, thresholdCheckInterval: 25,
  cooldownPeriodSeconds: 300, cpuThreshold: 80, memoryThreshold: 85, diskThreshold: 90,
};

describe('SystemSettingsModal', () => {
  beforeEach(() => mocks.protoFetch.mockReset());

  it('loads settings, edits fields, saves, resets, and confirms close', async () => {
    mocks.protoFetch
      .mockResolvedValueOnce({ success: true, settings })
      .mockResolvedValueOnce({ success: true, settings: { ...settings, cpuThreshold: 75 } })
      .mockResolvedValueOnce({ success: true, settings });
    const onClose = vi.fn();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    render(<SystemSettingsModal isOpen onClose={onClose} />);
    expect(await screen.findByText('System Monitor Settings')).toBeInTheDocument();
    const numbers = screen.getAllByRole('spinbutton');
    const interval = numbers[0];
    if (!interval) throw new Error('metric interval input was not rendered');
    expect(interval).toHaveValue(15);
    fireEvent.change(interval, { target: { value: '20' } });
    expect(screen.getByRole('button', { name: 'Save Settings' })).toBeEnabled();
    fireEvent.click(screen.getByRole('button', { name: 'Save Settings' }));
    await waitFor(() => expect(screen.getByText('Settings saved successfully!')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'Reset to Defaults' }));
    await waitFor(() => expect(screen.getByText('Settings reset to defaults!')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('shows loading and server errors and falls back to defaults', async () => {
    mocks.protoFetch.mockRejectedValueOnce(new Error('settings unavailable'));
    render(<SystemSettingsModal isOpen onClose={vi.fn()} />);
    expect(await screen.findByText('settings unavailable')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Save Settings' })).toBeDisabled();
  });

  it('covers field fallbacks, rejected confirmations, and save/reset failures', async () => {
    mocks.protoFetch.mockResolvedValueOnce({ success: true, settings });
    const onClose = vi.fn();
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    render(<SystemSettingsModal isOpen onClose={onClose} />);
    await screen.findByText('System Monitor Settings');

    const numbers = screen.getAllByRole('spinbutton');
    for (const input of numbers) fireEvent.change(input, { target: { value: '' } });
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByRole('button', { name: 'Reset to Defaults' }));
    expect(confirm).toHaveBeenCalled();
    expect(mocks.protoFetch).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).not.toHaveBeenCalled();
    confirm.mockReturnValue(true);
    mocks.protoFetch.mockResolvedValueOnce({ success: false, error: 'save failed' });
    fireEvent.click(screen.getByRole('button', { name: 'Save Settings' }));
    expect(await screen.findByText('save failed')).toBeInTheDocument();

    mocks.protoFetch.mockResolvedValueOnce({ success: false, error: 'reset failed' });
    fireEvent.click(screen.getByRole('button', { name: 'Reset to Defaults' }));
    expect(await screen.findByText('reset failed')).toBeInTheDocument();
  });
});
