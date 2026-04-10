import { execSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const tempDir = mkdtempSync(join(tmpdir(), 'web-console-selectors-'));

try {
  execSync(
    `pnpm exec tsc src/consts/selectors.ts --module nodenext --moduleResolution nodenext --target es2022 --outDir ${tempDir}`,
    { stdio: 'pipe' },
  );

  const selectorsModule = await import(pathToFileURL(join(tempDir, 'selectors.js')).href);
  const manifestPath = 'src/consts/selectors.manifest.json';
  const manifestJson = `${JSON.stringify(selectorsModule.selectorsManifest, null, 2)}\n`;

  writeFileSync(manifestPath, manifestJson, 'utf8');
  process.stdout.write(`Generated ${manifestPath}\n`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
