/**
 * Stable UI contract for Browser Automation Studio playbooks.
 *
 * This is the single source of truth for semantic test IDs. The generated
 * selectors.manifest.json lets BAS resolve @selector/<group>.<name> without
 * coupling workflows to CSS or visible copy.
 */
const literalSelectors = {
  app: {
    root: "scenario-to-desktop-app",
    generateTab: "scenario-to-desktop-tab-generate",
    signingTab: "scenario-to-desktop-tab-signing",
  },
  generator: {
    scenarioPicker: "generator-scenario-picker",
    templatePicker: "generator-template-picker",
    generateSubmit: "generator-generate-submit",
    preflightRun: "generator-preflight-run",
    buildStart: "generator-build-start",
    smokeTestRun: "generator-smoke-test-run",
    liveDesktopLaunch: "generator-live-desktop-launch",
    buildSection: "generator-build-section",
    smokeTestSection: "generator-smoke-test-section",
    evidenceFixtureRun: "generator-evidence-fixture-run",
    evidenceFixtureResult: "generator-evidence-fixture-result",
  },
  signing: {
    scenarioSelect: "signing-scenario-select",
    enabled: "signing-enabled",
    windowsForm: "signing-windows-form",
    macosForm: "signing-macos-form",
    linuxForm: "signing-linux-form",
    validate: "signing-validate",
  },
  liveDesktop: {
    start: "live-desktop-start",
    canvas: "live-desktop-canvas",
    close: "live-desktop-close",
    error: "live-desktop-error",
  },
} as const;

type SelectorTree = { readonly [key: string]: string | SelectorTree };

const flattenSelectors = (
  tree: SelectorTree,
  prefix: string[] = [],
  result: Record<string, { testId: string; selector: string }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const path = [...prefix, key];
    if (typeof value === "string") {
      result[path.join(".")] = {
        testId: value,
        selector: `[data-testid="${value}"]`,
      };
      continue;
    }
    flattenSelectors(value, path, result);
  }
  return result;
};

export const selectors = literalSelectors;
export const selectorsManifest = {
  selectors: flattenSelectors(literalSelectors),
};
