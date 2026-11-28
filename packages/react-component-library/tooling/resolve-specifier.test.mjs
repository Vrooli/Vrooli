import {
  mkdirSync,
  mkdtempSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { createRequire } from "node:module";
import { join } from "node:path";
import { tmpdir } from "node:os";
import test from "node:test";
import assert from "node:assert/strict";
import { resolveLibrarySpecifier, sourceLibraryResolver } from "./resolve-specifier.mjs";

test("bare and major imports resolve the latest active release, never a draft", async () => {
  const root = mkdtempSync(join(tmpdir(), "rcl-source-resolver-"));
  const assetRoot = join(root, "components", "Panel");
  try {
    mkdirSync(join(assetRoot, "versions", "1.0.0"), { recursive: true });
    mkdirSync(join(assetRoot, "versions", "1.1.0"), { recursive: true });
    mkdirSync(join(assetRoot, "versions", "1.2.0-draft.1"), { recursive: true });
    writeFileSync(join(assetRoot, "component.json"), JSON.stringify({
      libraryId: "react-component-library:Panel",
      latest: "1.1.0",
      draft: "1.2.0-draft.1",
      deprecatedVersions: [],
      evictedVersions: [],
    }));
    writeFileSync(join(assetRoot, "versions", "1.0.0", "Panel.tsx"), "export const Panel = () => null;\n");
    writeFileSync(join(assetRoot, "versions", "1.1.0", "Panel.tsx"), "export const Panel = () => null;\n");
    writeFileSync(join(assetRoot, "versions", "1.2.0-draft.1", "Panel.tsx"), "export const Panel = () => null;\n");

    const bare = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel", { libraryRoot: root });
    const major = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel/1", { libraryRoot: root });
    assert.equal(bare.version, "1.1.0");
    assert.equal(major.version, "1.1.0");
    const draft = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel/1", {
      libraryRoot: root,
      draftOverrides: { Panel: "1.2.0-draft.1" },
    });
    assert.equal(draft.version, "1.2.0-draft.1");
    assert.equal(draft.draft, true);
    assert.equal(draft.sourcePath, join(assetRoot, "versions", "1.2.0-draft.1", "Panel.tsx"));
    const exactRelease = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel/1.1.0", {
      libraryRoot: root,
      draftOverrides: { Panel: "1.2.0-draft.1" },
    });
    assert.equal(exactRelease.version, "1.1.0");
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
    writeFileSync(join(assetRoot, "component.json"), JSON.stringify({ libraryId: "react-component-library:Panel", latest: "2.0.0", retiredMajorAliases: ["1"] }));
    await assert.rejects(resolveLibrarySpecifier("@vrooli/react-component-library/Panel/1", { libraryRoot: root }), /major alias is retired/);
    const exact = await resolveLibrarySpecifier("@vrooli/react-component-library/Panel/1.0.0", { libraryRoot: root });
    assert.equal(exact.version, "1.0.0");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("versioned usage-contract wrappers are served in DEV and omitted from production builds", async () => {
  const root = mkdtempSync(join(tmpdir(), "rcl-runtime-contract-"));
  const assetRoot = join(root, "components", "Panel");
  const versionRoot = join(assetRoot, "versions", "1.0.0");
  mkdirSync(versionRoot, { recursive: true });
  writeFileSync(join(assetRoot, "component.json"), JSON.stringify({
    libraryId: "react-component-library:Panel",
    latest: "1.0.0",
    deprecatedVersions: [],
    evictedVersions: [],
  }));
  writeFileSync(join(versionRoot, "Panel.tsx"), "export const Panel = () => null;\n");
  writeFileSync(join(versionRoot, "usage-contract.json"), JSON.stringify({
    componentId: "react-component-library:Panel",
    version: "1.0.0",
    children: { maximumCount: 2 },
  }));
  try {
    const development = sourceLibraryResolver({ libraryRoot: root });
    development.configResolved({ command: "serve" });
    const virtualId = await development.resolveId("@vrooli/react-component-library/Panel/1");
    assert.match(virtualId, /^\0rcl-usage-contract:/);
    const wrapper = development.load(virtualId);
    assert.match(wrapper, /wrapUsageContract/);
    assert.match(wrapper, /"version":"1\.0\.0"/);
    assert.match(wrapper, /__rcl_usage_contract_original/);

    const production = sourceLibraryResolver({ libraryRoot: root });
    production.configResolved({ command: "build" });
    const sourcePath = await production.resolveId("@vrooli/react-component-library/Panel/1");
    assert.equal(sourcePath, join(versionRoot, "Panel.tsx"));
    assert.equal(production.load(sourcePath), null);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("serve-only resolution routes RCL source dependencies through the consumer before the shared package fallback", async () => {
  const root = mkdtempSync(join(tmpdir(), "rcl-runtime-dependencies-"));
  const assetRoot = join(root, "components", "Panel");
  const versionRoot = join(assetRoot, "versions", "1.0.0");
  mkdirSync(versionRoot, { recursive: true });
  writeFileSync(join(assetRoot, "component.json"), JSON.stringify({
    libraryId: "react-component-library:Panel",
    latest: "1.0.0",
    deprecatedVersions: [],
    evictedVersions: [],
  }));
  writeFileSync(join(versionRoot, "Panel.tsx"), "export const Panel = () => null;\n");
  writeFileSync(join(versionRoot, "usage-contract.json"), JSON.stringify({
    componentId: "react-component-library:Panel",
    version: "1.0.0",
    children: { maximumCount: 2 },
  }));
  try {
    const resolver = sourceLibraryResolver({ libraryRoot: root, devOnly: true });
    assert.equal(resolver.apply, "serve");
    resolver.configResolved({ command: "serve", mode: "development", root: join(root, "consumer") });
    let request;
    const context = {
      async resolve(id, importer, options) {
        request = { id, importer, options };
        return { id: join(root, "consumer", "node_modules", id, "index.js") };
      },
    };
    const resolved = await resolver.resolveId.call(context, "lucide-react", join(versionRoot, "Panel.tsx"));
    assert.equal(resolved, join(root, "consumer", "node_modules", "lucide-react", "index.js"));
    assert.equal(request.id, "lucide-react");
    assert.equal(request.importer, join(root, "consumer", "__rcl-usage-contract-dependency__.mjs"));
    assert.deepEqual(request.options, { skipSelf: true });

    const testMode = sourceLibraryResolver({ libraryRoot: root, devOnly: true });
    testMode.configResolved({ command: "serve", mode: "test", root: join(root, "consumer") });
    assert.equal(await testMode.resolveId("@vrooli/react-component-library/Panel/1"), join(versionRoot, "Panel.tsx"));

    const packageContext = {
      async resolve() { return null; },
    };
    const packageFallback = await resolver.resolveId.call(packageContext, "lucide-react", join(versionRoot, "Panel.tsx"));
    assert.equal(realpathSync(packageFallback), realpathSync(createRequire(import.meta.url).resolve("lucide-react")));
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
