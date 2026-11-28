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
  fs.writeFileSync(
    path.join(component, "versions", version, "FixturePanel.tsx"),
    `
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
  `,
  );
  return { root, library, version };
}
function run({ library }: { library: string }) {
  return spawnSync(
    process.execPath,
    [
      path.resolve(import.meta.dirname, "../../scripts/check-catalog-usage-contracts.mjs"),
      "--library-root",
      library,
      "--json",
    ],
    { encoding: "utf8" },
  );
}
afterEach(() =>
  tempDirs.splice(0).forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })),
);

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

    fs.writeFileSync(
      path.join(
        fixtureData.library,
        "components",
        "FixturePanel",
        "versions",
        fixtureData.version,
        "usage-contract.json",
      ),
      JSON.stringify({
        componentId: "react-component-library:FixturePanel",
        version: fixtureData.version,
        children: {
          typicalCount: 1,
          maximumCount: 4,
          allowedChildComponents: ["Text", "Button"],
          excludedChildComponents: ["AlertDialog"],
        },
        items: {
          property: "actions",
          typicalCount: 2,
          maximumCount: 3,
          requiredFields: ["id", "label"],
          maximumLabelCharacters: 24,
          primaryField: "primary",
          primaryCount: 1,
        },
        collections: [
          {
            property: "rows",
            typicalCount: 5,
            maximumCount: 20,
            stableIdField: "key",
            labelField: "title",
            maximumLabelCharacters: 64,
            mobileMaximumCount: 5,
            mobileField: "mobile",
            phoneLabelField: "shortTitle",
            primaryAction: "not-applicable",
          },
        ],
      }),
    );
    const passed = run(fixtureData);
    expect(passed.status).toBe(0);
    expect(JSON.parse(passed.stdout).errors).toEqual([]);

    const contractPath = path.join(
      fixtureData.library,
      "components",
      "FixturePanel",
      "versions",
      fixtureData.version,
      "usage-contract.json",
    );
    const invalid = JSON.parse(fs.readFileSync(contractPath, "utf8"));
    invalid.collections[0].mobileMaximumCount = 21;
    fs.writeFileSync(contractPath, JSON.stringify(invalid));
    const rejected = run(fixtureData);
    expect(rejected.status).toBe(1);
    expect(JSON.parse(rejected.stdout).errors).toContain(
      "FixturePanel.rows: mobileMaximumCount must be a non-negative integer no larger than maximumCount",
    );
  });

  it.each([
    { name: "AppShell", version: "2.1.8", contractTemplateVersion: "2.1.8" },
    { name: "AsyncBoundary", version: "1.0.12", contractTemplateVersion: "1.0.12" },
    { name: "BottomNav", version: "1.5.8", contractTemplateVersion: "1.5.8" },
    { name: "List", version: "1.2.0", contractTemplateVersion: "1.2.0" },
    { name: "AppNotificationSettings", version: "1.0.6", contractTemplateVersion: "1.0.7-draft.1" },
    { name: "page-section", version: "0.1.0", contractTemplateVersion: "0.1.1-draft.1" },
  ])(
    "fails closed on missing and wrong-version contracts for $name@$version",
    ({ name, version, contractTemplateVersion }) => {
      const libraryRoot = fs.mkdtempSync(path.join(os.tmpdir(), `rcl-${name}-identity-`));
      tempDirs.push(libraryRoot);
      const sourceLibrary = path.resolve(import.meta.dirname, "../../../library");
      const sourceVersion = path.join(sourceLibrary, "components", name, "versions", version);
      const componentRoot = path.join(libraryRoot, "components", name);
      const targetVersion = path.join(componentRoot, "versions", version);
      fs.mkdirSync(targetVersion, { recursive: true });
      fs.writeFileSync(
        path.join(componentRoot, "component.json"),
        JSON.stringify({ latest: version }),
      );
      const sourceFiles = fs
        .readdirSync(sourceVersion)
        .filter((file) => file.endsWith(".tsx") && !file.endsWith(".test.tsx"));
      const sourceName =
        sourceFiles.find((file) => file === `${name}.tsx`) ||
        sourceFiles.find((file) => file !== "story.tsx");
      if (!sourceName) throw new Error(`No TSX source found for ${name}@${version}`);
      fs.copyFileSync(path.join(sourceVersion, sourceName), path.join(targetVersion, sourceName));

      const contractTemplatePath = path.join(
        sourceLibrary,
        "components",
        name,
        "versions",
        contractTemplateVersion,
        "usage-contract.json",
      );
      const contractPath = path.join(targetVersion, "usage-contract.json");
      const missing = run({ library: libraryRoot });
      expect(missing.status).toBe(1);
      expect(JSON.parse(missing.stdout).errors).toContain(
        `${name}: missing usage-contract.json for latest ${version}`,
      );

      const contract = JSON.parse(fs.readFileSync(contractTemplatePath, "utf8"));
      fs.writeFileSync(contractPath, JSON.stringify({ ...contract, version }, null, 2));
      const exact = run({ library: libraryRoot });
      expect(exact.status).toBe(0);
      expect(JSON.parse(exact.stdout).errors).toEqual([]);

      fs.writeFileSync(
        contractPath,
        JSON.stringify({ ...contract, version: `${version}-draft.1` }, null, 2),
      );
      const wrongIdentity = run({ library: libraryRoot });
      expect(wrongIdentity.status).toBe(1);
      expect(JSON.parse(wrongIdentity.stdout).errors).toContain(
        `${name}: contract version must equal latest ${version}`,
      );
    },
  );
});
