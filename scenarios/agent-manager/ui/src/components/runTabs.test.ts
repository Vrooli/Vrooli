import assert from "node:assert/strict";
import { test } from "vitest";
import { isRunTab, RUN_TABS } from "./runTabs";

test("every run detail tab, including Usage and Report, is deep-linkable and nothing else is", () => {
  assert.deepEqual(RUN_TABS.map((tab) => tab.id), ["task", "timeline", "diff", "usage", "report"]);
  for (const tab of RUN_TABS) assert.equal(isRunTab(tab.id), true);
  for (const other of ["cost", "", null, "Usage"]) assert.equal(isRunTab(other), false);
});
