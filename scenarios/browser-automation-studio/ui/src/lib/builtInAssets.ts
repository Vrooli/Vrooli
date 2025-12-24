/**
 * Built-in assets that are always available regardless of user uploads.
 * These assets are bundled with the application and cannot be deleted.
 */

import type { BrandAsset } from './storage';

// Import the Vrooli logo from public assets
// This will be resolved by Vite to the correct path
const VROOLI_LOGO_URL = '/manifest-icon-192.maskable.png';

/**
 * Special asset ID prefix for built-in assets.
 * Used to distinguish from user-uploaded assets.
 */
export const BUILTIN_ASSET_PREFIX = 'builtin:';

/**
 * Built-in asset IDs
 */
export const BUILTIN_ASSET_IDS = {
  VROOLI_ASCENSION: `${BUILTIN_ASSET_PREFIX}vrooli-ascension`,
} as const;

export type BuiltInAssetId = (typeof BUILTIN_ASSET_IDS)[keyof typeof BUILTIN_ASSET_IDS];

/**
 * Check if an asset ID refers to a built-in asset
 */
export const isBuiltInAssetId = (id: string | null | undefined): id is BuiltInAssetId => {
  return typeof id === 'string' && id.startsWith(BUILTIN_ASSET_PREFIX);
};

/**
 * Built-in asset definitions
 */
export interface BuiltInAssetDefinition extends Omit<BrandAsset, 'data' | 'filePath' | 'createdAt' | 'updatedAt'> {
  /** URL to the asset (can be relative to public folder) */
  url: string;
  /** Whether this asset is the default for watermarks on free/solo tiers */
  isDefaultWatermark?: boolean;
  /** Description shown in the UI */
  description?: string;
}

/**
 * All built-in assets
 */
export const BUILTIN_ASSETS: Record<BuiltInAssetId, BuiltInAssetDefinition> = {
  [BUILTIN_ASSET_IDS.VROOLI_ASCENSION]: {
    id: BUILTIN_ASSET_IDS.VROOLI_ASCENSION,
    name: 'Vrooli Ascension',
    type: 'logo',
    mimeType: 'image/png',
    width: 192,
    height: 192,
    sizeBytes: 0, // Not tracked for built-ins
    url: VROOLI_LOGO_URL,
    thumbnail: VROOLI_LOGO_URL,
    isDefaultWatermark: true,
    description: 'The Vrooli brand logo - always available',
  },
};

/**
 * Get a built-in asset by ID
 */
export const getBuiltInAsset = (id: string): BuiltInAssetDefinition | null => {
  if (!isBuiltInAssetId(id)) return null;
  return BUILTIN_ASSETS[id] ?? null;
};

/**
 * Get all built-in assets as an array
 */
export const getAllBuiltInAssets = (): BuiltInAssetDefinition[] => {
  return Object.values(BUILTIN_ASSETS);
};

/**
 * Get built-in assets filtered by type
 */
export const getBuiltInAssetsByType = (type: 'logo' | 'background' | 'other'): BuiltInAssetDefinition[] => {
  return getAllBuiltInAssets().filter((asset) => asset.type === type);
};

/**
 * Get the default watermark asset (Vrooli Ascension)
 */
export const getDefaultWatermarkAsset = (): BuiltInAssetDefinition => {
  return BUILTIN_ASSETS[BUILTIN_ASSET_IDS.VROOLI_ASCENSION];
};

/**
 * Get the URL for an asset (handles both built-in and user assets)
 * For built-in assets, returns the URL directly.
 * For user assets, returns null (caller should use assetStore.getAssetUrl)
 */
export const getBuiltInAssetUrl = (id: string): string | null => {
  const asset = getBuiltInAsset(id);
  return asset?.url ?? null;
};
