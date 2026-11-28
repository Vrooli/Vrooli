import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';

const driverRoot = join(__dirname, '../..');
const ensureScript = join(driverRoot, 'scripts/ensure-browsers.mjs');
const driverRequire = createRequire(join(driverRoot, 'package.json'));

describe('versioned browser closure verification', () => {
  it('checks the package inventory and staged runtime identity without downloading', async () => {
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
    const expectedRelative = require('node:path').relative(cache, expected);
    const runEnsure = (cachePath: string, extraEnv: NodeJS.ProcessEnv = {}) => spawnSync(process.execPath, [ensureScript], {
      cwd: driverRoot,
      encoding: 'utf8',
      env: {
        ...process.env,
        PATH: '',
        HOME: join(sandbox, 'home'),
        PLAYWRIGHT_BROWSERS_PATH: cachePath,
        BROWSER_EXECUTABLE_PATH: join(cachePath, expectedRelative),
        ...extraEnv,
      },
      timeout: 30_000,
    });

    try {
      mkdirSync(dirname(expected), { recursive: true });
      const executableBytes = 'fixture executable';
      writeFileSync(expected, executableBytes, { mode: 0o755 });
      const crypto = require('node:crypto');
      const executableSha = crypto.createHash('sha256').update(executableBytes).digest('hex');
      const payloadFrame = `${Buffer.byteLength(expectedRelative)}:${expectedRelative}:0755:${Buffer.byteLength(executableBytes)}:${executableSha}\n`;
      const identity = {
        schema_version: 'v1', engine: 'chromium', browser_version: String(revision),
        driver: 'rebrowser-playwright-core', driver_version: JSON.parse(require('node:fs').readFileSync(corePackage, 'utf8')).version,
        bundle_version: `br1-chromium-${revision}-rebrowser-playwright-core-${JSON.parse(require('node:fs').readFileSync(corePackage, 'utf8')).version}`,
        sha256: crypto.createHash('sha256').update(payloadFrame).digest('hex'),
      };
      const closure = {
        schema_version: 'v3', support_matrix_schema: 'v3', platform: process.platform === 'win32' ? `windows-${process.arch}` : `${process.platform}-${process.arch}`,
        target_lifecycle_status: Object.fromEntries(['runtime_launch', 'browser_task', 'web_console_pane', 'update_rollback', 'uninstall'].map((capability) => [capability, process.platform === 'linux' && process.arch === 'x64' ? 'unqualified-no-receipt' : 'unqualified-no-node'])),
        runtime_identity: identity,
        headless_shell_revision: String(revision), executable_path: expectedRelative,
        executable_sha256: executableSha,
      };
      writeFileSync(join(cache, 'browser-closure.json'), JSON.stringify(closure));
      const ready = runEnsure(cache);
      expect(ready.error).toBeUndefined();
      expect(ready.status).toBe(0);
      expect(ready.stdout).toContain(identity.bundle_version);
      expect(ready.stderr).toBe('');
      expect(existsSync(expected)).toBe(true);

      const promotedLifecycle = { ...closure, target_lifecycle_status: { ...closure.target_lifecycle_status, browser_task: 'supported' } };
      writeFileSync(join(cache, 'browser-closure.json'), JSON.stringify(promotedLifecycle));
      const lifecycleMismatch = runEnsure(cache);
      expect(lifecycleMismatch.status).toBe(1);
      expect(lifecycleMismatch.stderr).toContain('does not match the installed driver and browser inventory');
      writeFileSync(join(cache, 'browser-closure.json'), JSON.stringify(closure));

      const mismatchedDriver = {
        ...identity,
        driver_version: '1.99.0',
        bundle_version: `br1-chromium-${revision}-rebrowser-playwright-core-1.99.0`,
      };
      writeFileSync(join(cache, 'browser-closure.json'), JSON.stringify({ ...closure, runtime_identity: mismatchedDriver }));
      const mismatch = runEnsure(cache);
      expect(mismatch.status).toBe(1);
      expect(mismatch.stderr).toContain('does not match the installed driver and browser inventory');
      writeFileSync(join(cache, 'browser-closure.json'), JSON.stringify(closure));

      const receiptPath = join(cache, '.offline-install-receipt.json');
      const receipt = { schema_version: 'v3', archive_sha256: 'a'.repeat(64), runtime_identity: identity };
      writeFileSync(receiptPath, JSON.stringify(receipt));
      const receiptReady = runEnsure(cache);
      expect(receiptReady.status).toBe(0);
      const mismatchedReceiptIdentity = {
        ...identity,
        driver_version: '1.99.0',
        bundle_version: `br1-chromium-${revision}-rebrowser-playwright-core-1.99.0`,
      };
      writeFileSync(receiptPath, JSON.stringify({ ...receipt, runtime_identity: mismatchedReceiptIdentity }));
      const receiptMismatch = runEnsure(cache);
      expect(receiptMismatch.status).toBe(1);
      expect(receiptMismatch.stderr).toContain('offline install receipt mismatch');
      writeFileSync(receiptPath, JSON.stringify(receipt));

      const pathMismatch = runEnsure(cache, { BROWSER_EXECUTABLE_PATH: join(cache, 'elsewhere', 'chrome') });
      expect(pathMismatch.status).toBe(1);
      expect(pathMismatch.stderr).toContain('BROWSER_EXECUTABLE_PATH must match');

      const missingCache = join(sandbox, 'missing-browser-cache');
      const missing = runEnsure(missingCache);
      expect(missing.error).toBeUndefined();
      expect(missing.status).toBe(1);
      expect(missing.stderr).toContain('invalid versioned browser closure');
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
