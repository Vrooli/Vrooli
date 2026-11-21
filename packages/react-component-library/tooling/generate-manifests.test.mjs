import { readFile, mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import test from "node:test";
import assert from "node:assert/strict";
import { generateManifests } from "./generate-manifests.mjs";

async function writeFixture(root, name, manifest) {
  const directory = join(root, "components", name);
  const path = join(directory, "component.json");
  await mkdir(directory, { recursive: true });
  await writeFile(path, `${JSON.stringify(manifest, null, 2)}\n`);
  return path;
}

test("manifest generation preserves authored fields and is checkable/idempotent", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "rcl-manifest-generation-"));
  t.after(async () => rm(root, { recursive: true, force: true }));
  const path = await writeFixture(root, "ClassMerge", {
    "$schema": "../../../../../.vrooli/schemas/component-manifest.schema.json",
    assetKind: "foundation",
    libraryId: "react-component-library:class-merge",
    displayName: "Class Merge",
    kind: "surface",
    description: "Conditional class composition with consumer overrides.",
    tags: ["foundations", "styling"],
    entry: "ClassMerge.tsx",
    fileSlots: { source: "versions/1.0.0/ClassMerge.tsx" },
    latest: "1.0.0",
    deprecatedVersions: [],
    draft: "",
    evictedVersions: [],
  });
  const before = await readFile(path, "utf8");

  const stale = await generateManifests({ root, check: true });
  assert.equal(stale.stale.length, 1);
  assert.equal(await readFile(path, "utf8"), before, "check mode must not rewrite source manifests");

  const generated = await generateManifests({ root });
  assert.equal(generated.written, 1);
  const generatedBytes = await readFile(path, "utf8");
  const manifest = JSON.parse(generatedBytes);
  assert.equal(manifest.catalogId, "react-component-library:class-merge");
  assert.equal(manifest.displayName, "Class Merge");
  assert.equal(manifest.kind, "surface");
  assert.equal(manifest.description, "Conditional class composition with consumer overrides.");
  assert.deepEqual(manifest.tags, ["foundations", "styling"]);
  assert.equal(manifest.entry, "ClassMerge.tsx");
  assert.deepEqual(manifest.fileSlots, { source: "versions/1.0.0/ClassMerge.tsx" });
  assert.deepEqual(Object.keys(manifest), Object.keys(manifest).sort(), "top-level serialization is stable");

  assert.deepEqual(await generateManifests({ root, check: true }), { written: 0, stale: [] });
  await generateManifests({ root });
  assert.equal(await readFile(path, "utf8"), generatedBytes, "a second generation must be byte-identical");
});

test("supplemental manifests retain a null catalog identity", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "rcl-manifest-supplemental-"));
  t.after(async () => rm(root, { recursive: true, force: true }));
  const path = await writeFixture(root, "AuthClient", {
    assetKind: "component",
    libraryId: "react-component-library:auth-client",
    catalogId: null,
    displayName: "Auth Client",
    kind: "surface",
    latest: "1.0.0",
    deprecatedVersions: [],
    supplemental: true,
  });

  await generateManifests({ root });

  const manifest = JSON.parse(await readFile(path, "utf8"));
  assert.equal(manifest.supplemental, true);
  assert.equal(manifest.catalogId, null);
});
