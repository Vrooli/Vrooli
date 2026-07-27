import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { buildDefaultStorageForm } from '../../services/downloads.service';
import { StepConfiguration } from './StepConfiguration';

function renderConfiguration(provider: 'aws-s3' | 'cloudflare-r2' | 'minio' | 'custom', form = buildDefaultStorageForm()) {
  const onFormChange = vi.fn();
  const onCloudflareAccountIdChange = vi.fn();
  render(<StepConfiguration provider={provider} form={form} cloudflareAccountId="" onFormChange={onFormChange} onCloudflareAccountIdChange={onCloudflareAccountIdChange} />);
  return { onFormChange, onCloudflareAccountIdChange };
}

describe('StepConfiguration', () => {
  it('uses an AWS region control and preserves bucket updates', () => {
    const { onFormChange } = renderConfiguration('aws-s3');
    expect(screen.getByText('Configure AWS S3')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('my-download-bucket'), { target: { value: 'releases' } });
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'eu-west-1' } });
    expect(onFormChange).toHaveBeenCalledWith({ bucket: 'releases' });
    expect(onFormChange).toHaveBeenCalledWith({ region: 'eu-west-1' });
  });

  it('collects the R2 account identity separately and explains the generated endpoint', () => {
    const { onCloudflareAccountIdChange } = renderConfiguration('cloudflare-r2', { ...buildDefaultStorageForm(), endpoint: 'https://acct.r2.cloudflarestorage.com' });
    expect(screen.getByText('Configure Cloudflare R2')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('abc123def456'), { target: { value: 'acct' } });
    expect(onCloudflareAccountIdChange).toHaveBeenCalledWith('acct');
    expect(screen.getByText(/Endpoint: https:\/\/acct.r2.cloudflarestorage.com/)).toBeInTheDocument();
  });

  it('exposes path-style controls for MinIO and custom S3 endpoints', () => {
    const minio = renderConfiguration('minio');
    fireEvent.change(screen.getByPlaceholderText('https://minio.example.com:9000'), { target: { value: 'https://minio.local:9000' } });
    fireEvent.click(screen.getByRole('checkbox'));
    expect(minio.onFormChange).toHaveBeenCalledWith({ endpoint: 'https://minio.local:9000' });
    expect(minio.onFormChange).toHaveBeenCalledWith({ forcePathStyle: true });

    const custom = renderConfiguration('custom');
    fireEvent.change(screen.getByPlaceholderText('https://s3.provider.com'), { target: { value: 'https://objects.example.com' } });
    expect(custom.onFormChange).toHaveBeenCalledWith({ endpoint: 'https://objects.example.com' });
  });
});
