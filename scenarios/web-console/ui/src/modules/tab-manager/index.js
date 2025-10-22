export { configureTabManager } from "./callbacks.js";
export { setTerminalEventHandlers } from "./terminal-handlers.js";
export { initializeShortcutButtons } from "./shortcuts.js";
export {
  initializeTabCustomizationUI,
  openTabCustomization,
  closeTabCustomization,
  positionTabContextMenu,
} from "./customization.js";
export {
  createTerminalTab,
  destroyTerminalTab,
  getActiveTab,
  findTab,
  setActiveTab,
  renderTabs,
  applyTabAppearance,
  refreshTabButton,
  isDetachedTab,
  getDetachedTabs,
  getTabs,
} from "./tabs.js";
