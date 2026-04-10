import fs from "node:fs";
import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";
import { createRequire } from "node:module";
import tsParser from "@typescript-eslint/parser";

const __dirname = dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
const pluginRoot = dirname(require.resolve("@typescript-eslint/eslint-plugin/package.json"));
const { plugin: tsPlugin } = require(join(pluginRoot, "dist/raw-plugin.js"));
const importGraphCache = new Map();
const sourceExtensions = [".ts", ".tsx", ".js", ".jsx", ".mts", ".cts"];

function isFile(path) {
  try {
    return fs.statSync(path).isFile();
  } catch {
    return false;
  }
}

function resolveRelativeImport(fromFile, specifier) {
  const basePath = resolve(dirname(fromFile), specifier);
  const candidates = [
    basePath,
    ...sourceExtensions.map((ext) => `${basePath}${ext}`),
    ...sourceExtensions.map((ext) => join(basePath, `index${ext}`))
  ];

  return candidates.find((candidate) => isFile(candidate)) ?? null;
}

function collectRelativeImports(filePath) {
  if (importGraphCache.has(filePath)) {
    return importGraphCache.get(filePath);
  }

  let source = "";
  try {
    source = fs.readFileSync(filePath, "utf8");
  } catch {
    importGraphCache.set(filePath, []);
    return [];
  }

  const specifiers = new Set();
  const staticImportPattern = /(?:import|export)\s+(?:[^"'`]*?\s+from\s*)?["'](\.[^"']+)["']/g;
  const dynamicImportPattern = /import\(\s*["'](\.[^"']+)["']\s*\)/g;

  for (const pattern of [staticImportPattern, dynamicImportPattern]) {
    let match;
    while ((match = pattern.exec(source)) !== null) {
      if (match[1]) {
        specifiers.add(match[1]);
      }
    }
  }

  const imports = Array.from(specifiers)
    .map((specifier) => resolveRelativeImport(filePath, specifier))
    .filter((resolvedPath) => resolvedPath !== null);

  importGraphCache.set(filePath, imports);
  return imports;
}

function importsBackTo(startFile, currentFile, visited = new Set()) {
  if (currentFile === startFile) {
    return true;
  }
  if (visited.has(currentFile)) {
    return false;
  }

  visited.add(currentFile);
  return collectRelativeImports(currentFile).some((dependency) => importsBackTo(startFile, dependency, visited));
}

const importPlugin = {
  rules: {
    // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
    "no-cycle": {
      meta: {
        type: "problem",
        schema: [],
        docs: {
          description: "Detect relative import cycles that cause runtime initialization failures."
        }
      },
      create(context) {
        const filename = context.filename;
        if (!filename || filename === "<input>") {
          return {};
        }

        const checkNode = (node) => {
          const sourceValue = node.source?.value;
          if (typeof sourceValue !== "string" || !sourceValue.startsWith(".")) {
            return;
          }

          const resolvedImport = resolveRelativeImport(filename, sourceValue);
          if (!resolvedImport) {
            return;
          }

          if (importsBackTo(filename, resolvedImport)) {
            context.report({
              node,
              message: `Relative import cycle detected through '${sourceValue}'.`
            });
          }
        };

        return {
          ImportDeclaration: checkNode,
          ExportAllDeclaration: checkNode,
          ExportNamedDeclaration: checkNode
        };
      }
    }
  }
};

const nodeGlobals = {
  AbortController: "readonly",
  Buffer: "readonly",
  clearInterval: "readonly",
  clearTimeout: "readonly",
  console: "readonly",
  fetch: "readonly",
  global: "readonly",
  module: "readonly",
  process: "readonly",
  require: "readonly",
  setInterval: "readonly",
  setTimeout: "readonly",
  __dirname: "readonly",
  __filename: "readonly"
};

const browserGlobals = {
  AbortController: "readonly",
  Headers: "readonly",
  Request: "readonly",
  Response: "readonly",
  URL: "readonly",
  URLSearchParams: "readonly",
  WebSocket: "readonly",
  clearTimeout: "readonly",
  console: "readonly",
  document: "readonly",
  fetch: "readonly",
  FormData: "readonly",
  localStorage: "readonly",
  navigator: "readonly",
  setTimeout: "readonly",
  window: "readonly"
};

/** @type {import("eslint").Linter.FlatConfig[]} */
export default [
  {
    ignores: ["dist", "coverage", "node_modules"]
  },
  js.configs.recommended,
  {
    files: ["*.js", "*.mjs"],
    languageOptions: {
      globals: nodeGlobals
    }
  },
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: 2021,
        sourceType: "module",
        project: "./tsconfig.json",
        tsconfigRootDir: __dirname
      },
      globals: browserGlobals
    },
    plugins: {
      "@typescript-eslint": tsPlugin,
      import: importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh
    },
    rules: {
      "no-unused-vars": "off",
      "no-undef": "off",
      "no-redeclare": "off",

      // ════════════════════════════════════════════════════════════════════════
      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
      //
      // These rules keep Test Genie stable at runtime by protecting hook
      // ordering, nullability, and typed transport/state boundaries.
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Keeps unchecked values from leaking through UI decision boundaries.
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",

      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",

      // CRITICAL: Prevents explicit any from disabling type safety.
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
      "import/no-cycle": "error",

      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "@typescript-eslint/no-redeclare": "error",
      "@typescript-eslint/consistent-type-imports": "warn",
      "react-hooks/exhaustive-deps": "warn",
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }]
    }
  },
  {
    files: ["src/**/*.test.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-unused-vars": "off",
      "@typescript-eslint/no-non-null-assertion": "off"
    }
  }
];
