import { existsSync } from 'node:fs';
import { readFile, writeFile } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';
import { join } from 'node:path';

const repoRoot = process.argv[2] ?? '../../../';
const ledgerPath = join(repoRoot, 'scenarios/react-component-library/library/released-version-hashes.json');
const originalLedgerPath = 'scenarios/react-component-library/library/released-version-hashes.json';
const current = JSON.parse(await readFile(ledgerPath, 'utf8'));
const prior = spawnSync('git', ['show', `HEAD:${originalLedgerPath}`], { cwd: repoRoot, encoding: 'utf8' });
if (prior.status !== 0) throw new Error(`cannot read prior ledger: ${prior.stderr}`);
const previous = JSON.parse(prior.stdout);
const known = new Set(current.entries.map((entry) => entry.path));
let restored = 0;
for (const entry of previous.entries) {
  const name = entry.path.split('/').at(-1) ?? '';
  if (name !== 'story.json' || known.has(entry.path)) continue;
  const path = join(repoRoot, 'scenarios/react-component-library/library', entry.path);
  if (!existsSync(path)) continue;
  current.entries.push(entry);
  known.add(entry.path);
  restored += 1;
}
current.entries.sort((left, right) => left.path.localeCompare(right.path));
await writeFile(ledgerPath, `${JSON.stringify(current, null, 2)}\n`);
console.log(JSON.stringify({ restored, remaining: current.entries.length }));
