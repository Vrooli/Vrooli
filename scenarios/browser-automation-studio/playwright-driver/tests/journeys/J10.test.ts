import { existsSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';

const driverRoot = join(__dirname, '../..');
const ensureScript = join(driverRoot, 'scripts/ensure-browsers.mjs');
const driverRequire = createRequire(join(driverRoot, 'package.json'));

describe('[REQ:BAS-RH-J10] clean-machine browser readiness', () => {
  it('uses the declared Playwright inventory when PATH and the browser cache are empty', () => {
    const sandbox = mkdtempSync(join(tmpdir(), 'bas-j10-'));
    const cache = join(sandbox, 'browser-cache');
    const preload = join(sandbox, 'mock-installer.cjs');
    const wrapperPackage = driverRequire.resolve('rebrowser-playwright/package.json');
    const corePackage = createRequire(wrapperPackage).resolve('playwright-core/package.json');
    const coreCLI = join(dirname(corePackage), 'cli.js');

    writeFileSync(preload, `
      const fs = require('node:fs');
      const childProcess = require('node:child_process');
      childProcess.spawnSync = (command, args) => {
        if (command !== process.execPath || args[0] !== ${JSON.stringify(coreCLI)} || args[1] !== 'install' || args[2] !== 'chromium') {
          return { status: 91 };
        }
        const installStatus = Number(process.env.BAS_J10_INSTALL_STATUS || 0);
        if (installStatus !== 0) return { status: installStatus };
        const path = require('node:path');
        const inventory = JSON.parse(fs.readFileSync(path.join(path.dirname(args[0]), 'browsers.json'), 'utf8'));
        const revision = inventory.browsers.find((browser) => browser.name === 'chromium').revision;
        fs.mkdirSync(path.join(process.env.PLAYWRIGHT_BROWSERS_PATH, 'chromium-' + revision), { recursive: true });
        return { status: 0 };
      };
      require('node:module').syncBuiltinESMExports();
    `);

    const runEnsure = (cachePath: string, installStatus: number) => spawnSync(process.execPath, [ensureScript], {
        cwd: driverRoot,
        encoding: 'utf8',
        env: {
          ...process.env,
          PATH: '',
          HOME: join(sandbox, 'home'),
          PLAYWRIGHT_BROWSERS_PATH: cachePath,
          BAS_J10_INSTALL_STATUS: String(installStatus),
          NODE_OPTIONS: `--require=${preload}`,
        },
        timeout: 30_000,
      });

    try {
      // The child runs the real ensure script against the package's declared
      // core, while this test double prevents any browser download or install.
      const result = runEnsure(cache, 0);
      expect(result.error).toBeUndefined();
      if (result.status !== 0) throw new Error(`${result.error ?? ''}\n${result.stdout}\n${result.stderr}`);
      expect(result.stdout).toContain('chromium build');
      expect(result.stdout).toContain('installed');
      expect(result.stderr).toBe('');
      const inventory = JSON.parse(require('node:fs').readFileSync(join(dirname(corePackage), 'browsers.json'), 'utf8'));
      const revision = inventory.browsers.find((browser: { name: string }) => browser.name === 'chromium').revision;
      expect(existsSync(join(cache, `chromium-${revision}`))).toBe(true);

      const missingCache = join(sandbox, 'missing-browser-cache');
      const failed = runEnsure(missingCache, 37);
      expect(failed.error).toBeUndefined();
      expect(failed.status).toBe(37);
      expect(failed.stderr).toContain('ensure-browsers: install failed with status 37');
      expect(existsSync(join(missingCache, `chromium-${revision}`))).toBe(false);
    } finally {
      rmSync(sandbox, { recursive: true, force: true });
    }
  });
});
