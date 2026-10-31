import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { forwardedDOMBindings } from "./dom-bindings.mjs";

test("literal prop chains require a host attribute and respect overrides", async () => {
  const dir = mkdtempSync(join(tmpdir(), "rcl-bindings-"));
  try {
    const root = join(dir, "App.tsx");
    writeFileSync(join(dir, "Surface.tsx"), `export function Surface({surfaceId, as:Tag="section"}) { return <Tag data-experience-surface={surfaceId}/>; }
export function Dropped({surfaceId}) {return <div/>;}
export function Reassigned({surfaceId}) {surfaceId = "different"; return <div data-experience-surface={surfaceId}/>;}
export const Opaque = arbitraryWrapper(({surfaceId}) => <div data-experience-surface={surfaceId}/>);`);
    writeFileSync(join(dir, "Browser.tsx"), `import {Surface} from './Surface'; export function Browser({surfaceId}) { return <Surface surfaceId={surfaceId}/>; }`);
    writeFileSync(root, `import {Browser} from './Browser'; import {Surface,Dropped,Opaque,Reassigned} from './Surface';
export function App() { return <><Browser surfaceId="results"/><Dropped surfaceId="dropped"/><Reassigned surfaceId="reassigned"/><Opaque surfaceId="opaque"/><Surface surfaceId="overridden" {...unknown}/><Surface {...unknown} surfaceId="unknown-host"/><Surface {...unknown} as="section" surfaceId="explicit-host"/></>; }`);
    const found = (await forwardedDOMBindings([root])).get(root);
    assert.deepEqual(found.map(item => item.value).sort(), ["explicit-host", "results"]);
    assert.equal(found.find(item => item.value === "results").via.length, 3);
  } finally { rmSync(dir, { recursive: true, force: true }); }
});

test("catalog route reaches ExperienceSurface through its real source imports", async () => {
  const root = resolve("scenarios/react-component-library/ui/src/App.tsx");
  const found = (await forwardedDOMBindings([root])).get(root);
  const binding = found.find(item => item.attribute === "data-experience-surface" && item.value === "catalog-results");
  assert.ok(binding, JSON.stringify(found));
  assert.ok(binding.via.some(file => file.endsWith("CatalogBrowser.tsx")));
  assert.ok(binding.via.at(-1).endsWith("ExperienceSurface.tsx"));
});
