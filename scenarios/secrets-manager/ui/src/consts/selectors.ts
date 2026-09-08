import { librarySelectors } from "./selectors.library";
export { librarySelectors };
export const selectors = {
  navigation: {
    vault: '[aria-label="Vault"]',
    access: '[aria-label="Access"]',
    activity: '[aria-label="Activity"]',
    recovery: '[aria-label="Recovery"]',
    sources: '[aria-label="Sources"]',
    settings: '[aria-label="Settings"]'
  },
  vault: {
    collection: '[data-experience-surface="vault-collection"]',
    collectionLoading: '[data-experience-surface="vault-collection"][data-experience-state="loading"]',
    collectionReady: '[data-experience-surface="vault-collection"][data-experience-state="ready"]',
    inspector: '[data-experience-surface="vault-inspector"]',
    item: '[data-testid^="vault-item-"]',
    newItem: '[data-testid="vault-new-item"]',
    reveal: '[data-testid="vault-reveal"]',
    lock: '[data-testid="vault-lock"]',
    unlock: '[data-testid="vault-unlock"]',
    search: 'input[aria-label="Search vault"]',
    filter: 'select[aria-label="Filter item type"]'
  },
  recovery: { status: "recovery-status" }
} as const;

export const selectorsManifest = {
  schemaVersion: "2025.11",
  generatedAt: "managed-by-secrets-manager",
  selectors: {
    ...Object.fromEntries(Object.entries(selectors.navigation).map(([id, selector]) => [`navigation.${id}`, { selector }])),
    ...Object.fromEntries(Object.entries(selectors.navigation).map(([id, selector]) => [id, { selector }])),
    ...Object.fromEntries(Object.entries(selectors.vault).map(([id, selector]) => [`vault.${id}`, { selector }])),
    "vault.collection.loading": { selector: selectors.vault.collectionLoading },
    "vault.collection.ready": { selector: selectors.vault.collectionReady },
    "recovery.status": { testId: selectors.recovery.status, selector: `[data-testid="${selectors.recovery.status}"]` }
  },
  dynamicSelectors: {}
} as const;
