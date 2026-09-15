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

/** Severity of an operator-authored signing notice. */
export type SigningNoticeSeverity = 'info' | 'warning';

/** Whether a platform inherits, overrides, or hides the app-level signing notice. */
export type SigningNoticeMode = 'inherit' | 'override' | 'hide';

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
  /** Inherit the app-level signing notice, override it, or hide it for this platform. */
  signingNoticeMode: SigningNoticeMode;
  signingNoticeTitle: string;
  signingNoticeBody: string;
  signingNoticeLinkLabel: string;
  signingNoticeLinkUrl: string;
  signingNoticeSeverity: SigningNoticeSeverity;
  // Read-only artifact metadata (populated from API when source is 'managed')
  artifactFilename?: string;
  artifactSizeBytes?: number;
  artifactCount?: number;
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
  /** Controls public visibility without deleting the app's catalog record. */
  enabled?: boolean;
  /** Preserve catalog metadata such as plugin publication and capability links. */
  metadata?: Record<string, unknown>;
  /** Optional same-origin or hosted URL for a live web app. */
  webUrl: string;
  /** Comma-separated monetization capabilities shown to operators. */
  featureGates: string;
  /** Operator-controlled readiness label, e.g. live, enabling, planned. */
  catalogStatus: string;
  /** Marks a catalog entry as an agent-facing plugin capability. */
  agentPlugin: boolean;
  /** App-level signing-pending notice shown as the default for every platform. */
  signingNoticeEnabled: boolean;
  signingNoticeTitle: string;
  signingNoticeBody: string;
  signingNoticeLinkLabel: string;
  signingNoticeLinkUrl: string;
  signingNoticeSeverity: SigningNoticeSeverity;
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

function isDownloadAsset(value: unknown): value is DownloadAsset {
  return typeof value === 'object' && value !== null && 'platform' in value;
}

function normalizeDownloadAssets(value: unknown): DownloadAsset[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter(isDownloadAsset);
}

function normalizeMetadataValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(normalizeMetadataValue);
  if (!value || typeof value !== 'object') return value;

  const record = value as Record<string, unknown>;
  if ('stringValue' in record || 'string_value' in record) return record.stringValue ?? record.string_value;
  if ('boolValue' in record || 'bool_value' in record) return record.boolValue ?? record.bool_value;
  if ('numberValue' in record || 'number_value' in record) return record.numberValue ?? record.number_value;
  if ('intValue' in record || 'int_value' in record) return record.intValue ?? record.int_value;
  if ('listValue' in record || 'list_value' in record) {
    const list = (record.listValue ?? record.list_value) as Record<string, unknown> | undefined;
    return Array.isArray(list?.values) ? list.values.map(normalizeMetadataValue) : [];
  }
  if ('structValue' in record || 'struct_value' in record) return normalizeMetadata(record.structValue ?? record.struct_value);
  if ('fields' in record) return normalizeMetadata(record);
  return value;
}

function normalizeMetadata(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  const record = value as Record<string, unknown>;
  const fields = record.fields;
  if (!fields || typeof fields !== 'object' || Array.isArray(fields)) return record;
  return Object.fromEntries(
    Object.entries(fields as Record<string, unknown>).map(([key, field]) => [key, normalizeMetadataValue(field)]),
  );
}

interface SigningNoticeFields {
  title: string;
  body: string;
  linkLabel: string;
  linkUrl: string;
  severity: SigningNoticeSeverity;
}

const EMPTY_SIGNING_NOTICE: SigningNoticeFields = { title: '', body: '', linkLabel: '', linkUrl: '', severity: 'info' };

function readString(record: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string') return value;
  }
  return '';
}

/**
 * Reads the operator-authored signing notice from normalized catalog metadata.
 * Absent copy means "inherit"; `enabled: false` means "hide".
 */
function readSigningNotice(metadata: Record<string, unknown>): { mode: SigningNoticeMode; values: SigningNoticeFields } {
  const raw = metadata.signing_notice;
  if (raw === undefined || raw === null) return { mode: 'inherit', values: { ...EMPTY_SIGNING_NOTICE } };
  const notice = normalizeMetadata(raw);
  if (Object.keys(notice).length === 0) return { mode: 'inherit', values: { ...EMPTY_SIGNING_NOTICE } };
  if (notice.enabled === false) return { mode: 'hide', values: { ...EMPTY_SIGNING_NOTICE } };
  return {
    mode: 'override',
    values: {
      title: readString(notice, 'title'),
      body: readString(notice, 'body'),
      linkLabel: readString(notice, 'link_label', 'linkLabel'),
      linkUrl: readString(notice, 'link_url', 'linkUrl'),
      severity: notice.severity === 'warning' ? 'warning' : 'info',
    },
  };
}

/** Serializes notice copy, returning undefined when required copy is missing. */
function serializeSigningNotice(values: SigningNoticeFields): Record<string, unknown> | undefined {
  const title = values.title.trim();
  const body = values.body.trim();
  if (!title || !body) return undefined;
  const notice: Record<string, unknown> = { enabled: true, title, body, severity: values.severity };
  const label = values.linkLabel.trim();
  if (label) notice.link_label = label;
  const url = values.linkUrl.trim();
  if (url) notice.link_url = url;
  return notice;
}

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
  const metadata = normalizeMetadata(asset?.metadata);
  const hasContent = Boolean(asset?.artifact_url || asset?.artifact_id || asset?.release_version);
  const explicitEnabled = metadata.enabled;
  const enabled = explicitEnabled !== undefined ? Boolean(explicitEnabled) : hasContent;
  const artifactSource = asset?.artifact_source ?? (asset?.artifact_id ? 'managed' : 'direct');
  const notice = readSigningNotice(metadata);

  return {
    platform,
    enabled,
    artifactSource,
    artifactUrl: asset?.artifact_url ?? '',
    artifactId: asset?.artifact_id ? String(asset.artifact_id) : '',
    releaseVersion: asset?.release_version ?? '',
    releaseNotes: asset?.release_notes ?? '',
    requiresEntitlement: asset?.requires_entitlement ?? false,
    signingNoticeMode: notice.mode,
    signingNoticeTitle: notice.values.title,
    signingNoticeBody: notice.values.body,
    signingNoticeLinkLabel: notice.values.linkLabel,
    signingNoticeLinkUrl: notice.values.linkUrl,
    signingNoticeSeverity: notice.values.severity,
    // Read-only artifact metadata from API
    artifactFilename: asset?.artifact_filename,
    artifactSizeBytes: asset?.artifact_size_bytes,
    artifactCount: asset?.artifact_count,
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
  // The generated contract marks these required, but persisted legacy records may
  // still omit them. Normalize the untrusted response at this boundary.
  const rawPlatforms: unknown = app.platforms;
  const platforms = normalizeDownloadAssets(rawPlatforms);
  const rawName: unknown = app.name;
  const metadata = normalizeMetadata(app.metadata);
  const signingNotice = readSigningNotice(metadata);

  const platformMap: Record<PlatformKey, PlatformFormValues> = PLATFORM_KEYS.reduce((acc, key) => {
    const asset = platforms.find((platform) => platform.platform === key);
    acc[key] = buildPlatformForm(key, asset);
    return acc;
  }, {} as Record<PlatformKey, PlatformFormValues>);

  // Storefront enabled status: true if URL exists
  const appleEnabled = appleStore ? Boolean(appleStore.url) : false;
  const googleEnabled = googleStore ? Boolean(googleStore.url) : false;

  return {
    appKey: app.app_key,
    name: typeof rawName === 'string' ? rawName : '',
    tagline: app.tagline ?? '',
    description: app.description ?? '',
    iconUrl: app.icon_url ?? '',
    screenshotUrl: app.screenshot_url ?? '',
    installOverview: app.install_overview ?? '',
    installSteps: (app.install_steps ?? []).join('\n'),
    displayOrder: app.display_order ?? 0,
    enabled: metadata.enabled !== false,
    metadata,
    webUrl: typeof metadata.web_url === 'string' ? metadata.web_url : '',
    featureGates: Array.isArray(metadata.feature_gates)
      ? metadata.feature_gates.filter((value): value is string => typeof value === 'string').join(', ')
      : '',
    catalogStatus: typeof metadata.catalog_status === 'string' ? metadata.catalog_status : '',
    agentPlugin: metadata.agent_plugin === true,
    signingNoticeEnabled: signingNotice.mode === 'override',
    signingNoticeTitle: signingNotice.values.title,
    signingNoticeBody: signingNotice.values.body,
    signingNoticeLinkLabel: signingNotice.values.linkLabel,
    signingNoticeLinkUrl: signingNotice.values.linkUrl,
    signingNoticeSeverity: signingNotice.values.severity,
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
    enabled: true,
    metadata: {},
    webUrl: '',
    featureGates: '',
    catalogStatus: 'planned',
    agentPlugin: false,
    signingNoticeEnabled: false,
    signingNoticeTitle: '',
    signingNoticeBody: '',
    signingNoticeLinkLabel: '',
    signingNoticeLinkUrl: '',
    signingNoticeSeverity: 'info',
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
    const platformMetadata: Record<string, unknown> = { enabled: entry.enabled };
    if (entry.signingNoticeMode === 'hide') {
      platformMetadata.signing_notice = { enabled: false };
    } else if (entry.signingNoticeMode === 'override') {
      const notice = serializeSigningNotice({ title: entry.signingNoticeTitle, body: entry.signingNoticeBody, linkLabel: entry.signingNoticeLinkLabel, linkUrl: entry.signingNoticeLinkUrl, severity: entry.signingNoticeSeverity });
      if (notice) platformMetadata.signing_notice = notice;
    }
    return {
      platform: entry.platform,
      artifact_source: artifactSource,
      artifact_id: artifactSource === 'managed' ? Number(entry.artifactId) || undefined : undefined,
      artifact_url: artifactSource === 'direct' ? entry.artifactUrl.trim() : '',
      release_version: entry.releaseVersion.trim(),
      release_notes: entry.releaseNotes.trim(),
      requires_entitlement: entry.requiresEntitlement,
      metadata: platformMetadata,
    };
  }).filter((platform) => {
    if (!platform.metadata.enabled) return false;
    if (!platform.release_version.length) return false;
    if (platform.artifact_source === 'managed') return Boolean(platform.artifact_id);
    return platform.artifact_url.length > 0;
  });

  const metadata: Record<string, unknown> = { ...(values.metadata ?? {}) };
  metadata.enabled = values.enabled !== false;
  const webUrl = values.webUrl.trim();
  if (webUrl) metadata.web_url = webUrl;
  else delete metadata.web_url;
  const featureGates = values.featureGates.split(',').map((gate) => gate.trim()).filter(Boolean);
  if (featureGates.length > 0) metadata.feature_gates = featureGates;
  else delete metadata.feature_gates;
  const catalogStatus = values.catalogStatus.trim();
  if (catalogStatus) metadata.catalog_status = catalogStatus;
  else delete metadata.catalog_status;
  if (values.agentPlugin) metadata.agent_plugin = true;
  else delete metadata.agent_plugin;
  if (values.signingNoticeEnabled) {
    const notice = serializeSigningNotice({ title: values.signingNoticeTitle, body: values.signingNoticeBody, linkLabel: values.signingNoticeLinkLabel, linkUrl: values.signingNoticeLinkUrl, severity: values.signingNoticeSeverity });
    if (notice) metadata.signing_notice = notice;
    else delete metadata.signing_notice;
  } else {
    delete metadata.signing_notice;
  }

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
    metadata,
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

/**
 * Storage provider identifiers for the wizard
 */
export type StorageProviderId = 'aws-s3' | 'cloudflare-r2' | 'minio' | 'custom';

/**
 * Provider preset configuration
 */
export interface ProviderPreset {
  region: string;
  endpoint: string;
  forcePathStyle: boolean;
}

/**
 * AWS regions for S3
 */
export const AWS_REGIONS = [
  { value: 'us-east-1', label: 'US East (N. Virginia)' },
  { value: 'us-east-2', label: 'US East (Ohio)' },
  { value: 'us-west-1', label: 'US West (N. California)' },
  { value: 'us-west-2', label: 'US West (Oregon)' },
  { value: 'eu-west-1', label: 'EU (Ireland)' },
  { value: 'eu-west-2', label: 'EU (London)' },
  { value: 'eu-west-3', label: 'EU (Paris)' },
  { value: 'eu-central-1', label: 'EU (Frankfurt)' },
  { value: 'ap-northeast-1', label: 'Asia Pacific (Tokyo)' },
  { value: 'ap-northeast-2', label: 'Asia Pacific (Seoul)' },
  { value: 'ap-southeast-1', label: 'Asia Pacific (Singapore)' },
  { value: 'ap-southeast-2', label: 'Asia Pacific (Sydney)' },
  { value: 'ap-south-1', label: 'Asia Pacific (Mumbai)' },
  { value: 'sa-east-1', label: 'South America (Sao Paulo)' },
  { value: 'ca-central-1', label: 'Canada (Central)' },
] as const;

/**
 * Get provider defaults based on provider ID
 */
export function getProviderDefaults(providerId: StorageProviderId): ProviderPreset {
  switch (providerId) {
    case 'aws-s3':
      return { region: 'us-east-1', endpoint: '', forcePathStyle: false };
    case 'cloudflare-r2':
      return { region: 'auto', endpoint: '', forcePathStyle: false };
    case 'minio':
      return { region: '', endpoint: '', forcePathStyle: true };
    case 'custom':
    default:
      return { region: '', endpoint: '', forcePathStyle: false };
  }
}

/**
 * Generate Cloudflare R2 endpoint from account ID
 */
export function generateR2Endpoint(accountId: string): string {
  if (!accountId.trim()) return '';
  return `https://${accountId.trim()}.r2.cloudflarestorage.com`;
}
