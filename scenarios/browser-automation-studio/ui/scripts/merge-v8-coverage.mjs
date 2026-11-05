import { readFile, readdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { fileFingerprint, sourceFingerprint } from './coverage-inputs.mjs';

// The library reconciles counters by source location, as Vitest itself does.
// Numeric IDs are local to each transformed map and cannot be added directly.
export async function mergeCoverageRun(runRoot, uiRoot) {
  const manifest = JSON.parse(await readFile(path.join(runRoot, 'manifest.json'), 'utf8'));
  if (manifest.version !== 1 || !Array.isArray(manifest.projects) || manifest.projects.length === 0 ||
      new Set(manifest.projects).size !== manifest.projects.length ||
      manifest.projects.some(name => typeof name !== 'string' || !/^[a-z][a-z0-9-]*$/.test(name))) {
    throw new Error('Incomplete coverage: invalid expected-project manifest.');
  }
  if (manifest.sourceFingerprint !== sourceFingerprint(uiRoot)) throw new Error('Incomplete coverage: source inputs changed during the run.');
  const rawRoot = path.join(runRoot, 'raw');
  const actual = (await readdir(rawRoot, { withFileTypes: true })).filter(entry => entry.isDirectory()).map(entry => entry.name).sort();
  if (JSON.stringify(actual) !== JSON.stringify([...manifest.projects].sort())) throw new Error('Incomplete coverage: missing or unexpected project shards.');
  const { default: libCoverage } = await import('istanbul-lib-coverage');
  const merged = libCoverage.createCoverageMap({});
  for (const name of manifest.projects) {
    const reportPath = path.join(rawRoot, name, 'coverage-final.json');
    const receipt = manifest.completed?.[name];
    if (!receipt || receipt.sourceFingerprint !== manifest.sourceFingerprint || receipt.reportFingerprint !== fileFingerprint(reportPath)) {
      throw new Error(`Incomplete coverage: missing or stale receipt for ${name}.`);
    }
    merged.merge(JSON.parse(await readFile(reportPath, 'utf8')));
  }
  if (!manifest.projects.includes(manifest.sourceClosureProject) || !manifest.completed[manifest.sourceClosureProject]?.sourceClosure) {
    throw new Error('Incomplete coverage: no complete-source instrumentation shard.');
  }
  if (merged.files().length === 0) throw new Error('Incomplete coverage: no executable source coverage was produced.');
  const total = libCoverage.createCoverageSummary();
  const summary = {};
  for (const file of merged.files().sort()) {
    const fileSummary = merged.fileCoverageFor(file).toSummary();
    summary[file] = fileSummary.toJSON();
    total.merge(fileSummary);
  }
  summary.total = total.toJSON();
  await writeFile(path.join(runRoot, 'coverage-final.json'), `${JSON.stringify(merged.toJSON())}\n`);
  await writeFile(path.join(runRoot, 'coverage-summary.json'), `${JSON.stringify(summary, null, 2)}\n`);
  // One explicit scenario-wide contract, unchanged from vite.config.ts.
  const { coverage: { thresholds } } = JSON.parse(await readFile(new URL('../vitest.projects.json', import.meta.url), 'utf8'));
  const floor = thresholds.lines;
  const failed = Object.entries(summary.total).filter(([name, value]) => value.pct < thresholds[name]);
  return { status: failed.length ? 'below-floor' : 'passed', floor, failed: failed.map(([name, value]) => `${name} ${value.pct}%`), summary: summary.total };
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try {
    const result = await mergeCoverageRun(path.resolve(process.argv[2] ?? 'coverage'), path.resolve(process.argv[3] ?? '.'));
    if (result.status === 'below-floor') {
      console.error(`Merged UI coverage is below the ${result.floor}% floor: ${result.failed.join(', ')}.`);
      process.exitCode = 1;
    }
  } catch (error) {
    console.error(`UI coverage merge error: ${error.message}`);
    process.exitCode = 2;
  }
}
