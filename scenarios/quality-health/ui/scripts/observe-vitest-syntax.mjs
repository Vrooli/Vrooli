import { readFileSync } from 'node:fs';
import { observeVitestSyntax } from '../eslint-rules/vitest-profile.mjs';

// Input source is already bounded and scoped by the Quality Health service.
// This process performs one native lint observation per requested file.
const input = JSON.parse(readFileSync(0, 'utf8'));
if (!Array.isArray(input) || input.length > 100) throw new Error('invalid observation batch');
const output = input.map(file => ({
  ...observeVitestSyntax(file.source, file.file), sourceDigest: file.sourceDigest,
}));
process.stdout.write(JSON.stringify(output));
