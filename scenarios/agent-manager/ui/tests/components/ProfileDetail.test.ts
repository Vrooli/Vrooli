import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ProfileDetail } from "../../src/components/ProfileDetail.js";
import { SandboxMode } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("ProfileDetail renders operational configuration and dispatches edit/delete actions", async () => {
  const user = userEvent.setup();
  const onEdit = vi.fn();
  const onDelete = vi.fn();
  const profile = {
    id: "profile-1", name: "Investigator", description: "", roleRef: "read-only", profileKey: "investigate",
    maxTurns: 12, effort: "high", sandboxConfig: { mode: SandboxMode.PROTECTED, manualReview: true },
    networkAccess: 2, features: { enableBrowser: true }, extraFlags: { codex: { flags: ["--fast"] } },
    createdAt: undefined, updatedAt: undefined,
  } as any;
  renderWithProviders(createElement(ProfileDetail, { profile, onEdit, onDelete }));
  assert.ok(screen.getByText("No description provided"));
  assert.ok(screen.getByText("Manual Review"));
  assert.ok(screen.getByText("Browser"));
  assert.ok(screen.getByText("codex: --fast"));
  assert.ok(screen.getByText("12"));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  await user.click(screen.getByRole("button", { name: "Delete" }));
  assert.deepEqual(onEdit.mock.calls, [[profile]]);
  assert.deepEqual(onDelete.mock.calls, [["profile-1"]]);
});
