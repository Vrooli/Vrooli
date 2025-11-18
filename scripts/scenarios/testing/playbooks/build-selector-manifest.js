#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const Module = require('module');

const MANIFEST_FILENAME = 'selectors.manifest.json';

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
  return moduleWrapper.exports;
};

const buildManifestPayload = (selectorsModule) => {
  if (!selectorsModule || !selectorsModule.selectorsManifest) {
    throw new Error('selectors.ts must export selectorsManifest');
  }
  return selectorsModule.selectorsManifest;
};

const writeManifest = (scenarioDir, payload) => {
  const manifestPath = path.resolve(
    scenarioDir,
    'ui',
    'src',
    'consts',
    MANIFEST_FILENAME,
  );
  const envelope = {
    schemaVersion: '2025.11',
    generatedAt: new Date().toISOString(),
    ...payload,
  };
  fs.writeFileSync(manifestPath, `${JSON.stringify(envelope, null, 2)}\n`, 'utf8');
  return manifestPath;
};

const args = process.argv.slice(2);
let scenarioDir = '';
for (let i = 0; i < args.length; i += 1) {
  if (args[i] === '--scenario') {
    scenarioDir = args[i + 1] || '';
    i += 1;
  }
}

if (!scenarioDir) {
  console.error('Usage: build-selector-manifest.js --scenario <path>');
  process.exit(1);
}

try {
  const selectorsPath = resolveSelectorsPath(scenarioDir);
  const selectorsModule = requireSelectorsModule(selectorsPath, scenarioDir);
  const payload = buildManifestPayload(selectorsModule);
  const manifestPath = writeManifest(scenarioDir, payload);
  console.log(`✅ Selector manifest written to ${manifestPath}`);
} catch (error) {
  console.error(error.message || String(error));
  process.exit(1);
}
