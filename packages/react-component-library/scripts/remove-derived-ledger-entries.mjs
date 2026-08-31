import { readFile, writeFile } from 'node:fs/promises';

const ledgerPath = process.argv[2] ?? '../../../scenarios/react-component-library/library/released-version-hashes.json';
const original = await readFile(ledgerPath, 'utf8');
const ledger = JSON.parse(original);
const before = ledger.entries.length;
ledger.entries = ledger.entries.filter((entry) => {
  const name = entry.path.split('/').at(-1) ?? '';
  return name === 'experience-contract.json' || name === 'story.json' || name.endsWith('.ts') || name.endsWith('.tsx');
});
const removed = before - ledger.entries.length;
await writeFile(ledgerPath, `${JSON.stringify(ledger, null, 2)}\n`);
console.log(JSON.stringify({ removed, remaining: ledger.entries.length }));
