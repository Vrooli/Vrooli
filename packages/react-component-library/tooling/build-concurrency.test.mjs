import { chmod, cp, mkdir, mkdtemp, readdir, rm, symlink, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { spawn } from "node:child_process";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import assert from "node:assert/strict";

const packageRoot = join(fileURLToPath(new URL("..", import.meta.url)));
const repoRoot = join(packageRoot, "..", "..");

// A wrapper in place of the package's tsc. The first invocation announces that
// it is about to compile and waits for a release file; every other invocation
// compiles at once. That freezes one build exactly where its generated compiler
// inputs exist but have not been read, so a second build can run to completion
// (and clean up) in between, deterministically.
const gatedTsc = (nodePath, realTsc) => `#!${nodePath}
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
const gate = process.env.RCL_TEST_TSC_GATE;
let first = false;
try { mkdirSync(gate + ".claimed"); first = true; } catch {}
if (first) {
  writeFileSync(gate + ".held", "");
  while (!existsSync(gate + ".release")) await new Promise((resolve) => setTimeout(resolve, 20));
}
process.exit(spawnSync(process.execPath, [${JSON.stringify(realTsc)}, ...process.argv.slice(2)], { stdio: "inherit" }).status ?? 1);
`;

/** A throwaway package root beside a one-asset library: the real build scripts, a real asset, no shared state. */
async function hermeticPackage() {
  const root = await mkdtemp(join(tmpdir(), "rcl-build-concurrency-"));
  const pkg = join(root, "packages", "react-component-library");
  const library = join(root, "scenarios", "react-component-library", "library", "foundations");
  await mkdir(pkg, { recursive: true });
  await mkdir(library, { recursive: true });
  await cp(join(packageRoot, "tooling"), join(pkg, "tooling"), { recursive: true });
  for (const file of ["tsconfig.build.json", "package.json"]) await cp(join(packageRoot, file), join(pkg, file));
  await cp(join(repoRoot, "scenarios", "react-component-library", "library", "foundations", "StyleSheet"), join(library, "StyleSheet"), { recursive: true });
  const modules = join(pkg, "node_modules");
  await mkdir(join(modules, ".bin"), { recursive: true });
  for (const entry of await readdir(join(packageRoot, "node_modules"))) {
    if (entry !== ".bin") await symlink(join(packageRoot, "node_modules", entry), join(modules, entry));
  }
  await writeFile(join(modules, ".bin", "tsc"), gatedTsc(process.execPath, join(packageRoot, "node_modules", "typescript", "bin", "tsc")));
  await chmod(join(modules, ".bin", "tsc"), 0o755);
  return { root, pkg };
}

function runBuild(pkg, args, env) {
  const child = spawn(process.execPath, [join(pkg, "tooling", "build.mjs"), ...args], { cwd: pkg, env: { ...process.env, ...env }, stdio: ["ignore", "pipe", "pipe"] });
  let output = "";
  child.stdout.on("data", (chunk) => { output += chunk; });
  child.stderr.on("data", (chunk) => { output += chunk; });
  const done = new Promise((resolve) => child.on("close", (code) => resolve({ code, output })));
  return { done };
}

async function until(condition, what) {
  const deadline = Date.now() + 30000;
  while (!condition()) {
    if (Date.now() > deadline) throw new Error(`timed out waiting for ${what}`);
    await new Promise((resolve) => setTimeout(resolve, 20));
  }
}

// Two builds of the package overlap (a scenario start, a refresh and an agent's
// build can all be running). Each one's cleanup used to delete the fixed-name
// tsconfig and `process` declaration the other's tsc was about to read, so the
// second compile failed (TS2559 on code that declares nothing wrong).
test("a build's compiler inputs survive another build finishing and cleaning up first", async () => {
  const { root, pkg } = await hermeticPackage();
  try {
    const gate = join(root, "gate");
    const env = { RCL_TEST_TSC_GATE: gate };
    const first = runBuild(pkg, ["--check-only"], env);
    await until(() => existsSync(`${gate}.held`), "the first build to reach its compiler");

    const second = await runBuild(pkg, ["--check-only"], env).done;
    assert.equal(second.code, 0, second.output);
    assert.match(second.output, /Checked @vrooli\/react-component-library sources\./);

    await writeFile(`${gate}.release`, "");
    const result = await first.done;
    assert.equal(result.code, 0, `the first build lost its compiler inputs:\n${result.output}`);
    assert.match(result.output, /Checked @vrooli\/react-component-library sources\./);
    assert.deepEqual((await readdir(pkg)).filter((name) => name.startsWith(".build-") || name.startsWith(".rcl-candidate-")), [], "no build leaves scratch files in the package root");
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
