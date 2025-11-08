const fs = require('fs');
const path = require('path');

const rootDir = __dirname;
const distDir = path.join(rootDir, '..', 'dist');
const uiDir = path.join(rootDir, '..');

function cleanDist(dir) {
  if (fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
  fs.mkdirSync(dir, { recursive: true });
}

function copyAsset(asset) {
  const source = path.join(uiDir, asset);
  const dest = path.join(distDir, asset);
  if (!fs.existsSync(source)) {
    throw new Error(`Missing asset: ${asset}`);
  }
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.copyFileSync(source, dest);
}

function main() {
  cleanDist(distDir);
  ['index.html', 'app.js', 'styles.css'].forEach(copyAsset);
  console.log(`Copied UI assets to ${distDir}`);
}

main();
