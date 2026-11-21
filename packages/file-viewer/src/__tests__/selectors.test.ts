import { describe, expect, it } from "vitest";

import { fileViewerSelectors, selectors } from "../selectors.js";

// The viewer's test ids are public test API: Web Console's browser-automation
// workflows and its generated selector manifest depend on these exact values.
describe("viewer selectors", () => {
  it("keeps every test id stable", () => {
    expect(fileViewerSelectors).toMatchSnapshot();
  });

  it("gives every test id once, so a selector never matches two controls", () => {
    const ids: string[] = [];
    const collect = (node: object) => { for (const value of Object.values(node)) { if (typeof value === "string") ids.push(value); else collect(value as object); } };
    collect(fileViewerSelectors);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("is the registry the components read", () => {
    expect(selectors).toBe(fileViewerSelectors);
  });
});
