const assert = require('node:assert/strict');
const { test } = require('node:test');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const owner = path.resolve(__dirname, '../write-build-stamp.js');

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'desktop-stamp-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  fs.mkdirSync(path.join(root, 'dist'));
  fs.copyFileSync(owner, path.join(root, 'write-build-stamp.js'));
  for (const name of ['template-generator.ts', 'package.json', 'tsconfig.json']) fs.writeFileSync(path.join(root, name), '{}\n');
  fs.writeFileSync(path.join(root, 'dist/template-generator.js'), '// compiled fixture\n');
  return root;
}
function invoke(root, ...args) { return spawnSync(process.execPath, [path.join(root, 'write-build-stamp.js'), ...args], { encoding: 'utf8' }); }
test('read-only check refuses stale stamp without rewriting any bytes', t => {
  const root = fixture(t);
  assert.equal(invoke(root).status, 0);
  const stamp = path.join(root, 'dist/.build-stamp.json');
  const before = fs.readFileSync(stamp, 'utf8');
  fs.appendFileSync(path.join(root, 'template-generator.ts'), '// changed\n');
  const result = invoke(root, '--check');
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /stale/);
  assert.equal(fs.readFileSync(stamp, 'utf8'), before);
});

test('current read-only check reports identity while preserving stamp bytes and mtime', t => {
  const root = fixture(t);
  assert.equal(invoke(root).status, 0);
  const stamp = path.join(root, 'dist/.build-stamp.json');
  const before = fs.readFileSync(stamp, 'utf8'); const mtime = fs.statSync(stamp).mtimeMs;
  const result = invoke(root, '--check');
  assert.equal(result.status, 0, result.stderr);
  const report = JSON.parse(result.stdout);
  assert.equal(report.freshness, 'current');
  assert.match(report.inputs_sha256, /^[a-f0-9]{64}$/);
  assert.equal(fs.readFileSync(stamp, 'utf8'), before);
  assert.equal(fs.statSync(stamp).mtimeMs, mtime);
});
for (const kind of ['missing', 'malformed', 'wrong-schema', 'missing-entry', 'nonregular-entry']) {
  test(`refuses ${kind} without producing a fresh stamp`, t => {
    const root = fixture(t); assert.equal(invoke(root).status, 0);
    const stamp = path.join(root, 'dist/.build-stamp.json');
    if (kind === 'missing') fs.unlinkSync(stamp);
    if (kind === 'malformed') fs.writeFileSync(stamp, '{invalid');
    if (kind === 'wrong-schema') fs.writeFileSync(stamp, JSON.stringify({ schema: 'future', inputs_sha256: '0'.repeat(64) }));
    if (kind.includes('entry')) { fs.unlinkSync(path.join(root, 'dist/template-generator.js')); if (kind === 'nonregular-entry') fs.mkdirSync(path.join(root, 'dist/template-generator.js')); }
    const before = fs.existsSync(stamp) ? fs.readFileSync(stamp, 'utf8') : null;
    const result = invoke(root, '--check');
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /vrooli scenario setup scenario-to-desktop/);
    assert.equal(fs.existsSync(stamp) ? fs.readFileSync(stamp, 'utf8') : null, before);
  });
}
test('source, package, tsconfig and producer edits invalidate, tests and outputs do not', t => {
  const { inputHash } = require(owner);
  const root = fixture(t); const before = inputHash(root);
  fs.writeFileSync(path.join(root, 'ignored.test.ts'), 'test');
  fs.writeFileSync(path.join(root, 'dist/ignored.ts'), 'output');
  fs.mkdirSync(path.join(root, 'node_modules')); fs.writeFileSync(path.join(root, 'node_modules/ignored.ts'), 'dependency');
  assert.equal(inputHash(root), before);
  for (const name of ['template-generator.ts', 'package.json', 'tsconfig.json', 'write-build-stamp.js']) {
    const file = path.join(root, name); const bytes = fs.readFileSync(file);
    fs.appendFileSync(file, '\n'); assert.notEqual(inputHash(root), before, name); fs.writeFileSync(file, bytes);
  }
});
test('unknown command flags refuse without writing', t => {
  const root = fixture(t);
  const result = invoke(root, '--unexpected');
  assert.equal(result.status, 2);
  assert.match(result.stderr, /Usage/);
  assert.doesNotMatch(result.stderr, /scenario setup/);
  assert.equal(fs.existsSync(path.join(root, 'dist/.build-stamp.json')), false);
});
