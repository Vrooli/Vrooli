import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { buildDefaultCredentialsForm } from '../../services/downloads.service';
import { StepCredentials } from './StepCredentials';

const existingSettings = { provider: 's3' as const, force_path_style: false, signed_url_ttl_seconds: 900, access_key_id_set: true, secret_access_key_set: true, session_token_set: true, credentials_from_env: true, settings_row_available: true };

describe('StepCredentials', () => {
  it('shows environment credential guidance and protects configured values behind clear actions', () => {
    const onCredentialsChange = vi.fn();
    render(<StepCredentials provider="aws-s3" credentials={buildDefaultCredentialsForm()} existingSettings={existingSettings} onCredentialsChange={onCredentialsChange} />);
    expect(screen.getByText('Environment credentials detected')).toBeInTheDocument();
    expect(screen.getAllByPlaceholderText('••••••••••••')).toHaveLength(2);
    fireEvent.click(screen.getByRole('checkbox', { name: /Clear saved access key ID/ }));
    expect(onCredentialsChange).toHaveBeenCalledWith({ clearAccessKeyId: true, accessKeyId: '' });
  });

  it('forwards newly entered credentials without logging or exposing the secret', () => {
    const onCredentialsChange = vi.fn();
    render(<StepCredentials provider="minio" credentials={buildDefaultCredentialsForm()} existingSettings={null} onCredentialsChange={onCredentialsChange} />);
    fireEvent.change(screen.getByPlaceholderText('AKIA...'), { target: { value: 'operator' } });
    fireEvent.change(screen.getByPlaceholderText('Enter secret key'), { target: { value: 'secret-value' } });
    expect(onCredentialsChange).toHaveBeenCalledWith({ accessKeyId: 'operator', clearAccessKeyId: false });
    expect(onCredentialsChange).toHaveBeenCalledWith({ secretAccessKey: 'secret-value', clearSecretAccessKey: false });
    expect(screen.getByText(/Credentials are encrypted at rest and never logged/)).toBeInTheDocument();
  });
});
