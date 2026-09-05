import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const requestedOutDir = process.argv.find((arg) => arg.startsWith('--outDir='))?.slice('--outDir='.length)
  ?? (() => {
    const index = process.argv.indexOf('--outDir');
    return index >= 0 ? process.argv[index + 1] : null;
  })();
const dist = requestedOutDir ? resolve(uiRoot, requestedOutDir) : join(uiRoot, 'dist');
const assets = ['index.html', 'styles.css', 'api-resolver.js', 'bridge-init.js', 'app.js'];

rmSync(dist, { recursive: true, force: true });
mkdirSync(dist, { recursive: true });

for (const asset of assets) {
  const source = join(uiRoot, asset);
  if (!existsSync(source)) throw new Error(`Required asset is missing: ${source}`);
  cpSync(source, join(dist, asset));
}

const bridgePackage = join(uiRoot, 'node_modules', '@vrooli', 'iframe-bridge');
if (existsSync(bridgePackage)) {
  cpSync(bridgePackage, join(dist, 'node_modules', '@vrooli', 'iframe-bridge'), { recursive: true });
}

console.log(`Staged ${assets.length} asset(s) into ${dist}`);
