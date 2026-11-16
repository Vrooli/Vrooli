import { copyFileSync, mkdirSync, readdirSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { build } from 'esbuild';

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const outputIndex = process.argv.indexOf('--output-dir');
const modeIndex = process.argv.indexOf('--mode');
const profileBuild = process.argv.includes('--profile') || (modeIndex >= 0 && process.argv[modeIndex + 1] === 'profile');
const outputDir = resolve(
  packageRoot,
  outputIndex >= 0 ? process.argv[outputIndex + 1] : 'dist'
);
if (outputIndex >= 0 && (!process.argv[outputIndex + 1] || process.argv[outputIndex + 1].startsWith('--'))) {
  throw new Error('--output-dir requires a directory path');
}

mkdirSync(outputDir, { recursive: true });
await build({
  entryPoints: [join(packageRoot, 'src/server.ts')],
  bundle: true,
  platform: 'node',
  target: 'node18',
  outfile: join(outputDir, 'server.js'),
  sourcemap: profileBuild ? 'linked' : false,
  keepNames: profileBuild,
  external: [
    'rebrowser-playwright',
    'winston',
    'prom-client',
    'uuid',
    'zod',
    '@bufbuild/protobuf',
    '@bufbuild/protobuf/*',
    'ws',
  ],
});

const browserScripts = join(packageRoot, 'src/recording/capture/browser-scripts');
const stagedScripts = join(outputDir, 'browser-scripts');
mkdirSync(stagedScripts, { recursive: true });
for (const entry of readdirSync(browserScripts).filter((name) => name.endsWith('.js'))) {
  copyFileSync(join(browserScripts, entry), join(stagedScripts, entry));
}

// Compilation produces JavaScript and browser helper scripts only. The browser
// closure is a separately versioned runtime artifact acquired by its governed
// owner and verified when the bundle is staged/launched. Never download it as
// a side effect of compiling the driver.
