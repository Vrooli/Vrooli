/**
 * Zustand store for brand assets
 *
 * Wraps the storage abstraction layer and provides reactive state for the UI.
 * Manages blob URL lifecycle to prevent memory leaks.
 */

import { create } from 'zustand';
import type { BrandAsset, AssetType, StorageInfo } from '../lib/storage';
import { getAssetStorage, AssetValidationError } from '../lib/storage';

interface AssetStore {
  // State
  assets: BrandAsset[];
  isLoading: boolean;
  isInitialized: boolean;
  error: string | null;
  storageInfo: StorageInfo | null;

  // Cached URLs for rendering (managed lifecycle)
  assetUrls: Map<string, string>;

  // Actions
  initialize: () => Promise<void>;
  uploadAsset: (file: File, type?: AssetType) => Promise<BrandAsset>;
  deleteAsset: (id: string) => Promise<void>;
  renameAsset: (id: string, name: string) => Promise<void>;
  updateAssetType: (id: string, type: AssetType) => Promise<void>;
  getAssetUrl: (id: string) => Promise<string | null>;
  revokeAssetUrl: (id: string) => void;
  revokeAllUrls: () => void;
  refreshStorageInfo: () => Promise<void>;
  getAssetById: (id: string) => BrandAsset | undefined;
  getAssetsByType: (type: AssetType) => BrandAsset[];
  clearError: () => void;
}

export const useAssetStore = create<AssetStore>((set, get) => ({
  assets: [],
  isLoading: false,
  isInitialized: false,
  error: null,
  storageInfo: null,
  assetUrls: new Map(),

  initialize: async () => {
    // Don't re-initialize
    if (get().isInitialized || get().isLoading) return;

    set({ isLoading: true, error: null });

    try {
      const storage = await getAssetStorage();
      const assets = await storage.list();
      const storageInfo = await storage.getStorageInfo();
      set({ assets, storageInfo, isLoading: false, isInitialized: true });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to initialize asset storage';
      set({ error: message, isLoading: false, isInitialized: true });
    }
  },

  uploadAsset: async (file, type = 'other') => {
    set({ error: null });

    try {
      const storage = await getAssetStorage();
      const asset = await storage.save(file, { type });

      set((state) => ({
        assets: [asset, ...state.assets],
      }));

      // Refresh storage info in background
      get().refreshStorageInfo();

      return asset;
    } catch (err) {
      let message: string;
      if (err instanceof AssetValidationError) {
        message = err.message;
      } else if (err instanceof Error) {
        message = err.message;
      } else {
        message = 'Failed to upload asset';
      }
      set({ error: message });
      throw err;
    }
  },

  deleteAsset: async (id) => {
    set({ error: null });

    try {
      const storage = await getAssetStorage();
      await storage.delete(id);

      // Revoke URL if cached
      get().revokeAssetUrl(id);

      set((state) => ({
        assets: state.assets.filter((a) => a.id !== id),
      }));

      // Refresh storage info in background
      get().refreshStorageInfo();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete asset';
      set({ error: message });
      throw err;
    }
  },

  renameAsset: async (id, name) => {
    set({ error: null });

    try {
      const storage = await getAssetStorage();
      const updated = await storage.update(id, { name });

      if (updated) {
        set((state) => ({
          assets: state.assets.map((a) => (a.id === id ? { ...a, name, updatedAt: updated.updatedAt } : a)),
        }));
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to rename asset';
      set({ error: message });
      throw err;
    }
  },

  updateAssetType: async (id, type) => {
    set({ error: null });

    try {
      const storage = await getAssetStorage();
      const updated = await storage.update(id, { type });

      if (updated) {
        set((state) => ({
          assets: state.assets.map((a) => (a.id === id ? { ...a, type, updatedAt: updated.updatedAt } : a)),
        }));
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update asset type';
      set({ error: message });
      throw err;
    }
  },

  getAssetUrl: async (id) => {
    // Check cache first
    const cached = get().assetUrls.get(id);
    if (cached) return cached;

    try {
      const storage = await getAssetStorage();
      const url = await storage.getUrl(id);

      if (url) {
        set((state) => {
          const newUrls = new Map(state.assetUrls);
          newUrls.set(id, url);
          return { assetUrls: newUrls };
        });
      }

      return url;
    } catch {
      return null;
    }
  },

  revokeAssetUrl: (id) => {
    const url = get().assetUrls.get(id);
    if (url && url.startsWith('blob:')) {
      URL.revokeObjectURL(url);
    }
    set((state) => {
      const newUrls = new Map(state.assetUrls);
      newUrls.delete(id);
      return { assetUrls: newUrls };
    });
  },

  revokeAllUrls: () => {
    const urls = get().assetUrls;
    urls.forEach((url) => {
      if (url.startsWith('blob:')) {
        URL.revokeObjectURL(url);
      }
    });
    set({ assetUrls: new Map() });
  },

  refreshStorageInfo: async () => {
    try {
      const storage = await getAssetStorage();
      const storageInfo = await storage.getStorageInfo();
      set({ storageInfo });
    } catch {
      // Silent fail for background refresh
    }
  },

  getAssetById: (id) => {
    return get().assets.find((a) => a.id === id);
  },

  getAssetsByType: (type) => {
    return get().assets.filter((a) => a.type === type);
  },

  clearError: () => {
    set({ error: null });
  },
}));

export default useAssetStore;
