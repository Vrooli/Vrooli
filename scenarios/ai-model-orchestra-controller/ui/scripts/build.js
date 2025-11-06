const fs = require('fs');
const path = require('path');

const projectRoot = path.resolve(__dirname, '..');
const distDir = path.join(projectRoot, 'dist');
const sourceFiles = [
  { src: 'dashboard.html', dest: 'index.html' },
  { src: 'dashboard.css', dest: 'dashboard.css' },
  { src: 'dashboard.js', dest: 'dashboard.js' }
];
const assetDirectories = ['components', 'services', 'utils'];

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function copyFile(src, dest) {
  if (!fs.existsSync(src)) {
    console.warn(`[build-ui] Skipping missing file: ${src}`);
    return;
  }
  ensureDir(path.dirname(dest));
  fs.copyFileSync(src, dest);
}

function copyDirectory(src, dest) {
  if (!fs.existsSync(src)) {
    console.warn(`[build-ui] Skipping missing directory: ${src}`);
    return;
  }
  ensureDir(dest);
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      copyDirectory(srcPath, destPath);
    } else {
      ensureDir(path.dirname(destPath));
      fs.copyFileSync(srcPath, destPath);
    }
  }
}

function cleanDist() {
  if (fs.existsSync(distDir)) {
    fs.rmSync(distDir, { recursive: true, force: true });
  }
  ensureDir(distDir);
}

function build() {
  console.log('[build-ui] Creating production bundle...');
  cleanDist();

  sourceFiles.forEach(({ src, dest }) => {
    copyFile(path.join(projectRoot, src), path.join(distDir, dest));
  });

  assetDirectories.forEach((dir) => {
    copyDirectory(path.join(projectRoot, dir), path.join(distDir, dir));
  });

  console.log(`[build-ui] Bundle created in ${distDir}`);
}

build();
