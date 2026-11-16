import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';

const driverRoot = join(__dirname, '../..');
const ensureScript = join(driverRoot, 'scripts/ensure-browsers.mjs');
const driverRequire = createRequire(join(driverRoot, 'package.json'));

describe('versioned browser closure verification', () => {
  it('checks the package inventory without downloading or modifying host cache', () => {
    const sandbox = mkdtempSync(join(tmpdir(), 'bas-browser-closure-'));
    const cache = join(sandbox, 'browser-cache');
    const wrapperPackage = driverRequire.resolve('rebrowser-playwright/package.json');
    const corePackage = createRequire(wrapperPackage).resolve('playwright-core/package.json');
    const inventory = JSON.parse(require('node:fs').readFileSync(join(dirname(corePackage), 'browsers.json'), 'utf8'));
    const revision = inventory.browsers.find((browser: { name: string }) => browser.name === 'chromium').revision;
    const executable = spawnSync(process.execPath, ['-e', "process.stdout.write(require('rebrowser-playwright').chromium.executablePath())"], {
      cwd: driverRoot,
      encoding: 'utf8',
      env: { ...process.env, PLAYWRIGHT_BROWSERS_PATH: cache },
    }).stdout;
    const expected = executable.trim();
    const runEnsure = (cachePath: string, extraEnv: NodeJS.ProcessEnv = {}) => spawnSync(process.execPath, [ensureScript], {
      cwd: driverRoot,
      encoding: 'utf8',
      env: {
        ...process.env,
        PATH: '',
        HOME: join(sandbox, 'home'),
        PLAYWRIGHT_BROWSERS_PATH: cachePath,
        ...extraEnv,
      },
      timeout: 30_000,
    });

    try {
      mkdirSync(dirname(expected), { recursive: true });
      writeFileSync(expected, 'fixture executable');
      const ready = runEnsure(cache);
      expect(ready.error).toBeUndefined();
      expect(ready.status).toBe(0);
      expect(ready.stdout).toContain(`chromium build ${revision} executable present`);
      expect(ready.stderr).toBe('');
      expect(existsSync(expected)).toBe(true);

      const missingCache = join(sandbox, 'missing-browser-cache');
      const missing = runEnsure(missingCache);
      expect(missing.error).toBeUndefined();
      expect(missing.status).toBe(1);
      expect(missing.stderr).toContain(`chromium build ${revision} missing`);
      expect(missing.stderr).toContain('acquisition is not performed here');
      expect(existsSync(missingCache)).toBe(false);

      const forced = runEnsure(cache, { PLAYWRIGHT_ENSURE_BROWSERS: 'force' });
      expect(forced.status).toBe(1);
      expect(forced.stderr).toContain('acquisition is not performed here');
      expect(existsSync(expected)).toBe(true);
    } finally {
      rmSync(sandbox, { recursive: true, force: true });
    }
  });
});
