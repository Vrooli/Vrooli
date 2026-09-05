import { execSync } from 'node:child_process';
import { mkdtempSync, renameSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
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

  // The unit and performance providers may build/test the same UI in
  // parallel. Publish the complete manifest in one rename so neither
  // provider can observe a truncated JSON file during generation.
  const outputDir = mkdtempSync(join(dirname(manifestPath), '.selectors-manifest-'));
  const tempPath = join(outputDir, 'selectors.manifest.json');
  try {
    writeFileSync(tempPath, manifestJson, 'utf8');
    renameSync(tempPath, manifestPath);
  } finally {
    rmSync(outputDir, { recursive: true, force: true });
  }
  process.stdout.write(`Generated ${manifestPath}\n`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
