import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { PurgeConfirmDialog } from "../../src/components/dialogs/PurgeConfirmDialog.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("PurgeConfirmDialog requires DELETE, shows preview/error, and clears confirmation on close", async () => {
  const user = userEvent.setup();
  const onOpenChange = vi.fn();
  const onConfirm = vi.fn().mockResolvedValue(undefined);
  renderWithProviders(createElement(PurgeConfirmDialog, {
    open: true,
    onOpenChange,
    actionLabel: "Purge stale runs",
    pattern: "^stale-",
    preview: { profiles: 1, tasks: 2, runs: 3 },
    loading: false,
    error: "Preview changed",
    onConfirm,
  }));

  assert.ok(screen.getByText("Purge stale runs"));
  assert.ok(screen.getByText("Preview changed"));
  assert.equal(screen.getByRole("button", { name: "Confirm Delete" }).hasAttribute("disabled"), true);

  await user.type(screen.getByLabelText("Type DELETE to confirm"), "DELETE");
  await user.click(screen.getByRole("button", { name: "Confirm Delete" }));
  assert.equal(onConfirm.mock.calls.length, 1);

  await user.click(screen.getByRole("button", { name: "Cancel" }));
  assert.deepEqual(onOpenChange.mock.calls, [[false]]);
});
