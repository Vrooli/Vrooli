import { readFile, mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import test from "node:test";
import assert from "node:assert/strict";
import { generateManifests } from "./generate-manifests.mjs";

test("manifest projections preserve an explicit entry file", async () => {
  const root = await mkdtemp(join(tmpdir(), "rcl-manifest-projection-"));
  const path = join(root, "components", "MarkdownRenderer", "component.json");
  await mkdir(join(root, "components", "MarkdownRenderer"), { recursive: true });
  await writeFile(path, JSON.stringify({
    libraryId: "react-component-library:markdown-renderer",
    assetKind: "component",
    entry: "markdown-renderer.tsx",
    latest: "0.4.2",
    deprecatedVersions: [],
    authorField: "preserve only when governed",
  }, null, 2) + "\n");

  await generateManifests({ root });

  const projected = JSON.parse(await readFile(path, "utf8"));
  assert.equal(projected.entry, "markdown-renderer.tsx");
  assert.equal(projected.authorField, undefined);
});
