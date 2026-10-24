import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ReviewModal } from "./ReviewModal.js";
import { ApprovalState, RunStatus } from "../types.js";
import type { RunDiff } from "../types.js";
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


test("deleted-sandbox review explains unavailable decisions without directing to the inbox", () => {
  const run = makeRun({
    approvalState: ApprovalState.PENDING,
    actions: { ...makeRun().actions, canReview: true, canApprove: false, canReject: false,
      reviewUnavailableReason: "The workspace sandbox was deleted before this decision was completed. Approve and reject are unavailable; the archived diff remains read-only." },
  });
  renderWithProviders(createElement(ReviewModal, {
    open: true, onOpenChange: vi.fn(), run, diff: null, diffLoading: false,
  }));
  assert.ok(screen.getByRole("status").textContent?.includes("sandbox was deleted"));
  assert.equal(screen.queryByText(/Web Console approvals inbox/), null);
  assert.ok(screen.getByText("No diff available"));
});

test("ReviewModal labels a pending approval count unavailable even outside needs-review status", () => {
  const run = makeRun({ status: RunStatus.COMPLETE, approvalState: ApprovalState.PENDING, changedFiles: 0 });
  const rendered = renderWithProviders(createElement(ReviewModal, {
    open: true, onOpenChange: vi.fn(), run, diff: null, diffLoading: false,
  }));
  assert.ok(screen.getByText("File count unavailable"));
  assert.ok(screen.getByText("Unavailable"));
  assert.equal(screen.queryByText("None"), null);
  rendered.rerender(createElement(ReviewModal, {
    open: true, onOpenChange: vi.fn(), run, diff: null, diffLoading: true,
  }));
  assert.ok(screen.getByText("File count unavailable"));
  assert.equal(screen.queryByText("None"), null);
});

test("ReviewModal does not treat an unverified empty diff response as zero files", () => {
  const run = makeRun({ status: RunStatus.NEEDS_REVIEW, approvalState: ApprovalState.PENDING, changedFiles: 0 });
  const diff = { runId: run.id, files: [], fileCountVerified: false } as RunDiff;
  renderWithProviders(createElement(ReviewModal, {
    open: true, onOpenChange: vi.fn(), run, diff, diffLoading: false,
  }));
  assert.ok(screen.getByText("File count unavailable"));
  assert.ok(screen.getByText("Unavailable"));
  assert.equal(screen.queryByText("None"), null);
  assert.ok(screen.getByText("Diff evidence is incomplete; changes and file count are unverified."));
});

test("ReviewModal ignores a verified diff from a different run", () => {
  const run = makeRun({ status: RunStatus.NEEDS_REVIEW, approvalState: ApprovalState.PENDING, changedFiles: 0 });
  const diff = { runId: "another-run", files: [], fileCountVerified: true } as RunDiff;
  renderWithProviders(createElement(ReviewModal, {
    open: true, onOpenChange: vi.fn(), run, diff, diffLoading: false,
  }));
  assert.ok(screen.getByText("File count unavailable"));
  assert.ok(screen.getByText("Unavailable"));
});

test("ReviewModal shows zero only for an explicitly verified empty diff", () => {
  const run = makeRun({ status: RunStatus.NEEDS_REVIEW, approvalState: ApprovalState.PENDING, changedFiles: 0 });
  const diff = { runId: run.id, files: [], fileCountVerified: true } as RunDiff;
  renderWithProviders(createElement(ReviewModal, {
    open: true, onOpenChange: vi.fn(), run, diff, diffLoading: false,
  }));
  assert.ok(screen.getByText("0 changed files in the verified diff"));
  assert.ok(screen.getByText("0", { selector: "div" }));
  assert.equal(screen.queryByText("File count unavailable"), null);
});
