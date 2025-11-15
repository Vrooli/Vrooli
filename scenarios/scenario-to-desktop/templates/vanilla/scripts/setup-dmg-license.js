#!/usr/bin/env node

/**
 * Setup dmg-license stub for cross-platform builds
 *
 * dmg-license is macOS-only but dmg-builder requires it even on Linux.
 * This script creates a stub module on non-Mac platforms to allow builds.
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

// Only run on non-macOS platforms
if (os.platform() === 'darwin') {
  console.log('Running on macOS - dmg-license should install normally');
  process.exit(0);
}

const dmgLicensePath = path.join(__dirname, '..', 'node_modules', 'dmg-license');

// Check if dmg-license already exists
if (fs.existsSync(dmgLicensePath)) {
  console.log('dmg-license already installed');
  process.exit(0);
}

console.log('Creating dmg-license stub for cross-platform builds...');

// Create directory
fs.mkdirSync(dmgLicensePath, { recursive: true });

// Create package.json
const packageJson = {
  name: 'dmg-license',
  version: '1.0.11',
  description: 'Stub for cross-platform electron-builder support',
  main: 'index.js'
};

fs.writeFileSync(
  path.join(dmgLicensePath, 'package.json'),
  JSON.stringify(packageJson, null, 2)
);

// Create stub index.js
fs.writeFileSync(
  path.join(dmgLicensePath, 'index.js'),
  '// Stub module for cross-platform electron-builder support\nmodule.exports = {};\n'
);

console.log('✓ dmg-license stub created successfully');
