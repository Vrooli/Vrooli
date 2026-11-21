// Provision a disposable native fixture using already-installed governed
// dependencies. This runner does not install packages or consult expectations.
import { cpSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { spawnSync } from "node:child_process";

const [modulesArgument, outputArgument, reporterArgument] = process.argv.slice(2);
if (!modulesArgument || !outputArgument) throw new Error("usage: node run-native.mjs <installed-node_modules> <observation-output.json>");
const modules = resolve(modulesArgument);
const output = resolve(outputArgument);
const version = JSON.parse(readFileSync(join(modules, "vitest/package.json"), "utf8")).version;
if (version !== "2.1.9") {
  writeFileSync(output, JSON.stringify({ version, support: "unsupported-version", cases: [] }, null, 2));
  process.exitCode = 2;
} else {
  const root = mkdtempSync(join(tmpdir(), "unit-health-native-"));
  try {
    cpSync(join(dirname(fileURLToPath(import.meta.url)), "native/vitest-v2"), root, { recursive: true });
    symlinkSync(modules, join(root, "node_modules"), "dir");
    const result = spawnSync(process.execPath, [join(modules, "vitest/vitest.mjs"), "run", "--root", root,
      "--config", join(root, "vitest.config.mjs"), "--outputFile", join(root, "results.json")], {
      cwd: root, encoding: "utf8", timeout: 45000, maxBuffer: 8 * 1024 * 1024,
      env: { ...process.env, QUALITY_NATIVE_EVENTS: join(root, "events.json"),
        VROOLI_TEST_RUN_ID: `probe-${root.split('/').at(-1)}`,
        VROOLI_TEST_QUALITY_OUTPUT: join(root, 'native-handoff.json'),
        QUALITY_REQUIREMENT_REPORTER: reporterArgument ? pathToFileURL(resolve(reporterArgument)).href : '' },
    });
    if (result.error) throw result.error;
    const raw = JSON.parse(readFileSync(join(root, "results.json"), "utf8"));
    const events = JSON.parse(readFileSync(join(root, "events.json"), "utf8"));
    const requirementReporter = reporterArgument ? JSON.parse(readFileSync(join(root, "requirements.json"), "utf8")) : undefined;
    const nativeHandoff = reporterArgument ? JSON.parse(readFileSync(join(root, "native-handoff.json"), "utf8")) : undefined;
    const cases = raw.testResults.flatMap(file => file.assertionResults.map(test => ({
      name: test.fullName, status: test.status, failureMessages: test.failureMessages,
    }))).sort((a, b) => a.name.localeCompare(b.name));
    writeFileSync(output, JSON.stringify({ version, support: "observed-not-certified", runnerExitCode: result.status,
      cases, reporter: events, requirementReporter, nativeHandoff, stderr: result.stderr }, null, 2));
    console.log(JSON.stringify({ version, runnerExitCode: result.status, cases: cases.map(({ name, status }) => ({ name, status })) }));
  } finally {
    // Only this runner-created mkdtemp root is removed, never installed packages.
    rmSync(root, { recursive: true, force: true });
  }
}
