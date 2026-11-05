// Serial project processes avoid the full-suite heap exhaustion. Each run owns
// its outputs; focused behavior checks cannot erase prior coverage evidence.
import { spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, renameSync, writeFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { mergeCoverageRun } from './merge-v8-coverage.mjs';
import { fileFingerprint, sourceFingerprint } from './coverage-inputs.mjs';

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
export const inventory = JSON.parse(readFileSync(join(uiRoot, 'vitest.projects.json'), 'utf8'));
export function parseArgs(argv, env = process.env) {
  const suiteFlag = argv.indexOf('--suite');
  const suite = suiteFlag === -1 ? (env.BAS_VITEST_SUITE || 'smoke') : argv[suiteFlag + 1];
  if (!['smoke', 'full'].includes(suite)) throw new Error(`Unknown suite ${JSON.stringify(suite)}: expected smoke or full`);
  const forwarded = suiteFlag === -1 ? argv : [...argv.slice(0, suiteFlag), ...argv.slice(suiteFlag + 2)];
  let coverage = false;
  const remaining = [];
  // Consume enable/disable flags here. Forwarding them as well as the final
  // normalized flag makes Vitest reject duplicate scalar coverage values.
  for (let index = 0; index < forwarded.length; index++) {
    const arg = forwarded[index];
    if (arg === '--coverage' || arg === '--coverage.enabled') {
      const value = forwarded[index + 1];
      coverage = value !== 'false';
      if (value === 'true' || value === 'false') index++;
    } else if (arg === '--coverage=true' || arg === '--coverage.enabled=true') coverage = true;
    else if (['--no-coverage', '--coverage=false', '--coverage.enabled=false', '--no-coverage.enabled'].includes(arg)) coverage = false;
    else remaining.push(arg);
  }
  if (forwarded.some(arg => /^(--config|--project|--coverage\.(?:all|include|exclude|provider|reportsDirectory|thresholds|clean))(?:=|$|\.)/.test(arg))) {
    throw new Error('Suite project/configuration and coverage source contract are owner-selected; use direct Vitest for custom experiments.');
  }
  return { suite, forwarded: remaining, coverage };
}
function writeJson(file, value) { writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`); }
function publish(file, destination) {
  const temporary = `${destination}.${randomUUID()}.tmp`;
  writeFileSync(temporary, readFileSync(file));
  renameSync(temporary, destination);
}
export async function runSuite({ argv = process.argv.slice(2), env = process.env, root = uiRoot, execute = spawnSync } = {}) {
  const options = parseArgs(argv, env);
  const projects = options.suite === 'full' ? inventory.projects.map(project => project.name) : inventory.smoke;
  if (new Set(projects).size !== projects.length || inventory.smoke.some(name => !inventory.projects.some(project => project.name === name))) {
    throw new Error('Invalid Vitest project inventory.');
  }
  const coverageDir = join(root, 'coverage');
  const runId = randomUUID();
  const runRoot = join(coverageDir, 'runs', runId);
  const executionRunId = env.VROOLI_TEST_RUN_ID || runId;
  mkdirSync(runRoot, { recursive: true });
  const inputFingerprint = sourceFingerprint(root);
  const manifest = { version: 1, runId, runRoot, executionRunId, nativeObservationProjects: [], suite: options.suite, projects, sourceClosureProject: inventory.sourceClosureProject,
    sourceFingerprint: inputFingerprint, completed: {}, status: 'running' };
  writeJson(join(runRoot, 'manifest.json'), manifest);
  let combined = null;
  let exitCode = 0;
  try {
    for (const project of projects) {
      console.log(`\n=== Running Vitest project: ${project} ===`);
      const projectRoot = join(runRoot, 'raw', project);
      mkdirSync(projectRoot, { recursive: true });
      const requirementsFile = join(projectRoot, 'vitest-requirements.json');
      const extra = options.coverage ? ['--config', 'vitest.coverage.config.ts', '--coverage.reportsDirectory', projectRoot, '--coverage=true'] : ['--coverage=false'];
      const sourceClosure = options.coverage && project === inventory.sourceClosureProject;
      const result = execute('pnpm', ['exec', 'vitest', 'run', '--project', project, ...options.forwarded, ...extra], {
        cwd: root, stdio: 'inherit', shell: process.platform === 'win32',
        env: { ...env, VROOLI_TEST_RUN_ID: executionRunId, NODE_OPTIONS: env.NODE_OPTIONS || '--max-old-space-size=8192',
          BAS_COLLECT_RAW_COVERAGE: options.coverage ? '1' : '0',
          BAS_COVERAGE_SOURCE_CLOSURE: sourceClosure ? '1' : '0',
          BAS_COVERAGE_REPORTS_DIRECTORY: projectRoot,
          BAS_REQUIREMENTS_OUTPUT_FILE: requirementsFile,
          BAS_REQUIREMENTS_ARTIFACTS_DIR: join(projectRoot, 'unit'),
          ...(env.VROOLI_TEST_QUALITY_OUTPUT ? { VROOLI_TEST_QUALITY_OUTPUT: join(projectRoot, 'vitest-native.json') } : {}) },
      });
      if (existsSync(requirementsFile)) {
        const report = JSON.parse(readFileSync(requirementsFile, 'utf8'));
        combined = mergeReports(combined, report);
        if (report.native_observations?.length) manifest.nativeObservationProjects.push(project);
      }
      else if (!result.error && result.status === 0) throw new Error(`Missing behavior requirement report for ${project}.`);
      if (result.error) throw result.error;
      if (result.status !== 0) { exitCode = result.status ?? 1; manifest.status = 'tests-failed'; break; }
      if (inputFingerprint !== sourceFingerprint(root)) throw new Error('Source inputs changed during the run; reports cannot be merged.');
      manifest.completed[project] = { sourceFingerprint: inputFingerprint, sourceClosure,
        ...(options.coverage ? { reportFingerprint: fileFingerprint(join(projectRoot, 'coverage-final.json')) } : {}) };
      writeJson(join(runRoot, 'manifest.json'), manifest);
    }
    if (exitCode === 0 && options.coverage) {
      manifest.status = 'merging';
      writeJson(join(runRoot, 'manifest.json'), manifest);
      const result = await mergeCoverageRun(runRoot, root);
      manifest.status = result.status;
      manifest.coverage = result;
      for (const name of ['coverage-final.json', 'coverage-summary.json']) publish(join(runRoot, name), join(coverageDir, name));
      if (result.status === 'below-floor') {
        console.error(`Merged UI coverage is below the ${result.floor}% floor: ${result.failed.join(', ')}.`);
        exitCode = 1;
      }
    } else if (exitCode === 0) manifest.status = 'passed';
  } catch (error) {
    manifest.status = 'incomplete';
    manifest.error = error.message;
    console.error(`UI suite incomplete: ${error.message}`);
    exitCode = 2;
  } finally {
    // Behavior evidence survives a failing floor and partial-suite errors.
    if (combined) {
      writeJson(join(runRoot, 'vitest-requirements.json'), combined);
      publish(join(runRoot, 'vitest-requirements.json'), join(coverageDir, 'vitest-requirements.json'));
    }
    if (env.VROOLI_TEST_QUALITY_OUTPUT) {
      try {
        const { mergeNativeObservationReports, nativeObservations } = await import('@vrooli/vitest-requirement-reporter');
        const reports = combined?.native_observations ?? [];
        const missing = projects.filter(project => !manifest.nativeObservationProjects.includes(project));
        let observation;
        try {
          observation = reports.length ? mergeNativeObservationReports(reports)
            : nativeObservations([], 'unknown', () => false, ['No project-native observation reports were produced.'], executionRunId);
        } catch (error) {
          observation = nativeObservations([], 'unknown', () => false, [`Invalid project-native reports: ${error.message}`], executionRunId);
          manifest.status = 'incomplete';
          manifest.error = error.message;
          exitCode = 2;
        }
        if (missing.length) observation.unhandled_errors.push(`Incomplete BAS suite: missing native reports for selected projects: ${missing.join(', ')}.`);
        writeJson(join(runRoot, 'vitest-native.json'), observation);
        mkdirSync(dirname(env.VROOLI_TEST_QUALITY_OUTPUT), { recursive: true });
        publish(join(runRoot, 'vitest-native.json'), env.VROOLI_TEST_QUALITY_OUTPUT);
      } catch (error) {
        console.error(`Native quality handoff unavailable: ${error.message}`);
        manifest.status = 'incomplete';
        manifest.error = error.message;
        exitCode = 2;
      }
    }
    writeJson(join(runRoot, 'manifest.json'), manifest);
    publish(join(runRoot, 'manifest.json'), join(coverageDir, 'vitest-run.json'));
    console.log(`UI run evidence: ${runRoot} (${manifest.status})`);
  }
  return exitCode;
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try { process.exitCode = await runSuite(); }
  catch (error) { console.error(error.message); process.exitCode = 2; }
}

function mergeReports(a, b) {
  if (!a) return b;
  const merged = new Map();

  const addReq = (req) => {
    const existing = merged.get(req.id);
    if (!existing) {
      merged.set(req.id, { ...req });
      return;
    }
    // A requirement covered by several projects fails if it fails anywhere.
    if (req.status === "failed") existing.status = "failed";
    if (req.evidence) {
      existing.evidence = existing.evidence ? `${existing.evidence}; ${req.evidence}` : req.evidence;
    }
    existing.duration_ms = (existing.duration_ms || 0) + (req.duration_ms || 0);
    existing.test_count = (existing.test_count || 0) + (req.test_count || 0);
  };

  a.requirements?.forEach(addReq);
  b.requirements?.forEach(addReq);

  return {
    generated_at: b.generated_at,
    scenario: b.scenario || a.scenario,
    phase: b.phase || a.phase,
    test_framework: b.test_framework || a.test_framework,
    total_tests: (a.total_tests || 0) + (b.total_tests || 0),
    passed_tests: (a.passed_tests || 0) + (b.passed_tests || 0),
    failed_tests: (a.failed_tests || 0) + (b.failed_tests || 0),
    skipped_tests: (a.skipped_tests || 0) + (b.skipped_tests || 0),
    duration_ms: (a.duration_ms || 0) + (b.duration_ms || 0),
    native_observations: [...(a.native_observations ?? []), ...(b.native_observations ?? [])],
    requirements: Array.from(merged.values()).sort((x, y) => x.id.localeCompare(y.id)),
  };
}
