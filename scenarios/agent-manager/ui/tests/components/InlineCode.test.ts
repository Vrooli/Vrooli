import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { createElement } from "react";
import { test } from "vitest";
import { InlineCode } from "../../src/components/markdown/components/InlineCode.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("InlineCode derives copyable text from strings, nested nodes, arrays, and empty values", () => {
  const { rerender } = renderWithProviders(createElement(InlineCode, { children: "run-123" }));
  assert.ok(screen.getByRole("button", { name: "Copy inline code" }));

  rerender(createElement(InlineCode, { children: ["run-", createElement("strong", { key: "id" }, "456")] }));
  assert.equal(screen.getByRole("button", { name: "Copy inline code" }).tagName, "BUTTON");

  rerender(createElement(InlineCode, { children: createElement("em", null, "nested") }));
  assert.ok(screen.getByRole("button", { name: "Copy inline code" }));

  rerender(createElement(InlineCode, { children: null }));
  assert.equal(screen.queryByRole("button", { name: "Copy inline code" }), null);
});
