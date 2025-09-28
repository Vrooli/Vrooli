import { create } from 'zustand';
import { resourceService } from '@/services/api';
import type { ApiResponse, Resource } from '@/types';

type ResourceUpdater = Resource[] | ((current: Resource[]) => Resource[]);

interface ResourcesStoreState {
  resources: Resource[];
  loading: boolean;
  error: string | null;
  hasInitialized: boolean;
  loadResources: (options?: { force?: boolean }) => Promise<void>;
  setResourcesState: (updater: ResourceUpdater) => void;
  updateResource: (update: Partial<Resource> & { id?: string }) => void;
  startResource: (id: string) => Promise<ApiResponse<Resource> | null>;
  stopResource: (id: string) => Promise<ApiResponse<Resource> | null>;
  refreshResource: (id: string) => Promise<Resource | null>;
  clearError: () => void;
}

type ResourceLikeUpdate = Partial<Resource> & { id: string };

const upsertResource = (resources: Resource[], next: ResourceLikeUpdate): Resource[] => {
  let replaced = false;
  const updated = resources.map((resource) => {
    if (resource.id === next.id) {
      replaced = true;
      return { ...resource, ...next } as Resource;
    }
    return resource;
  });

  if (!replaced) {
    updated.push(next as Resource);
  }

  return updated;
};

const extractErrorMessage = (response: ApiResponse<unknown> | null): string | null => {
  if (!response) {
    return null;
  }

  if (response.success === false && response.error && response.error.trim().length > 0) {
    return response.error;
  }

  if (response.success === false && response.message) {
    return response.message;
  }

  return null;
};

export const useResourcesStore = create<ResourcesStoreState>((set, get) => ({
  resources: [],
  loading: false,
  error: null,
  hasInitialized: false,

  loadResources: async ({ force = false } = {}) => {
    const { loading, hasInitialized, resources } = get();
    if (loading || (!force && hasInitialized && resources.length > 0)) {
      return;
    }

    set({ loading: true, error: null });
    try {
      const fetched = await resourceService.getResources();
      if (Array.isArray(fetched)) {
        set({ resources: fetched });
      } else if (force) {
        set({ resources: [] });
      }
    } catch (error) {
      console.warn('[resourcesStore] Failed to fetch resources', error);
      if (!get().hasInitialized) {
        set({ error: 'Unable to load resources.' });
      }
    } finally {
      set({ loading: false, hasInitialized: true });
    }
  },

  setResourcesState: (updater) => {
    set((state) => ({
      resources: typeof updater === 'function'
        ? (updater as (current: Resource[]) => Resource[])(state.resources)
        : updater,
    }));
  },

  updateResource: (update) => {
    const identifier = update.id?.trim();
    if (!identifier) {
      return;
    }

    set((state) => ({
      resources: upsertResource(state.resources, { ...update, id: identifier }),
    }));
  },

  startResource: async (id) => {
    const response = await resourceService.startResource(id);
    if (response?.data) {
      set((state) => ({ resources: upsertResource(state.resources, response.data as ResourceLikeUpdate) }));
    } else if (response?.success) {
      void get().loadResources({ force: true });
    }

    const errorMessage = extractErrorMessage(response ?? null);
    if (errorMessage) {
      set({ error: errorMessage });
    } else if (response?.success) {
      set({ error: null });
    }

    return response ?? null;
  },

  stopResource: async (id) => {
    const response = await resourceService.stopResource(id);
    if (response?.data) {
      set((state) => ({ resources: upsertResource(state.resources, response.data as ResourceLikeUpdate) }));
    } else if (response?.success) {
      void get().loadResources({ force: true });
    }

    const errorMessage = extractErrorMessage(response ?? null);
    if (errorMessage) {
      set({ error: errorMessage });
    } else if (response?.success) {
      set({ error: null });
    }

    return response ?? null;
  },

  refreshResource: async (id) => {
    const resource = await resourceService.getResourceStatus(id);
    if (resource) {
      set((state) => ({ resources: upsertResource(state.resources, resource) }));
      set({ error: null });
    }
    return resource;
  },

  clearError: () => set({ error: null }),
}));
