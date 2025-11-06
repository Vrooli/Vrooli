#!/usr/bin/env node
const fsp = require('fs/promises');
const path = require('path');

const rootDir = path.resolve(__dirname, '..');
const srcDir = path.join(rootDir, 'src');
const distDir = path.join(rootDir, 'dist');

async function ensureSourceExists() {
  try {
    const stats = await fsp.stat(srcDir);
    if (!stats.isDirectory()) {
      throw new Error('ui/src must be a directory');
    }
  } catch (error) {
    throw new Error(`Source directory missing: ${srcDir}. ${error.message}`);
  }
}

async function copyRecursive(source, destination) {
  await fsp.mkdir(destination, { recursive: true });
  const entries = await fsp.readdir(source, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = path.join(source, entry.name);
    const destPath = path.join(destination, entry.name);
    if (entry.isDirectory()) {
      await copyRecursive(srcPath, destPath);
      continue;
    }
    await fsp.copyFile(srcPath, destPath);
  }
}

async function cleanDist() {
  await fsp.rm(distDir, { recursive: true, force: true });
}

async function build() {
  await ensureSourceExists();
  await cleanDist();
  await copyRecursive(srcDir, distDir);
  console.log(`[fall-foliage-explorer-ui] build complete -> ${distDir}`);
}

build().catch((error) => {
  console.error('[fall-foliage-explorer-ui] build failed:', error);
  process.exitCode = 1;
});
