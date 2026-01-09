/**
 * Workflow Node Components
 *
 * All node types organized by category for clean imports.
 * Categories follow the structure defined in constants/nodeCategories.ts
 */

// =============================================================================
// Navigation & Context Nodes
// =============================================================================
export { default as NavigateNode } from "./NavigateNode";
export { default as ScrollNode } from "./ScrollNode";
export { default as TabSwitchNode } from "./TabSwitchNode";
export { default as FrameSwitchNode } from "./FrameSwitchNode";
export { default as RotateNode } from "./RotateNode";
export { default as GestureNode } from "./GestureNode";

// =============================================================================
// Pointer & Gesture Nodes
// =============================================================================
export { default as ClickNode } from "./ClickNode";
export { default as HoverNode } from "./HoverNode";
export { default as DragDropNode } from "./DragDropNode";
export { default as FocusNode } from "./FocusNode";
export { default as BlurNode } from "./BlurNode";

// =============================================================================
// Forms & Input Nodes
// =============================================================================
export { default as TypeNode } from "./TypeNode";
export { default as SelectNode } from "./SelectNode";
export { default as UploadFileNode } from "./UploadFileNode";
export { default as ShortcutNode } from "./ShortcutNode";
export { default as KeyboardNode } from "./KeyboardNode";

// =============================================================================
// Data & Variables Nodes
// =============================================================================
export { default as ExtractNode } from "./ExtractNode";
export { default as SetVariableNode } from "./SetVariableNode";
export { default as UseVariableNode } from "./UseVariableNode";
export { default as ScriptNode } from "./ScriptNode";

// =============================================================================
// Assertions & Observability Nodes
// =============================================================================
export { default as AssertNode } from "./AssertNode";
export { default as WaitNode } from "./WaitNode";
export { default as ScreenshotNode } from "./ScreenshotNode";

// =============================================================================
// Workflow Logic Nodes
// =============================================================================
export { default as ConditionalNode } from "./ConditionalNode";
export { default as LoopNode } from "./LoopNode";
export { default as SubflowNode } from "./SubflowNode";

// =============================================================================
// Storage & Network Nodes
// =============================================================================
export { default as SetCookieNode } from "./SetCookieNode";
export { default as GetCookieNode } from "./GetCookieNode";
export { default as ClearCookieNode } from "./ClearCookieNode";
export { default as SetStorageNode } from "./SetStorageNode";
export { default as GetStorageNode } from "./GetStorageNode";
export { default as ClearStorageNode } from "./ClearStorageNode";
export { default as NetworkMockNode } from "./NetworkMockNode";

// =============================================================================
// Shared Node Components
// =============================================================================
export { default as ResiliencePanel } from "./ResiliencePanel";

// =============================================================================
// Internal/Utility Nodes
// =============================================================================
export { default as BrowserActionNode } from "./BrowserActionNode";
