import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ContextAttachmentModal } from "../../src/components/ContextAttachmentModal.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

vi.mock("../../src/components/markdown/components/CodeBlock.js", () => ({ CodeBlock: ({ code }: { code: string }) => createElement("pre", null, code) }));

test("ContextAttachmentModal presents formatted and JSON evidence, copies it, and closes", async () => {
  const user = userEvent.setup();
  const onOpenChange = vi.fn();
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
  const attachment = { type: "link", key: "report", label: "Run report", path: "/tmp/report", url: "https://example.test/report", content: "details", tags: ["run", "evidence"] };
  const { rerender } = renderWithProviders(createElement(ContextAttachmentModal, { attachment, open: true, onOpenChange }));
  assert.equal(screen.getAllByText("Run report").length, 2);
  assert.ok(screen.getByText("https://example.test/report"));
  await user.click(screen.getByRole("button", { name: "Raw JSON" }));
  assert.ok(screen.getByText(/"report"/));
  await user.click(screen.getByRole("button", { name: "Copy JSON" }));
  assert.equal(writeText.mock.calls.length, 1);
  assert.ok(screen.getByRole("button", { name: "Copied" }));
  await user.click(screen.getAllByRole("button", { name: "Close" })[1]!);
  assert.deepEqual(onOpenChange.mock.calls, [[false]]);
  rerender(createElement(ContextAttachmentModal, { attachment: null, open: true, onOpenChange }));
  assert.equal(screen.queryByRole("dialog"), null);
});
