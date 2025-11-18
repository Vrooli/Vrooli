#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const Module = require('module');

const cache = new Map();

const resolveModule = (moduleName, scenarioDir) => {
  const searchRoots = [
    path.join(scenarioDir, 'ui'),
    scenarioDir,
    path.dirname(scenarioDir),
    process.cwd(),
  ];
  for (const root of searchRoots) {
    try {
      return require.resolve(moduleName, { paths: [root] });
    } catch (error) {
      // keep searching
    }
  }
  throw new Error(
    `Unable to locate module '${moduleName}'. Install dependencies for ${scenarioDir} (pnpm install) and try again.`,
  );
};

const resolveSelectorsPath = (scenarioDir) => {
  if (!scenarioDir) {
    throw new Error('Scenario directory is required');
  }
  const resolved = path.resolve(scenarioDir, 'ui', 'src', 'consts', 'selectors.ts');
  if (!fs.existsSync(resolved)) {
    throw new Error(`Selector registry not found at ${resolved}`);
  }
  return resolved;
};

const requireSelectorsModule = (selectorsPath, scenarioDir) => {
  const cacheEntry = cache.get(selectorsPath);
  const stats = fs.statSync(selectorsPath);
  if (cacheEntry && cacheEntry.mtimeMs === stats.mtimeMs) {
    return cacheEntry.exports;
  }

  const tsPath = resolveModule('typescript', scenarioDir);
  const typescript = require(tsPath);
  const source = fs.readFileSync(selectorsPath, 'utf8');
  const compiled = typescript.transpileModule(source, {
    compilerOptions: {
      module: typescript.ModuleKind.CommonJS,
      target: typescript.ScriptTarget.ES2019,
      esModuleInterop: true,
    },
  }).outputText;

  if (!compiled) {
    throw new Error('Failed to compile selector registry');
  }

  const moduleWrapper = new Module(selectorsPath, module);
  moduleWrapper.filename = selectorsPath;
  moduleWrapper.paths = Module._nodeModulePaths(path.dirname(selectorsPath));
  moduleWrapper._compile(compiled, selectorsPath);
  const exported = moduleWrapper.exports;
  if (!exported || !exported.testIds || !exported.workflowSelectors) {
    throw new Error('Selector registry must export testIds and workflowSelectors');
  }

  if (!exported.dynamicSelectors) {
    exported.dynamicSelectors = {};
  }

  cache.set(selectorsPath, { mtimeMs: stats.mtimeMs, exports: exported });
  return exported;
};

const loadSelectorRegistry = (scenarioDir) => {
  const selectorsPath = resolveSelectorsPath(scenarioDir);
  const exports = requireSelectorsModule(selectorsPath, scenarioDir);
  return {
    selectorsPath,
    testIds: exports.testIds,
    workflowSelectors: exports.workflowSelectors,
    dynamicSelectors: exports.dynamicSelectors || {},
  };
};

const args = process.argv.slice(2);
if (require.main === module) {
  let scenarioDir = '';
  for (let i = 0; i < args.length; i += 1) {
    if (args[i] === '--scenario') {
      scenarioDir = args[i + 1] || '';
      i += 1;
    }
  }
  if (!scenarioDir) {
    console.error('Usage: selector-registry.js --scenario <path>');
    process.exit(1);
  }
  try {
    const registry = loadSelectorRegistry(scenarioDir);
    process.stdout.write(
      JSON.stringify(
        {
          selectorsPath: registry.selectorsPath,
          testIds: registry.testIds,
          workflowSelectors: registry.workflowSelectors,
          dynamicSelectors: registry.dynamicSelectors,
        },
        null,
        2,
      ),
    );
  } catch (error) {
    console.error(error.message || String(error));
    process.exit(1);
  }
}

module.exports = {
  loadSelectorRegistry,
};
