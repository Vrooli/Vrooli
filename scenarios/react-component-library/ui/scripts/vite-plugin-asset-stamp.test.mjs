import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  adoptedAssetIndex,
  assetStampMetadata,
  buildStampReport,
  stampSource,
} from "./vite-plugin-asset-stamp.mjs";

const scenarioRoot = new URL("../..", import.meta.url).pathname.replace(/\/$/, "");
const fixtures = JSON.parse(
  readFileSync(`${scenarioRoot}/contracts/asset-stamp.fixtures.json`, "utf8"),
);

function countOccurrences(haystack, needle) {
  return haystack.split(needle).length - 1;
}

// The shared contract is the anti-divergence device. The Go preview injector
// runs the identical cases; a behaviour that only one implementation has is a
// behaviour the preview harness and the production build disagree about.
describe("shared asset-stamp contract", () => {
  for (const testCase of fixtures.cases) {
    it(testCase.name, () => {
      const result = stampSource(testCase.source, {
        asset: testCase.asset,
        version: testCase.version,
        componentName: testCase.componentName,
      });
      for (const fragment of testCase.mustContain || []) {
        assert.ok(
          result.code.includes(fragment),
          `expected output to contain ${JSON.stringify(fragment)}\ngot: ${result.code}`,
        );
      }
      for (const fragment of testCase.mustNotContain || []) {
        assert.ok(
          !result.code.includes(fragment),
          `expected output to omit ${JSON.stringify(fragment)}\ngot: ${result.code}`,
        );
      }
      if (typeof testCase.assetMarkerCount === "number") {
        assert.equal(
          countOccurrences(result.code, "data-rcl-asset"),
          testCase.assetMarkerCount,
          `wrong marker count in: ${result.code}`,
        );
      }
    });
  }

  it("emits syntactically valid output for every fixture", () => {
    for (const testCase of fixtures.cases) {
      const result = stampSource(testCase.source, {
        asset: testCase.asset,
        version: testCase.version,
        componentName: testCase.componentName,
      });
      // JSX cannot be evaluated here, but a dynamic-root case compiles to a
      // plain object literal and must parse. A hyphenated bare key does not.
      if (!result.code.includes("<")) {
        const body = result.code.replace(/^import[^;]*;/, "").replaceAll("export ", "");
        assert.doesNotThrow(() => new Function(body), `invalid output: ${result.code}`);
      }
    }
  });
});

describe("resolver chain", () => {
  it("resolves a library entry through component.json", () => {
    const metadata = assetStampMetadata(
      `${scenarioRoot}/library/components/Card/versions/1.1.0/Card.tsx`,
      scenarioRoot,
    );
    assert.equal(metadata?.asset, "primitives.card");
    assert.equal(metadata?.version, "1.1.0");
    assert.equal(metadata?.strategy, "library");
  });

  it("resolves an adopted copy through its re-export shim marker", () => {
    const adoptedIndex = adoptedAssetIndex(`${scenarioRoot}/ui/src/components`);
    const metadata = assetStampMetadata(
      `${scenarioRoot}/ui/src/components/ui/Button/versions/2.0.0/Button.tsx`,
      scenarioRoot,
      undefined,
      { adoptedIndex },
    );
    assert.equal(metadata?.asset, "controls.button");
    assert.equal(metadata?.version, "2.0.0");
    assert.equal(metadata?.strategy, "adopted");
  });

  it("indexes only re-export shims, never files that merely import an asset", () => {
    const adoptedIndex = adoptedAssetIndex(`${scenarioRoot}/ui/src/components`);
    // Sidebar.tsx imports AppNavigation and NavigationTree to compose its own
    // markup. Composition is bespoke authorship: it must not inherit either
    // asset's identity, or hand-written code would be scored as adopted.
    const composed = `${scenarioRoot}/ui/src/components/Sidebar.tsx`;
    assert.equal(adoptedIndex.has(composed), false);
  });

  it("gives every indexed adopted implementation a unique catalog identity", () => {
    const adoptedIndex = adoptedAssetIndex(`${scenarioRoot}/ui/src/components`);
    const seen = new Map();
    for (const entry of adoptedIndex.values()) {
      if (entry.asset.includes(":")) continue; // backlog: no catalog id yet
      seen.set(entry.asset, (seen.get(entry.asset) || 0) + 1);
    }
    const duplicates = [...seen].filter(([, count]) => count > 1);
    assert.deepEqual(duplicates, [], "two shims claim the same catalog asset");
  });
});

describe("exemption policy", () => {
  const exemptions = JSON.parse(
    readFileSync(new URL("./asset-stamp-exemptions.json", import.meta.url), "utf8"),
  );

  it("classifies every exemption as permanent or backlog", () => {
    for (const item of exemptions) {
      assert.ok(
        item.kind === "permanent" || item.kind === "backlog",
        `${item.asset} has kind ${JSON.stringify(item.kind)}`,
      );
      assert.ok(item.reason?.trim(), `${item.asset} has no reason`);
    }
  });

  it("keeps backlog reasons specific rather than boilerplate", () => {
    const backlog = exemptions.filter((item) => item.kind === "backlog");
    const reasons = new Set(backlog.map((item) => item.reason));
    assert.equal(
      reasons.size,
      backlog.length,
      "backlog entries share a copy-pasted reason; each needs its own catalog next step",
    );
  });
});

describe("stamp report", () => {
  it("separates unbundled assets from exempt ones", () => {
    const declared = new Map([
      ["a.one", { asset: "a.one", libraryId: null, version: "1.0.0", exempt: false, exemptKind: null, reason: null, strategy: "library" }],
      ["a.two", { asset: "a.two", libraryId: null, version: "1.0.0", exempt: true, exemptKind: "permanent", reason: "no DOM root", strategy: "library" }],
      ["a.three", { asset: "a.three", libraryId: null, version: "1.0.0", exempt: true, exemptKind: "backlog", reason: "not catalogued", strategy: "library" }],
    ]);
    const stamped = new Map([["a.one", { asset: "a.one", version: "1.0.0", strategy: "library" }]]);
    const report = buildStampReport({ declared, stamped, generatedAt: "2026-01-01T00:00:00.000Z" });
    assert.equal(report.totals.stamped, 1);
    assert.equal(report.totals["exempt-permanent"], 1);
    assert.equal(report.totals["exempt-backlog"], 1);
    assert.equal(report.totals.unbundled, 0);
  });

  it("reports a declared, non-exempt asset that no build reached as unbundled", () => {
    const declared = new Map([
      ["a.missed", { asset: "a.missed", libraryId: null, version: "2.0.0", exempt: false, exemptKind: null, reason: null, strategy: "library" }],
    ]);
    const report = buildStampReport({ declared, stamped: new Map(), generatedAt: "2026-01-01T00:00:00.000Z" });
    assert.equal(report.totals.unbundled, 1);
    assert.equal(report.assets[0].state, "unbundled");
  });
});
