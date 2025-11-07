const fs = require('fs');
const path = require('path');

const UI_ROOT = __dirname;
const DIST_DIR = path.join(UI_ROOT, 'dist');
const IGNORE_DIRS = new Set(['node_modules', 'dist', 'coverage', '__tests__']);
const IGNORE_FILES = new Set(['build.js', 'server.js', 'package.json', 'package-lock.json', 'pnpm-lock.yaml', '.DS_Store']);

function copyRecursive(src, dest) {
  const stats = fs.statSync(src);
  if (stats.isDirectory()) {
    if (IGNORE_DIRS.has(path.basename(src))) {
      return;
    }
    fs.mkdirSync(dest, { recursive: true });
    for (const entry of fs.readdirSync(src)) {
      copyRecursive(path.join(src, entry), path.join(dest, entry));
    }
    return;
  }

  const basename = path.basename(src);
  if (IGNORE_FILES.has(basename)) {
    return;
  }

  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.copyFileSync(src, dest);
}

function ensureDist() {
  fs.rmSync(DIST_DIR, { recursive: true, force: true });
  fs.mkdirSync(DIST_DIR, { recursive: true });

  for (const entry of fs.readdirSync(UI_ROOT)) {
    if (IGNORE_DIRS.has(entry) || IGNORE_FILES.has(entry)) {
      continue;
    }
    copyRecursive(path.join(UI_ROOT, entry), path.join(DIST_DIR, entry));
  }

  const indexPath = path.join(DIST_DIR, 'index.html');
  if (!fs.existsSync(indexPath)) {
    throw new Error(`Expected ${indexPath} to exist after build.`);
  }

  console.log(`Visited Tracker UI assets built to ${DIST_DIR}`);
}

ensureDist();
