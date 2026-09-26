import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ReviewModal } from "../../src/components/ReviewModal.js";
import { ApprovalState } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";

test("ReviewModal submits trimmed approvals, rejects, and exposes the sandbox review context", async () => {
  const onOpenChange = vi.fn();
  const onApprove = vi.fn().mockResolvedValue(undefined);
  const onReject = vi.fn().mockResolvedValue(undefined);
  const onOpenSandbox = vi.fn();
  const run = makeRun({
    changedFiles: 2,
    sandboxId: "sandbox-1234567890",
    approvalState: ApprovalState.PENDING,
    actions: { ...makeRun().actions, canApprove: true, canReject: true },
  });
  const rendered = renderWithProviders(createElement(ReviewModal, {
    open: true, onOpenChange, run, diff: null, diffLoading: false, onApprove, onReject, onOpenSandbox,
  }));

  assert.ok(screen.getByText("2 changed files awaiting review"));
  assert.ok(screen.getByText("Pending Review"));
  assert.ok(screen.getByText("No diff available"));
  fireEvent.click(screen.getByRole("button", { name: /Open in Sandbox/ }));
  assert.equal(onOpenSandbox.mock.calls.length, 1);

  fireEvent.click(screen.getByRole("button", { name: "Approve" }));
  fireEvent.change(screen.getByLabelText("Your Name (optional)"), { target: { value: "  reviewer  " } });
  fireEvent.change(screen.getByLabelText("Commit Message (optional)"), { target: { value: "ship it" } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm Approval" }));
  await waitFor(() => assert.deepEqual(onApprove.mock.calls, [[{ actor: "reviewer", commitMsg: "ship it" }]]));
  assert.deepEqual(onOpenChange.mock.calls.at(-1), [false]);

  rendered.rerender(createElement(ReviewModal, {
    open: false, onOpenChange, run, diff: null, diffLoading: true, onApprove, onReject,
  }));
  rendered.rerender(createElement(ReviewModal, {
    open: true, onOpenChange, run, diff: null, diffLoading: true, onApprove, onReject,
  }));
  assert.ok(screen.getByText("Loading diff..."));
  fireEvent.click(screen.getByRole("button", { name: "Reject" }));
  fireEvent.change(screen.getByLabelText("Your Name (optional)"), { target: { value: "  operator " } });
  fireEvent.change(screen.getByLabelText("Rejection Reason"), { target: { value: "needs tests" } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm Rejection" }));
  await waitFor(() => assert.deepEqual(onReject.mock.calls, [[{ actor: "operator", reason: "needs tests" }]]));
});
