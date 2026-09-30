import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { InvestigateModal } from "../../src/components/InvestigateModal.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("InvestigateModal submits depth, selected context, scope defaults, and accumulated focus guidance", async () => {
  const submit = vi.fn().mockResolvedValue(undefined); const close = vi.fn();
  renderWithProviders(createElement(InvestigateModal, {
    open: true, onOpenChange: close, title: "Investigate run", description: "inspect evidence", confirmLabel: "Start Investigation",
    defaultProjectRoot: "/repo", defaultScopePaths: ["api", "cli"], onSubmit: submit, error: "prior failure",
  }));
  assert.ok(screen.getByText("prior failure"));
  fireEvent.click(screen.getByRole("radio", { name: /deep/i }));
  fireEvent.click(screen.getByText("Context to Include"));
  fireEvent.click(screen.getByText("Full logs"));
  fireEvent.click(screen.getByRole("button", { name: /Agent crashed/ }));
  fireEvent.click(screen.getByRole("button", { name: /Slow \/ excessive tokens/ }));
  fireEvent.click(screen.getByRole("button", { name: "Start Investigation" }));
  await waitFor(() => assert.equal(submit.mock.calls.length, 1));
  const [context, depth, flags, root, paths] = submit.mock.calls[0]!;
  assert.equal(depth, "deep"); assert.equal(root, "/repo"); assert.deepEqual(paths, ["api", "cli"]);
  assert.equal(flags.fullLogs, true);
  assert.match(context, /agent crashed/i); assert.match(context, /excessive tokens/i);
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  assert.deepEqual(close.mock.calls.at(-1), [false]);
});
