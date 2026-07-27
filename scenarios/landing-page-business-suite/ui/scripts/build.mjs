import { spawnSync } from 'node:child_process';

const buildMode = process.env.VROOLI_BUILD_MODE;
if (!buildMode) {
  run('vite', ['build']);
} else if (buildMode !== 'profile') {
  throw new Error(`Unsupported VROOLI_BUILD_MODE: ${buildMode}`);
} else {
  run('vite', ['build', '--mode', 'profile']);
}

function run(command, args) {
  const result = spawnSync(command, args, { stdio: 'inherit' });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}
