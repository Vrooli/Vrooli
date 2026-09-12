import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ResumeFromFailureModal } from "../../src/components/ResumeFromFailureModal.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";

test("ResumeFromFailureModal submits trimmed guidance and explains unavailable recovery", async () => {
  const onOpenChange = vi.fn();
  const onSubmit = vi.fn().mockResolvedValue(undefined);
  const failedRun = makeRun({ actions: { ...makeRun().actions, canResumeFromFailure: false, canResumeFromFailureReason: "The transcript has expired" } });
  const rendered = renderWithProviders(createElement(ResumeFromFailureModal, {
    open: true, onOpenChange, failedRun, onSubmit, error: "prior resume failed",
  }));
  assert.ok(screen.getByText("prior resume failed"));
  assert.ok(screen.getByText("The transcript has expired"));
  fireEvent.change(screen.getByLabelText("Additional Guidance (optional)"), { target: { value: "  finish the migration  " } });
  fireEvent.click(screen.getByRole("button", { name: "Resume Run" }));
  await waitFor(() => assert.deepEqual(onSubmit.mock.calls, [["finish the migration", undefined]]));
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  assert.deepEqual(onOpenChange.mock.calls.at(-1), [false]);

  rendered.rerender(createElement(ResumeFromFailureModal, { open: true, onOpenChange, failedRun: null, onSubmit, loading: true }));
  assert.equal((screen.getByRole("button", { name: "Resuming..." }) as HTMLButtonElement).disabled, true);
});
