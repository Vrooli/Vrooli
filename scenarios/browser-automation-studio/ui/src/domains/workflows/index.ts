/**
 * Workflow feature - handles visual workflow building and node management
 *
 * Structure:
 * - builder/: WorkflowBuilder canvas and toolbar components
 * - components/: Shared workflow components (ElementPicker, ConnectionLine, etc.)
 * - nodes/: Individual node type components organized by category
 */

// Builder components
export { default as WorkflowBuilder } from "./builder/WorkflowBuilder";
export { default as NodePalette } from "./builder/NodePalette";
export { default as WorkflowToolbar } from "./builder/WorkflowToolbar";

// Workflow-specific shared components
export * from "./components";

// Workflow creation dialog
export { WorkflowCreationDialog, type WorkflowCreationType } from "./WorkflowCreationDialog";

// Node types - use direct imports for tree-shaking:
// import { ClickNode, NavigateNode } from "@/domains/workflows/nodes";
// Or individual: import ClickNode from "@/domains/workflows/nodes/ClickNode";
