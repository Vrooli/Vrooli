import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync } from "node:fs";
import { mkdir, mkdtemp, readFile, rm, stat, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import test from "node:test";
import assert from "node:assert/strict";
import { defaultArtifactStoreRoot, directoryDigest } from "./runtime-artifact.mjs";

const packageRoot = join(fileURLToPath(new URL("..", import.meta.url)));
const repoRoot = join(packageRoot, "..", "..");
const authoredRoot = join(repoRoot, "scenarios", "react-component-library", "library");
const component = "ClassMerge";
const version = "0.1.0-draft.1";
const manifest = join(authoredRoot, "foundations", component, "component.json");
const buildTool = join(packageRoot, "tooling", "build.mjs");

function runDraft(manifestPath, sourcePath, selectedVersion, selectedComponent, testRoot) {
  const env = { ...process.env };
  delete env.RCL_DRAFT_TYPECHECK_TEST_ROOT;
  env.NODE_ENV = "test";
  env.RCL_DRAFT_TYPECHECK_TEST_ROOT = testRoot;
  return spawnSync(process.execPath, [buildTool, "--draft-typecheck", "--component", selectedComponent, "--version", selectedVersion,
    "--manifest", manifestPath, "--source", sourcePath], { cwd: repoRoot, encoding: "utf8", env });
}

async function fixtureFor(sourceText, fixtureVersion = version, fixtureComponent = component) {
  const fixtureRoot = await mkdtemp(join(tmpdir(), "rcl-classmerge-contract-"));
  const fixtureDir = join(fixtureRoot, "foundations", fixtureComponent);
  const fixtureManifest = join(fixtureDir, "component.json");
  const fixtureSource = join(fixtureDir, "versions", fixtureVersion, `${fixtureComponent}.tsx`);
  await mkdir(join(fixtureDir, "versions", fixtureVersion), { recursive: true });
  await writeFile(fixtureManifest, `${JSON.stringify({ libraryId: `react-component-library:${fixtureComponent}`, draft: fixtureVersion })}\n`);
  await writeFile(fixtureSource, sourceText);
  return { fixtureRoot, fixtureManifest, fixtureSource, fixtureDir };
}

async function currentSource() {
  const descriptor = JSON.parse(await readFile(manifest, "utf8"));
  const selectedVersion = descriptor.draft || descriptor.latest;
  assert.ok(selectedVersion, "ClassMerge manifest identifies a current source version");
  const path = join(authoredRoot, "foundations", component, "versions", selectedVersion, `${component}.tsx`);
  return { path, text: await readFile(path, "utf8") };
}

async function state(path) {
  if (!existsSync(path)) return "absent";
  if ((await stat(path)).isDirectory()) {
    return `directory:${await directoryDigest(path)}`;
  }
  return `file:${createHash("sha256").update(await readFile(path)).digest("hex")}`;
}

const classNameConsumer = `
const ClassMergeInner: ComponentType<{ label: string; count: number }> = ({ label, count }) => null;
const ClassMergeWrapped = withClassName(ClassMergeInner);
const validClassMergeConsumer = <ClassMergeWrapped label="notice" count={2} className="text-sm" />;
// @ts-expect-error required inner props remain required
const missingClassMergeInnerProp = <ClassMergeWrapped label="notice" className="text-sm" />;
// @ts-expect-error unknown props do not leak through the wrapper
const unknownClassMergeProp = <ClassMergeWrapped label="notice" count={2} className="text-sm" extra />;
function GenericRows<T>({items, renderItem, className}: {items: readonly T[]; renderItem: (item: T) => ReactNode; className?: string}) { return null; }
const WrappedGenericRows = withClassName(GenericRows);
const validGenericRows = <WrappedGenericRows items={[{id: "a"}]} renderItem={(row) => row.id} className="rows" />;
// @ts-expect-error generic item members remain checked
const invalidGenericRows = <WrappedGenericRows items={[{id: "a"}]} renderItem={(row) => row.missing} className="rows" />;
const Bare = ({required}: {required: string}) => null;
const WrappedBare = withClassName(Bare);
const validBare = <WrappedBare required="x" className="rows" />;
// @ts-expect-error required bare-component props remain required
const missingBareRequired = <WrappedBare className="rows" />;
// @ts-expect-error unknown bare-component props remain rejected
const extraBareProp = <WrappedBare required="x" className="rows" extra />;
`;

test("active draft compiler selection is exact and fails closed", async () => {
  const fixture = await fixtureFor("export const ClassMergeFixture = 1;\n");
  try {
    const wrongVersion = runDraft(fixture.fixtureManifest, fixture.fixtureSource, "0.1.0-draft.2", component, fixture.fixtureRoot);
    assert.notEqual(wrongVersion.status, 0);
    assert.match(wrongVersion.stderr, /not the exact active draft/);

    const wrongSource = join(fixture.fixtureDir, "versions", "0.1.0", `${component}.tsx`);
    await mkdir(join(fixture.fixtureDir, "versions", "0.1.0"), { recursive: true });
    await writeFile(wrongSource, "export const ClassMergeFixture = 1;\n");
    const wrongPath = runDraft(fixture.fixtureManifest, wrongSource, version, component, fixture.fixtureRoot);
    assert.notEqual(wrongPath.status, 0);
    assert.match(wrongPath.stderr, /inside the selected active version record/);
  } finally {
    await rm(fixture.fixtureRoot, { recursive: true, force: true });
  }
});

test("current ClassMerge source preserves generic and bare function component props", async () => {
  const original = await currentSource();
  const fixture = await fixtureFor(`${original.text}\n${classNameConsumer}`);
  try {
    const checked = runDraft(fixture.fixtureManifest, fixture.fixtureSource, version, component, fixture.fixtureRoot);
    assert.equal(checked.status, 0, checked.stdout + checked.stderr);
    assert.equal((await readFile(original.path, "utf8")), original.text, "checking current source leaves it unchanged");
  } finally {
    await rm(fixture.fixtureRoot, { recursive: true, force: true });
  }
});

test("no-emit package configuration accepts valid draft input and reports compiler diagnostics", async () => {
  const fixtureRoot = await mkdtemp(join(tmpdir(), "rcl-draft-typecheck-test-"));
  try {
    const fixtureVersion = "0.1.0-draft.1";
    const fixtureComponent = "TypecheckFixture";
    const fixtureDir = join(fixtureRoot, "foundations", fixtureComponent);
    const fixtureManifest = join(fixtureDir, "component.json");
    const fixtureSource = join(fixtureDir, "versions", fixtureVersion, `${fixtureComponent}.tsx`);
    await mkdir(join(fixtureDir, "versions", fixtureVersion), { recursive: true });
    await writeFile(fixtureManifest, `${JSON.stringify({ libraryId: `react-component-library:${fixtureComponent}`, draft: fixtureVersion })}\n`);
    await writeFile(fixtureSource, "export const TypecheckFixture: number = 42;\n");

    const valid = runDraft(fixtureManifest, fixtureSource, fixtureVersion, fixtureComponent, fixtureRoot);
    assert.equal(valid.status, 0, valid.stdout + valid.stderr);
    assert.match(valid.stdout, /Checked active draft TypecheckFixture@0\.1\.0-draft\.1 without emitting files/);
    assert.equal(existsSync(join(fixtureRoot, "dist")), false, "valid active draft no-emit leaves output absent");

    await writeFile(fixtureSource, 'export const TypecheckFixture: number = "not a number";\n');
    const invalid = runDraft(fixtureManifest, fixtureSource, fixtureVersion, fixtureComponent, fixtureRoot);
    assert.notEqual(invalid.status, 0);
    assert.match(invalid.stdout + invalid.stderr, /TS2322/);
    assert.equal(existsSync(join(fixtureRoot, "dist")), false, "invalid active draft no-emit leaves output absent");
  } finally {
    await rm(fixtureRoot, { recursive: true, force: true });
  }
});

test("draft checking leaves package and runtime artifacts plus current source inputs unchanged", async () => {
  const original = await currentSource();
  const fixture = await fixtureFor(`${original.text}\n${classNameConsumer}`);
  try {
    const artifactRoot = defaultArtifactStoreRoot();
    const paths = [
      join(packageRoot, "dist"),
      join(packageRoot, "package.json"),
      join(artifactRoot, "runtime-status.json"),
      join(artifactRoot, "current.json"),
      join(artifactRoot, "previous.json"),
      original.path,
      manifest,
    ];
    const beforeAll = await Promise.all(paths.map(state));
    const checked = runDraft(fixture.fixtureManifest, fixture.fixtureSource, version, component, fixture.fixtureRoot);
    assert.equal(checked.status, 0, checked.stdout + checked.stderr);
    assert.deepEqual(await Promise.all(paths.map(state)), beforeAll);
  } finally {
    await rm(fixture.fixtureRoot, { recursive: true, force: true });
  }
});
