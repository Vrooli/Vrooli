/**
 * Import Domain
 *
 * Unified import functionality for projects, assets, and routines.
 * Provides shared components, hooks, and modals for consistent import UX.
 */

// Types
export * from './types';

// Constants
export * from './constants';

// Hooks
export { useDropZone, type UseDropZoneOptions, type UseDropZoneReturn } from './hooks/useDropZone';
export {
  useFolderScanner,
  type UseFolderScannerOptions,
  type UseFolderScannerReturn,
} from './hooks/useFolderScanner';
export {
  useRoutineImport,
  type UseRoutineImportOptions,
  type UseRoutineImportReturn,
  type InspectRoutineResponse,
  type ImportRoutineRequest,
  type ImportRoutineResponse,
  type WorkflowPreview,
  type RoutineEntry,
  type ScanRoutinesResponse,
} from './hooks/useRoutineImport';
export {
  useProjectImport,
  type UseProjectImportOptions,
  type UseProjectImportReturn,
  type InspectFolderResponse,
  type ImportProjectRequest,
} from './hooks/useProjectImport';

// Components
export { DropZone, type DropZoneProps } from './components/DropZone';
export { FolderBrowser, type FolderBrowserProps } from './components/FolderBrowser';
export { ImportWizard, type ImportWizardProps } from './components/ImportWizard';
export {
  ValidationStatus,
  StatusBadge,
  ValidationBadge,
  AlertBox,
  type ValidationStatusProps,
  type StatusBadgeProps,
  type AlertBoxProps,
} from './components/ValidationStatus';

// Modals
export { RoutineImportModal } from './modals/RoutineImportModal';
export { ProjectImportModal } from './modals/ProjectImportModal';
export { AssetImportModal } from './modals/AssetImportModal';
