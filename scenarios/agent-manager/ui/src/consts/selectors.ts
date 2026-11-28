import { createSelectorRegistry } from "@vrooli/ui-selectors";

const registry = createSelectorRegistry({
  quickRun: {
    open: "quick-run-open",
    description: "quick-run-description",
    title: "quick-run-title",
    projectRoot: "quick-run-project-root",
    role: "quick-run-role",
    maxTurns: "quick-run-max-turns",
    timeoutMinutes: "quick-run-timeout-minutes",
    runMode: "quick-run-mode",
    network: "quick-run-network",
    next: "quick-run-next",
    start: "quick-run-start",
  },
  run: {
    detail: "run-detail",
    assistantMessage: "run-assistant-message",
  },
}, {});

export const selectors = registry.selectors;
export const selectorsManifest = registry.manifest;
