#!/usr/bin/env node

/**
 * Post-build script for Windows to manually set icon
 *
 * This runs after electron-builder finishes packaging but before creating the installer.
 * It manually embeds the icon into the exe since we disable signAndEditExecutable
 * to work around rcedit empty string issues.
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

const RCEDIT_CACHE = path.join(os.homedir(), '.cache', 'electron-builder', 'winCodeSign');

function resolveRceditBinary() {
  if (!fs.existsSync(RCEDIT_CACHE)) {
    return null;
  }

  const entries = fs.readdirSync(RCEDIT_CACHE, { withFileTypes: true });
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    const candidate = path.join(RCEDIT_CACHE, entry.name, 'rcedit-x64.exe');
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  return null;
}

// electron-builder afterPack hook
exports.default = async function(context) {
  // Only run for Windows
  if (context.electronPlatformName !== 'win32') {
    return;
  }

  const productFilename = context.packager?.appInfo?.productFilename || context.packager?.appInfo?.productName || 'app';
  const exePath = path.join(context.appOutDir, `${productFilename}.exe`);

  // electron-builder creates the .ico file in the dist directory
  const icoPath = path.join(context.outDir, '.icon-ico', 'icon.ico');

  if (!fs.existsSync(exePath)) {
    console.log('⚠️  Exe not found, skipping icon embedding');
    return;
  }

  if (!fs.existsSync(icoPath)) {
    console.log('⚠️  Icon file not found, skipping icon embedding');
    return;
  }

  const rceditBinary = resolveRceditBinary();
  if (!rceditBinary) {
    console.log('⚠️  rcedit not found, skipping icon embedding');
    return;
  }

  try {
    console.log('🎨 Embedding icon into Windows executable...');
    execSync(`wine "${rceditBinary}" "${exePath}" --set-icon "${icoPath}"`, {
      stdio: 'pipe',
      encoding: 'utf8'
    });
    console.log('✓ Icon embedded successfully');
  } catch (error) {
    console.error('❌ Failed to embed icon:', error.message);
    // Don't fail the build, just warn
  }
};
