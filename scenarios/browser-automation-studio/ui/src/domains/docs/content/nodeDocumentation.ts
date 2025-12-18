/**
 * Node documentation content - bundled at build time from docs/nodes/*.md
 *
 * This file imports all node documentation markdown files and exports them
 * as a typed map for use in the documentation hub and node palette help.
 */

// Import markdown files as raw strings using Vite's ?raw suffix
import navigateDoc from "../../../../../docs/nodes/navigate.md?raw";
import clickDoc from "../../../../../docs/nodes/click.md?raw";
import hoverDoc from "../../../../../docs/nodes/hover.md?raw";
import dragDropDoc from "../../../../../docs/nodes/drag-drop.md?raw";
import focusDoc from "../../../../../docs/nodes/focus.md?raw";
import blurDoc from "../../../../../docs/nodes/blur.md?raw";
import tabSwitchDoc from "../../../../../docs/nodes/tab-switch.md?raw";
import frameSwitchDoc from "../../../../../docs/nodes/frame-switch.md?raw";
import scrollDoc from "../../../../../docs/nodes/scroll.md?raw";
import rotateDoc from "../../../../../docs/nodes/rotate.md?raw";
import gestureDoc from "../../../../../docs/nodes/gesture.md?raw";
import selectDoc from "../../../../../docs/nodes/select.md?raw";
import uploadFileDoc from "../../../../../docs/nodes/upload-file.md?raw";
import typeDoc from "../../../../../docs/nodes/type.md?raw";
import shortcutDoc from "../../../../../docs/nodes/shortcut.md?raw";
import keyboardDoc from "../../../../../docs/nodes/keyboard.md?raw";
import setVariableDoc from "../../../../../docs/nodes/set-variable.md?raw";
import useVariableDoc from "../../../../../docs/nodes/use-variable.md?raw";
import scriptDoc from "../../../../../docs/nodes/script.md?raw";
import screenshotDoc from "../../../../../docs/nodes/screenshot.md?raw";
import waitDoc from "../../../../../docs/nodes/wait.md?raw";
import extractDoc from "../../../../../docs/nodes/extract.md?raw";
import assertDoc from "../../../../../docs/nodes/assert.md?raw";
import conditionalDoc from "../../../../../docs/nodes/conditional.md?raw";
import loopDoc from "../../../../../docs/nodes/loop.md?raw";
import workflowCallDoc from "../../../../../docs/nodes/workflow-call.md?raw";
import cookieStorageDoc from "../../../../../docs/nodes/cookie-storage.md?raw";
import networkMockDoc from "../../../../../docs/nodes/network-mock.md?raw";

export interface NodeDocEntry {
  /** Node type identifier matching WORKFLOW_NODE_DEFINITIONS */
  type: string;
  /** Display name for the node */
  name: string;
  /** Brief description for quick reference */
  summary: string;
  /** Category this node belongs to */
  category: string;
  /** Full markdown documentation */
  content: string;
}

/**
 * Map of node type to documentation content.
 * Keys match the node types in WORKFLOW_NODE_DEFINITIONS.
 */
export const NODE_DOCUMENTATION: Record<string, NodeDocEntry> = {
  navigate: {
    type: "navigate",
    name: "Navigate",
    summary: "Open a URL or jump to another scenario runbook",
    category: "Navigation & Context",
    content: navigateDoc,
  },
  click: {
    type: "click",
    name: "Click",
    summary: "Click DOM elements with timing and retry controls",
    category: "Pointer & Gestures",
    content: clickDoc,
  },
  hover: {
    type: "hover",
    name: "Hover",
    summary: "Move cursor and hold to trigger hover-only UI",
    category: "Pointer & Gestures",
    content: hoverDoc,
  },
  dragDrop: {
    type: "dragDrop",
    name: "Drag & Drop",
    summary: "Simulate HTML5 drag/drop or pointer drag sequences",
    category: "Pointer & Gestures",
    content: dragDropDoc,
  },
  focus: {
    type: "focus",
    name: "Focus",
    summary: "Focus an element before typing to trigger UI hooks",
    category: "Pointer & Gestures",
    content: focusDoc,
  },
  blur: {
    type: "blur",
    name: "Blur",
    summary: "Defocus to fire validation/onBlur handlers",
    category: "Pointer & Gestures",
    content: blurDoc,
  },
  tabSwitch: {
    type: "tabSwitch",
    name: "Tab Switch",
    summary: "Activate an existing tab or wait for a popup target",
    category: "Navigation & Context",
    content: tabSwitchDoc,
  },
  frameSwitch: {
    type: "frameSwitch",
    name: "Frame Switch",
    summary: "Enter/exit iframes using selector, index, or URL heuristics",
    category: "Navigation & Context",
    content: frameSwitchDoc,
  },
  scroll: {
    type: "scroll",
    name: "Scroll",
    summary: "Scroll the viewport or a container until a target is visible",
    category: "Navigation & Context",
    content: scrollDoc,
  },
  rotate: {
    type: "rotate",
    name: "Rotate",
    summary: "Swap viewport orientation/angle mid-run for mobile flows",
    category: "Navigation & Context",
    content: rotateDoc,
  },
  gesture: {
    type: "gesture",
    name: "Gesture",
    summary: "Dispatch swipe, pinch, tap, and long-press gestures",
    category: "Navigation & Context",
    content: gestureDoc,
  },
  select: {
    type: "select",
    name: "Select",
    summary: "Choose dropdown options by value/text/index",
    category: "Forms & Input",
    content: selectDoc,
  },
  uploadFile: {
    type: "uploadFile",
    name: "Upload File",
    summary: "Attach local files to file input elements",
    category: "Forms & Input",
    content: uploadFileDoc,
  },
  type: {
    type: "type",
    name: "Type",
    summary: "Type text with delay and randomization controls",
    category: "Forms & Input",
    content: typeDoc,
  },
  shortcut: {
    type: "shortcut",
    name: "Shortcut",
    summary: "Dispatch keyboard shortcuts (Ctrl/Cmd + key combos)",
    category: "Forms & Input",
    content: shortcutDoc,
  },
  keyboard: {
    type: "keyboard",
    name: "Keyboard",
    summary: "Low-level keydown/keyup/keypress simulation",
    category: "Forms & Input",
    content: keyboardDoc,
  },
  setVariable: {
    type: "setVariable",
    name: "Set Variable",
    summary: "Create/update a workflow variable from static or extracted data",
    category: "Data & Variables",
    content: setVariableDoc,
  },
  useVariable: {
    type: "useVariable",
    name: "Use Variable",
    summary: "Validate or transform variables for downstream nodes",
    category: "Data & Variables",
    content: useVariableDoc,
  },
  evaluate: {
    type: "evaluate",
    name: "Script",
    summary: "Run arbitrary JavaScript within the current frame",
    category: "Data & Variables",
    content: scriptDoc,
  },
  screenshot: {
    type: "screenshot",
    name: "Screenshot",
    summary: "Capture viewport or element screenshots for artifacts",
    category: "Assertions & Observability",
    content: screenshotDoc,
  },
  wait: {
    type: "wait",
    name: "Wait",
    summary: "Block until selectors, requests, or custom probes settle",
    category: "Assertions & Observability",
    content: waitDoc,
  },
  extract: {
    type: "extract",
    name: "Extract",
    summary: "Grab text/attributes/JSON and optionally store as variables",
    category: "Data & Variables",
    content: extractDoc,
  },
  assert: {
    type: "assert",
    name: "Assert",
    summary: "Validate expressions, selectors, or variable comparisons",
    category: "Assertions & Observability",
    content: assertDoc,
  },
  conditional: {
    type: "conditional",
    name: "Conditional",
    summary: "Branch workflows using expression/element/variable checks",
    category: "Workflow Logic",
    content: conditionalDoc,
  },
  loop: {
    type: "loop",
    name: "Loop",
    summary: "Execute a sub-graph repeatedly via for-each, repeat, or while logic",
    category: "Workflow Logic",
    content: loopDoc,
  },
  subflow: {
    type: "subflow",
    name: "Subflow",
    summary: "Run another saved workflow inline inside the same session",
    category: "Workflow Logic",
    content: workflowCallDoc,
  },
  setCookie: {
    type: "setCookie",
    name: "Set Cookie",
    summary: "Create/update browser cookies",
    category: "Storage & Network",
    content: cookieStorageDoc,
  },
  getCookie: {
    type: "getCookie",
    name: "Get Cookie",
    summary: "Read cookies into variables for assertions or reuse",
    category: "Storage & Network",
    content: cookieStorageDoc,
  },
  clearCookie: {
    type: "clearCookie",
    name: "Clear Cookie",
    summary: "Delete specific cookies or flush the jar",
    category: "Storage & Network",
    content: cookieStorageDoc,
  },
  setStorage: {
    type: "setStorage",
    name: "Set Storage",
    summary: "Write to localStorage/sessionStorage with JSON validation",
    category: "Storage & Network",
    content: cookieStorageDoc,
  },
  getStorage: {
    type: "getStorage",
    name: "Get Storage",
    summary: "Read storage values and store them as variables",
    category: "Storage & Network",
    content: cookieStorageDoc,
  },
  clearStorage: {
    type: "clearStorage",
    name: "Clear Storage",
    summary: "Remove storage keys or wipe the namespace",
    category: "Storage & Network",
    content: cookieStorageDoc,
  },
  networkMock: {
    type: "networkMock",
    name: "Network Mock",
    summary: "Stub, modify, or block HTTP traffic via Fetch interception",
    category: "Storage & Network",
    content: networkMockDoc,
  },
};

/**
 * Get documentation for a specific node type
 */
export function getNodeDocumentation(nodeType: string): NodeDocEntry | undefined {
  return NODE_DOCUMENTATION[nodeType];
}

/**
 * Get all node documentation entries grouped by category
 */
export function getNodeDocumentationByCategory(): Record<string, NodeDocEntry[]> {
  const byCategory: Record<string, NodeDocEntry[]> = {};

  for (const entry of Object.values(NODE_DOCUMENTATION)) {
    if (!byCategory[entry.category]) {
      byCategory[entry.category] = [];
    }
    byCategory[entry.category].push(entry);
  }

  return byCategory;
}

/**
 * Category ordering for consistent display
 */
export const CATEGORY_ORDER = [
  "Navigation & Context",
  "Pointer & Gestures",
  "Forms & Input",
  "Data & Variables",
  "Assertions & Observability",
  "Workflow Logic",
  "Storage & Network",
];
