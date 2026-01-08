// Domain hooks
export { useSessionNavigation } from './navigation';
export type { UseSessionNavigationProps, UseSessionNavigationReturn } from './navigation';

export { useProfileSettings } from './settings';
export type { UseProfileSettingsProps, UseProfileSettingsReturn } from './settings';

export { useSessionPersistence } from './persistence';
export type { UseSessionPersistenceProps, UseSessionPersistenceReturn } from './persistence';

export { useProfileBoundResources } from './resources';
export type {
  UseProfileBoundResourcesProps,
  UseProfileBoundResourcesReturn,
  BoundStorageResource,
  BoundServiceWorkersResource,
  BoundHistoryResource,
  BoundTabsResource,
} from './resources';
