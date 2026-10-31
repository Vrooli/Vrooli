// Read-only conservation check: Jest discovery runs, no test bodies/browser work.
const { spawnSync } = require('node:child_process');
const { readdirSync, readFileSync } = require('node:fs');
const { join, relative } = require('node:path');
const { createHash } = require('node:crypto');
const root = join(__dirname, '..');
const jest = join(require.resolve('jest/package.json'), '..', 'bin', 'jest.js');
const configs = {
  unit: 'jest.config.js',
  native: 'jest.integration.config.js',
  managed: 'jest.managed.config.js',
  audio_device: 'jest.audio-device.config.js',
  motion: 'jest.motion.config.js',
  journeys: 'jest.journeys.config.js',
};
function walk(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const path = join(directory, entry.name);
    return entry.isDirectory() ? walk(path) : entry.isFile() && entry.name.endsWith('.test.ts') ? [relative(root, path)] : [];
  });
}
const selections = Object.fromEntries(Object.entries(configs).map(([kind, config]) => {
  const result = spawnSync(process.execPath, [jest, '--config', config, '--listTests', '--json', '--runInBand'], { cwd: root, encoding: 'utf8', timeout: 30000, maxBuffer: 4 << 20 });
  if (result.error || result.status !== 0) throw new Error(`Jest selection ${kind} failed: ${result.error || result.stderr}`);
  return [kind, JSON.parse(result.stdout).map(path => relative(root, path)).sort()];
}));
const expected = walk(join(root, 'tests')).sort();
const owners = new Map();
for (const [kind, paths] of Object.entries(selections)) {
  for (const path of paths) {
    if (owners.has(path)) throw new Error(`Test selected twice: ${path} (${owners.get(path)}, ${kind})`);
    owners.set(path, kind);
  }
}
const missing = expected.filter(path => !owners.has(path));
const extra = [...owners.keys()].filter(path => !expected.includes(path));
if (missing.length || extra.length) throw new Error(`Selection lost/added files: ${JSON.stringify({ missing, extra })}`);
const files = expected.map(path => ({ path, owner: owners.get(path), sha256: createHash('sha256').update(readFileSync(join(root, path))).digest('hex') }));
process.stdout.write(JSON.stringify({ counts: Object.fromEntries(Object.entries(selections).map(([kind, paths]) => [kind, paths.length])), configs, files }, null, 2) + '\n');
