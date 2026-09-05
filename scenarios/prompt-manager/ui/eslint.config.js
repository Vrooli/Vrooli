import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import importPlugin from "eslint-plugin-import";
import globals from "globals";

/**
 * ESLint flat config for prompt-manager UI
 *
 * STABILITY CRITICAL RULES (DO NOT REMOVE):
 * The rules below preserve hook ordering, dependency freshness, null safety,
 * checked value access, and module initialization ordering.
 *
 * See ui-health skill for rationale.
 */
/**
 * World layer rule (src/world/README.md). Each layer lists the layers it must
 * never import. Enforced with import/no-restricted-paths; the fixtures under
 * src/world/** /__lint__ prove the rule fires (src/world/__lint__/layerRule.test.ts).
 */
const WORLD_LAYER_FORBIDS = {
  config: ["sim", "engine", "scene", "hud", "data"],
  sim: ["engine", "scene", "hud", "data"],
  engine: ["sim", "scene", "hud", "data"],
  scene: ["hud", "data"],
  hud: ["scene", "engine"],
  data: ["scene", "hud", "engine"],
};
const WORLD_LAYER_ZONES = Object.entries(WORLD_LAYER_FORBIDS).flatMap(([layer, forbidden]) =>
  forbidden.map((from) => ({
    target: `./src/world/${layer}`,
    from: `./src/world/${from}`,
    message: `world/${layer} must not import world/${from}; see src/world/README.md for the layer rule.`,
  })),
);
// Only the route component (src/world/index.tsx) composes scene and hud.
const WORLD_PRESENTATION_ZONES = [
  "./src/components",
  "./src/hooks",
  "./src/services",
  "./src/stores",
  "./src/lib",
  "./src/app",
  "./src/test",
  "./src/test-utils",
  "./src/types",
  "./src/constants",
].flatMap((target) => [
  { target, from: "./src/world/scene", message: "Only src/world/index.tsx mounts world/scene." },
  { target, from: "./src/world/hud", message: "Only src/world/index.tsx mounts world/hud." },
]);
// The simulation and the control surface are renderer-free and framework-free.
const RENDERER_FREE_IMPORTS = {
  paths: ["three", "react", "react-dom", "@react-three/fiber", "@react-three/drei", "@react-three/postprocessing", "postprocessing", "n8ao", "camera-controls", "zustand"].map((name) => ({
    name,
    message: "world/sim and world/config are renderer-free: no three, react or store imports.",
  })),
  patterns: [
    { group: ["three/*", "@react-three/*", "react/*", "react-dom/*"], message: "world/sim and world/config are renderer-free." },
  ],
};

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage", "*.config.js", "**/__lint__/**"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.strictTypeChecked],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        project: ["./tsconfig.json", "./tsconfig.node.json"],
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "import": importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
          project: ["./tsconfig.json", "./tsconfig.node.json"],
        },
      },
    },
    rules: {
      // ════════════════════════════════════════════════════════════════════════
      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
      //
      // These rules prevent runtime crashes. If you encounter errors:
      // ✅ DO: Fix the code with optional chaining (?.), null checks, or proper types
      // ❌ DON'T: Disable the rule, use "as" casts, or use non-null assertion (!)
      //
      // Removing these rules WILL cause production crashes that are much harder
      // to debug than the lint errors they produce at development time.
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      // Detects early returns before hooks, conditional hook calls, and unstable hook ordering.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Catches temporal-dead-zone crashes ("Cannot access 'x' before
      // initialization"): a lazy useState initializer or callback reading a
      // const declared later in the same component. tsc does not flag these
      // inside closures; the world view shipped one and crashed on mount.
      "@typescript-eslint/no-use-before-define": ["error", { functions: false, classes: false, variables: true, allowNamedExports: true, ignoreTypeReferences: true }],

      // CRITICAL: Catches stale-closure bugs when dependencies drift from actual usage.
      "react-hooks/exhaustive-deps": "warn",

      // CRITICAL: Prevents explicit 'any' from disabling type safety at UI boundaries.
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Prevents non-null assertion (!) from bypassing TypeScript null checks.
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches unsafe arguments flowing from unchecked values into typed APIs.
      "@typescript-eslint/no-unsafe-argument": "warn",

      // CRITICAL: Catches assigning unchecked values that spread `any` through the codebase.
      "@typescript-eslint/no-unsafe-assignment": "warn",

      // CRITICAL: Catches invoking unchecked values that will crash at runtime.
      "@typescript-eslint/no-unsafe-call": "warn",

      // CRITICAL: Catches member access on unchecked values that will crash at runtime.
      "@typescript-eslint/no-unsafe-member-access": "warn",

      // CRITICAL: Catches returning unchecked values that leak unsafe types to callers.
      "@typescript-eslint/no-unsafe-return": "warn",

      // CRITICAL: Detects circular dependencies that produce initialization-order failures.
      "import/no-cycle": "error",

      // React refresh for HMR
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],

      // Allow unused variables prefixed with underscore
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }
      ],

      // Relax some strict rules that are too noisy
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/no-confusing-void-expression": "off",

      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["**/test-utils", "**/test-utils/*", "@/test-utils", "@/test-utils/*"],
              message: "Production code must not import test-only helpers from src/test-utils.",
            },
            {
              group: ["**/features/*/mocks", "**/features/*/mocks/*"],
              message: "Production code must not import feature test mocks.",
            },
          ],
        },
      ],
    },
  },
  // World layer boundary (see src/world/README.md)
  {
    files: ["src/**/*.{ts,tsx}"],
    rules: {
      "import/no-restricted-paths": [
        "error",
        { zones: [...WORLD_LAYER_ZONES, ...WORLD_PRESENTATION_ZONES] },
      ],
    },
  },
  {
    files: ["src/world/sim/**/*.ts", "src/world/config/**/*.ts"],
    rules: {
      "no-restricted-imports": ["error", RENDERER_FREE_IMPORTS],
    },
  },
  // Test file overrides
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}", "**/*.integration.test.{ts,tsx}", "**/test/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": "off",
      // Allow unbound methods in tests (common pattern with vi.mocked)
      "@typescript-eslint/unbound-method": "off",
      // Allow act from @testing-library/react (deprecation refers to react-dom/test-utils)
      "@typescript-eslint/no-deprecated": "off",
      // Relax unsafe rules in tests since we often work with mocks
      "@typescript-eslint/no-unsafe-call": "off",
      "@typescript-eslint/no-unsafe-member-access": "off",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-unsafe-return": "off",
      // Test utilities aren't meant for HMR
      "react-refresh/only-export-components": "off",
    },
  }
);
