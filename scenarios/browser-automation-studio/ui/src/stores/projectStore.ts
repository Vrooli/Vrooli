// ============================================================================
// Backward Compatibility Re-export
// ============================================================================
// This file re-exports from the new domains/projects/ location.
// All imports from '@stores/projectStore' continue to work unchanged.
//
// Prefer importing from '@/domains/projects' for new code.

export { useProjectStore, buildProjectFolderPath } from '../domains/projects/store';
export type { Project, ProjectState } from '../domains/projects/store';
