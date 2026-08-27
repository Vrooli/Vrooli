import assert from "node:assert/strict";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";

import { buildCensus, extractUtilityClasses } from "./overlay-census.mjs";

const fixtureRoot = resolve(fileURLToPath(new URL("./testdata/census-fixture", import.meta.url)));

test("detects utilities only from class-bearing expressions", () => {
  const hits = extractUtilityClasses(`
    const styles = \`[data-card] { display: grid; }\`;
    const classes = cn("md:inset-x-8", "bg-app-surface");
    export const Card = () => <div data-token="touch-target" className={classes} />;
  `);
  assert.deepEqual(hits.map((hit) => hit.class), ["bg-app-surface", "md:inset-x-8"]);
});

test("rejects malformed template fragments that merely contain utility punctuation", () => {
  const hits = extractUtilityClasses(`
    const classes = condition ? "md:inset-x-8" : ":";
    const interpolated = \`top-${"${offset}"}\`;
    export const Card = () => <div className={cn(classes, interpolated)} />;
  `);
  assert.deepEqual(hits.map((hit) => hit.class), ["md:inset-x-8"]);
});

test("census distinguishes purged classes and transitive token gaps", () => {
  const census = buildCensus(fixtureRoot, "2026-08-27T00:00:00.000Z");
  assert.equal(census.summary.scenarios_importing_rcl, 1);
  assert.equal(census.summary.library_files_emitting_utility_classes, 1);
  assert.equal(census.summary.purged_classes_total, 1);
  assert.equal(census.summary.tailwind_configs_covering_package, 0);
  assert.equal(census.summary.overlay_catalog_assets_without_implementation, 1);
  const consumer = census.consumers[0];
  assert.deepEqual(consumer.purge[0].purged_classes, ["md:inset-x-8"]);
  assert.deepEqual(consumer.transitive_token_gaps, [{ property: "--token-a", missing: "--token-missing" }]);
  assert.deepEqual(consumer.unsatisfied_tokens, []);
});
