import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { MarkdownRenderer } from "./index.js";
import { renderWithProviders } from "../../test-utils/index.js";

test("MarkdownRenderer renders rich operational markdown safely and copies inline code", async () => {
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
  const content = [
    "# Finding", "", "**Important** and *useful* with ~~stale~~ `run-123`.", "",
    "> Inspect the report before retrying.", "", "- first", "- second", "", "1. ordered", "2. evidence", "",
    "[Open report](https://example.test/report)", "", "---", "", "| Key | Value |", "| --- | --- |", "| status | failed |",
  ].join("\n");
  const { container } = renderWithProviders(createElement(MarkdownRenderer, { content, className: "report-markdown" }));
  assert.equal(container.querySelector("h1")?.textContent, "Finding");
  assert.equal(container.querySelector("blockquote")?.textContent?.trim(), "Inspect the report before retrying.");
  assert.equal(container.querySelectorAll("li").length, 4);
  assert.equal(screen.getByRole("link", { name: "Open report" }).getAttribute("target"), "_blank");
  assert.equal(container.querySelectorAll("th").length, 2);
  fireEvent.click(screen.getByRole("button", { name: "Copy inline code" }));
  await waitFor(() => assert.deepEqual(writeText.mock.calls, [["run-123"]]));
  assert.ok(screen.getByRole("button", { name: "Copied" }));
});

test("MarkdownRenderer omits empty content and normalizes non-string content at its boundary", () => {
  const { container, rerender } = renderWithProviders(createElement(MarkdownRenderer, { content: "" }));
  assert.equal(container.innerHTML, "");
  rerender(createElement(MarkdownRenderer, { content: 42 as unknown as string }));
  assert.ok(screen.getByText("42"));
});


test("MarkdownRenderer updates streaming content and stable display classes while retaining code blocks", () => {
  const { container, rerender } = renderWithProviders(createElement(MarkdownRenderer, { content: "## Two\n### Three\n#### Four\n\n```ts\nconst value = 1;\n```", isStreaming: true }));
  assert.equal(container.querySelectorAll("h2,h3,h4").length, 3);
  assert.ok(container.textContent?.includes("const value = 1;"));
  rerender(createElement(MarkdownRenderer, { content: "first", isStreaming: true }));
  assert.ok(screen.getByText("first"));
  rerender(createElement(MarkdownRenderer, { content: "other", isStreaming: true }));
  assert.ok(screen.getByText("other"));
  rerender(createElement(MarkdownRenderer, { content: "other", isStreaming: true }));
  assert.ok(screen.getByText("other"));
  rerender(createElement(MarkdownRenderer, { content: "settled", className: "settled" }));
  rerender(createElement(MarkdownRenderer, { content: "other", className: "expanded" }));
  assert.equal(container.querySelector(".markdown-content")?.className, "markdown-content expanded");
  rerender(createElement(MarkdownRenderer, { content: "other", className: "expanded" }));
  assert.ok(screen.getByText("other"));
});
