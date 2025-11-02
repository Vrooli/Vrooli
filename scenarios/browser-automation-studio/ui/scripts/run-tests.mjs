import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { mkdirSync, existsSync } from 'node:fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = dirname(__dirname);
const testsDir = join(projectRoot, 'tests');

if (!existsSync(testsDir)) {
  console.log('ℹ️  No UI test directory found, skipping');
  process.exit(0);
}

const filteredArgs = process.argv
  .slice(2)
  .filter(
    (arg) =>
      ![
        '--',
        '--coverage',
        '--silent',
        '--watch',
        '--watchAll',
      ].includes(arg),
  );

const coverageDir = join(projectRoot, 'coverage', 'node');
mkdirSync(coverageDir, { recursive: true });

const nodeArgs = [
  '--test',
  '--experimental-test-coverage',
  '--test-reporter=spec',
  testsDir,
  ...filteredArgs,
];

const result = spawnSync(process.execPath, nodeArgs, {
  stdio: 'inherit',
  env: {
    ...process.env,
    NODE_V8_COVERAGE: coverageDir,
  },
});

if (result.error) {
  throw result.error;
}

process.exit(result.status ?? 0);
