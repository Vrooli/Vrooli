import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ArtifactUploader } from './ArtifactUploader';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => ({
  ...(await vi.importActual('../../../shared/api')),
  presignDownloadArtifactUploadAdmin: vi.fn(),
  commitDownloadArtifactAdmin: vi.fn(),
}));

const apps = [{ bundle_key: 'desktop', app_key: 'desktop-app', name: 'Desktop App', platforms: [] }];
const installer = new File(['binary'], 'desktop-win-v1.2.3.exe', { type: 'application/octet-stream' });

function selectInstaller() {
  fireEvent.change(document.querySelector('input[type="file"]')!, { target: { files: [installer] } });
}

describe('ArtifactUploader', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn());
    vi.mocked(api.presignDownloadArtifactUploadAdmin).mockResolvedValue({
      upload_url: 'https://storage.example.test/upload', required_headers: { host: 'storage.example.test', 'x-upload-token': 'token' },
      bucket: 'releases', object_key: 'desktop/1.2.3/app.exe', expires_at: '2026-01-01T00:00:00Z', stable_object_uri: 's3://releases/desktop/1.2.3/app.exe',
    });
    vi.mocked(api.commitDownloadArtifactAdmin).mockResolvedValue({ id: 42, provider: 's3', bundle_key: 'desktop', bucket: 'releases', object_key: 'desktop/1.2.3/app.exe', platform: 'windows', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' });
  });

  it('requires a selected installer before an upload can begin', () => {
    render(<ArtifactUploader apps={apps} />);
    expect(screen.getByRole('button', { name: 'Upload' })).toBeDisabled();
    expect(screen.getByText('Drag & drop your installer')).toBeInTheDocument();
  });

  it('detects platform and semantic version from an installer filename and lets operators clear it', () => {
    render(<ArtifactUploader apps={apps} />);
    selectInstaller();
    expect(screen.getByText('desktop-win-v1.2.3.exe')).toBeInTheDocument();
    expect(screen.getByText('WINDOWS')).toBeInTheDocument();
    expect(screen.getByText('v1.2.3')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g. 2.1.0')).toHaveValue('1.2.3');
    fireEvent.click(screen.getByRole('button', { name: /Remove/ }));
    expect(screen.queryByText('desktop-win-v1.2.3.exe')).not.toBeInTheDocument();
  });

  it('uploads only after presigning, strips unsafe host headers, commits metadata, and reports the artifact id', async () => {
    const onUploadComplete = vi.fn();
    vi.mocked(fetch).mockResolvedValue(new Response('', { status: 200 }));
    render(<ArtifactUploader apps={apps} onUploadComplete={onUploadComplete} />);
    selectInstaller();
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));

    await waitFor(() => { expect(api.presignDownloadArtifactUploadAdmin).toHaveBeenCalledWith(expect.objectContaining({ app_key: 'desktop-app', platform: 'windows', release_version: '1.2.3' })); });
    expect(fetch).toHaveBeenCalledWith('https://storage.example.test/upload', expect.objectContaining({ method: 'PUT', body: installer }));
    await waitFor(() => { expect(api.commitDownloadArtifactAdmin).toHaveBeenCalledWith(expect.objectContaining({ bucket: 'releases', set_as_current: true })); });
    expect(onUploadComplete).toHaveBeenCalledWith(42);
    expect(await screen.findByText(/Uploaded desktop-win-v1.2.3.exe and set as latest version/)).toBeInTheDocument();
  });

  it('retains the selected file and surfaces storage upload errors for correction', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response('', { status: 403 }));
    render(<ArtifactUploader apps={apps} />);
    selectInstaller();
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));
    expect(await screen.findByText('Upload failed (403)')).toBeInTheDocument();
    expect(screen.getByText('desktop-win-v1.2.3.exe')).toBeInTheDocument();
    expect(api.commitDownloadArtifactAdmin).not.toHaveBeenCalled();
  });

  it('allows release managers to override detected metadata and avoid replacing the current artifact', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response('', { status: 200 }));
    render(<ArtifactUploader apps={apps} />);
    selectInstaller();
    const [, platform] = screen.getAllByRole('combobox');
    fireEvent.change(platform!, { target: { value: 'linux' } });
    fireEvent.change(screen.getByPlaceholderText('e.g. 2.1.0'), { target: { value: '1.2.4' } });
    fireEvent.click(screen.getByRole('checkbox'));
    expect(screen.getByText('Detected: WINDOWS (changed)')).toBeInTheDocument();
    expect(screen.getByText('Detected: v1.2.3 (changed)')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));

    await waitFor(() => { expect(api.presignDownloadArtifactUploadAdmin).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'linux', release_version: '1.2.4',
    })); });
    await waitFor(() => { expect(api.commitDownloadArtifactAdmin).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'linux', release_version: '1.2.4', set_as_current: false,
    })); });
  });

  it('keeps an operator-safe failure message when presigning rejects with a non-Error value', async () => {
    vi.mocked(api.presignDownloadArtifactUploadAdmin).mockRejectedValue('storage unavailable');
    render(<ArtifactUploader apps={apps} />);
    selectInstaller();
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));

    expect(await screen.findByText('Upload failed')).toBeInTheDocument();
    expect(screen.getByText('desktop-win-v1.2.3.exe')).toBeInTheDocument();
    expect(api.commitDownloadArtifactAdmin).not.toHaveBeenCalled();
  });

  it('supports drag-and-drop file selection and lets the operator cancel before upload', () => {
    const onCancel = vi.fn();
    const linuxInstaller = new File(['binary'], 'desktop-linux-2.0.0.AppImage', { type: '' });
    render(<ArtifactUploader apps={apps} onCancel={onCancel} />);
    const dropZone = screen.getByText('Drag & drop your installer').parentElement?.parentElement;
    if (!dropZone) throw new Error('upload drop zone is missing');

    fireEvent.dragOver(dropZone, { dataTransfer: { files: [linuxInstaller] } });
    fireEvent.drop(dropZone, { dataTransfer: { files: [linuxInstaller] } });
    expect(screen.getByText('LINUX')).toBeInTheDocument();
    expect(screen.getByText('v2.0.0')).toBeInTheDocument();
    expect(screen.getByText('6 B')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
