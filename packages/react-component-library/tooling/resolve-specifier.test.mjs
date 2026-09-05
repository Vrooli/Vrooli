import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import test from "node:test";
import assert from "node:assert/strict";
import { resolveLibrarySpecifier } from "./resolve-specifier.mjs";

test("bare and major imports resolve the latest active release, never a draft", async () => {
  const root = mkdtempSync(join(tmpdir(), "rcl-source-resolver-"));
  const assetRoot = join(root, "components", "Panel");
  try {
    mkdirSync(join(assetRoot, "versions", "1.0.0"), { recursive: true });
    mkdirSync(join(assetRoot, "versions", "1.1.0"), { recursive: true });
    mkdirSync(join(assetRoot, "versions", "2.0.0-draft.1"), { recursive: true });
    writeFileSync(join(assetRoot, "component.json"), JSON.stringify({
      libraryId: "react-component-library:Panel",
      latest: "1.1.0",
      draft: "2.0.0-draft.1",
      deprecatedVersions: [],
      evictedVersions: [],
    }));
    writeFileSync(join(assetRoot, "versions", "1.0.0", "Panel.tsx"), "export const Panel = () => null;\n");
    writeFileSync(join(assetRoot, "versions", "1.1.0", "Panel.tsx"), "export const Panel = () => null;\n");
    writeFileSync(join(assetRoot, "versions", "2.0.0-draft.1", "Panel.tsx"), "export const Panel = () => null;\n");

    const bare = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel", { libraryRoot: root });
    const major = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel/1", { libraryRoot: root });
    assert.equal(bare.version, "1.1.0");
    assert.equal(major.version, "1.1.0");
    await assert.rejects(
      resolveLibrarySpecifier("@vrooli/react-component-library/Panel/2", { libraryRoot: root }),
      /no active released version/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("an explicit deprecated major remains reproducible when no active release has that major", async () => {
  const root = mkdtempSync(join(tmpdir(), "rcl-resolve-history-"));
  const assetRoot = join(root, "components", "Panel");
  mkdirSync(join(assetRoot, "versions", "1.0.0"), { recursive: true });
  mkdirSync(join(assetRoot, "versions", "2.0.0"), { recursive: true });
  writeFileSync(join(assetRoot, "component.json"), JSON.stringify({ libraryId: "react-component-library:Panel", kind: "components", latest: "2.0.0", deprecatedVersions: ["1.0.0"] }));
  writeFileSync(join(assetRoot, "versions", "1.0.0", "Panel.tsx"), "export const Panel = () => null;\n");
  writeFileSync(join(assetRoot, "versions", "2.0.0", "Panel.tsx"), "export const Panel = () => null;\n");
  try {
    const historical = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel/1", { libraryRoot: root });
    assert.equal(historical.version, "1.0.0");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
