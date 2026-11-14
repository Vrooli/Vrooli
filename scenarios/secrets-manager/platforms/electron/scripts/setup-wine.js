#!/usr/bin/env node

/**
 * Auto-install Wine AppImage for cross-platform Windows builds
 *
 * Wine is required for building Windows executables on Linux.
 * This script downloads and sets up Wine AppImage if:
 * 1. Running on Linux
 * 2. Wine is not already available
 * 3. User is building for Windows target
 *
 * Wine AppImage requires NO sudo - fully portable installation.
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');
const os = require('os');

const WINE_DIR = path.join(os.homedir(), '.local', 'share', 'wine-appimage');
const WINE_BIN_DIR = path.join(os.homedir(), '.local', 'bin');
const WINE_APPIMAGE_PATH = path.join(WINE_DIR, 'wine.AppImage');
const WINE_WRAPPER_PATH = path.join(WINE_BIN_DIR, 'wine');

// Resolve the current stable asset instead of pinning a versioned filename.
const WINE_RELEASE_API_URL = 'https://api.github.com/repos/mmtrt/WINE_AppImage/releases/tags/continuous-stable';

function wineEnvironment() {
  return { ...process.env, PATH: `${WINE_BIN_DIR}${path.delimiter}${process.env.PATH || ''}` };
}

function latestWineAppImageUrl() {
  return new Promise((resolve, reject) => {
    const request = https.get(WINE_RELEASE_API_URL, { headers: { 'User-Agent': 'vrooli-scenario-to-desktop' } }, (response) => {
      let body = '';
      response.setEncoding('utf8');
      response.on('data', (chunk) => { body += chunk; });
      response.on('end', () => {
        if (response.statusCode !== 200) {
          reject(new Error(`Failed to resolve Wine release: HTTP ${response.statusCode}`));
          return;
        }
        try {
          const assets = JSON.parse(body).assets || [];
          const asset = assets.find((candidate) => typeof candidate.browser_download_url === 'string' && /^wine-stable_.*-x86_64\.AppImage$/.test(candidate.name || ''));
          if (!asset) throw new Error('stable Wine AppImage asset not found');
          resolve(asset.browser_download_url);
        } catch (error) {
          reject(error);
        }
      });
    });
    request.on('error', reject);
  });
}

async function checkWineInstalled() {
  try {
    execSync('wine --version', { stdio: 'pipe', env: wineEnvironment() });
    return true;
  } catch {
    return false;
  }
}

async function downloadFile(url, destPath) {
  return new Promise((resolve, reject) => {
    console.log(`📥 Downloading Wine AppImage...`);
    console.log(`   URL: ${url}`);
    console.log(`   Destination: ${destPath}`);

    https.get(url, (response) => {
      // Follow redirects
      if (response.statusCode === 301 || response.statusCode === 302) {
        const redirectUrl = response.headers.location;
        console.log(`   Following redirect to: ${redirectUrl}`);
        response.resume();
        downloadFile(redirectUrl, destPath).then(resolve).catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        response.resume();
        fs.unlink(destPath, () => {});
        reject(new Error(`Failed to download: HTTP ${response.statusCode}`));
        return;
      }
      const file = fs.createWriteStream(destPath);
      const totalBytes = parseInt(response.headers['content-length'] || '0', 10);
      let downloadedBytes = 0;
      let lastPercent = 0;

      response.on('data', (chunk) => {
        downloadedBytes += chunk.length;
        const percent = Math.floor((downloadedBytes / totalBytes) * 100);

        if (percent >= lastPercent + 10) {
          console.log(`   Progress: ${percent}% (${Math.floor(downloadedBytes / 1024 / 1024)}MB / ${Math.floor(totalBytes / 1024 / 1024)}MB)`);
          lastPercent = percent;
        }
      });

      response.pipe(file);

      file.on('finish', () => {
        file.close();
        console.log(`✓ Download complete`);
        resolve();
      });
    }).on('error', (err) => {
      fs.unlink(destPath, () => {});
      reject(err);
    });
  });
}

async function setupWineAppImage() {
  console.log('\n🍷 Wine Auto-Installer for Windows Builds\n');

  // Only run on Linux
  if (os.platform() !== 'linux') {
    console.log('ℹ️  Not on Linux - Wine not required');
    return;
  }

  // Check if Wine is already available
  if (await checkWineInstalled()) {
    console.log('✓ Wine is already installed');
    return;
  }

  console.log('⚠️  Wine not found - required for Windows builds');
  console.log('📦 Installing Wine AppImage (no sudo required)...\n');

  try {
    // Create directories
    fs.mkdirSync(WINE_DIR, { recursive: true });
    fs.mkdirSync(WINE_BIN_DIR, { recursive: true });

    // Download the current Wine AppImage
    const wineUrl = await latestWineAppImageUrl();
    await downloadFile(wineUrl, WINE_APPIMAGE_PATH);

    const downloaded = fs.statSync(WINE_APPIMAGE_PATH);
    if (!downloaded.isFile() || downloaded.size < 1024 * 1024) {
      throw new Error(`Wine AppImage is unexpectedly small (${downloaded.size} bytes)`);
    }

    // Make executable
    fs.chmodSync(WINE_APPIMAGE_PATH, 0o755);
    console.log('✓ Made Wine AppImage executable');

    // Create wrapper script
    const wrapperScript = `#!/bin/bash
# Wine AppImage wrapper for electron-builder
# Auto-generated by scenario-to-desktop
exec "${WINE_APPIMAGE_PATH}" "$@"
`;

    fs.writeFileSync(WINE_WRAPPER_PATH, wrapperScript);
    fs.chmodSync(WINE_WRAPPER_PATH, 0o755);

    // npm scripts prepend node_modules/.bin to PATH, so expose the same
    // validated wrapper to electron-builder in this generated package.
    const localBin = path.join(process.cwd(), 'node_modules', '.bin');
    fs.mkdirSync(localBin, { recursive: true });
    const localWrapper = path.join(localBin, 'wine');
    fs.writeFileSync(localWrapper, `#!/bin/sh\nexec "${WINE_WRAPPER_PATH}" "$@"\n`);
    fs.chmodSync(localWrapper, 0o755);
    console.log('✓ Created Wine wrapper script');

    // Verify installation
    try {
      const version = execSync(`"${WINE_WRAPPER_PATH}" --version`, {
        encoding: 'utf8',
        stdio: 'pipe'
      }).trim();
      console.log(`\n✅ Wine installed successfully!`);
      console.log(`   Version: ${version}`);
      console.log(`   Location: ${WINE_APPIMAGE_PATH}`);
      console.log(`   Wrapper: ${WINE_WRAPPER_PATH}`);
      console.log(`\n💡 Make sure ~/.local/bin is in your PATH:`);
      console.log(`   export PATH="$HOME/.local/bin:$PATH"`);
      console.log(`\n🚀 You can now build Windows executables with: npm run dist:win`);
    } catch (err) {
      console.error('⚠️  Wine installed but verification failed:', err.message);
      console.log('   Try running manually:', WINE_WRAPPER_PATH, '--version');
    }

  } catch (error) {
    console.error('❌ Failed to install Wine AppImage:', error.message);
    console.log('\n📖 Manual installation:');
    console.log('   1. Download: https://github.com/mmtrt/WINE_AppImage/releases');
    console.log('   2. chmod +x wine-*.AppImage');
    console.log('   3. mv wine-*.AppImage ~/.local/share/wine-appimage/wine.AppImage');
    console.log('   4. Create wrapper script at ~/.local/bin/wine');
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  setupWineAppImage().catch(err => {
    console.error('Fatal error:', err);
    process.exit(1);
  });
}

module.exports = { setupWineAppImage };
