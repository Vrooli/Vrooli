import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, existsSync } from 'node:fs';
import path from 'node:path';

// Bind reports to the complete source/test/config input, including unimported
// files. Output directories and node_modules are never inputs.
export function sourceFingerprint(root) {
  const files = [];
  function visit(relative) {
    const full = path.join(root, relative);
    if (!existsSync(full)) return;
    for (const entry of readdirSync(full, { withFileTypes: true })) {
      const next = path.join(relative, entry.name);
      if (entry.isDirectory()) visit(next);
      else if (entry.isFile()) files.push(next);
    }
  }
  for (const directory of ['src', 'scripts', 'vitest']) visit(directory);
  for (const file of ['vite.config.ts', 'vitest.coverage.config.ts', 'vitest.projects.json', 'tsconfig.json', 'tsconfig.node.json', 'package.json', 'pnpm-lock.yaml']) {
    if (existsSync(path.join(root, file))) files.push(file);
  }
  const hash = createHash('sha256');
  for (const relative of files.sort()) hash.update(relative).update('\0').update(readFileSync(path.join(root, relative))).update('\0');
  return hash.digest('hex');
}
export const fileFingerprint = file => createHash('sha256').update(readFileSync(file)).digest('hex');
