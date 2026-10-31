import { mkdir, mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";
import ts from "typescript";
import test from "node:test";
import assert from "node:assert/strict";
import { packageBuildEnvironmentDeclarations, packageBuildTypes } from "./build-environment.mjs";

const packageRoot = join(fileURLToPath(new URL("..", import.meta.url)));

test("browser-target build environment keeps process optional and active checks strict", async () => {
  const buildSource = await readFile(join(packageRoot, "tooling", "build.mjs"), "utf8");
  assert.match(buildSource, /compilerOptions\.types = \[\.\.\.packageBuildTypes\]/);
  assert.match(buildSource, /writeFile\(environmentDeclarationPath, packageBuildEnvironmentDeclarations\)/);
  assert.deepEqual(packageBuildTypes, ["react", "react-dom"]);
  assert.match(packageBuildEnvironmentDeclarations, /declare var process: \{ env\?: \{ NODE_ENV\?: string \} \} \| undefined;/);

  const repoRoot = join(packageRoot, "..", "..");
  const sourcePath = join(repoRoot, "scenarios", "react-component-library", "library", "foundations", "StyleSheet", "versions", "1.0.1", "StyleSheet.ts");
  const source = await readFile(sourcePath, "utf8");
  assert.match(source, /runtimeGlobal\.process\?\.env\?\.NODE_ENV/);

  const fixtureRoot = await mkdtemp(join(tmpdir(), "rcl-package-build-environment-"));
  try {
    const oldDir = join(fixtureRoot, "deprecated");
    await mkdir(oldDir, { recursive: true });
    await writeFile(join(fixtureRoot, "build-environment.d.ts"), packageBuildEnvironmentDeclarations);
    await writeFile(join(oldDir, "StyleSheet.ts"), source);
    const typeRoot = join(packageRoot, "node_modules", "@types");
    const options = (outDir) => ({
      target: ts.ScriptTarget.ES2022,
      lib: ["lib.es2022.d.ts", "lib.dom.d.ts", "lib.dom.iterable.d.ts"],
      module: ts.ModuleKind.ESNext,
      moduleResolution: ts.ModuleResolutionKind.Bundler,
      jsx: ts.JsxEmit.ReactJSX,
      strict: true,
      skipLibCheck: true,
      declaration: true,
      rootDir: fixtureRoot,
      outDir,
      baseUrl: packageRoot,
      typeRoots: [typeRoot],
      types: [...packageBuildTypes],
      paths: {
        react: [join(typeRoot, "react", "index.d.ts")],
        "react/jsx-runtime": [join(typeRoot, "react", "jsx-runtime.d.ts")],
      },
    });
    const programFor = (files, config) => {
      const program = ts.createProgram({ rootNames: files, options: config });
      return { program, diagnostics: ts.getPreEmitDiagnostics(program) };
    };
    const oldOut = join(fixtureRoot, "old-out");
    const old = programFor([join(fixtureRoot, "build-environment.d.ts"), join(oldDir, "StyleSheet.ts")], options(oldOut));
    assert.deepEqual(old.diagnostics.map((item) => item.code), []);
    assert.equal(old.program.emit().emitSkipped, false);
    const oldDeclaration = await readFile(join(oldOut, "deprecated", "StyleSheet.d.ts"), "utf8");
    assert.doesNotMatch(oldDeclaration, /\bprocess\b/, "build-only globals must not leak to consumers");
    await assert.rejects(readFile(join(oldOut, "build-environment.d.ts")), { code: "ENOENT" });

    const invalidActive = join(fixtureRoot, "active-invalid.ts");
    await writeFile(invalidActive, 'export const invalidActive: number = "not a number";\n');
    const active = programFor([join(fixtureRoot, "build-environment.d.ts"), invalidActive], options(join(fixtureRoot, "active-out")));
    assert.ok(active.diagnostics.some((item) => item.code === 2322), "active semantic errors remain fatal");

    const unguarded = join(fixtureRoot, "unguarded-process.ts");
    await writeFile(unguarded, "export const mode = globalThis.process.env.NODE_ENV;\n");
    const unsafe = programFor([join(fixtureRoot, "build-environment.d.ts"), unguarded], options(join(fixtureRoot, "unsafe-out")));
    assert.ok(unsafe.diagnostics.length > 0, "unguarded process access is invalid in browser-target types");

    const instrumented = source.replace(
      'const isDevelopment = runtimeGlobal.process?.env?.NODE_ENV !== "production";',
      'const isDevelopment = runtimeGlobal.process?.env?.NODE_ENV !== "production";\nexport const __developmentForTest = isDevelopment;',
    );
    assert.notEqual(instrumented, source);
    const js = ts.transpileModule(instrumented, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
    const developmentMode = (processValue) => {
      const exports = {};
      const context = { exports, require: () => ({ useInsertionEffect() {} }), console };
      if (processValue !== undefined) context.process = processValue;
      vm.runInNewContext(js, vm.createContext(context));
      return exports.__developmentForTest;
    };
    assert.equal(developmentMode(undefined), true);
    assert.equal(developmentMode({ env: { NODE_ENV: "production" } }), false);
    assert.equal(developmentMode({ env: { NODE_ENV: "development" } }), true);
  } finally {
    await rm(fixtureRoot, { recursive: true, force: true });
  }
});

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

test("ordinary package compilation keeps every draft tree excluded", async () => {
  const buildSource = await readFile(join(packageRoot, "tooling", "build.mjs"), "utf8");
  assert.match(buildSource, /versions\/\*\*\/\*-draft\.\*/);
  assert.match(buildSource, /if \(draftTypecheck\)/, "draft checks use a separate no-emit branch");
  assert.match(buildSource, /workbenchBrokenImports\.length > 0 && !draftTypecheck/, "draft checks cannot enter the build failure/fallback status writer");
  assert.match(buildSource, /process\.exit\(draftExitCode\)/, "draft checks exit before candidate creation and export synchronization");
});

test("package build never rewrites authored library sources", async () => {
  const buildSource = await readFile(join(packageRoot, "tooling", "build.mjs"), "utf8");
  assert.doesNotMatch(buildSource, /(?:readFile|writeFile)\([^\n]*sourceRoot/);
  assert.doesNotMatch(buildSource, /writeFile\([^\n]*sourceRoot/);
  assert.match(buildSource, /const rewritten = source\.replace\(/, "only the disposable emitted artifact may be normalized");
});
