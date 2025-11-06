#!/usr/bin/env node
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { promises as fs } from 'node:fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const uiDir = __dirname;
const distDir = path.join(uiDir, 'dist');

const EXCLUDES = new Set([
  'dist',
  'node_modules',
  'package-lock.json',
  'package.json',
  'build.mjs'
]);

async function ensureDir(dir) {
  await fs.mkdir(dir, { recursive: true });
}

async function copyRecursive(src, dest) {
  const stats = await fs.stat(src);
  if (stats.isDirectory()) {
    await ensureDir(dest);
    const entries = await fs.readdir(src);
    for (const entry of entries) {
      const entrySrc = path.join(src, entry);
      const entryDest = path.join(dest, entry);
      await copyRecursive(entrySrc, entryDest);
    }
    return;
  }
  await ensureDir(path.dirname(dest));
  await fs.copyFile(src, dest);
}

async function cleanDist() {
  await fs.rm(distDir, { recursive: true, force: true });
  await ensureDir(distDir);
}

async function build() {
  console.log('\u001b[34m[device-sync-hub-ui]\u001b[0m Building static assets...');
  await cleanDist();
  const entries = await fs.readdir(uiDir);
  for (const entry of entries) {
    if (EXCLUDES.has(entry)) {
      continue;
    }
    const src = path.join(uiDir, entry);
    const dest = path.join(distDir, entry);
    await copyRecursive(src, dest);
  }
  console.log(`\u001b[32m[device-sync-hub-ui]\u001b[0m Build complete -> ${distDir}`);
}

build().catch((err) => {
  console.error('\u001b[31m[device-sync-hub-ui]\u001b[0m Build failed:', err);
  process.exitCode = 1;
});
