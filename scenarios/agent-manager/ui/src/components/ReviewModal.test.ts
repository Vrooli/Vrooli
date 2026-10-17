import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ReviewModal } from "./ReviewModal.js";
import { ApprovalState } from "../types.js";
import { renderWithProviders } from "../test-utils/index.js";
import { makeRun } from "../test-utils/fixtures/runs.js";

// ReviewModal is now READ-ONLY: it shows the run summary + diff and points the
// operator at the Web Console approvals inbox. Approving/rejecting a run is no
// longer offered here (the inbox is the single operator surface), so the modal
// exposes no approve/reject controls.
test("ReviewModal shows a read-only review and points approvals to the inbox", async () => {
  const onOpenChange = vi.fn();
  const onOpenSandbox = vi.fn();
  const run = makeRun({
    changedFiles: 2,
    sandboxId: "sandbox-1234567890",
    approvalState: ApprovalState.PENDING,
    actions: { ...makeRun().actions, canApprove: true, canReject: true },
  });
  const rendered = renderWithProviders(createElement(ReviewModal, {
    open: true, onOpenChange, run, diff: null, diffLoading: false, onOpenSandbox,
  }));

  assert.ok(screen.getByText("2 changed files awaiting review"));
  assert.ok(screen.getByText("Pending Review"));
  assert.ok(screen.getByText("No diff available"));

  // Sandbox review (read-only) is still offered.
  fireEvent.click(screen.getByRole("button", { name: /Open in Sandbox/ }));
  assert.equal(onOpenSandbox.mock.calls.length, 1);

  // The inbox pointer is present; approve/reject controls are NOT.
  assert.ok(screen.getByText(/Web Console approvals inbox/));
  assert.equal(screen.queryByRole("button", { name: "Approve" }), null);
  assert.equal(screen.queryByRole("button", { name: "Reject" }), null);
  assert.equal(screen.queryByRole("button", { name: /Confirm Approval/ }), null);
  assert.equal(screen.queryByRole("button", { name: /Confirm Rejection/ }), null);

  // Loading state still renders.
  rendered.rerender(createElement(ReviewModal, {
    open: true, onOpenChange, run, diff: null, diffLoading: true, onOpenSandbox,
  }));
  assert.ok(screen.getByText("Loading diff..."));

  // Close button dismisses the modal. (Both the header "X" and the footer share
  // the accessible name "Close"; the footer button is the last one.)
  const closeButtons = screen.getAllByRole("button", { name: "Close" });
  fireEvent.click(closeButtons[closeButtons.length - 1]!);
  assert.deepEqual(onOpenChange.mock.calls.at(-1), [false]);
});
