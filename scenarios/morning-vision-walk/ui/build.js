#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const projectDir = __dirname;
const distDir = path.join(projectDir, 'dist');
const pkg = (() => {
  try {
    return require('./package.json');
  } catch (error) {
    return {};
  }
})();
const label = pkg.displayName || pkg.name || 'ui';

function copyRecursive(source, target) {
  const stat = fs.statSync(source);
  if (stat.isDirectory()) {
    fs.mkdirSync(target, { recursive: true });
    for (const entry of fs.readdirSync(source)) {
      copyRecursive(path.join(source, entry), path.join(target, entry));
    }
  } else {
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.copyFileSync(source, target);
  }
}

fs.rmSync(distDir, { recursive: true, force: true });
fs.mkdirSync(distDir, { recursive: true });

const assets = ['index.html', 'script.js', 'styles.css', 'assets', 'public'];
let copied = 0;

for (const asset of assets) {
  const sourcePath = path.join(projectDir, asset);
  if (fs.existsSync(sourcePath)) {
    const targetPath = path.join(distDir, asset);
    copyRecursive(sourcePath, targetPath);
    copied += 1;
  }
}

if (!copied) {
  const placeholder = `<!doctype html><html><head><meta charset=\"utf-8\"><title>${label}</title></head><body><div id=\"app\">${label} bundle placeholder</div></body></html>`;
  fs.writeFileSync(path.join(distDir, 'index.html'), placeholder);
}

console.log(`Built ${label} assets -> ${distDir}`);
