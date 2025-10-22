export const tabCallbacks = {
  onActiveTabChanged: null,
  onTabCloseRequested: null,
  onTabCreated: null,
  onTabMetadataChanged: null,
  onTabCustomizationOpened: null,
  onTabCustomizationClosed: null,
  onShortcut: null,
};

export function configureTabManager(options = {}) {
  Object.entries(options).forEach(([key, value]) => {
    tabCallbacks[key] = typeof value === "function" ? value : null;
  });
}
