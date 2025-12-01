#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const MANIFEST_FILENAME = 'selectors.manifest.json';
const SELECTOR_DIR_CANDIDATES = ['consts', 'constants'];

const resolveManifestPath = (scenarioDir) => {
  if (!scenarioDir) {
    throw new Error('Scenario directory is required');
  }
  const tried = [];
  for (const dirName of SELECTOR_DIR_CANDIDATES) {
    const manifestPath = path.resolve(
      scenarioDir,
      'ui',
      'src',
      dirName,
      MANIFEST_FILENAME,
    );
    tried.push(manifestPath);
    if (fs.existsSync(manifestPath)) {
      return manifestPath;
    }
  }
  throw new Error(
    `Selector manifest not found. Tried:\n${tried
      .map((candidate) => `  - ${candidate}`)
      .join('\n')}`,
  );
};

const loadManifest = (manifestPath) => {
  const raw = fs.readFileSync(manifestPath, 'utf8');
  try {
    return JSON.parse(raw);
  } catch (error) {
    throw new Error(`Failed to parse selector manifest: ${error.message}`);
  }
};

const loadSelectorRegistry = (scenarioDir) => {
  const manifestPath = resolveManifestPath(scenarioDir);
  const manifest = loadManifest(manifestPath);
  if (!manifest || typeof manifest !== 'object') {
    throw new Error('Selector manifest payload is invalid');
  }
  return {
    manifestPath,
    manifest,
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
          manifestPath: registry.manifestPath,
          manifest: registry.manifest,
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
