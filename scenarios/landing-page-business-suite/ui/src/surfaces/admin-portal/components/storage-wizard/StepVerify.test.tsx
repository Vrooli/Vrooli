import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { buildDefaultStorageForm } from '../../services/downloads.service';
import { StepVerify } from './StepVerify';

function verifyProps(overrides: Partial<React.ComponentProps<typeof StepVerify>> = {}) {
  return {
    provider: 'minio' as const, form: { ...buildDefaultStorageForm(), bucket: 'releases', endpoint: 'https://minio.local:9000', defaultPrefix: 'apps/', forcePathStyle: true }, cloudflareAccountId: '',
    testStatus: 'idle' as const, testError: null, saveStatus: 'idle' as const, saveError: null,
    onFormChange: vi.fn(), onTestConnection: vi.fn().mockResolvedValue(undefined), onSave: vi.fn().mockResolvedValue(undefined), ...overrides,
  };
}

describe('StepVerify', () => {
  it('requires a saved configuration before allowing a storage connection test', () => {
    render(<StepVerify {...verifyProps()} />);
    expect(screen.getByText('MinIO')).toBeInTheDocument();
    expect(screen.getByText('https://minio.local:9000')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Test Connection' })).toBeDisabled();
    expect(screen.getByText('Save your configuration first to enable testing')).toBeInTheDocument();
  });

  it('enables testing after save and exposes advanced TTL/CDN updates', () => {
    const onTestConnection = vi.fn().mockResolvedValue(undefined);
    const onFormChange = vi.fn();
    render(<StepVerify {...verifyProps({ saveStatus: 'success', onTestConnection, onFormChange })} />);
    fireEvent.click(screen.getByRole('button', { name: 'Test Connection' }));
    expect(onTestConnection).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText('Advanced Settings'));
    fireEvent.change(screen.getByDisplayValue('900'), { target: { value: '3600' } });
    fireEvent.change(screen.getByPlaceholderText('https://downloads.example.com'), { target: { value: 'https://cdn.example.com' } });
    expect(onFormChange).toHaveBeenCalledWith({ signedUrlTtlSeconds: 3600 });
    expect(onFormChange).toHaveBeenCalledWith({ publicBaseUrl: 'https://cdn.example.com' });
  });

  it('presents save and connection errors with troubleshooting rather than hiding failures', () => {
    render(<StepVerify {...verifyProps({ testStatus: 'error', testError: 'Access denied', saveStatus: 'error', saveError: 'Unable to persist settings' })} />);
    expect(screen.getByText('Connection failed')).toBeInTheDocument();
    expect(screen.getByText('Access denied')).toBeInTheDocument();
    expect(screen.getByText('Unable to persist settings')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Troubleshooting guide' }));
    expect(screen.getByText('Troubleshooting Connection Issues')).toBeInTheDocument();
  });
});
