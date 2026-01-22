import { describe, it, expect } from 'vitest';
import type { DownloadApp, DownloadAsset } from '../../../shared/api';
import {
  buildPlatformForm,
  deserializeApp,
  buildDefaultAppValues,
  serializeApp,
  normalizePayload,
  isFormDirty,
  computeDownloadHealthFromForms,
  buildDefaultStorageForm,
  buildDefaultCredentialsForm,
  buildStorageUpdatePayload,
  PLATFORM_KEYS,
  type AppFormValues,
  type StorageFormValues,
  type CredentialsFormValues,
} from './downloads.service';

// Helper to create partial test assets
const createAsset = (overrides: Partial<DownloadAsset>): DownloadAsset => ({
  bundle_key: 'test-bundle',
  app_key: 'test-app',
  platform: 'windows',
  artifact_url: '',
  release_version: '',
  requires_entitlement: false,
  ...overrides,
} as DownloadAsset);

// Helper to create partial test apps
const createApp = (overrides: Partial<DownloadApp>): DownloadApp => ({
  bundle_key: 'test-bundle',
  app_key: 'test-app',
  name: 'Test App',
  platforms: [],
  ...overrides,
} as DownloadApp);

describe('downloads.service', () => {
  describe('PLATFORM_KEYS', () => {
    it('contains all three platforms in order', () => {
      expect(PLATFORM_KEYS).toEqual(['windows', 'mac', 'linux']);
    });
  });

  describe('buildPlatformForm', () => {
    it('returns default values when no asset provided', () => {
      const result = buildPlatformForm('windows');

      expect(result).toEqual({
        platform: 'windows',
        enabled: false,
        artifactSource: 'direct',
        artifactUrl: '',
        artifactId: '',
        releaseVersion: '',
        releaseNotes: '',
        requiresEntitlement: false,
        sizeMb: '',
      });
    });

    it('enables platform when asset has content', () => {
      const asset = createAsset({
        platform: 'mac',
        artifact_url: 'https://example.com/app.dmg',
        release_version: '1.0.0',
      });

      const result = buildPlatformForm('mac', asset);

      expect(result.enabled).toBe(true);
      expect(result.artifactUrl).toBe('https://example.com/app.dmg');
      expect(result.releaseVersion).toBe('1.0.0');
    });

    it('respects explicit enabled metadata', () => {
      const asset = createAsset({
        platform: 'windows',
        artifact_url: 'https://example.com/app.exe',
        metadata: { enabled: false },
      });

      const result = buildPlatformForm('windows', asset);

      expect(result.enabled).toBe(false);
    });

    it('sets artifactSource to managed when artifact_id is present', () => {
      const asset = createAsset({
        platform: 'linux',
        artifact_id: 42,
        release_version: '2.0.0',
      });

      const result = buildPlatformForm('linux', asset);

      expect(result.artifactSource).toBe('managed');
      expect(result.artifactId).toBe('42');
    });

    it('uses artifact_source from asset when provided', () => {
      const asset = createAsset({
        platform: 'windows',
        artifact_source: 'managed',
        artifact_id: 123,
      });

      const result = buildPlatformForm('windows', asset);

      expect(result.artifactSource).toBe('managed');
    });

    it('parses size_mb from metadata', () => {
      const asset = createAsset({
        platform: 'mac',
        metadata: { size_mb: 150 },
      });

      const result = buildPlatformForm('mac', asset);

      expect(result.sizeMb).toBe('150');
    });

    it('handles requiresEntitlement flag', () => {
      const asset = createAsset({
        platform: 'linux',
        requires_entitlement: true,
      });

      const result = buildPlatformForm('linux', asset);

      expect(result.requiresEntitlement).toBe(true);
    });
  });

  describe('deserializeApp', () => {
    it('deserializes basic app fields', () => {
      const app = createApp({
        app_key: 'test-app',
        name: 'Test App',
        tagline: 'A great app',
        description: 'Full description',
        display_order: 1,
      });

      const result = deserializeApp(app);

      expect(result.appKey).toBe('test-app');
      expect(result.name).toBe('Test App');
      expect(result.tagline).toBe('A great app');
      expect(result.description).toBe('Full description');
      expect(result.displayOrder).toBe(1);
    });

    it('deserializes icon and screenshot URLs', () => {
      const app = createApp({
        app_key: 'test',
        icon_url: 'https://example.com/icon.png',
        screenshot_url: 'https://example.com/screenshot.png',
      });

      const result = deserializeApp(app);

      expect(result.iconUrl).toBe('https://example.com/icon.png');
      expect(result.screenshotUrl).toBe('https://example.com/screenshot.png');
    });

    it('deserializes install steps as newline-separated string', () => {
      const app = createApp({
        app_key: 'test',
        install_overview: 'Easy to install',
        install_steps: ['Step 1', 'Step 2', 'Step 3'],
      });

      const result = deserializeApp(app);

      expect(result.installOverview).toBe('Easy to install');
      expect(result.installSteps).toBe('Step 1\nStep 2\nStep 3');
    });

    it('deserializes Apple App Store storefront', () => {
      const app = createApp({
        app_key: 'test',
        storefronts: [
          {
            store: 'app_store',
            label: 'Download on App Store',
            url: 'https://apps.apple.com/app/123',
            badge: 'Featured',
          },
        ],
      });

      const result = deserializeApp(app);

      expect(result.appleEnabled).toBe(true);
      expect(result.appleLabel).toBe('Download on App Store');
      expect(result.appleUrl).toBe('https://apps.apple.com/app/123');
      expect(result.appleBadge).toBe('Featured');
    });

    it('deserializes Google Play storefront', () => {
      const app = createApp({
        app_key: 'test',
        storefronts: [
          {
            store: 'play_store',
            label: 'Get it on Google Play',
            url: 'https://play.google.com/store/apps/123',
          },
        ],
      });

      const result = deserializeApp(app);

      expect(result.googleEnabled).toBe(true);
      expect(result.googleLabel).toBe('Get it on Google Play');
      expect(result.googleUrl).toBe('https://play.google.com/store/apps/123');
    });

    it('deserializes platforms', () => {
      const app = createApp({
        app_key: 'test',
        platforms: [
          createAsset({
            platform: 'windows',
            artifact_url: 'https://example.com/app.exe',
            release_version: '1.0.0',
          }),
        ],
      });

      const result = deserializeApp(app);

      expect(result.platforms.windows.artifactUrl).toBe('https://example.com/app.exe');
      expect(result.platforms.windows.releaseVersion).toBe('1.0.0');
      expect(result.platforms.mac.artifactUrl).toBe('');
      expect(result.platforms.linux.artifactUrl).toBe('');
    });

    it('handles empty/null values gracefully', () => {
      const app = createApp({
        app_key: 'test',
        name: null as unknown as string,
        storefronts: undefined,
        platforms: undefined as unknown as DownloadAsset[],
      });

      const result = deserializeApp(app);

      expect(result.name).toBe('');
      expect(result.appleEnabled).toBe(false);
      expect(result.googleEnabled).toBe(false);
      expect(result.platforms.windows.enabled).toBe(false);
    });
  });

  describe('buildDefaultAppValues', () => {
    it('returns default values with empty appKey', () => {
      const result = buildDefaultAppValues();

      expect(result.appKey).toBe('');
      expect(result.name).toBe('');
      expect(result.appleEnabled).toBe(false);
      expect(result.googleEnabled).toBe(false);
    });

    it('uses provided appKey', () => {
      const result = buildDefaultAppValues('my-app');

      expect(result.appKey).toBe('my-app');
    });

    it('initializes all platforms with defaults', () => {
      const result = buildDefaultAppValues();

      expect(result.platforms.windows.platform).toBe('windows');
      expect(result.platforms.mac.platform).toBe('mac');
      expect(result.platforms.linux.platform).toBe('linux');
      expect(result.platforms.windows.enabled).toBe(false);
    });

    it('sets default storefront labels', () => {
      const result = buildDefaultAppValues();

      expect(result.appleLabel).toBe('App Store');
      expect(result.googleLabel).toBe('Google Play');
    });
  });

  describe('serializeApp', () => {
    it('serializes basic fields', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues('test-app'),
        name: 'Test App',
        tagline: 'A tagline',
        description: 'Description',
      };

      const result = serializeApp(values);

      expect(result.app_key).toBe('test-app');
      expect(result.name).toBe('Test App');
      expect(result.tagline).toBe('A tagline');
      expect(result.description).toBe('Description');
    });

    it('trims whitespace from fields', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        appKey: '  test-app  ',
        name: '  Test App  ',
      };

      const result = serializeApp(values);

      expect(result.app_key).toBe('test-app');
      expect(result.name).toBe('Test App');
    });

    it('omits icon_url and screenshot_url when empty', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        iconUrl: '',
        screenshotUrl: '   ',
      };

      const result = serializeApp(values);

      expect(result.icon_url).toBeUndefined();
      expect(result.screenshot_url).toBeUndefined();
    });

    it('parses install_steps from newline-separated string', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        installSteps: 'Step 1\n  Step 2  \n\nStep 3',
      };

      const result = serializeApp(values);

      expect(result.install_steps).toEqual(['Step 1', 'Step 2', 'Step 3']);
    });

    it('includes App Store storefront when enabled with URL', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        appleEnabled: true,
        appleUrl: 'https://apps.apple.com/123',
        appleLabel: 'Get on App Store',
        appleBadge: 'New',
      };

      const result = serializeApp(values);

      expect(result.storefronts).toHaveLength(1);
      expect(result.storefronts![0]).toEqual({
        store: 'app_store',
        label: 'Get on App Store',
        url: 'https://apps.apple.com/123',
        badge: 'New',
      });
    });

    it('excludes storefront when enabled but URL is empty', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        appleEnabled: true,
        appleUrl: '',
      };

      const result = serializeApp(values);

      expect(result.storefronts).toHaveLength(0);
    });

    it('excludes badge when empty', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        googleEnabled: true,
        googleUrl: 'https://play.google.com/123',
        googleBadge: '  ',
      };

      const result = serializeApp(values);

      expect(result.storefronts![0].badge).toBeUndefined();
    });

    it('includes enabled platforms with required fields', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        platforms: {
          windows: {
            platform: 'windows',
            enabled: true,
            artifactSource: 'direct',
            artifactUrl: 'https://example.com/app.exe',
            artifactId: '',
            releaseVersion: '1.0.0',
            releaseNotes: 'Initial release',
            requiresEntitlement: false,
            sizeMb: '100',
          },
          mac: buildPlatformForm('mac'),
          linux: buildPlatformForm('linux'),
        },
      };

      const result = serializeApp(values);

      expect(result.platforms).toHaveLength(1);
      expect(result.platforms[0]).toMatchObject({
        platform: 'windows',
        artifact_source: 'direct',
        artifact_url: 'https://example.com/app.exe',
        release_version: '1.0.0',
        release_notes: 'Initial release',
        requires_entitlement: false,
        metadata: { size_mb: 100, enabled: true },
      });
    });

    it('excludes disabled platforms', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        platforms: {
          windows: {
            platform: 'windows',
            enabled: false,
            artifactSource: 'direct',
            artifactUrl: 'https://example.com/app.exe',
            artifactId: '',
            releaseVersion: '1.0.0',
            releaseNotes: '',
            requiresEntitlement: false,
            sizeMb: '',
          },
          mac: buildPlatformForm('mac'),
          linux: buildPlatformForm('linux'),
        },
      };

      const result = serializeApp(values);

      expect(result.platforms).toHaveLength(0);
    });

    it('excludes platforms without release version', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        platforms: {
          windows: {
            platform: 'windows',
            enabled: true,
            artifactSource: 'direct',
            artifactUrl: 'https://example.com/app.exe',
            artifactId: '',
            releaseVersion: '',
            releaseNotes: '',
            requiresEntitlement: false,
            sizeMb: '',
          },
          mac: buildPlatformForm('mac'),
          linux: buildPlatformForm('linux'),
        },
      };

      const result = serializeApp(values);

      expect(result.platforms).toHaveLength(0);
    });

    it('uses artifact_id for managed platforms', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        platforms: {
          windows: {
            platform: 'windows',
            enabled: true,
            artifactSource: 'managed',
            artifactUrl: '',
            artifactId: '42',
            releaseVersion: '1.0.0',
            releaseNotes: '',
            requiresEntitlement: false,
            sizeMb: '',
          },
          mac: buildPlatformForm('mac'),
          linux: buildPlatformForm('linux'),
        },
      };

      const result = serializeApp(values);

      expect(result.platforms).toHaveLength(1);
      expect(result.platforms[0].artifact_id).toBe(42);
      expect(result.platforms[0].artifact_url).toBe('');
    });

    it('excludes managed platforms without artifact_id', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues(),
        platforms: {
          windows: {
            platform: 'windows',
            enabled: true,
            artifactSource: 'managed',
            artifactUrl: '',
            artifactId: '',
            releaseVersion: '1.0.0',
            releaseNotes: '',
            requiresEntitlement: false,
            sizeMb: '',
          },
          mac: buildPlatformForm('mac'),
          linux: buildPlatformForm('linux'),
        },
      };

      const result = serializeApp(values);

      expect(result.platforms).toHaveLength(0);
    });
  });

  describe('normalizePayload', () => {
    it('delegates to serializeApp', () => {
      const values = buildDefaultAppValues('test');
      const serialized = serializeApp(values);
      const normalized = normalizePayload(values);

      expect(normalized).toEqual(serialized);
    });
  });

  describe('isFormDirty', () => {
    it('returns false when values are identical', () => {
      const original = buildDefaultAppValues('test');
      const current = { ...original };

      expect(isFormDirty(current, original)).toBe(false);
    });

    it('returns true when name changes', () => {
      const original = buildDefaultAppValues('test');
      const current = { ...original, name: 'Changed Name' };

      expect(isFormDirty(current, original)).toBe(true);
    });

    it('returns true when platform is enabled', () => {
      const original = buildDefaultAppValues('test');
      const current: AppFormValues = {
        ...original,
        platforms: {
          ...original.platforms,
          windows: {
            ...original.platforms.windows,
            enabled: true,
            artifactUrl: 'https://example.com/app.exe',
            releaseVersion: '1.0.0',
          },
        },
      };

      expect(isFormDirty(current, original)).toBe(true);
    });

    it('returns false when only whitespace differs', () => {
      const original: AppFormValues = {
        ...buildDefaultAppValues('test'),
        name: 'App Name',
      };
      const current: AppFormValues = {
        ...original,
        name: '  App Name  ',
      };

      // After trimming, they should be equal
      expect(isFormDirty(current, original)).toBe(false);
    });
  });

  describe('computeDownloadHealthFromForms', () => {
    it('returns zero counts for empty forms', () => {
      const result = computeDownloadHealthFromForms([]);

      expect(result).toEqual({
        appCount: 0,
        platformsConfigured: 0,
        platformsMissing: 0,
        storefrontsConfigured: 0,
        hasApps: false,
      });
    });

    it('counts apps correctly', () => {
      const forms = [
        { values: buildDefaultAppValues('app1') },
        { values: buildDefaultAppValues('app2') },
      ];

      const result = computeDownloadHealthFromForms(forms);

      expect(result.appCount).toBe(2);
      expect(result.hasApps).toBe(true);
    });

    it('counts configured platforms', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues('test'),
        platforms: {
          windows: {
            platform: 'windows',
            enabled: true,
            artifactSource: 'direct',
            artifactUrl: 'https://example.com/app.exe',
            artifactId: '',
            releaseVersion: '1.0.0',
            releaseNotes: '',
            requiresEntitlement: false,
            sizeMb: '',
          },
          mac: buildPlatformForm('mac'),
          linux: buildPlatformForm('linux'),
        },
      };

      const result = computeDownloadHealthFromForms([{ values }]);

      expect(result.platformsConfigured).toBe(1);
      expect(result.platformsMissing).toBe(2);
    });

    it('counts storefronts from URL presence', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues('test'),
        appleUrl: 'https://apps.apple.com/123',
        googleUrl: 'https://play.google.com/123',
      };

      const result = computeDownloadHealthFromForms([{ values }]);

      expect(result.storefrontsConfigured).toBe(2);
    });

    it('counts managed platforms as configured', () => {
      const values: AppFormValues = {
        ...buildDefaultAppValues('test'),
        platforms: {
          windows: {
            platform: 'windows',
            enabled: true,
            artifactSource: 'managed',
            artifactUrl: '',
            artifactId: '42',
            releaseVersion: '1.0.0',
            releaseNotes: '',
            requiresEntitlement: false,
            sizeMb: '',
          },
          mac: buildPlatformForm('mac'),
          linux: buildPlatformForm('linux'),
        },
      };

      const result = computeDownloadHealthFromForms([{ values }]);

      expect(result.platformsConfigured).toBe(1);
    });
  });

  describe('buildDefaultStorageForm', () => {
    it('returns default storage form values', () => {
      const result = buildDefaultStorageForm();

      expect(result).toEqual({
        bucket: '',
        region: '',
        endpoint: '',
        forcePathStyle: false,
        defaultPrefix: '',
        signedUrlTtlSeconds: 900,
        publicBaseUrl: '',
      });
    });
  });

  describe('buildDefaultCredentialsForm', () => {
    it('returns default credentials form values', () => {
      const result = buildDefaultCredentialsForm();

      expect(result).toEqual({
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearAccessKeyId: false,
        clearSecretAccessKey: false,
        clearSessionToken: false,
      });
    });
  });

  describe('buildStorageUpdatePayload', () => {
    it('builds basic payload from storage form', () => {
      const storageForm: StorageFormValues = {
        bucket: 'my-bucket',
        region: 'us-east-1',
        endpoint: 'https://s3.example.com',
        forcePathStyle: true,
        defaultPrefix: 'downloads/',
        signedUrlTtlSeconds: 3600,
        publicBaseUrl: 'https://cdn.example.com',
      };
      const credentialsForm = buildDefaultCredentialsForm();

      const result = buildStorageUpdatePayload(storageForm, credentialsForm);

      expect(result).toEqual({
        provider: 's3',
        bucket: 'my-bucket',
        region: 'us-east-1',
        endpoint: 'https://s3.example.com',
        force_path_style: true,
        default_prefix: 'downloads/',
        signed_url_ttl_seconds: 3600,
        public_base_url: 'https://cdn.example.com',
      });
    });

    it('includes credentials when provided', () => {
      const storageForm = buildDefaultStorageForm();
      const credentialsForm: CredentialsFormValues = {
        accessKeyId: 'AKIAEXAMPLE',
        secretAccessKey: 'secretkey123',
        sessionToken: 'token123',
        clearAccessKeyId: false,
        clearSecretAccessKey: false,
        clearSessionToken: false,
      };

      const result = buildStorageUpdatePayload(storageForm, credentialsForm);

      expect(result.access_key_id).toBe('AKIAEXAMPLE');
      expect(result.secret_access_key).toBe('secretkey123');
      expect(result.session_token).toBe('token123');
    });

    it('clears credentials when clear flags are set', () => {
      const storageForm = buildDefaultStorageForm();
      const credentialsForm: CredentialsFormValues = {
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearAccessKeyId: true,
        clearSecretAccessKey: true,
        clearSessionToken: true,
      };

      const result = buildStorageUpdatePayload(storageForm, credentialsForm);

      expect(result.access_key_id).toBe('');
      expect(result.secret_access_key).toBe('');
      expect(result.session_token).toBe('');
    });

    it('trims credential values', () => {
      const storageForm = buildDefaultStorageForm();
      const credentialsForm: CredentialsFormValues = {
        accessKeyId: '  AKIAEXAMPLE  ',
        secretAccessKey: '  secret  ',
        sessionToken: '',
        clearAccessKeyId: false,
        clearSecretAccessKey: false,
        clearSessionToken: false,
      };

      const result = buildStorageUpdatePayload(storageForm, credentialsForm);

      expect(result.access_key_id).toBe('AKIAEXAMPLE');
      expect(result.secret_access_key).toBe('secret');
      expect(result.session_token).toBeUndefined();
    });

    it('omits empty credentials when not clearing', () => {
      const storageForm = buildDefaultStorageForm();
      const credentialsForm: CredentialsFormValues = {
        accessKeyId: '',
        secretAccessKey: '   ',
        sessionToken: '',
        clearAccessKeyId: false,
        clearSecretAccessKey: false,
        clearSessionToken: false,
      };

      const result = buildStorageUpdatePayload(storageForm, credentialsForm);

      expect(result.access_key_id).toBeUndefined();
      expect(result.secret_access_key).toBeUndefined();
      expect(result.session_token).toBeUndefined();
    });
  });
});
