/**
 * IndexedDB implementation of AssetStorage
 *
 * Used for web browsers and PWA environments.
 * Stores assets as Blobs with metadata in IndexedDB.
 */

import type { AssetStorage, BrandAsset, AssetMetadata, StorageInfo, AssetType } from './types';
import { validateFile, getImageDimensions, generateThumbnail } from './imageUtils';

const DB_NAME = 'browserAutomationStudio';
const DB_VERSION = 1;
const ASSETS_STORE = 'brandAssets';
const BLOBS_STORE = 'assetBlobs';

/**
 * IndexedDB record for asset metadata
 */
interface AssetRecord extends AssetMetadata {
  thumbnail: string; // Base64 data URL
}

/**
 * IndexedDB record for asset blob data
 */
interface BlobRecord {
  id: string;
  data: Blob;
}

export class IndexedDbAssetStorage implements AssetStorage {
  private db: IDBDatabase | null = null;
  private dbPromise: Promise<IDBDatabase> | null = null;

  /**
   * Get or open the database connection
   */
  private async getDb(): Promise<IDBDatabase> {
    if (this.db) return this.db;

    // Prevent multiple simultaneous opens
    if (this.dbPromise) return this.dbPromise;

    this.dbPromise = new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);

      request.onerror = () => {
        this.dbPromise = null;
        reject(new Error(`Failed to open database: ${request.error?.message}`));
      };

      request.onsuccess = () => {
        this.db = request.result;
        this.dbPromise = null;

        // Handle connection close
        this.db.onclose = () => {
          this.db = null;
        };

        resolve(this.db);
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;

        // Store for asset metadata
        if (!db.objectStoreNames.contains(ASSETS_STORE)) {
          const assetsStore = db.createObjectStore(ASSETS_STORE, { keyPath: 'id' });
          assetsStore.createIndex('type', 'type', { unique: false });
          assetsStore.createIndex('createdAt', 'createdAt', { unique: false });
          assetsStore.createIndex('name', 'name', { unique: false });
        }

        // Separate store for blob data (keeps metadata queries fast)
        if (!db.objectStoreNames.contains(BLOBS_STORE)) {
          db.createObjectStore(BLOBS_STORE, { keyPath: 'id' });
        }
      };
    });

    return this.dbPromise;
  }

  async save(file: File, metadata?: Partial<Pick<AssetMetadata, 'name' | 'type'>>): Promise<BrandAsset> {
    // Validate the file
    await validateFile(file);

    const db = await this.getDb();
    const id = crypto.randomUUID();
    const now = Date.now();

    // Get image dimensions
    const dimensions = await getImageDimensions(file);

    // Generate thumbnail
    const thumbnail = await generateThumbnail(file);

    // Create metadata record
    const assetRecord: AssetRecord = {
      id,
      name: metadata?.name || file.name.replace(/\.[^/.]+$/, ''), // Remove extension
      type: metadata?.type || 'other',
      mimeType: file.type,
      width: dimensions.width,
      height: dimensions.height,
      sizeBytes: file.size,
      createdAt: now,
      updatedAt: now,
      thumbnail,
    };

    // Create blob record
    const blobRecord: BlobRecord = {
      id,
      data: file,
    };

    // Store both in a transaction
    return new Promise((resolve, reject) => {
      const tx = db.transaction([ASSETS_STORE, BLOBS_STORE], 'readwrite');

      tx.onerror = () => reject(new Error(`Failed to save asset: ${tx.error?.message}`));
      tx.oncomplete = () => {
        resolve({
          ...assetRecord,
          data: file,
        });
      };

      tx.objectStore(ASSETS_STORE).put(assetRecord);
      tx.objectStore(BLOBS_STORE).put(blobRecord);
    });
  }

  async get(id: string): Promise<BrandAsset | null> {
    const db = await this.getDb();

    return new Promise((resolve, reject) => {
      const tx = db.transaction([ASSETS_STORE, BLOBS_STORE], 'readonly');

      let assetRecord: AssetRecord | undefined;
      let blobRecord: BlobRecord | undefined;

      const assetRequest = tx.objectStore(ASSETS_STORE).get(id);
      assetRequest.onsuccess = () => {
        assetRecord = assetRequest.result;
      };

      const blobRequest = tx.objectStore(BLOBS_STORE).get(id);
      blobRequest.onsuccess = () => {
        blobRecord = blobRequest.result;
      };

      tx.onerror = () => reject(new Error(`Failed to get asset: ${tx.error?.message}`));
      tx.oncomplete = () => {
        if (!assetRecord) {
          resolve(null);
          return;
        }
        resolve({
          ...assetRecord,
          data: blobRecord?.data,
        });
      };
    });
  }

  async getUrl(id: string): Promise<string | null> {
    const db = await this.getDb();

    return new Promise((resolve, reject) => {
      const request = db.transaction(BLOBS_STORE, 'readonly').objectStore(BLOBS_STORE).get(id);

      request.onerror = () => reject(new Error(`Failed to get asset URL: ${request.error?.message}`));
      request.onsuccess = () => {
        const record = request.result as BlobRecord | undefined;
        if (!record?.data) {
          resolve(null);
          return;
        }
        resolve(URL.createObjectURL(record.data));
      };
    });
  }

  async list(): Promise<BrandAsset[]> {
    const db = await this.getDb();

    return new Promise((resolve, reject) => {
      const request = db.transaction(ASSETS_STORE, 'readonly').objectStore(ASSETS_STORE).getAll();

      request.onerror = () => reject(new Error(`Failed to list assets: ${request.error?.message}`));
      request.onsuccess = () => {
        const records = request.result as AssetRecord[];
        // Sort by createdAt descending (newest first)
        records.sort((a, b) => b.createdAt - a.createdAt);
        resolve(records);
      };
    });
  }

  async update(id: string, updates: Partial<Pick<AssetMetadata, 'name' | 'type'>>): Promise<BrandAsset | null> {
    const asset = await this.get(id);
    if (!asset) return null;

    const db = await this.getDb();
    const now = Date.now();

    const updatedRecord: AssetRecord = {
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
    };

    return new Promise((resolve, reject) => {
      const request = db.transaction(ASSETS_STORE, 'readwrite').objectStore(ASSETS_STORE).put(updatedRecord);

      request.onerror = () => reject(new Error(`Failed to update asset: ${request.error?.message}`));
      request.onsuccess = () => {
        resolve({
          ...updatedRecord,
          data: asset.data,
        });
      };
    });
  }

  async delete(id: string): Promise<boolean> {
    const db = await this.getDb();

    return new Promise((resolve, reject) => {
      const tx = db.transaction([ASSETS_STORE, BLOBS_STORE], 'readwrite');

      tx.onerror = () => reject(new Error(`Failed to delete asset: ${tx.error?.message}`));
      tx.oncomplete = () => resolve(true);

      tx.objectStore(ASSETS_STORE).delete(id);
      tx.objectStore(BLOBS_STORE).delete(id);
    });
  }

  async getStorageInfo(): Promise<StorageInfo> {
    const assets = await this.list();
    const used = assets.reduce((sum, a) => sum + a.sizeBytes, 0);

    // Try to get quota estimate
    let total = 0;
    try {
      if (navigator.storage?.estimate) {
        const estimate = await navigator.storage.estimate();
        total = estimate.quota || 0;
      }
    } catch {
      // Quota API not available
    }

    return {
      used,
      total,
      count: assets.length,
    };
  }

  async isAvailable(): Promise<boolean> {
    try {
      // Check if IndexedDB is available
      if (typeof indexedDB === 'undefined') return false;

      // Try to open (this also tests if we're in a context that allows it)
      await this.getDb();
      return true;
    } catch {
      return false;
    }
  }
}
