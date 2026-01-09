/**
 * Projects Domain
 *
 * Project organization and management.
 * This domain handles project CRUD operations, file tree navigation,
 * and project-workflow relationships.
 */

// Store
export { useProjectStore, buildProjectFolderPath } from './store';
export type { Project, ProjectState } from './store';

// Main components
export { default as ProjectDetail } from './ProjectDetail';
export { default as ProjectModal } from './ProjectModal';
export { ProjectsTab } from './ProjectsTab';

// Import modals (re-exported from import domain)
export { ProjectImportModal, AssetImportModal as AssetUploadModal } from '@/domains/import';

// Decomposed ProjectDetail components
export { ProjectDetailHeader } from './ProjectDetailHeader';
export { ProjectDetailTabs } from './ProjectDetailTabs';
export { ExecutionPanel } from './ExecutionPanel';
export { WorkflowCardGrid } from './WorkflowCardGrid';
export { WorkflowCard } from './WorkflowCard';
export { ProjectFileTree } from './ProjectFileTree';

// FileTree components
export { FileTreeItem } from './FileTree';

// Hooks
export { useProjectDetailStore } from './hooks/useProjectDetailStore';
export { useFileTreeOperations } from './hooks/useFileTreeOperations';

// Import hooks (re-exported from import domain)
export { useProjectImport, useFolderScanner as useFolderBrowser } from '@/domains/import';
export type { InspectFolderResponse, ImportProjectRequest } from '@/domains/import';
export type { FolderEntry, ScanResult } from '@/domains/import';

// Folder Browser (re-exported from import domain)
export { FolderBrowser as FolderBrowserPanel } from '@/domains/import';

export type {
  WorkflowWithStats,
  ProjectEntry,
  ProjectEntryKind,
  ViewMode,
  ActiveTab,
} from './hooks/useProjectDetailStore';
