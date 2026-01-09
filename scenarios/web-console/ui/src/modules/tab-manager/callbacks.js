// @ts-check

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * @typedef {Object} TabManagerCallbacks
 * @property {((tab: TerminalTab, previousId: string | null) => void) | null} onActiveTabChanged
 * @property {((tab: TerminalTab) => void) | null} onTabCloseRequested
 * @property {((tab: TerminalTab) => void) | null} onTabCreated
 * @property {((tab: TerminalTab) => void) | null} onTabMetadataChanged
 * @property {((tab: TerminalTab) => void) | null} onTabCustomizationOpened
 * @property {(() => void) | null} onTabCustomizationClosed
 * @property {((payload: { id: string; command: string }) => void) | null} onShortcut
 */

/** @type {TabManagerCallbacks} */
export const tabCallbacks = {
  onActiveTabChanged: null,
  onTabCloseRequested: null,
  onTabCreated: null,
  onTabMetadataChanged: null,
  onTabCustomizationOpened: null,
  onTabCustomizationClosed: null,
  onShortcut: null,
};

/**
 * @param {Partial<TabManagerCallbacks>} [options={}]
 */
export function configureTabManager(options = {}) {
  tabCallbacks.onActiveTabChanged =
    typeof options.onActiveTabChanged === "function"
      ? options.onActiveTabChanged
      : null;
  tabCallbacks.onTabCloseRequested =
    typeof options.onTabCloseRequested === "function"
      ? options.onTabCloseRequested
      : null;
  tabCallbacks.onTabCreated =
    typeof options.onTabCreated === "function"
      ? options.onTabCreated
      : null;
  tabCallbacks.onTabMetadataChanged =
    typeof options.onTabMetadataChanged === "function"
      ? options.onTabMetadataChanged
      : null;
  tabCallbacks.onTabCustomizationOpened =
    typeof options.onTabCustomizationOpened === "function"
      ? options.onTabCustomizationOpened
      : null;
  tabCallbacks.onTabCustomizationClosed =
    typeof options.onTabCustomizationClosed === "function"
      ? options.onTabCustomizationClosed
      : null;
  tabCallbacks.onShortcut =
    typeof options.onShortcut === "function" ? options.onShortcut : null;
}
