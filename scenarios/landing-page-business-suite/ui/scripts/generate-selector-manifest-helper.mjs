import { execSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

process.chdir(join(fileURLToPath(new URL('..', import.meta.url))));

const tempDir = mkdtempSync(join(tmpdir(), 'landing-page-business-suite-selectors-'));

try {
  execSync(
    `pnpm exec tsc src/shared/consts/selectors.ts --module commonjs --moduleResolution node --target es2022 --skipLibCheck --outDir ${tempDir}`,
    { stdio: 'pipe' },
  );

  const selectorsModule = await import(
    pathToFileURL(join(tempDir, 'selectors.js')).href,
  );
  const manifestPath = 'src/shared/consts/selectors.manifest.json';
  writeFileSync(manifestPath, `${JSON.stringify(selectorsModule.selectorsManifest, null, 2)}\n`, 'utf8');
  process.stdout.write(`Generated ${manifestPath}\n`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
