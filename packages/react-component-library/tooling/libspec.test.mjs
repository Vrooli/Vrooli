import test from "node:test";
import assert from "node:assert/strict";
import { parseLibrarySpecifier, parseLibrarySpecifiers } from "./libspec.mjs";

test("shared library grammar parses bare, major, and exact selectors", () => {
  assert.deepEqual(parseLibrarySpecifiers(`import "@vrooli/react-component-library/Button"; import "@vrooli/react-component-library/Panel/2"; import "@vrooli/react-component-library/Panel/2.1.0"`), [
    { name: "Button", selector: "" }, { name: "Panel", selector: "2" }, { name: "Panel", selector: "2.1.0" },
  ]);
  assert.equal(parseLibrarySpecifier("@vrooli/react-component-library/Panel/2.1"), null);
});
