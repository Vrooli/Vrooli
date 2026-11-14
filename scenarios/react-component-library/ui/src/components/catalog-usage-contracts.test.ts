import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { afterEach, describe, expect, it } from "vitest";

const tempDirs: string[] = [];
function fixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "rcl-contract-catalog-"));
  tempDirs.push(root);
  const library = path.join(root, "library");
  const component = path.join(library, "components", "FixturePanel");
  const version = "1.2.3";
  fs.mkdirSync(path.join(component, "versions", version), { recursive: true });
  fs.writeFileSync(path.join(component, "component.json"), JSON.stringify({ latest: version }));
  fs.writeFileSync(path.join(component, "versions", version, "FixturePanel.tsx"), `
    type Action = { id: string; label: string; onSelect: () => void };
    type ActionList = readonly [Action, ...Action[]];
    type Row = { key: string; title: string };
    export interface FixturePanelProps {
      children?: React.ReactNode;
      actions: ActionList;
      rows: ReadonlyArray<Row>;
      searchFields?: string[];
      defaultExpandedIds?: string[];
    }
  `);
  return { root, library, version };
}
function run({ library }: { library: string }) {
  return spawnSync(process.execPath, [
    path.resolve(import.meta.dirname, "../../scripts/check-catalog-usage-contracts.mjs"),
    "--library-root", library,
    "--json",
  ], { encoding: "utf8" });
}
afterEach(() => tempDirs.splice(0).forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })));

describe("catalog usage-contract coverage", () => {
  it("fails with actionable missing-contract findings, then accepts a complete contract", () => {
    const fixtureData = fixture();
    const failed = run(fixtureData);
    expect(failed.status).toBe(1);
    expect(JSON.parse(failed.stdout).eligible[0].shapes).toEqual([
      { prop: "children", kind: "children" },
      { prop: "actions", kind: "actions" },
      { prop: "rows", kind: "items" },
    ]);
    expect(JSON.parse(failed.stdout).errors[0]).toContain("missing usage-contract.json");

    fs.writeFileSync(path.join(fixtureData.library, "components", "FixturePanel", "versions", fixtureData.version, "usage-contract.json"), JSON.stringify({
      componentId: "react-component-library:FixturePanel",
      version: fixtureData.version,
      children: { typicalCount: 1, maximumCount: 4, allowedChildComponents: ["Text", "Button"] },
      items: { property: "actions", typicalCount: 2, maximumCount: 3, requiredFields: ["id", "label"], maximumLabelCharacters: 24, primaryField: "primary", primaryCount: 1 },
      collections: [{ property: "rows", typicalCount: 5, maximumCount: 20, stableIdField: "key", labelField: "title", maximumLabelCharacters: 64, primaryAction: "not-applicable" }],
    }));
    const passed = run(fixtureData);
    expect(passed.status).toBe(0);
    expect(JSON.parse(passed.stdout).errors).toEqual([]);
  });
});
