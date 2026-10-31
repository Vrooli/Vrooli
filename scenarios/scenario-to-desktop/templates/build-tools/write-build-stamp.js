const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

const schema = 'scenario-to-desktop-template-generator-build-v1';
function inputHash(root) {
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
  // Track the producer/verifier itself as well as the compiled TypeScript inputs.
  for (const name of ['write-build-stamp.js', 'package.json', 'tsconfig.json']) selected.push(path.join(root, name));
  const relative = file => path.relative(root, file).split(path.sep).join('/');
  selected.sort((a, b) => Buffer.compare(Buffer.from(relative(a)), Buffer.from(relative(b))));
  const hash = crypto.createHash('sha256');
  for (const file of selected) {
    if (!fs.statSync(file).isFile()) throw new Error(`generator input is not a regular file: ${relative(file)}`);
    hash.update(relative(file)).update('\0').update(fs.readFileSync(file)).update('\0');
  }
  return hash.digest('hex');
}
function writeStamp(root) {
  const stamp = { schema, inputs_sha256: inputHash(root), built_at: new Date().toISOString() };
  fs.writeFileSync(path.join(root, 'dist', '.build-stamp.json'), `${JSON.stringify(stamp, null, 2)}\n`);
  return stamp;
}
function checkStamp(root) {
  let stamp;
  try {
    const stampPath = path.join(root, 'dist/.build-stamp.json');
    if (!fs.statSync(stampPath).isFile()) throw new Error('stamp is not a regular file');
    stamp = JSON.parse(fs.readFileSync(stampPath, 'utf8'));
  }
  catch (error) { throw new Error(`template generator build stamp is missing or invalid: ${error.message}`); }
  if (!stamp || stamp.schema !== schema || typeof stamp.inputs_sha256 !== 'string' || !/^[a-f0-9]{64}$/.test(stamp.inputs_sha256)) throw new Error('template generator build stamp is invalid');
  const current = inputHash(root);
  if (stamp.inputs_sha256 !== current) {
    throw new Error(`template generator is stale: recorded hash ${JSON.stringify(stamp.inputs_sha256)}, current hash ${JSON.stringify(current)}`);
  }
  if (!fs.statSync(path.join(root, 'dist/template-generator.js')).isFile()) throw new Error('compiled template generator is not a regular file');
  return { schema, inputs_sha256: current, freshness: 'current', generator: 'dist/template-generator.js' };
}
if (require.main === module) {
  const args = process.argv.slice(2);
  if (args.length > 1 || (args.length === 1 && args[0] !== '--check')) {
    console.error('Usage: node write-build-stamp.js [--check]');
    process.exitCode = 2;
  } else {
    try {
      if (args.length === 0) writeStamp(__dirname);
      else console.log(JSON.stringify(checkStamp(__dirname)));
    } catch (error) {
      console.error(`${error.message}; run vrooli scenario setup scenario-to-desktop`);
      process.exitCode = 1;
    }
  }
}
module.exports = { inputHash, writeStamp, checkStamp };
