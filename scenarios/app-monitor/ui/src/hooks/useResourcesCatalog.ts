import { useMemo } from 'react';
import { useResourcesStore } from '@/state/resourcesStore';
import type { Resource } from '@/types';

const RESOURCE_STATUS_PRIORITY: Record<string, number> = {
  online: 0,
  running: 0,
  healthy: 0,
  degraded: 1,
  warning: 1,
  syncing: 1,
  offline: 2,
  stopped: 2,
  error: 2,
  unknown: 3,
  unregistered: 3,
};

const sortResources = (resources: Resource[]): Resource[] => {
  return [...resources].sort((a, b) => {
    const aStatus = (a.status ?? '').toLowerCase();
    const bStatus = (b.status ?? '').toLowerCase();
    const aPriority = RESOURCE_STATUS_PRIORITY[aStatus] ?? 3;
    const bPriority = RESOURCE_STATUS_PRIORITY[bStatus] ?? 3;
    if (aPriority !== bPriority) {
      return aPriority - bPriority;
    }
    return (a.name ?? a.id ?? '').localeCompare(b.name ?? b.id ?? '');
  });
};

export const useResourcesCatalog = () => {
  const resources = useResourcesStore(state => state.resources);
  const loading = useResourcesStore(state => state.loading);
  const error = useResourcesStore(state => state.error);
  const loadResources = useResourcesStore(state => state.loadResources);
  const startResource = useResourcesStore(state => state.startResource);
  const stopResource = useResourcesStore(state => state.stopResource);
  const refreshResource = useResourcesStore(state => state.refreshResource);
  const sortedResources = useMemo(() => sortResources(resources), [resources]);

  return {
    resources,
    sortedResources,
    loading,
    error,
    loadResources,
    startResource,
    stopResource,
    refreshResource,
  };
};

export type { Resource } from '@/types';
