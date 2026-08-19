import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { assetStampMetadata, stampSource } from "./vite-plugin-asset-stamp.mjs";

const scenarioRoot = new URL("../..", import.meta.url).pathname.replace(/\/$/, "");

describe("RCL build asset stamp", () => {
  it("derives the catalog id and version from the entry path", () => {
    const metadata = assetStampMetadata(
      `${scenarioRoot}/library/components/Card/versions/1.1.0/Card.tsx`,
      scenarioRoot,
    );
    assert.equal(metadata?.asset, "primitives.card");
    assert.equal(metadata?.version, "1.1.0");
    assert.equal(metadata?.exempt, false);
  });

  it("removes authored markers and adds only the build-owned root marker", () => {
    const result = stampSource(
      `export function Exploit() { return <div data-rcl-asset="forged"><span data-rcl-asset="forged-child" /></div>; }`,
      { asset: "fixtures.marker-integrity", version: "1.0.0", componentName: "Exploit" },
    );
    assert.equal(result.changed, true);
    assert.match(result.code, /data-rcl-asset="fixtures\.marker-integrity"/);
    assert.match(result.code, /data-rcl-version="1\.0\.0"/);
    assert.match(result.code, /data-rcl-stamp="vite"/);
    assert.equal((result.code.match(/data-rcl-asset=/g) || []).length, 1);
    assert.doesNotMatch(result.code, /forged/);
  });

  it("stamps the owned element beneath a fragment or provider root", () => {
    const result = stampSource(
      `import { createContext } from "react"; const C = createContext(null); export function Exploit() { return <C.Provider value={null}><style>{""}</style><section data-rcl-asset="forged" /></C.Provider>; }`,
      { asset: "fixtures.provider-root", version: "1.0.0", componentName: "Exploit" },
    );
    assert.equal((result.code.match(/data-rcl-asset=/g) || []).length, 1);
    assert.match(result.code, /data-rcl-asset="fixtures\.provider-root"/);
    assert.doesNotMatch(result.code, /forged/);
  });

  it("stamps a dynamic createElement root", () => {
    const result = stampSource(
      `import { createElement } from "react"; export function Exploit({ Component }) { return createElement(Component, { className: "owned" }); }`,
      { asset: "fixtures.dynamic-root", version: "1.0.0", componentName: "Exploit" },
    );
    assert.match(result.code, /data-rcl-asset: "fixtures\.dynamic-root"/);
    assert.match(result.code, /data-rcl-version: "1\.0\.0"/);
  });

  it("does not let an earlier style helper hide the component root", () => {
    const result = stampSource(
      `import { createElement } from "react"; export function Exploit({ Component }) { createElement("style", {}); return createElement(Component, {}); }`,
      { asset: "fixtures.dynamic-after-helper", version: "1.0.0", componentName: "Exploit" },
    );
    assert.match(result.code, /data-rcl-asset: "fixtures\.dynamic-after-helper"/);
  });

  it("searches past conditional fragments for an owned JSX root", () => {
    const result = stampSource(
      `export function Exploit({ visible }) { if (!visible) return <>{null}</>; return <main data-rcl-asset="forged" />; }`,
      { asset: "fixtures.conditional-root", version: "1.0.0", componentName: "Exploit" },
    );
    assert.match(result.code, /data-rcl-asset="fixtures\.conditional-root"/);
    assert.doesNotMatch(result.code, /forged/);
  });

  it("does not pretend a clone-only asset owns a root", () => {
    const result = stampSource(
      `export function Exploit({ children, ...props }) { return cloneElement(children, props); }`,
      { asset: "fixtures.clone-only", version: "1.0.0", componentName: "Exploit" },
    );
    assert.equal(result.changed, false);
    assert.equal(result.reason, "no-owned-root");
  });
});
