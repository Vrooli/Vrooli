/**
 * Storage types for brand assets
 *
 * Supports hybrid storage: IndexedDB for web/PWA, file system for Electron desktop.
 */

// Asset types
export type AssetType = 'logo' | 'background' | 'other';

// Supported MIME types (SVG excluded due to security risks - can contain scripts)
export const ALLOWED_MIME_TYPES = [
  'image/png',
  'image/jpeg',
  'image/jpg',
  'image/webp',
] as const;

export type AllowedMimeType = (typeof ALLOWED_MIME_TYPES)[number];

// Validation constants
export const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5MB
export const MAX_DIMENSION = 4096; // Max width/height in pixels (supports 4K)
export const THUMBNAIL_SIZE = 200; // Thumbnail dimension in pixels

/**
 * Metadata stored with each asset (persisted)
 */
export interface AssetMetadata {
  id: string;
  name: string;
  type: AssetType;
  mimeType: string;
  width: number;
  height: number;
  sizeBytes: number;
  createdAt: number; // Unix timestamp ms
  updatedAt: number; // Unix timestamp ms
}

/**
 * Full brand asset including runtime data
 */
export interface BrandAsset extends AssetMetadata {
  // Runtime-only fields (not persisted in metadata JSON)
  data?: Blob; // For IndexedDB storage
  filePath?: string; // For file system storage
  thumbnail?: string; // Base64 data URL for gallery preview
}

/**
 * Storage usage statistics
 */
export interface StorageInfo {
  used: number; // Bytes used
  total: number; // Total available (estimate)
  count: number; // Number of assets
}

/**
 * Abstract storage interface for brand assets
 *
 * Implementations:
 * - IndexedDbAssetStorage: Web/PWA using IndexedDB
 * - DesktopAssetStorage: Electron using file system via window.desktop
 */
export interface AssetStorage {
  /**
   * Save an asset from a File object
   * @param file - The file to save
   * @param metadata - Partial metadata (type, name override)
   * @returns The created BrandAsset with generated ID
   */
  save(file: File, metadata?: Partial<Pick<AssetMetadata, 'name' | 'type'>>): Promise<BrandAsset>;

  /**
   * Get asset by ID (includes metadata and thumbnail)
   * @param id - Asset ID
   * @returns Asset or null if not found
   */
  get(id: string): Promise<BrandAsset | null>;

  /**
   * Get a URL suitable for rendering in <img src="">
   * - IndexedDB: Returns blob: URL (must be revoked when done)
   * - File System: Returns file:// URL or data URL
   * @param id - Asset ID
   * @returns URL string or null if not found
   */
  getUrl(id: string): Promise<string | null>;

  /**
   * List all assets (metadata only, includes thumbnails)
   * @returns Array of assets sorted by createdAt descending
   */
  list(): Promise<BrandAsset[]>;

  /**
   * Update asset metadata (name, type)
   * @param id - Asset ID
   * @param updates - Fields to update
   * @returns Updated asset or null if not found
   */
  update(id: string, updates: Partial<Pick<AssetMetadata, 'name' | 'type'>>): Promise<BrandAsset | null>;

  /**
   * Delete asset by ID
   * @param id - Asset ID
   * @returns true if deleted, false if not found
   */
  delete(id: string): Promise<boolean>;

  /**
   * Get storage usage statistics
   * @returns Storage info with used/total bytes and asset count
   */
  getStorageInfo(): Promise<StorageInfo>;

  /**
   * Check if storage is available and working
   * @returns true if storage can be used
   */
  isAvailable(): Promise<boolean>;
}

/**
 * Validation error for asset uploads
 */
export class AssetValidationError extends Error {
  constructor(
    message: string,
    public readonly code: 'INVALID_TYPE' | 'FILE_TOO_LARGE' | 'DIMENSION_TOO_LARGE' | 'INVALID_IMAGE' | 'STORAGE_FULL',
  ) {
    super(message);
    this.name = 'AssetValidationError';
  }
}
