import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ApplyInvestigationModal } from "../../src/components/ApplyInvestigationModal.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { RunStatus, StructuredResultStatus } from "../../src/types.js";
import { makeRun } from "../testutil/runs.js";

test("ApplyInvestigationModal distinguishes in-progress, terminal-unstructured, error, and view-run paths", async () => {
  const open = vi.fn(); const submit = vi.fn().mockResolvedValue(undefined); const view = vi.fn();
  const running = makeRun({ id: "investigation-1", status: RunStatus.RUNNING });
  const rendered = renderWithProviders(createElement(ApplyInvestigationModal, { open: true, onOpenChange: open, investigationRun: running, onSubmit: submit }));
  assert.ok(screen.getByText("Investigation in progress..."));
  assert.equal((screen.getByRole("button", { name: "Apply Investigation" }) as HTMLButtonElement).disabled, true);
  rendered.rerender(createElement(ApplyInvestigationModal, { open: true, onOpenChange: open, investigationRun: makeRun({ id: "investigation-1", status: RunStatus.COMPLETE }), onSubmit: submit, onViewRun: view, error: "apply failed" }));
  assert.ok(screen.getByText("No structured recommendations available"));
  assert.ok(screen.getByText("apply failed"));
  fireEvent.click(screen.getByRole("button", { name: "View investigation run" }));
  assert.deepEqual(view.mock.calls, [["investigation-1"]]);
  fireEvent.change(screen.getByLabelText("Additional Context for Apply Agent"), { target: { value: "  focus on receipts  " } });
  fireEvent.click(screen.getByRole("button", { name: "Apply Investigation" }));
  await waitFor(() => assert.deepEqual(submit.mock.calls, [[[], "focus on receipts", undefined]]));
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  assert.deepEqual(open.mock.calls.at(-1), [false]);
});

test("ApplyInvestigationModal defaults structured findings to selected and permits intentional narrowing", async () => {
  const submit = vi.fn().mockResolvedValue(undefined); const close = vi.fn();
  const investigation = makeRun({
    id: "investigation-structured", status: RunStatus.COMPLETE,
    result: { structured: { status: StructuredResultStatus.SUCCESS, value: { summary: "Two actionable fixes", primaryCategory: "Both", confidence: "High", categories: [
      { name: "Environment", recommendations: [{ text: "Repair CLI setup", severity: "Critical", evidence: "tool failed" }] },
      { name: "Agent", recommendations: [{ text: "Reduce retries", severity: "Major" }] },
    ] } } },
  });
  renderWithProviders(createElement(ApplyInvestigationModal, { open: true, onOpenChange: close, investigationRun: investigation, onSubmit: submit }));
  assert.ok(screen.getByText("Two actionable fixes")); assert.ok(screen.getByText(/Confidence:\s*High/)); assert.ok(screen.getByText("Critical"));
  assert.ok(screen.getByRole("button", { name: "Apply 2 Recommendations" }));
  fireEvent.click(screen.getByRole("button", { name: "Repair CLI setup" }));
  assert.ok(screen.getByRole("button", { name: "Apply 1 Recommendation" }));
  fireEvent.change(screen.getByLabelText("Additional Context for Apply Agent (optional)"), { target: { value: "  preserve receipts  " } });
  fireEvent.click(screen.getByRole("button", { name: "Apply 1 Recommendation" }));
  await waitFor(() => assert.deepEqual(submit.mock.calls, [[["Reduce retries"], "preserve receipts", undefined]]));
});

test("ApplyInvestigationModal can collapse categories and intentionally clear then restore all recommendations", async () => {
  const submit = vi.fn().mockResolvedValue(undefined);
  const investigation = makeRun({
    id: "investigation-selection", status: RunStatus.COMPLETE,
    result: { structured: { status: StructuredResultStatus.SUCCESS, value: {
      summary: "Selection controls are actionable", primaryCategory: "Agent",
      categories: [
        { name: "Agent", recommendations: [{ text: "Keep the useful receipt", severity: "Minor" }] },
        { name: "Environment", recommendations: [{ text: "Repair the local tool", severity: "Major" }] },
      ],
    } } },
  });
  renderWithProviders(createElement(ApplyInvestigationModal, {
    open: true, onOpenChange: vi.fn(), investigationRun: investigation, onSubmit: submit,
  }));

  assert.ok(screen.getByText("2 of 2 selected"));
  fireEvent.click(screen.getByRole("button", { name: /Environment/ }));
  assert.equal(screen.queryByRole("button", { name: "Repair the local tool" }), null);
  fireEvent.click(screen.getByRole("button", { name: /Environment/ }));
  assert.ok(screen.getByRole("button", { name: "Repair the local tool" }));

  const selectAll = screen.getAllByRole("checkbox")[0]!;
  fireEvent.click(selectAll);
  assert.ok(screen.getByText("0 of 2 selected"));
  assert.equal((screen.getByRole("button", { name: "Apply 0 Recommendations" }) as HTMLButtonElement).disabled, true);
  fireEvent.click(selectAll);
  await waitFor(() => assert.ok(screen.getByText("2 of 2 selected")));
  fireEvent.click(screen.getByRole("button", { name: "Apply 2 Recommendations" }));
  await waitFor(() => assert.deepEqual(submit.mock.calls, [[["Keep the useful receipt", "Repair the local tool"], "", undefined]]));
});
