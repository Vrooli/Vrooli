/* eslint-disable @typescript-eslint/unbound-method -- assertions exercise Vitest/browser mocks, not detached production methods. */
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DownloadSettings } from './DownloadSettings';
import * as downloadsForm from '../hooks/useDownloadsForm';
import * as downloadHosting from '../hooks/useDownloadHosting';
import { presignDownloadArtifactGetAdmin } from '../../../shared/api';
import { buildDefaultAppValues } from '../services/downloads.service';

vi.mock('../hooks/useDownloadsForm');
vi.mock('../hooks/useDownloadHosting');
vi.mock('../../../shared/api', async (importOriginal) => ({ ...(await importOriginal<typeof import('../../../shared/api')>()), presignDownloadArtifactGetAdmin: vi.fn() }));
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));
vi.mock('../components/storage-wizard', () => ({ StorageWizard: () => <div>Storage wizard</div> }));
vi.mock('../components/ArtifactUploader', () => ({ ArtifactUploader: ({ onUploadComplete }: { onUploadComplete: () => void }) => <button onClick={onUploadComplete}>Artifact uploader</button> }));
vi.mock('../../../shared/ui/ImageUploader', () => ({ ImageUploader: ({ uploadLabel, onChange }: { uploadLabel: string; onChange: (url: string | null) => void }) => <button onClick={() => { onChange(`https://cdn.example/${uploadLabel}.png`); }}>{uploadLabel}</button> }));

function formState(overrides: Record<string, unknown> = {}) {
  return { forms: [], loading: false, error: null, dirtyCount: 0, downloadHealth: { appCount: 0, platformsConfigured: 0, platformsMissing: 0, storefrontsConfigured: 0 }, loadApps: vi.fn(), handleFieldChange: vi.fn(), handlePlatformChange: vi.fn(), handleAddApp: vi.fn(), handleReset: vi.fn(), handleDelete: vi.fn(), handleSave: vi.fn(), handleSaveAll: vi.fn(), savingAll: false, draggingKey: null, dragOverKey: null, handleDragStart: vi.fn(), handleDragOver: vi.fn(), handleDragLeave: vi.fn(), handleDrop: vi.fn(), handleDragEnd: vi.fn(), ...overrides } as unknown as ReturnType<typeof downloadsForm.useDownloadsForm>;
}
function hostingState(overrides: Record<string, unknown> = {}) {
  return { storageSettings: null, storageSuccess: null, loadStorage: vi.fn(), artifactsLoading: false, artifactsError: null, artifactsQuery: '', setArtifactsQuery: vi.fn(), artifactsPlatform: '', setArtifactsPlatform: vi.fn(), artifactsAppKey: '', setArtifactsAppKey: vi.fn(), artifacts: [], selectedArtifact: null, setSelectedArtifact: vi.fn(), applyTarget: { appKey: '', platform: 'windows', requiresEntitlement: false, releaseVersion: '', releaseNotes: '' }, setApplyTarget: vi.fn(), loadArtifacts: vi.fn(), handleApplyArtifact: vi.fn(), handleSetArtifactAsCurrent: vi.fn(), ...overrides } as unknown as ReturnType<typeof downloadHosting.useDownloadHosting>;
}

describe('DownloadSettings', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubGlobal('open', vi.fn()); vi.mocked(downloadHosting.useDownloadHosting).mockReturnValue(hostingState()); });

  it('guides an operator through the empty app state and exposes refresh, preview, and add actions', () => {
    const state = formState();
    vi.mocked(downloadsForm.useDownloadsForm).mockReturnValue(state);
    render(<DownloadSettings />);
    expect(screen.getByTestId('downloads-empty-state')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('downloads-add-app'));
    fireEvent.click(screen.getByRole('button', { name: 'Add Your First App' }));
    fireEvent.click(screen.getByTestId('downloads-refresh'));
    fireEvent.click(screen.getByTestId('downloads-preview'));
    expect(state.handleAddApp).toHaveBeenCalledTimes(2);
    expect(state.loadApps).toHaveBeenCalledOnce();
    expect(window.open).toHaveBeenCalledWith('/', '_blank', 'noopener,noreferrer');
  });

  it('shows hosted artifacts, supports applying a version, setting latest, and opening a presigned download', async () => {
    const values = { ...buildDefaultAppValues('desktop'), name: 'Desktop App' };
    const form = { key: 'app-1', values, original: values, saving: false };
    const artifact = { id: 'artifact-1', original_filename: 'desktop-1.2.3.exe', object_key: 'desktop.exe', app_key: 'desktop', platform: 'windows', release_version: '1.2.3', size_bytes: 1048576, created_at: '2026-01-01T00:00:00Z', is_current: false };
    const state = formState({ forms: [form] });
    const hosting = hostingState({ artifacts: [artifact] });
    vi.mocked(downloadsForm.useDownloadsForm).mockReturnValue(state);
    vi.mocked(downloadHosting.useDownloadHosting).mockReturnValue(hosting);
    vi.mocked(presignDownloadArtifactGetAdmin).mockResolvedValue({ url: 'https://downloads.example/artifact' });
    render(<DownloadSettings />);
    fireEvent.click(screen.getByRole('button', { name: 'Hosting' }));
    expect(screen.getByText('Storage wizard')).toBeInTheDocument();
    expect(screen.getByText('desktop-1.2.3.exe')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Set as Latest' }));
    expect(hosting.handleSetArtifactAsCurrent).toHaveBeenCalledWith(artifact, 'desktop', 'windows');
    fireEvent.click(screen.getByRole('button', { name: 'Apply to App…' }));
    expect(hosting.setSelectedArtifact).toHaveBeenCalledWith(artifact);
    expect(hosting.setApplyTarget).toHaveBeenCalledWith(expect.objectContaining({ appKey: 'desktop', platform: 'windows', releaseVersion: '1.2.3' }));
    fireEvent.click(screen.getByRole('button', { name: 'Download' }));
    await waitFor(() => { expect(window.open).toHaveBeenCalledWith('https://downloads.example/artifact', '_blank'); });
  });

  it('edits app metadata and installer delivery settings, then saves a dirty app', async () => {
    const original = buildDefaultAppValues('desktop');
    const values = {
      ...original,
      name: 'Desktop App',
      platforms: {
        ...original.platforms,
        windows: { ...original.platforms.windows, enabled: true, artifactSource: 'direct' as const, artifactUrl: 'https://cdn.example/desktop.exe', releaseVersion: '2.0.0' },
      },
    };
    const state = formState({ forms: [{ key: 'app-1', values, original, saving: false, isNew: false }] });
    vi.mocked(downloadsForm.useDownloadsForm).mockReturnValue(state);
    render(<DownloadSettings />);
    fireEvent.change(screen.getByDisplayValue('Desktop App'), { target: { value: 'Desktop Pro' } });
    fireEvent.change(screen.getByDisplayValue('https://cdn.example/desktop.exe'), { target: { value: 'https://cdn.example/pro.exe' } });
    fireEvent.click(screen.getByTestId('download-save-app-1'));
    await waitFor(() => { expect(state.handleSave).toHaveBeenCalledWith('app-1'); });
    expect(state.handleFieldChange).toHaveBeenCalledWith('app-1', 'name', 'Desktop Pro');
    expect(state.handlePlatformChange).toHaveBeenCalledWith('app-1', 'windows', 'artifactUrl', 'https://cdn.example/pro.exe');
    expect(state.handleReset).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Hosting' }));
    expect(screen.getByTestId('downloads-hosting')).toBeInTheDocument();
  });

  it('supports dirty-card recovery, save-all, deletion, reordering, storefronts, and managed artifact settings', () => {
    const original = buildDefaultAppValues('desktop');
    const values = {
      ...original,
      name: 'Desktop App',
      appleEnabled: true,
      appleLabel: 'App Store',
      appleUrl: 'https://apps.example/apple',
      appleBadge: 'New',
      googleEnabled: true,
      googleLabel: 'Google Play',
      googleUrl: 'https://apps.example/google',
      googleBadge: 'Featured',
      platforms: {
        ...original.platforms,
        windows: { ...original.platforms.windows, enabled: true, artifactSource: 'managed' as const, artifactId: 'artifact-1', artifactFilename: 'desktop.exe', artifactSizeBytes: 2048, artifactCount: 2, releaseVersion: '2.0.0' },
        mac: { ...original.platforms.mac, enabled: true, artifactSource: 'direct' as const },
        linux: { ...original.platforms.linux, enabled: true, artifactSource: 'direct' as const },
      },
    };
    const secondValues = { ...buildDefaultAppValues('mobile'), name: 'Mobile App' };
    const state = formState({
      forms: [
        { key: 'app-1', values, original, saving: false, isNew: false, lastSavedAt: '2026-01-01T12:00:00Z', error: 'Needs review' },
        { key: 'app-2', values: secondValues, original: secondValues, saving: false, isNew: false },
      ],
      dirtyCount: 1,
      error: 'Load warning',
    });
    vi.mocked(downloadsForm.useDownloadsForm).mockReturnValue(state);
    render(<DownloadSettings />);
    fireEvent.click(screen.getByTestId('downloads-save-all'));
    fireEvent.click(screen.getByTestId('download-reset-app-1'));
    fireEvent.click(screen.getByTestId('download-delete-app-1'));
    fireEvent.dragStart(screen.getByTestId('download-card-app-1'));
    fireEvent.dragOver(screen.getByTestId('download-card-app-2'));
    fireEvent.dragLeave(screen.getByTestId('download-card-app-2'));
    fireEvent.drop(screen.getByTestId('download-card-app-2'));
    fireEvent.dragEnd(screen.getByTestId('download-card-app-1'));
    const checkboxes = screen.getAllByRole('checkbox');
    checkboxes.forEach((checkbox) => { fireEvent.click(checkbox); });
    const selects = screen.getAllByRole('combobox');
    selects.forEach((select) => { fireEvent.change(select, { target: { value: 'direct' } }); });
    fireEvent.change(screen.getAllByDisplayValue('App Store')[0]!, { target: { value: 'Store' } });
    fireEvent.change(screen.getAllByDisplayValue('https://apps.example/apple')[0]!, { target: { value: 'https://new.example/apple' } });
    fireEvent.change(screen.getAllByDisplayValue('New')[0]!, { target: { value: 'Updated' } });
    expect(state.handleSaveAll).toHaveBeenCalledOnce();
    expect(state.handleReset).toHaveBeenCalledWith('app-1');
    expect(state.handleDelete).toHaveBeenCalledWith('app-1');
    expect(state.handleDragStart).toHaveBeenCalledWith('app-1');
    expect(state.handleDragOver).toHaveBeenCalledWith('app-2');
    expect(state.handleDragLeave).toHaveBeenCalledOnce();
    expect(state.handleDrop).toHaveBeenCalledWith('app-2');
    expect(state.handleDragEnd).toHaveBeenCalledOnce();
    expect(state.handleFieldChange).toHaveBeenCalledWith('app-1', 'appleLabel', 'Store');
    expect(state.handlePlatformChange).toHaveBeenCalledWith('app-1', 'windows', 'artifactSource', 'direct');
  });

  it('filters hosted artifacts and edits an active artifact-application target', () => {
    const values = { ...buildDefaultAppValues('desktop'), name: 'Desktop App' };
    const form = { key: 'app-1', values, original: values, saving: false, isNew: false };
    const artifact = { id: 'artifact-1', original_filename: 'desktop-1.2.3.exe', object_key: 'desktop.exe', app_key: 'desktop', platform: 'windows', release_version: '1.2.3', size_bytes: 0, created_at: '2026-01-01T00:00:00Z', is_current: true };
    const state = formState({ forms: [form] });
    const hosting = hostingState({
      artifacts: [artifact],
      selectedArtifact: artifact,
      artifactsError: 'Storage unavailable',
      storageSuccess: 'Storage connected',
      applyTarget: { appKey: 'desktop', platform: 'windows', requiresEntitlement: false, releaseVersion: '1.2.3', releaseNotes: '' },
    });
    vi.mocked(downloadsForm.useDownloadsForm).mockReturnValue(state);
    vi.mocked(downloadHosting.useDownloadHosting).mockReturnValue(hosting);
    render(<DownloadSettings />);
    fireEvent.click(screen.getByRole('button', { name: 'Hosting' }));
    const selects = screen.getAllByRole('combobox');
    fireEvent.change(selects[0]!, { target: { value: 'desktop' } });
    fireEvent.change(selects[1]!, { target: { value: 'linux' } });
    fireEvent.change(screen.getByPlaceholderText('Search filename, version…'), { target: { value: '1.2.3' } });
    fireEvent.click(screen.getAllByRole('button', { name: 'Refresh' })[1]!);
    fireEvent.change(selects[2]!, { target: { value: 'desktop' } });
    fireEvent.change(selects[3]!, { target: { value: 'mac' } });
    fireEvent.change(screen.getByDisplayValue('1.2.3'), { target: { value: '2.0.0' } });
    fireEvent.change(screen.getAllByDisplayValue('')[1]!, { target: { value: 'Important fixes' } });
    fireEvent.click(screen.getByRole('checkbox', { name: 'Requires entitlement' }));
    fireEvent.click(screen.getByRole('button', { name: 'Apply to app' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(hosting.setArtifactsAppKey).toHaveBeenCalledWith('desktop');
    expect(hosting.setArtifactsPlatform).toHaveBeenCalledWith('linux');
    expect(hosting.setArtifactsQuery).toHaveBeenCalledWith('1.2.3');
    expect(hosting.setApplyTarget).toHaveBeenCalledWith(expect.any(Function));
    expect(hosting.handleApplyArtifact).toHaveBeenCalledOnce();
    expect(hosting.setSelectedArtifact).toHaveBeenCalledWith(null);
  });

  it('edits detailed app copy, images, storefront values, and managed release metadata', () => {
    const original = buildDefaultAppValues('desktop');
    const values = {
      ...original, name: 'Desktop App', tagline: 'Old tagline', description: 'Old description', installOverview: 'Old overview', installSteps: 'One',
      googleEnabled: true, googleLabel: 'Google Play', googleUrl: 'https://apps.example/google', googleBadge: 'Featured',
      platforms: { ...original.platforms, windows: { ...original.platforms.windows, enabled: true, artifactSource: 'managed' as const, releaseVersion: '1.0.0', releaseNotes: 'Old notes' } },
    };
    const state = formState({ forms: [{ key: 'app-1', values, original, saving: false, isNew: true }] });
    const hosting = hostingState();
    vi.mocked(downloadsForm.useDownloadsForm).mockReturnValue(state);
    vi.mocked(downloadHosting.useDownloadHosting).mockReturnValue(hosting);
    render(<DownloadSettings />);

    fireEvent.change(screen.getByDisplayValue('desktop'), { target: { value: 'desktop-pro' } });
    fireEvent.change(screen.getByDisplayValue('Old tagline'), { target: { value: 'New tagline' } });
    fireEvent.change(screen.getByDisplayValue('Old description'), { target: { value: 'New description' } });
    fireEvent.change(screen.getByDisplayValue('Old overview'), { target: { value: 'New overview' } });
    fireEvent.change(screen.getByDisplayValue('One'), { target: { value: 'One\nTwo' } });
    fireEvent.click(screen.getByRole('button', { name: 'Upload icon' }));
    fireEvent.click(screen.getByRole('button', { name: 'Upload screenshot' }));
    fireEvent.change(screen.getByDisplayValue('Google Play'), { target: { value: 'Play Store' } });
    fireEvent.change(screen.getByDisplayValue('https://apps.example/google'), { target: { value: 'https://new.example/google' } });
    fireEvent.change(screen.getByDisplayValue('Featured'), { target: { value: 'New' } });
    fireEvent.change(screen.getByDisplayValue('1.0.0'), { target: { value: '1.1.0' } });
    fireEvent.change(screen.getByDisplayValue('Old notes'), { target: { value: 'Fixes' } });
    fireEvent.click(screen.getByRole('button', { name: 'Hosting' }));
    fireEvent.click(screen.getByRole('button', { name: 'Artifact uploader' }));

    expect(state.handleFieldChange).toHaveBeenCalledWith('app-1', 'appKey', 'desktop-pro');
    expect(state.handleFieldChange).toHaveBeenCalledWith('app-1', 'iconUrl', 'https://cdn.example/Upload icon.png');
    expect(state.handleFieldChange).toHaveBeenCalledWith('app-1', 'googleLabel', 'Play Store');
    expect(state.handlePlatformChange).toHaveBeenCalledWith('app-1', 'windows', 'releaseVersion', '1.1.0');
    expect(state.handlePlatformChange).toHaveBeenCalledWith('app-1', 'windows', 'releaseNotes', 'Fixes');
    expect(hosting.loadArtifacts).toHaveBeenCalled();
  });
});
