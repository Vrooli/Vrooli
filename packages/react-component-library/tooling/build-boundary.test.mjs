import { readdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import assert from "node:assert/strict";

const packageRoot = join(fileURLToPath(new URL("..", import.meta.url)));

test("package build configuration has no static scenario-tree reach", async () => {
  const forbidden = ["scenario", "s/"].join("");
  const paths = [join(packageRoot, "tsconfig.build.json")];
  // These adapters are intentionally the package-to-scenario boundary: the
  // package build validates its workbench consumer and the DOM-binding probe
  // resolves authored component sources. They are not package-runtime reach.
  const scenarioAwareMaintenance = new Set(["build.mjs", "dom-bindings.mjs", "dom-bindings.test.mjs"]);
  for (const entry of await readdir(join(packageRoot, "tooling"), { withFileTypes: true })) {
    // These maintenance scripts intentionally operate on the scenario
    // tree. That is their governed boundary, not a package-build dependency.
    if (entry.isFile() && entry.name.endsWith(".mjs")
      && !scenarioAwareMaintenance.has(entry.name)
      && entry.name !== "reap-retired.mjs") {
      paths.push(join(packageRoot, "tooling", entry.name));
    }
  }
  const violations = [];
  for (const path of paths) {
    const source = await readFile(path, "utf8");
    if (source.includes(forbidden)) violations.push(path);
  }
  assert.deepEqual(violations, []);
});

test("package build uses its owned compiler and keeps React as a peer", async () => {
  const buildSource = await readFile(join(packageRoot, "tooling", "build.mjs"), "utf8");
  assert.match(buildSource, /join\(packageRoot, ["']node_modules["'], ["']\.bin["'], ["']tsc["']\)/);
  assert.doesNotMatch(buildSource, /scenarioNodeModules|ui["'].*node_modules|RCL_TYPESCRIPT_BIN/);

  const manifest = JSON.parse(await readFile(join(packageRoot, "package.json"), "utf8"));
  assert.equal(manifest.peerDependencies?.react, ">=18.0.0");
  assert.equal(manifest.dependencies?.react, undefined);
  assert.ok(manifest.devDependencies?.react, "React is available for the package build without becoming a runtime dependency");
  assert.ok(manifest.devDependencies?.typescript, "the package declares its own compiler");
});

test("package build never rewrites authored library sources", async () => {
  const buildSource = await readFile(join(packageRoot, "tooling", "build.mjs"), "utf8");
  assert.doesNotMatch(buildSource, /(?:readFile|writeFile)\([^\n]*sourceRoot/);
  assert.doesNotMatch(buildSource, /writeFile\([^\n]*sourceRoot/);
  assert.match(buildSource, /const rewritten = source\.replace\(/, "only the disposable emitted artifact may be normalized");
});
