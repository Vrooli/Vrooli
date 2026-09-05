import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const tempDir = mkdtempSync(join(tmpdir(), 'web-search-selectors-'));
try {
  execFileSync('pnpm', ['exec', 'tsc', 'src/consts/selectors.ts', '--rootDir', 'src', '--module', 'commonjs', '--moduleResolution', 'node', '--target', 'es2022', '--skipLibCheck', '--outDir', tempDir], { stdio: 'pipe' });
  const { selectorsManifest } = await import(pathToFileURL(join(tempDir, 'consts', 'selectors.js')).href);
  writeFileSync('src/consts/selectors.manifest.json', `${JSON.stringify(selectorsManifest, null, 2)}\n`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
