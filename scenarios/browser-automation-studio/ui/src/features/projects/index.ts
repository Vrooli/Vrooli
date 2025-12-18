/**
 * Projects Feature - Backward Compatibility Re-exports
 *
 * @deprecated Import from '@/domains/projects' instead
 *
 * This module re-exports from the new domains/projects location
 * for backward compatibility during the migration.
 */

// Dashboard re-export for backwards compatibility (already in views/)
export { DashboardView as Dashboard } from '@/views/DashboardView';

export {
  // Main components
  ProjectDetail,
  ProjectModal,
  // Decomposed components
  ProjectDetailHeader,
  ProjectDetailTabs,
  ExecutionPanel,
  WorkflowCardGrid,
  ProjectFileTree,
  // Hooks
  useProjectDetailStore,
  useFileTreeOperations,
} from '@/domains/projects';

export type {
  WorkflowWithStats,
  ProjectEntry,
  ProjectEntryKind,
  ViewMode,
  ActiveTab,
} from '@/domains/projects';
