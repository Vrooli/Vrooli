import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders as render } from '../../test-utils';
import { ImageUploader } from './ImageUploader';

const { mockUploadAsset } = vi.hoisted(() => ({ mockUploadAsset: vi.fn() }));
vi.mock('../api', () => ({
  uploadAsset: mockUploadAsset,
  getAssetUrl: (p: string) => `/resolved/${p}`,
}));

beforeEach(() => {
  vi.clearAllMocks();
  mockUploadAsset.mockResolvedValue({ url: 'uploads/logo.png' });
});

const pngFile = () => new File(['data'], 'logo.png', { type: 'image/png' });

describe('ImageUploader', () => {
  it('uploads a selected file and reports the new URL', async () => {
    const onChange = vi.fn();
    const onUploadComplete = vi.fn();
    const { container } = render(
      <ImageUploader value={null} onChange={onChange} onUploadComplete={onUploadComplete} />,
    );
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    await userEvent.upload(input, pngFile());
    expect(mockUploadAsset).toHaveBeenCalled();
    expect(onChange).toHaveBeenCalledWith('uploads/logo.png');
    expect(onUploadComplete).toHaveBeenCalled();
  });

  it('rejects files larger than the max size', async () => {
    const onChange = vi.fn();
    const { container } = render(<ImageUploader value={null} onChange={onChange} maxSize={10} />);
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const big = new File(['x'.repeat(100)], 'big.png', { type: 'image/png' });
    await userEvent.upload(input, big);
    expect(await screen.findByText(/File too large/)).toBeInTheDocument();
    expect(mockUploadAsset).not.toHaveBeenCalled();
  });

  it('surfaces an upload failure', async () => {
    mockUploadAsset.mockRejectedValue(new Error('upload boom'));
    const { container } = render(<ImageUploader value={null} onChange={vi.fn()} />);
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    await userEvent.upload(input, pngFile());
    expect(await screen.findByText('upload boom')).toBeInTheDocument();
  });

  it('sets an image via the URL input and submits with Enter', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<ImageUploader value={null} onChange={onChange} />);
    await user.click(screen.getByRole('button', { name: /or enter URL/i }));
    const urlInput = screen.getByPlaceholderText(/example.com\/image/);
    await user.type(urlInput, 'https://cdn/pic.png');
    fireEvent.keyDown(urlInput, { key: 'Enter' });
    expect(onChange).toHaveBeenCalledWith('https://cdn/pic.png');
  });

  it('cancels the URL input with Escape and the cancel button', async () => {
    const user = userEvent.setup();
    render(<ImageUploader value={null} onChange={vi.fn()} />);
    await user.click(screen.getByRole('button', { name: /or enter URL/i }));
    const urlInput = screen.getByPlaceholderText(/example.com\/image/);
    fireEvent.keyDown(urlInput, { key: 'Escape' });
    // Reopen and cancel via the button.
    await user.click(screen.getByRole('button', { name: /or enter URL/i }));
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByPlaceholderText(/example.com\/image/)).not.toBeInTheDocument();
  });

  it('clears the current image', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    const { container } = render(<ImageUploader value="uploads/logo.png" onChange={onChange} />);
    // The clear button is the rose-colored X.
    const clearBtn = container.querySelector('button.text-rose-400') as HTMLButtonElement;
    await user.click(clearBtn);
    expect(onChange).toHaveBeenCalledWith(null);
  });

  it('shows a fallback when the image fails to load', () => {
    render(<ImageUploader value="uploads/broken.png" onChange={vi.fn()} />);
    const img = screen.getByRole('img');
    fireEvent.error(img);
    expect(screen.getByText(/Failed to load image preview/)).toBeInTheDocument();
  });

  it('rejects a non-image dropped onto the preview area', async () => {
    const { container } = render(<ImageUploader value={null} onChange={vi.fn()} />);
    const dropZone = container.querySelector('[class*="items-center gap-4"]') as HTMLElement;
    fireEvent.dragOver(dropZone);
    fireEvent.drop(dropZone, { dataTransfer: { files: [new File(['x'], 'a.txt', { type: 'text/plain' })] } });
    expect(await screen.findByText(/Please drop an image file/)).toBeInTheDocument();
  });
});
