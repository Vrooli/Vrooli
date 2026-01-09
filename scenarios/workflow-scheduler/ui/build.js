#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const ROOT = __dirname;
const DIST = path.join(ROOT, 'dist');

function copyPath(src, dest) {
  const stat = fs.statSync(src);
  if (stat.isDirectory()) {
    fs.mkdirSync(dest, { recursive: true });
    for (const entry of fs.readdirSync(src)) {
      const srcEntry = path.join(src, entry);
      const destEntry = path.join(dest, entry);
      copyPath(srcEntry, destEntry);
    }
  } else {
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.copyFileSync(src, dest);
  }
}

function main() {
  fs.rmSync(DIST, { recursive: true, force: true });
  fs.mkdirSync(DIST, { recursive: true });

  const assets = [
    'index.html',
    'dashboard.html',
    'bridge-init.js',
    'assets',
    'public',
    'styles'
  ];

  let copied = 0;
  for (const asset of assets) {
    const assetPath = path.join(ROOT, asset);
    if (fs.existsSync(assetPath)) {
      copyPath(assetPath, path.join(DIST, asset));
      copied += 1;
    }
  }

  if (copied === 0) {
    console.warn('No UI assets were copied into dist/. Ensure source files exist.');
  } else {
    console.log(`Copied ${copied} UI asset(s) into dist/.`);
  }
}

main();
