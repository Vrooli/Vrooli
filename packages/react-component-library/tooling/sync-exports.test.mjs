import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { test } from "node:test";
import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { tmpdir } from "node:os";
import { resolveCatalogExports } from "./export-resolution.mjs";

const scriptsRoot = dirname(fileURLToPath(import.meta.url));

test("sync-exports check rejects no live imports and reports the inverse count", () => {
  const output = execFileSync(process.execPath, [join(scriptsRoot, "sync-exports.mjs"), "--check"], {
    encoding: "utf8",
  });
  const report = JSON.parse(output.trim().split("\n").at(-1));
  assert.equal(report.checked, true);
  assert.equal(report.brokenVersionImports, 0);
  const packageJSON = JSON.parse(readFileSync(join(scriptsRoot, "..", "package.json"), "utf8"));
  assert.equal(report.versionedExports > 0, true);
  assert.equal(Boolean(packageJSON.exports["./Button"]), true);
  assert.equal(Boolean(packageJSON.exports["./Button/2"]), true);
  assert.equal(Boolean(packageJSON.exports["./Button/2.2.4"]), false);
  assert.equal(Boolean(packageJSON.exports["./Button/2/2.2.4"]), false);
});

test("every advertised subpath has a declaration in the built artifact", () => {
  const packageRoot = join(scriptsRoot, "..");
  const packageJSON = JSON.parse(readFileSync(join(packageRoot, "package.json"), "utf8"));
  for (const [subpath, target] of Object.entries(packageJSON.exports)) {
    if (subpath === ".") continue;
    assert.equal(
      existsSync(join(packageRoot, target.types)),
      true,
      `${subpath} advertises a missing declaration ${target.types}`,
    );
  }
});

function fixture(manifest, versions) {
  const root = mkdtempSync(join(tmpdir(), "rcl-export-resolution-"));
  const asset = join(root, "components", "Panel");
  mkdirSync(asset, { recursive: true });
  writeFileSync(join(asset, "component.json"), `${JSON.stringify({
    libraryId: "react-component-library:Panel",
    displayName: "Panel",
    draft: "",
    deprecatedVersions: [],
    ...manifest,
  }, null, 2)}\n`);
  for (const version of versions) {
    const versionRoot = join(asset, "versions", version);
    mkdirSync(versionRoot, { recursive: true });
    writeFileSync(join(versionRoot, "Panel.tsx"), `export const Panel = () => null;\n`);
  }
  return root;
}

test("deprecated highest directory does not control bare resolution", async () => {
  const root = fixture({ latest: "1.0.0", deprecatedVersions: ["1.1.0"], latestRationale: "1.1.0 is withdrawn" }, ["1.0.0", "1.1.0"]);
  try {
    const { resolutions } = await resolveCatalogExports({ libraryRoot: root });
    assert.equal(resolutions["./Panel"].version, "1.0.0");
    assert.equal(resolutions["./Panel/1"].version, "1.0.0");
    assert.equal(resolutions["./Panel/1.1.0"].version, "1.1.0", "exact reproduction aliases remain available");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("incomplete evicted retirement mirrors are ignored during export resolution", async () => {
  const root = fixture(
    { latest: "2.0.0", deprecatedVersions: ["1.0.0"], evictedVersions: ["1.0.0"] },
    ["1.0.0", "2.0.0"],
  );
  try {
    rmSync(join(root, "components", "Panel", "versions", "1.0.0", "Panel.tsx"));
    writeFileSync(join(root, "components", "Panel", "versions", "1.0.0", "retired-source.ts"), "export {};\n");
    const { resolutions } = await resolveCatalogExports({ libraryRoot: root });
    assert.equal(resolutions["./Panel/1.0.0"], undefined);
    assert.equal(resolutions["./Panel/1/1.0.0"], undefined);
    assert.equal(resolutions["./Panel"].version, "2.0.0");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("complete evicted source is durable history, not a package export", async () => {
  const root = fixture(
    { latest: "2.0.0", deprecatedVersions: ["1.0.0"], evictedVersions: ["1.0.0"] },
    ["1.0.0", "2.0.0"],
  );
  try {
    const { resolutions } = await resolveCatalogExports({ libraryRoot: root });
    assert.equal(resolutions["./Panel/1.0.0"], undefined);
    assert.equal(resolutions["./Panel/1"], undefined);
    assert.equal(resolutions["./Panel"].version, "2.0.0");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("two supported majors emit an alias for each major", async () => {
  const root = fixture({ latest: "2.1.0" }, ["1.4.0", "2.0.0", "2.1.0"]);
  try {
    const { resolutions } = await resolveCatalogExports({ libraryRoot: root });
    assert.equal(resolutions["./Panel/1"].version, "1.4.0");
    assert.equal(resolutions["./Panel/2"].version, "2.1.0");
    assert.equal(resolutions["./Panel"].version, "2.1.0");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("a manifest naming a missing latest version fails resolution", async () => {
  const root = fixture({ latest: "2.0.0" }, ["1.0.0"]);
  try {
    await assert.rejects(
      resolveCatalogExports({ libraryRoot: root }),
      /latest "2\.0\.0" does not name a released version on disk/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
