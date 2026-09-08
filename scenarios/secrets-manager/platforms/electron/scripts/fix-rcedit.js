#!/usr/bin/env node

/**
 * Post-build script to fix rcedit issues with Wine
 *
 * This script wraps rcedit calls to handle empty OriginalFilename values
 * that cause issues with older rcedit versions under Wine.
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

const CACHE_DIR = path.join(os.homedir(), '.cache', 'electron-builder', 'winCodeSign');

async function patchRcedit() {
  if (!fs.existsSync(CACHE_DIR)) {
    console.log('ℹ️  rcedit cache directory not found; skipping patch until electron-builder populates it');
    return;
  }

  const winCodeSignDirs = fs.readdirSync(CACHE_DIR, { withFileTypes: true })
    .filter(dirent => dirent.isDirectory() && dirent.name.startsWith('winCodeSign-'))
    .map(dirent => path.join(CACHE_DIR, dirent.name));

  if (winCodeSignDirs.length === 0) {
    console.log('ℹ️  No winCodeSign cache directories detected; nothing to patch yet');
    return;
  }

  for (const dir of winCodeSignDirs) {
    const ia32Path = path.join(dir, 'rcedit-ia32.exe');
    const x64Path = path.join(dir, 'rcedit-x64.exe');
    const ia32BackupPath = path.join(dir, 'rcedit-ia32.exe.original');

    // Check if already patched
    if (fs.existsSync(ia32BackupPath)) {
      console.log(`✓ rcedit already patched in ${dir}`);
      continue;
    }

    // Backup original ia32 version
    if (fs.existsSync(ia32Path)) {
      fs.copyFileSync(ia32Path, ia32BackupPath);
    }

    // Create symlink from ia32 to x64 (x64 version works better with Wine)
    if (fs.existsSync(x64Path)) {
      if (fs.existsSync(ia32Path)) {
        fs.unlinkSync(ia32Path);
      }
      fs.symlinkSync(path.basename(x64Path), ia32Path);
      console.log(`✓ Patched rcedit in ${dir} (ia32 -> x64 symlink)`);
    }
  }
}

if (require.main === module) {
  patchRcedit().catch(err => {
    console.error('Failed to patch rcedit:', err);
    process.exit(1);
  });
}

module.exports = { patchRcedit };
