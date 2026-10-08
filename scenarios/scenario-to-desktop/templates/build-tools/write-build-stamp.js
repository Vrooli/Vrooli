const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

const root = __dirname;
const selected = [];
function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name === 'dist') continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full);
    else if (entry.isFile() && entry.name.endsWith('.ts') && !entry.name.endsWith('.test.ts')) selected.push(full);
  }
}
walk(root);
for (const name of ['package.json', 'tsconfig.json']) selected.push(path.join(root, name));
selected.sort((a, b) => path.relative(root, a).localeCompare(path.relative(root, b)));
const hash = crypto.createHash('sha256');
for (const file of selected) {
  hash.update(path.relative(root, file).split(path.sep).join('/'));
  hash.update('\0');
  hash.update(fs.readFileSync(file));
  hash.update('\0');
}
const stamp = { schema: 'scenario-to-desktop-template-generator-build-v1', inputs_sha256: hash.digest('hex'), built_at: new Date().toISOString() };
fs.writeFileSync(path.join(root, 'dist', '.build-stamp.json'), `${JSON.stringify(stamp, null, 2)}\n`);
