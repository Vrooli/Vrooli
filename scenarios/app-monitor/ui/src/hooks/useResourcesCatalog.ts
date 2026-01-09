import { useMemo } from 'react';
import { useResourcesStore } from '@/state/resourcesStore';
import type { Resource } from '@/types';

export type ResourceSortOption = 'status' | 'name-asc' | 'name-desc' | 'type';

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

const normalizeName = (value?: string | null) => value?.toLowerCase() ?? '';

const compareByName = (a: Resource, b: Resource) => (
  (a.name ?? a.id ?? '').localeCompare(b.name ?? b.id ?? '', undefined, { sensitivity: 'base' })
);

const compareByType = (a: Resource, b: Resource) => {
  const typeCompare = (a.type ?? '').localeCompare(b.type ?? '', undefined, { sensitivity: 'base' });
  if (typeCompare !== 0) {
    return typeCompare;
  }
  return compareByName(a, b);
};

const compareByStatus = (a: Resource, b: Resource) => {
  const aStatus = (a.status ?? '').toLowerCase();
  const bStatus = (b.status ?? '').toLowerCase();
  const aPriority = RESOURCE_STATUS_PRIORITY[aStatus] ?? 3;
  const bPriority = RESOURCE_STATUS_PRIORITY[bStatus] ?? 3;
  if (aPriority !== bPriority) {
    return aPriority - bPriority;
  }
  const nameCompare = compareByName(a, b);
  if (nameCompare !== 0) {
    return nameCompare;
  }
  return normalizeName(a.id).localeCompare(normalizeName(b.id));
};

const sortResources = (resources: Resource[], sort: ResourceSortOption): Resource[] => {
  const comparator = (() => {
    switch (sort) {
      case 'name-asc':
        return compareByName;
      case 'name-desc':
        return (a: Resource, b: Resource) => compareByName(b, a);
      case 'type':
        return compareByType;
      case 'status':
      default:
        return compareByStatus;
    }
  })();

  return [...resources].sort(comparator);
};

export const normalizeResourceSort = (value: string | null | undefined): ResourceSortOption => {
  if (value === 'name-asc' || value === 'name-desc' || value === 'type') {
    return value;
  }
  return 'status';
};

export const useResourcesCatalog = ({ sort = 'status' }: { sort?: ResourceSortOption } = {}) => {
  const resources = useResourcesStore(state => state.resources);
  const loading = useResourcesStore(state => state.loading);
  const error = useResourcesStore(state => state.error);
  const loadResources = useResourcesStore(state => state.loadResources);
  const startResource = useResourcesStore(state => state.startResource);
  const stopResource = useResourcesStore(state => state.stopResource);
  const refreshResource = useResourcesStore(state => state.refreshResource);
  const sortedResources = useMemo(() => sortResources(resources, sort), [resources, sort]);

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
