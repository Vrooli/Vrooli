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

// Node types - use direct imports for tree-shaking:
// import { ClickNode, NavigateNode } from "@features/workflows/nodes";
// Or individual: import ClickNode from "@features/workflows/nodes/ClickNode";
