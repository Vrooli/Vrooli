import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ImageUploader } from './ImageUploader';
import { uploadAsset } from '../api';

vi.mock('../api', () => ({ uploadAsset: vi.fn(), getAssetUrl: vi.fn((value: string) => value) }));

describe('ImageUploader', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('accepts a URL, clears it, and reports a failed preview safely', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ImageUploader value="https://cdn.example/logo.png" onChange={onChange} alt="Brand logo" />);

    fireEvent.error(screen.getByAltText('Brand logo'));
    expect(screen.getByText('Failed to load image preview')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'or enter URL' }));
    await user.type(screen.getByPlaceholderText('https://example.com/image.png'), ' https://images.example/new.png ');
    await user.click(screen.getByRole('button', { name: 'Set' }));
    expect(onChange).toHaveBeenCalledWith('https://images.example/new.png');
    await user.click(screen.getByRole('button', { name: 'Clear image' }));
    expect(onChange).toHaveBeenLastCalledWith(null);
  });

  it('uploads allowed assets, resets the same-file input, and returns full asset metadata', async () => {
    const onChange = vi.fn();
    const onUploadComplete = vi.fn();
    const asset = { url: '/api/assets/logo.png', id: 1, filename: 'logo.png', original_filename: 'logo.png', mime_type: 'image/png', size_bytes: 5, storage_path: 'logo.png', category: 'logo', created_at: '2026-01-01T00:00:00Z' };
    vi.mocked(uploadAsset).mockResolvedValue(asset);
    render(<ImageUploader onChange={onChange} category="logo" onUploadComplete={onUploadComplete} />);

    const input = document.querySelector('input[type="file"]')!;
    const file = new File(['image'], 'logo.png', { type: 'image/png' });
    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => expect(uploadAsset).toHaveBeenCalledWith(file, { category: 'logo' }));
    expect(onChange).toHaveBeenCalledWith('/api/assets/logo.png');
    expect(onUploadComplete).toHaveBeenCalledWith(asset);
  });

  it('rejects over-sized files and surfaces upload errors without changing the image', async () => {
    const onChange = vi.fn();
    render(<ImageUploader onChange={onChange} maxSize={1} />);
    const input = document.querySelector('input[type="file"]')!;
    fireEvent.change(input, { target: { files: [new File(['too large'], 'large.png', { type: 'image/png' })] } });
    expect(screen.getByText('File too large. Maximum size is 0MB')).toBeInTheDocument();
    expect(onChange).not.toHaveBeenCalled();

    vi.mocked(uploadAsset).mockRejectedValue(new Error('Storage unavailable'));
    fireEvent.change(input, { target: { files: [new File(['x'], 'small.png', { type: 'image/png' })] } });
    expect(await screen.findByText('Storage unavailable')).toBeInTheDocument();
  });
});
