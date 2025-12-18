/**
 * Project management feature - handles projects listing, creation, and details
 */
export { default as Dashboard } from "./Dashboard";
export { default as ProjectDetail } from "./ProjectDetail";
export { default as ProjectModal } from "./ProjectModal";

// Decomposed ProjectDetail components
export { ProjectDetailHeader } from "./ProjectDetailHeader";
export { ProjectDetailTabs } from "./ProjectDetailTabs";
export { ExecutionPanel } from "./ExecutionPanel";
export { WorkflowCardGrid } from "./WorkflowCardGrid";
export { ProjectFileTree } from "./ProjectFileTree";

// Hooks
export { useProjectDetailStore } from "./hooks/useProjectDetailStore";
export { useFileTreeOperations } from "./hooks/useFileTreeOperations";
export type {
  WorkflowWithStats,
  ProjectEntry,
  ProjectEntryKind,
  ViewMode,
  ActiveTab,
} from "./hooks/useProjectDetailStore";
