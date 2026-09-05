import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { describe, expect, it, vi } from 'vitest';
import { StepProvider } from './StepProvider';

describe('StepProvider', () => {
  it('presents the supported providers and sends an explicit operator selection', () => {
    const onSelectProvider = vi.fn();
    render(<StepProvider selectedProvider={null} onSelectProvider={onSelectProvider} />);
    expect(screen.getByText('AWS S3')).toBeInTheDocument();
    expect(screen.getByText('Cloudflare R2')).toBeInTheDocument();
    expect(screen.getByText('MinIO')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Cloudflare R2/ }));
    expect(onSelectProvider).toHaveBeenCalledWith('cloudflare-r2');
  });

  it('shows decision support without losing the current provider selection', () => {
    render(<StepProvider selectedProvider="minio" onSelectProvider={vi.fn()} />);
    expect(screen.getByText('Self-hosted object storage')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Learn more' }));
    expect(screen.getByText('Which Storage Provider Should I Choose?')).toBeInTheDocument();
  });
});
