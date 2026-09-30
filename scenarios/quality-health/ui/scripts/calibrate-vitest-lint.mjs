// Native observations only: expected outcomes remain in the independent corpus.
import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { observeVitestSyntax } from '../eslint-rules/vitest-profile.mjs';
const [root, output] = process.argv.slice(2);
if (!root || !output) throw new Error('usage: node calibrate-vitest-lint.mjs <corpus-root> <output.json>');
const cases = JSON.parse(readFileSync(join(root, 'development.json'), 'utf8'));
const observations = cases.filter(c => c.input.profile === 'react-vitest-v2').map(({ input }) => ({
  caseID: input.id,
  files: input.files.map(file => observeVitestSyntax(readFileSync(join(root, input.root, file), 'utf8'), file.replace(/\.txt$/, ''))),
}));
writeFileSync(output, JSON.stringify(observations, null, 2));
console.log(JSON.stringify({ cases: observations.length, unknown: observations.filter(c=>c.files.some(f=>f.status==='unknown')).length }));
