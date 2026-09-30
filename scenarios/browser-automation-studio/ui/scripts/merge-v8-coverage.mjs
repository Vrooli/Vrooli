import { readFile, readdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

const rawRoot = path.resolve('coverage/raw');
const outputRoot = path.resolve('coverage');
const floor = 85;
const reports = [];
for (const entry of await readdir(rawRoot, { withFileTypes: true })) {
  if (entry.isDirectory()) reports.push(JSON.parse(await readFile(path.join(rawRoot, entry.name, 'coverage-final.json'), 'utf8')));
}
if (reports.length === 0) throw new Error('No raw V8 coverage reports were produced.');

const merged = {};
for (const report of reports) for (const [file, coverage] of Object.entries(report)) {
  if (!merged[file]) { merged[file] = structuredClone(coverage); continue; }
  const destination = merged[file];
  for (const key of ['s', 'f']) for (const [id, count] of Object.entries(coverage[key])) destination[key][id] = (destination[key][id] ?? 0) + count;
  for (const [id, counts] of Object.entries(coverage.b)) destination.b[id] = (destination.b[id] ?? counts.map(() => 0)).map((count, index) => count + counts[index]);
}

const metric = (total, covered) => ({ total, covered, skipped: 0, pct: total === 0 ? 100 : Number(((covered / total) * 100).toFixed(2)) });
const summary = { total: {} };
const totals = { statements: [0, 0], functions: [0, 0], branches: [0, 0], lines: [0, 0] };
for (const [file, coverage] of Object.entries(merged)) {
  const lineCounts = new Map();
  for (const [id, count] of Object.entries(coverage.s)) { const line = coverage.statementMap[id].start.line; lineCounts.set(line, (lineCounts.get(line) ?? 0) + count); }
  const values = { statements: Object.values(coverage.s), functions: Object.values(coverage.f), branches: Object.values(coverage.b).flat(), lines: [...lineCounts.values()] };
  summary[file] = {};
  for (const [name, counts] of Object.entries(values)) { const value = metric(counts.length, counts.filter((count) => count > 0).length); summary[file][name] = value; totals[name][0] += value.total; totals[name][1] += value.covered; }
}
for (const [name, [total, covered]] of Object.entries(totals)) summary.total[name] = metric(total, covered);
await writeFile(path.join(outputRoot, 'coverage-final.json'), `${JSON.stringify(merged)}\n`);
await writeFile(path.join(outputRoot, 'coverage-summary.json'), `${JSON.stringify(summary, null, 2)}\n`);
const failed = Object.entries(summary.total).filter(([, value]) => value.pct < floor);
if (failed.length) throw new Error(`Merged UI coverage is below the ${floor}% floor: ${failed.map(([name, value]) => `${name} ${value.pct}%`).join(', ')}.`);
