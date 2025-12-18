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

// Decomposed ProjectDetail components
export { ProjectDetailHeader } from './ProjectDetailHeader';
export { ProjectDetailTabs } from './ProjectDetailTabs';
export { ExecutionPanel } from './ExecutionPanel';
export { WorkflowCardGrid } from './WorkflowCardGrid';
export { ProjectFileTree } from './ProjectFileTree';

// FileTree components
export { FileTreeItem } from './FileTree';

// Hooks
export { useProjectDetailStore } from './hooks/useProjectDetailStore';
export { useFileTreeOperations } from './hooks/useFileTreeOperations';
export type {
  WorkflowWithStats,
  ProjectEntry,
  ProjectEntryKind,
  ViewMode,
  ActiveTab,
} from './hooks/useProjectDetailStore';
