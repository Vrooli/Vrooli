/**
 * Desktop (Electron) implementation of AssetStorage
 *
 * Thin wrapper around window.desktop.storage API provided by scenario-to-desktop.
 * Stores assets in the file system at userData/app-storage/brand-assets/.
 *
 * This implementation is only used when running as an Electron app.
 */

import type { AssetStorage, BrandAsset, AssetMetadata, StorageInfo, AssetType } from './types';
import { validateFile, getImageDimensions, generateThumbnail, getFileExtension } from './imageUtils';

// Storage path prefix within app-storage
const STORAGE_PREFIX = 'brand-assets';

// Type declarations for window.desktop (provided by scenario-to-desktop)
interface DesktopStorageAPI {
  writeFile(path: string, data: string | ArrayBuffer): Promise<void>;
  readFile(path: string): Promise<ArrayBuffer | null>;
  deleteFile(path: string): Promise<boolean>;
  listDir(path: string): Promise<Array<{ name: string; isDirectory: boolean }> | null>;
  exists(path: string): Promise<boolean>;
  getStorageInfo(): Promise<{ used: number; total: number } | null>;
}

interface DesktopAPI {
  storage: DesktopStorageAPI;
  storeJSON(path: string, data: unknown): Promise<void>;
  loadStoredJSON(path: string): Promise<unknown>;
  storeBlob(path: string, blob: Blob): Promise<void>;
  loadStoredBlob(path: string, mimeType: string): Promise<Blob | null>;
  getStoredFileUrl(path: string, mimeType: string): Promise<string | null>;
}

declare global {
  interface Window {
    desktop?: DesktopAPI;
  }
}

/**
 * Metadata stored as JSON sidecar
 */
interface StoredMetadata extends AssetMetadata {
  thumbnail: string;
  filePath: string;
}

export class DesktopAssetStorage implements AssetStorage {
  private get desktop(): DesktopAPI {
    if (!window.desktop) {
      throw new Error('Desktop API not available. This storage is only for Electron apps.');
    }
    return window.desktop;
  }

  async save(file: File, metadata?: Partial<Pick<AssetMetadata, 'name' | 'type'>>): Promise<BrandAsset> {
    // Validate the file
    await validateFile(file);

    const id = crypto.randomUUID();
    const extension = getFileExtension(file.name);
    const now = Date.now();

    // Get image dimensions
    const dimensions = await getImageDimensions(file);

    // Generate thumbnail
    const thumbnail = await generateThumbnail(file);

    // File paths
    const filePath = `${STORAGE_PREFIX}/files/${id}.${extension}`;
    const thumbnailPath = `${STORAGE_PREFIX}/thumbnails/${id}.jpg`;
    const metadataPath = `${STORAGE_PREFIX}/metadata/${id}.json`;

    // Save the main file
    await this.desktop.storeBlob(filePath, file);

    // Save thumbnail (convert data URL to blob)
    const thumbnailBlob = await fetch(thumbnail).then((r) => r.blob());
    await this.desktop.storeBlob(thumbnailPath, thumbnailBlob);

    // Build asset metadata
    const storedMetadata: StoredMetadata = {
      id,
      name: metadata?.name || file.name.replace(/\.[^/.]+$/, ''),
      type: metadata?.type || 'other',
      mimeType: file.type,
      width: dimensions.width,
      height: dimensions.height,
      sizeBytes: file.size,
      createdAt: now,
      updatedAt: now,
      thumbnail,
      filePath,
    };

    // Save metadata JSON
    await this.desktop.storeJSON(metadataPath, storedMetadata);

    return {
      ...storedMetadata,
    };
  }

  async get(id: string): Promise<BrandAsset | null> {
    try {
      const metadataPath = `${STORAGE_PREFIX}/metadata/${id}.json`;
      const metadata = (await this.desktop.loadStoredJSON(metadataPath)) as StoredMetadata | null;
      if (!metadata) return null;
      return metadata;
    } catch {
      return null;
    }
  }

  async getUrl(id: string): Promise<string | null> {
    const asset = await this.get(id);
    if (!asset?.filePath) return null;
    return this.desktop.getStoredFileUrl(asset.filePath, asset.mimeType);
  }

  async list(): Promise<BrandAsset[]> {
    const metadataDir = `${STORAGE_PREFIX}/metadata`;

    // Check if directory exists
    const exists = await this.desktop.storage.exists(metadataDir);
    if (!exists) return [];

    const files = await this.desktop.storage.listDir(metadataDir);
    if (!files) return [];

    const assets: BrandAsset[] = [];

    for (const file of files) {
      if (file.name.endsWith('.json') && !file.isDirectory) {
        const id = file.name.replace('.json', '');
        const asset = await this.get(id);
        if (asset) assets.push(asset);
      }
    }

    // Sort by createdAt descending
    return assets.sort((a, b) => b.createdAt - a.createdAt);
  }

  async update(id: string, updates: Partial<Pick<AssetMetadata, 'name' | 'type'>>): Promise<BrandAsset | null> {
    const asset = await this.get(id);
    if (!asset) return null;

    const now = Date.now();
    const metadataPath = `${STORAGE_PREFIX}/metadata/${id}.json`;

    const updatedMetadata: StoredMetadata = {
      id: asset.id,
      name: updates.name ?? asset.name,
      type: (updates.type ?? asset.type) as AssetType,
      mimeType: asset.mimeType,
      width: asset.width,
      height: asset.height,
      sizeBytes: asset.sizeBytes,
      createdAt: asset.createdAt,
      updatedAt: now,
      thumbnail: asset.thumbnail || '',
      filePath: asset.filePath || '',
    };

    await this.desktop.storeJSON(metadataPath, updatedMetadata);
    return updatedMetadata;
  }

  async delete(id: string): Promise<boolean> {
    const asset = await this.get(id);
    if (!asset) return false;

    // Delete all files
    await this.desktop.storage.deleteFile(`${STORAGE_PREFIX}/metadata/${id}.json`);
    if (asset.filePath) {
      await this.desktop.storage.deleteFile(asset.filePath);
    }
    // Delete thumbnail (try both jpg and original extension)
    await this.desktop.storage.deleteFile(`${STORAGE_PREFIX}/thumbnails/${id}.jpg`);

    return true;
  }

  async getStorageInfo(): Promise<StorageInfo> {
    const assets = await this.list();
    const used = assets.reduce((sum, a) => sum + a.sizeBytes, 0);

    // Get system storage info
    const info = await this.desktop.storage.getStorageInfo();

    return {
      used,
      total: info?.total || 0,
      count: assets.length,
    };
  }

  async isAvailable(): Promise<boolean> {
    return typeof window.desktop?.storage !== 'undefined';
  }
}
