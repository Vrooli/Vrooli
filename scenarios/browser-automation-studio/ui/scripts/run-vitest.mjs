// Run the Vitest projects that make up a BAS suite, one project per process.
//
// The suite is split by project because running it as one process exhausts the
// heap, and per-project coverage has to be collected into separate raw
// directories before it can be merged. All of that used to live in bash; it is
// Node here so the suite runs the same on every host.
//
// The project list is selected by `--suite smoke|full` (default smoke), or by
// BAS_VITEST_SUITE for callers that already export it. The flag exists because
// the `VAR=value command` prefix this used to rely on is POSIX-only — cmd.exe
// has no equivalent. Any remaining CLI arguments are forwarded to vitest.
import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readdirSync, readFileSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");

const SMOKE_PROJECTS = ["boundaries", "stores", "features-core", "workflow-palette", "workflow-builder"];
const FULL_PROJECTS = [
  ...SMOKE_PROJECTS,
  "utils", "api-clients", "record-mode", "session-manager", "subscription",
  "components", "export-domain", "exports-domain", "shared",
];

const argv = process.argv.slice(2);
const suiteFlag = argv.indexOf("--suite");
const suite = suiteFlag === -1 ? (process.env.BAS_VITEST_SUITE || "smoke") : argv[suiteFlag + 1];
const forwarded = suiteFlag === -1 ? argv : [...argv.slice(0, suiteFlag), ...argv.slice(suiteFlag + 2)];

const projects = suite === "smoke" ? SMOKE_PROJECTS : suite === "full" ? FULL_PROJECTS : null;
if (projects === null) {
  console.error(`Unknown suite ${JSON.stringify(suite)}: expected 'smoke' or 'full'`);
  process.exit(2);
}
const wantsCoverage = forwarded.some(
  (arg) => arg === "--coverage" || arg.startsWith("--coverage=") || arg === "--no-coverage",
);

const coverageDir = join(uiRoot, "coverage");
rmSync(coverageDir, { recursive: true, force: true });
mkdirSync(coverageDir, { recursive: true });

const nodeOptions = process.env.NODE_OPTIONS || "--max-old-space-size=8192";

for (const project of projects) {
  console.log(`\n=== Running Vitest project: ${project} ===`);

  const extra = [];
  // The boundaries project is an architecture check, not a covered surface.
  if (project === "boundaries" && !wantsCoverage) extra.push("--coverage=false");
  if (wantsCoverage) {
    extra.push("--config", "vitest.coverage.config.ts",
               "--coverage.reportsDirectory", `coverage/raw/${project}`);
  }

  const env = { ...process.env, NODE_OPTIONS: nodeOptions };
  if (wantsCoverage) {
    env.BAS_COLLECT_RAW_COVERAGE = "1";
    env.BAS_COVERAGE_REPORTS_DIRECTORY = `coverage/raw/${project}`;
  }

  const result = spawnSync("pnpm", ["exec", "vitest", "run", "--project", project, ...extra, ...forwarded], {
    cwd: uiRoot,
    stdio: "inherit",
    env,
    shell: process.platform === "win32",
  });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);

  // Per-project requirements land in one of two places depending on whether
  // coverage redirected the reports directory. Stash them under a per-project
  // name so the next project cannot overwrite them.
  const target = join(coverageDir, `vitest-requirements-${project}.json`);
  const raw = join(coverageDir, "raw", project, "vitest-requirements.json");
  const flat = join(coverageDir, "vitest-requirements.json");
  if (existsSync(raw)) renameSync(raw, target);
  else if (existsSync(flat)) renameSync(flat, target);
}

if (wantsCoverage) {
  const merge = spawnSync("node", ["./scripts/merge-v8-coverage.mjs"], {
    cwd: uiRoot,
    stdio: "inherit",
    shell: process.platform === "win32",
  });
  if (merge.status !== 0) process.exit(merge.status ?? 1);
}

mergeRequirementReports();

// Fold the per-project requirement reports back into the single
// vitest-requirements.json the test harness reads.
function mergeRequirementReports() {
  const files = existsSync(coverageDir)
    ? readdirSync(coverageDir).filter((f) => f.startsWith("vitest-requirements-") && f.endsWith(".json"))
    : [];
  if (files.length === 0) return;

  let combined = null;
  for (const file of files) {
    const report = JSON.parse(readFileSync(join(coverageDir, file), "utf-8"));
    combined = mergeReports(combined, report);
  }
  if (combined) {
    writeFileSync(join(coverageDir, "vitest-requirements.json"), JSON.stringify(combined, null, 2));
  }
  for (const file of files) rmSync(join(coverageDir, file));
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
    requirements: Array.from(merged.values()).sort((x, y) => x.id.localeCompare(y.id)),
  };
}
