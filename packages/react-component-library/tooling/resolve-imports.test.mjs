import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { test } from "node:test";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const source = join(
  packageRoot,
  "..",
  "..",
  "scenarios",
  "react-component-library",
  "library",
  "components",
  "SidebarShell",
  "versions",
  "2.6.3",
  "SidebarShell.tsx",
);

test("structured TSX facts preserve computed overlay roles", () => {
  assert.ok(existsSync(source), `fixture source missing: ${source}`);
  const result = spawnSync(process.execPath, [join(packageRoot, "tooling", "resolve-imports.mjs"), "--facts", source], {
    encoding: "utf8",
  });
  assert.equal(result.status, 0, result.stderr);
  const facts = JSON.parse(result.stdout);
  const roleValues = facts.flatMap((file) => file.attributes.role ?? []);
  assert.ok(roleValues.some((value) => value.includes('"dialog"')), roleValues.join("\n"));
  assert.ok(roleValues.some((value) => value.includes('"complementary"')), roleValues.join("\n"));
  const sidebar = facts.find((file) => file.file.endsWith("SidebarShell.tsx"));
  assert.ok(sidebar.exports.includes("SidebarShell"), sidebar.exports.join("\n"));
  assert.ok(sidebar.hookCalls.includes("useEscapeKey"), sidebar.hookCalls.join("\n"));
  assert.ok(sidebar.hookCalls.includes("useResizablePanel"), sidebar.hookCalls.join("\n"));
  assert.ok(sidebar.calls.includes("assignRef"), sidebar.calls.join("\n"));
  assert.equal(sidebar.inlineStyleElements, (readFileSync(source, "utf8").match(/style=\{\{/g) ?? []).length);
  assert.ok(sidebar.imports.includes("react"), sidebar.imports.join("\n"));
  assert.ok(sidebar.imports.includes("@vrooli/react-component-library/ResizeHandle/1"), sidebar.imports.join("\n"));
});
