import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { spawnSync } from "node:child_process";
import { test } from "node:test";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
test("structured TSX facts preserve computed overlay roles", (t) => {
  const directory = mkdtempSync(join(tmpdir(), "rcl-source-facts-"));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const source = join(directory, "SidebarShell.tsx");
  writeFileSync(source, `import React from "react";
import ResizeHandle from "@vrooli/react-component-library/ResizeHandle/1";
export function SidebarShell({mobile}) {
  useEscapeKey(); useResizablePanel(); assignRef();
  return <aside role={mobile ? "dialog" : "complementary"} style={{width: 240}}><ResizeHandle /></aside>;
}`);
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
  assert.equal(sidebar.inlineStyleElements, 1);
  assert.ok(sidebar.imports.includes("react"), sidebar.imports.join("\n"));
  assert.ok(sidebar.imports.includes("@vrooli/react-component-library/ResizeHandle/1"), sidebar.imports.join("\n"));
});
