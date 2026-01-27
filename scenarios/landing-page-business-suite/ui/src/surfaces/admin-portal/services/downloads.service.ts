import type {
  DownloadApp,
  DownloadAppInput,
  DownloadAsset,
  DownloadStorefront,
  DownloadStorageSettingsUpdate,
} from '../../../shared/api';
import { isFormDirtyNormalized } from '../../../shared/lib/formUtils';

/**
 * Platform keys supported for desktop installers
 */
export type PlatformKey = 'windows' | 'mac' | 'linux';

/**
 * Platform form values for a single platform (Windows/Mac/Linux)
 */
export interface PlatformFormValues {
  platform: PlatformKey;
  enabled: boolean;
  artifactSource: 'direct' | 'managed';
  artifactUrl: string;
  artifactId: string;
  releaseVersion: string;
  releaseNotes: string;
  requiresEntitlement: boolean;
  sizeMb: string;
}

/**
 * App form values for the download settings form
 */
export interface AppFormValues {
  appKey: string;
  name: string;
  tagline: string;
  description: string;
  iconUrl: string;
  screenshotUrl: string;
  installOverview: string;
  installSteps: string;
  displayOrder: number;
  appleEnabled: boolean;
  appleLabel: string;
  appleUrl: string;
  appleBadge: string;
  googleEnabled: boolean;
  googleLabel: string;
  googleUrl: string;
  googleBadge: string;
  platforms: Record<PlatformKey, PlatformFormValues>;
}

/**
 * All platform keys in order
 */
export const PLATFORM_KEYS: PlatformKey[] = ['windows', 'mac', 'linux'];

/**
 * Build platform form values from a download asset
 *
 * A platform is considered enabled if it has content or explicitly enabled metadata.
 *
 * @param platform - The platform key (windows/mac/linux)
 * @param asset - Optional existing download asset data
 * @returns Platform form values for the form
 */
export function buildPlatformForm(platform: PlatformKey, asset?: DownloadAsset): PlatformFormValues {
  const hasContent = Boolean(asset?.artifact_url || asset?.artifact_id || asset?.release_version);
  const explicitEnabled = asset?.metadata?.enabled;
  const enabled = explicitEnabled !== undefined ? Boolean(explicitEnabled) : hasContent;
  const artifactSource = asset?.artifact_source ?? (asset?.artifact_id ? 'managed' : 'direct');

  return {
    platform,
    enabled,
    artifactSource,
    artifactUrl: asset?.artifact_url ?? '',
    artifactId: asset?.artifact_id ? String(asset.artifact_id) : '',
    releaseVersion: asset?.release_version ?? '',
    releaseNotes: asset?.release_notes ?? '',
    requiresEntitlement: asset?.requires_entitlement ?? false,
    sizeMb: asset?.metadata?.size_mb ? String(asset.metadata.size_mb) : '',
  };
}

/**
 * Deserialize a DownloadApp from the API into form values
 *
 * @param app - Download app from the API
 * @returns Form values for editing
 */
export function deserializeApp(app: DownloadApp): AppFormValues {
  const appleStore = app.storefronts?.find((store) => store.store === 'app_store');
  const googleStore = app.storefronts?.find((store) => store.store === 'play_store');

  const platformMap: Record<PlatformKey, PlatformFormValues> = PLATFORM_KEYS.reduce((acc, key) => {
    const asset = app.platforms?.find((platform) => platform.platform === key);
    acc[key] = buildPlatformForm(key, asset);
    return acc;
  }, {} as Record<PlatformKey, PlatformFormValues>);

  // Storefront enabled status: true if URL exists
  const appleEnabled = appleStore ? Boolean(appleStore.url) : false;
  const googleEnabled = googleStore ? Boolean(googleStore.url) : false;

  return {
    appKey: app.app_key,
    name: app.name ?? '',
    tagline: app.tagline ?? '',
    description: app.description ?? '',
    iconUrl: app.icon_url ?? '',
    screenshotUrl: app.screenshot_url ?? '',
    installOverview: app.install_overview ?? '',
    installSteps: (app.install_steps ?? []).join('\n'),
    displayOrder: app.display_order ?? 0,
    appleEnabled,
    appleLabel: appleStore?.label ?? 'App Store',
    appleUrl: appleStore?.url ?? '',
    appleBadge: appleStore?.badge ?? '',
    googleEnabled,
    googleLabel: googleStore?.label ?? 'Google Play',
    googleUrl: googleStore?.url ?? '',
    googleBadge: googleStore?.badge ?? '',
    platforms: platformMap,
  };
}

/**
 * Build default app form values for a new app
 *
 * @param appKey - Optional initial app key
 * @returns Default form values
 */
export function buildDefaultAppValues(appKey = ''): AppFormValues {
  const platforms = PLATFORM_KEYS.reduce(
    (acc, platform) => ({
      ...acc,
      [platform]: buildPlatformForm(platform),
    }),
    {} as Record<PlatformKey, PlatformFormValues>,
  );

  return {
    appKey,
    name: '',
    tagline: '',
    description: '',
    iconUrl: '',
    screenshotUrl: '',
    installOverview: '',
    installSteps: '',
    displayOrder: 0,
    appleEnabled: false,
    appleLabel: 'App Store',
    appleUrl: '',
    appleBadge: '',
    googleEnabled: false,
    googleLabel: 'Google Play',
    googleUrl: '',
    googleBadge: '',
    platforms,
  };
}

/**
 * Serialize app form values to API input format
 *
 * @param values - Form values to serialize
 * @returns API input format for creating/updating an app
 */
export function serializeApp(values: AppFormValues): DownloadAppInput {
  const storefronts: DownloadStorefront[] = [];

  // Only include storefronts that are enabled AND have a URL
  if (values.appleEnabled && values.appleUrl.trim()) {
    storefronts.push({
      store: 'app_store',
      label: values.appleLabel.trim() || 'App Store',
      url: values.appleUrl.trim(),
      badge: values.appleBadge.trim() || undefined,
    });
  }
  if (values.googleEnabled && values.googleUrl.trim()) {
    storefronts.push({
      store: 'play_store',
      label: values.googleLabel.trim() || 'Google Play',
      url: values.googleUrl.trim(),
      badge: values.googleBadge.trim() || undefined,
    });
  }

  const installSteps = values.installSteps
    .split('\n')
    .map((step) => step.trim())
    .filter(Boolean);

  // Only include platforms that are enabled AND have required fields
  const platforms = PLATFORM_KEYS.map((key) => {
    const entry = values.platforms[key];
    const artifactSource = entry.artifactSource;
    return {
      platform: entry.platform,
      artifact_source: artifactSource,
      artifact_id: artifactSource === 'managed' ? Number(entry.artifactId) || undefined : undefined,
      artifact_url: artifactSource === 'direct' ? entry.artifactUrl.trim() : '',
      release_version: entry.releaseVersion.trim(),
      release_notes: entry.releaseNotes.trim(),
      requires_entitlement: entry.requiresEntitlement,
      metadata: {
        ...(entry.sizeMb.trim() ? { size_mb: Number(entry.sizeMb) } : {}),
        enabled: entry.enabled,
      },
    };
  }).filter((platform) => {
    if (!platform.metadata.enabled) return false;
    if (!platform.release_version?.length) return false;
    if (platform.artifact_source === 'managed') return Boolean(platform.artifact_id);
    return platform.artifact_url.length > 0;
  });

  return {
    app_key: values.appKey.trim(),
    name: values.name.trim(),
    tagline: values.tagline.trim(),
    description: values.description.trim(),
    icon_url: values.iconUrl.trim() || undefined,
    screenshot_url: values.screenshotUrl.trim() || undefined,
    install_overview: values.installOverview.trim(),
    install_steps: installSteps,
    display_order: values.displayOrder,
    storefronts,
    platforms,
  };
}

/**
 * Normalize app form values to a comparable payload format
 *
 * Used for dirty checking - comparing current values to original values.
 *
 * @param values - Form values to normalize
 * @returns Normalized payload for comparison
 */
export function normalizePayload(values: AppFormValues): DownloadAppInput {
  return serializeApp(values);
}

/**
 * Check if form values have changed from original values
 *
 * @param current - Current form values
 * @param original - Original form values
 * @returns True if the form has unsaved changes
 */
export function isFormDirty(current: AppFormValues, original: AppFormValues): boolean {
  return isFormDirtyNormalized(current, original, normalizePayload);
}

/**
 * Compute download health metrics from form states
 *
 * @param forms - Array of form states with values
 * @returns Health metrics for the downloads section
 */
export function computeDownloadHealthFromForms(
  forms: Array<{ values: AppFormValues }>
): {
  appCount: number;
  platformsConfigured: number;
  platformsMissing: number;
  storefrontsConfigured: number;
  hasApps: boolean;
} {
  const appCount = forms.length;
  let platformsConfigured = 0;
  let platformsMissing = 0;
  let storefrontsConfigured = 0;

  forms.forEach((form) => {
    PLATFORM_KEYS.forEach((platform) => {
      const p = form.values.platforms[platform];
      const configured =
        Boolean(p.releaseVersion) &&
        (p.artifactSource === 'managed' ? Boolean(p.artifactId) : Boolean(p.artifactUrl));
      if (configured) {
        platformsConfigured++;
      } else {
        platformsMissing++;
      }
    });
    if (form.values.appleUrl) storefrontsConfigured++;
    if (form.values.googleUrl) storefrontsConfigured++;
  });

  return {
    appCount,
    platformsConfigured,
    platformsMissing,
    storefrontsConfigured,
    hasApps: appCount > 0,
  };
}

/**
 * Storage form values for the download hosting settings
 */
export interface StorageFormValues {
  bucket: string;
  region: string;
  endpoint: string;
  forcePathStyle: boolean;
  defaultPrefix: string;
  signedUrlTtlSeconds: number;
  publicBaseUrl: string;
}

/**
 * Credentials form values for storage authentication
 */
export interface CredentialsFormValues {
  accessKeyId: string;
  secretAccessKey: string;
  sessionToken: string;
  clearAccessKeyId: boolean;
  clearSecretAccessKey: boolean;
  clearSessionToken: boolean;
}

/**
 * Build default storage form values
 *
 * @returns Default storage form values
 */
export function buildDefaultStorageForm(): StorageFormValues {
  return {
    bucket: '',
    region: '',
    endpoint: '',
    forcePathStyle: false,
    defaultPrefix: '',
    signedUrlTtlSeconds: 900,
    publicBaseUrl: '',
  };
}

/**
 * Build default credentials form values
 *
 * @returns Default credentials form values
 */
export function buildDefaultCredentialsForm(): CredentialsFormValues {
  return {
    accessKeyId: '',
    secretAccessKey: '',
    sessionToken: '',
    clearAccessKeyId: false,
    clearSecretAccessKey: false,
    clearSessionToken: false,
  };
}

/**
 * Build storage update payload from form values
 *
 * @param storageForm - Storage form values
 * @param credentialsForm - Credentials form values
 * @returns Payload for updating storage settings
 */
export function buildStorageUpdatePayload(
  storageForm: StorageFormValues,
  credentialsForm: CredentialsFormValues
): DownloadStorageSettingsUpdate {
  const payload: DownloadStorageSettingsUpdate = {
    provider: 's3',
    bucket: storageForm.bucket,
    region: storageForm.region,
    endpoint: storageForm.endpoint,
    force_path_style: storageForm.forcePathStyle,
    default_prefix: storageForm.defaultPrefix,
    signed_url_ttl_seconds: storageForm.signedUrlTtlSeconds,
    public_base_url: storageForm.publicBaseUrl,
  };

  if (credentialsForm.clearAccessKeyId) {
    payload.access_key_id = '';
  } else if (credentialsForm.accessKeyId.trim()) {
    payload.access_key_id = credentialsForm.accessKeyId.trim();
  }

  if (credentialsForm.clearSecretAccessKey) {
    payload.secret_access_key = '';
  } else if (credentialsForm.secretAccessKey.trim()) {
    payload.secret_access_key = credentialsForm.secretAccessKey.trim();
  }

  if (credentialsForm.clearSessionToken) {
    payload.session_token = '';
  } else if (credentialsForm.sessionToken.trim()) {
    payload.session_token = credentialsForm.sessionToken.trim();
  }

  return payload;
}
