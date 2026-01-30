import { useCallback, useEffect, useState } from 'react';
import {
  applyDownloadArtifactAdmin,
  commitDownloadArtifactAdmin,
  getDownloadStorageAdmin,
  listDownloadArtifactsAdmin,
  listDownloadArtifactsByAppAdmin,
  presignDownloadArtifactUploadAdmin,
  setArtifactAsCurrentAdmin,
  testDownloadStorageAdmin,
  updateDownloadStorageAdmin,
  type DownloadArtifact,
  type DownloadStorageSettingsSnapshot,
} from '../../../shared/api';
import {
  buildDefaultStorageForm,
  buildDefaultCredentialsForm,
  buildStorageUpdatePayload,
  type PlatformKey,
  type StorageFormValues,
  type CredentialsFormValues,
} from '../services/downloads.service';

interface ApplyTargetState {
  appKey: string;
  platform: PlatformKey;
  requiresEntitlement: boolean;
  releaseVersion: string;
  releaseNotes: string;
}

interface UploadState {
  file: File | null;
  platform: PlatformKey | '';
  releaseVersion: string;
  appKey: string;
  busy: boolean;
  message: string;
  error: string;
}

interface UseDownloadHostingProps {
  activeTab: 'apps' | 'hosting';
  loadApps: () => Promise<void>;
  getFirstAppKey: () => string;
}

interface UseDownloadHostingReturn {
  // Storage
  storageSettings: DownloadStorageSettingsSnapshot | null;
  storageLoading: boolean;
  storageSaving: boolean;
  storageError: string | null;
  storageSuccess: string | null;
  storageForm: StorageFormValues;
  setStorageForm: React.Dispatch<React.SetStateAction<StorageFormValues>>;
  credentialsForm: CredentialsFormValues;
  setCredentialsForm: React.Dispatch<React.SetStateAction<CredentialsFormValues>>;
  loadStorage: () => Promise<void>;
  handleSaveStorage: () => Promise<void>;
  handleTestStorage: () => Promise<void>;

  // Artifacts
  artifactsLoading: boolean;
  artifactsError: string | null;
  setArtifactsError: React.Dispatch<React.SetStateAction<string | null>>;
  artifactsQuery: string;
  setArtifactsQuery: React.Dispatch<React.SetStateAction<string>>;
  artifactsPlatform: PlatformKey | '';
  setArtifactsPlatform: React.Dispatch<React.SetStateAction<PlatformKey | ''>>;
  artifactsAppKey: string;
  setArtifactsAppKey: React.Dispatch<React.SetStateAction<string>>;
  artifacts: DownloadArtifact[];
  selectedArtifact: DownloadArtifact | null;
  setSelectedArtifact: React.Dispatch<React.SetStateAction<DownloadArtifact | null>>;
  applyTarget: ApplyTargetState;
  setApplyTarget: React.Dispatch<React.SetStateAction<ApplyTargetState>>;
  loadArtifacts: () => Promise<void>;
  handleApplyArtifact: () => Promise<void>;
  handleSetArtifactAsCurrent: (artifact: DownloadArtifact, appKey: string, platform: PlatformKey) => Promise<void>;

  // Upload
  uploadState: UploadState;
  setUploadState: React.Dispatch<React.SetStateAction<UploadState>>;
  handleUploadArtifact: () => Promise<void>;
}

/**
 * Hook for managing download hosting state and operations.
 * Extracts storage settings, artifacts, and upload state management from DownloadSettings.
 */
export function useDownloadHosting({
  activeTab,
  loadApps,
  getFirstAppKey: _getFirstAppKey,
}: UseDownloadHostingProps): UseDownloadHostingReturn {
  // Storage state
  const [storageSettings, setStorageSettings] = useState<DownloadStorageSettingsSnapshot | null>(null);
  const [storageLoading, setStorageLoading] = useState(false);
  const [storageSaving, setStorageSaving] = useState(false);
  const [storageError, setStorageError] = useState<string | null>(null);
  const [storageSuccess, setStorageSuccess] = useState<string | null>(null);
  const [storageForm, setStorageForm] = useState<StorageFormValues>(buildDefaultStorageForm());
  const [credentialsForm, setCredentialsForm] = useState<CredentialsFormValues>(buildDefaultCredentialsForm());

  // Artifacts state
  const [artifactsLoading, setArtifactsLoading] = useState(false);
  const [artifactsError, setArtifactsError] = useState<string | null>(null);
  const [artifactsQuery, setArtifactsQuery] = useState('');
  const [artifactsPlatform, setArtifactsPlatform] = useState<PlatformKey | ''>('');
  const [artifactsAppKey, setArtifactsAppKey] = useState<string>('');
  const [artifacts, setArtifacts] = useState<DownloadArtifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState<DownloadArtifact | null>(null);
  const [applyTarget, setApplyTarget] = useState<ApplyTargetState>({
    appKey: '',
    platform: 'windows',
    requiresEntitlement: false,
    releaseVersion: '',
    releaseNotes: '',
  });

  // Upload state
  const [uploadState, setUploadState] = useState<UploadState>({
    file: null,
    platform: '',
    releaseVersion: '',
    appKey: '',
    busy: false,
    message: '',
    error: '',
  });

  const loadStorage = useCallback(async () => {
    setStorageLoading(true);
    setStorageError(null);
    setStorageSuccess(null);
    try {
      const { settings } = await getDownloadStorageAdmin();
      setStorageSettings(settings);
      setStorageForm({
        bucket: settings.bucket ?? '',
        region: settings.region ?? '',
        endpoint: settings.endpoint ?? '',
        forcePathStyle: settings.force_path_style ?? false,
        defaultPrefix: settings.default_prefix ?? '',
        signedUrlTtlSeconds: settings.signed_url_ttl_seconds ?? 900,
        publicBaseUrl: settings.public_base_url ?? '',
      });
      setCredentialsForm((prev) => ({
        ...prev,
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearAccessKeyId: false,
        clearSecretAccessKey: false,
        clearSessionToken: false,
      }));
    } catch (err) {
      setStorageError(err instanceof Error ? err.message : 'Failed to load storage settings');
    } finally {
      setStorageLoading(false);
    }
  }, []);

  const loadArtifacts = useCallback(async () => {
    setArtifactsLoading(true);
    setArtifactsError(null);
    try {
      // Use by-app endpoint if app key is filtered, otherwise use general list
      const response = artifactsAppKey
        ? await listDownloadArtifactsByAppAdmin({
            app_key: artifactsAppKey,
            platform: artifactsPlatform || undefined,
            page_size: 50,
          })
        : await listDownloadArtifactsAdmin({
            query: artifactsQuery.trim() || undefined,
            platform: artifactsPlatform || undefined,
            app_key: artifactsAppKey || undefined,
            page_size: 50,
          });
      setArtifacts(response.artifacts ?? []);
    } catch (err) {
      setArtifactsError(err instanceof Error ? err.message : 'Failed to load artifacts');
    } finally {
      setArtifactsLoading(false);
    }
  }, [artifactsPlatform, artifactsQuery, artifactsAppKey]);

  useEffect(() => {
    if (activeTab !== 'hosting') return;
    void loadStorage();
    void loadArtifacts();
  }, [activeTab, loadArtifacts, loadStorage]);

  const handleSaveStorage = useCallback(async () => {
    setStorageSaving(true);
    setStorageError(null);
    setStorageSuccess(null);
    try {
      const payload = buildStorageUpdatePayload(storageForm, credentialsForm);
      const { settings } = await updateDownloadStorageAdmin(payload);
      setStorageSettings(settings);
      setStorageSuccess('Saved storage settings.');
    } catch (err) {
      setStorageError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setStorageSaving(false);
    }
  }, [storageForm, credentialsForm]);

  const handleTestStorage = useCallback(async () => {
    setStorageError(null);
    setStorageSuccess(null);
    try {
      await testDownloadStorageAdmin();
      setStorageSuccess('Connection test succeeded.');
    } catch (err) {
      setStorageError(err instanceof Error ? err.message : 'Connection test failed');
    }
  }, []);

  const handleUploadArtifact = useCallback(async () => {
    if (!uploadState.file) {
      setUploadState((prev) => ({ ...prev, error: 'Choose a file first.' }));
      return;
    }
    setUploadState((prev) => ({ ...prev, busy: true, error: '', message: '' }));
    try {
      const presign = await presignDownloadArtifactUploadAdmin({
        filename: uploadState.file.name,
        content_type: uploadState.file.type || 'application/octet-stream',
        app_key: uploadState.appKey.trim() || undefined,
        platform: uploadState.platform || undefined,
        release_version: uploadState.releaseVersion.trim() || undefined,
      });
      const headers = new Headers();
      Object.entries(presign.required_headers ?? {}).forEach(([key, value]) => {
        if (key.toLowerCase() === 'host') return;
        headers.set(key, value);
      });
      if (!headers.has('Content-Type')) {
        headers.set('Content-Type', uploadState.file.type || 'application/octet-stream');
      }
      const uploadResp = await fetch(presign.upload_url, { method: 'PUT', headers, body: uploadState.file });
      if (!uploadResp.ok) throw new Error(`Upload failed (${uploadResp.status})`);
      await commitDownloadArtifactAdmin({
        bucket: presign.bucket,
        object_key: presign.object_key,
        original_filename: uploadState.file.name,
        content_type: uploadState.file.type || undefined,
        platform: uploadState.platform || undefined,
        release_version: uploadState.releaseVersion.trim() || undefined,
      });
      setUploadState((prev) => ({ ...prev, busy: false, file: null, message: 'Upload committed.', error: '' }));
      await loadArtifacts();
    } catch (err) {
      setUploadState((prev) => ({ ...prev, busy: false, error: err instanceof Error ? err.message : 'Upload failed' }));
    }
  }, [uploadState.file, uploadState.appKey, uploadState.platform, uploadState.releaseVersion, loadArtifacts]);

  const handleApplyArtifact = useCallback(async () => {
    if (!selectedArtifact) return;
    if (!applyTarget.appKey.trim()) {
      setArtifactsError('Select an app to apply to.');
      return;
    }
    try {
      await applyDownloadArtifactAdmin({
        app_key: applyTarget.appKey,
        platform: applyTarget.platform,
        artifact_id: selectedArtifact.id,
        release_version: applyTarget.releaseVersion.trim() || undefined,
        release_notes: applyTarget.releaseNotes.trim() || undefined,
        requires_entitlement: applyTarget.requiresEntitlement,
      });
      setSelectedArtifact(null);
      await loadApps();
      setStorageSuccess('Applied artifact to download asset.');
    } catch (err) {
      setArtifactsError(err instanceof Error ? err.message : 'Failed to apply artifact');
    }
  }, [selectedArtifact, applyTarget, loadApps]);

  const handleSetArtifactAsCurrent = useCallback(async (artifact: DownloadArtifact, appKey: string, platform: PlatformKey) => {
    try {
      await setArtifactAsCurrentAdmin({
        artifact_id: artifact.id,
        app_key: appKey,
        platform,
      });
      await loadArtifacts();
      await loadApps();
      setStorageSuccess(`Set ${artifact.original_filename || artifact.object_key} as the latest version.`);
    } catch (err) {
      setArtifactsError(err instanceof Error ? err.message : 'Failed to set artifact as current');
    }
  }, [loadArtifacts, loadApps]);

  return {
    // Storage
    storageSettings,
    storageLoading,
    storageSaving,
    storageError,
    storageSuccess,
    storageForm,
    setStorageForm,
    credentialsForm,
    setCredentialsForm,
    loadStorage,
    handleSaveStorage,
    handleTestStorage,

    // Artifacts
    artifactsLoading,
    artifactsError,
    setArtifactsError,
    artifactsQuery,
    setArtifactsQuery,
    artifactsPlatform,
    setArtifactsPlatform,
    artifactsAppKey,
    setArtifactsAppKey,
    artifacts,
    selectedArtifact,
    setSelectedArtifact,
    applyTarget,
    setApplyTarget,
    loadArtifacts,
    handleApplyArtifact,
    handleSetArtifactAsCurrent,

    // Upload
    uploadState,
    setUploadState,
    handleUploadArtifact,
  };
}
