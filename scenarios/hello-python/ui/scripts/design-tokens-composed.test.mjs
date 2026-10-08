import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const uiRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

// Walk up to the repository root so this works from the template tree
// (templates/scenarios/react-vite/ui) and from a generated scenario
// (scenarios/<id>/ui) alike; skip when the design kits are not present.
function findDesignRoot(start) {
  let dir = start;
  for (let depth = 0; depth < 8; depth += 1) {
    const candidate = path.join(dir, "templates", "design");
    if (fs.existsSync(path.join(candidate, "_base", "tokens.css"))) return candidate;
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return null;
}
const designRoot = findDesignRoot(uiRoot);
const MANAGED_BEGIN = "/* rcl:tokens:begin */";
const MANAGED_END = "/* rcl:tokens:end */";

// `template-manager lifecycle generate` writes `_base/tokens.css`, a newline,
// then the selected kit's adapter `tokens.css` into ui/src/design-tokens.css.
// The template keeps a copy of that output for the default kit so static
// readers can inspect the ramp; this test keeps the copy from drifting.
test("template design-tokens.css equals the composed default-kit output", { skip: designRoot === null && "templates/design not found" }, () => {
  const base = fs.readFileSync(path.join(designRoot, "_base", "tokens.css"), "utf8");
  const adapter = fs.readFileSync(path.join(designRoot, "vrooli-default", "adapters", "react-vite-tailwind", "tokens.css"), "utf8");
  const actual = withoutManagedRegion(fs.readFileSync(path.join(uiRoot, "src", "design-tokens.css"), "utf8"));
  assert.equal(actual, `${base}\n${adapter}`, "ui/src/design-tokens.css must be regenerated from templates/design (base + vrooli-default adapter)");
});

// `react-component-library adoptions link` owns one region of the ramp file,
// declared in ui/manifest.json as files.designTokens.managedRegion. Everything
// outside that region must still be the composed kit output.
function withoutManagedRegion(css) {
  const begin = css.indexOf(MANAGED_BEGIN);
  if (begin < 0) return css;
  const end = css.indexOf(MANAGED_END, begin);
  if (end < 0) throw new Error(`design-tokens.css opens ${MANAGED_BEGIN} without ${MANAGED_END}`);
  return css.slice(0, begin) + css.slice(end + MANAGED_END.length);
}

test("template declares the files the component library manages", () => {
  const manifest = JSON.parse(fs.readFileSync(path.join(uiRoot, "manifest.json"), "utf8"));
  for (const key of ["designTokens", "localeCatalogue", "selectorRegistry", "librarySelectors", "appEntry"]) {
    assert.ok(manifest.files?.[key]?.path, `ui/manifest.json files.${key}.path is required`);
  }
  assert.equal(manifest.files.designTokens.managedRegion.begin, MANAGED_BEGIN);
  assert.equal(manifest.files.designTokens.managedRegion.end, MANAGED_END);
});
