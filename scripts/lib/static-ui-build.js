#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const uiDir = process.cwd();
const distDir = path.join(uiDir, 'dist');

function copyRecursive(src, dest) {
  const stats = fs.statSync(src);
  if (stats.isDirectory()) {
    fs.mkdirSync(dest, { recursive: true });
    for (const entry of fs.readdirSync(src)) {
      copyRecursive(path.join(src, entry), path.join(dest, entry));
    }
    return;
  }

  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.copyFileSync(src, dest);
}

function copyEntry(name, { flatten = false } = {}) {
  const src = path.join(uiDir, name);
  if (!fs.existsSync(src)) {
    return false;
  }

  const stats = fs.statSync(src);
  if (flatten && stats.isDirectory()) {
    for (const entry of fs.readdirSync(src)) {
      copyRecursive(path.join(src, entry), path.join(distDir, entry));
    }
    return true;
  }

  const dest = path.join(distDir, name);
  copyRecursive(src, dest);
  return true;
}

function collectHtmlFiles() {
  return fs
    .readdirSync(uiDir)
    .filter((file) => file.endsWith('.html') && file !== 'index.html');
}

(function main() {
  console.log(`[static-ui-build] Building assets from ${uiDir}`);
  fs.rmSync(distDir, { recursive: true, force: true });
  fs.mkdirSync(distDir, { recursive: true });

  const directories = [
    { name: 'public', flatten: true },
    { name: 'static', flatten: true },
    { name: 'assets' },
    { name: 'images' },
    { name: 'styles' },
    { name: 'scripts' },
    { name: 'fonts' },
    { name: 'data' }
  ];

  const files = [
    'index.html',
    'styles.css',
    'script.js',
    'app.js',
    'app.css',
    'main.js',
    'favicon.ico',
    'manifest.json'
  ];

  let copied = 0;

  for (const file of files) {
    if (copyEntry(file)) {
      copied += 1;
      console.log(`[static-ui-build] Copied ${file}`);
    }
  }

  for (const dir of directories) {
    if (copyEntry(dir.name, { flatten: Boolean(dir.flatten) })) {
      copied += 1;
      console.log(
        `[static-ui-build] Copied ${dir.name}${dir.flatten ? ' (flattened)' : ''}`
      );
    }
  }

  for (const html of collectHtmlFiles()) {
    if (copyEntry(html)) {
      copied += 1;
      console.log(`[static-ui-build] Copied ${html}`);
    }
  }

  if (copied === 0) {
    console.error(
      '[static-ui-build] No static assets were copied. Add static files or update the build script.'
    );
    process.exit(1);
  }

  console.log(
    `[static-ui-build] ✅ Built ${copied} asset group(s) into ${path.relative(uiDir, distDir)}`
  );
})();
