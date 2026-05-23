import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ApplyConfirmDialog } from "./ApplyConfirmDialog";

afterEach(() => cleanup());

describe("ApplyConfirmDialog", () => {
  it("renders nothing when closed", () => {
    renderWithProviders(
      <ApplyConfirmDialog open={false} requiresNote={false} onClose={() => {}} onConfirm={() => {}} />,
    );
    expect(screen.queryByTestId(selectors.features.apply.confirmDialog.root)).toBeNull();
  });

  it("requires a note when requiresNote=true", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    renderWithProviders(
      <ApplyConfirmDialog open requiresNote onClose={() => {}} onConfirm={onConfirm} />,
    );
    await user.click(screen.getByTestId(selectors.features.apply.confirmDialog.confirmButton));
    expect(onConfirm).not.toHaveBeenCalled();
    // Note: confirm button is disabled until the note has content, so the
    // error sentinel renders only after user attempts submission with a
    // typed-then-cleared input. We assert that the click did not confirm.
  });

  it("calls onConfirm with the note when one is provided", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    renderWithProviders(
      <ApplyConfirmDialog open requiresNote onClose={() => {}} onConfirm={onConfirm} />,
    );
    const noteInput = screen.getByTestId(selectors.features.apply.confirmDialog.noteInput);
    await user.type(noteInput, "covered by audit ticket #42");
    await user.click(screen.getByTestId(selectors.features.apply.confirmDialog.confirmButton));
    expect(onConfirm).toHaveBeenCalledWith({ note: "covered by audit ticket #42" });
  });

  it("calls onClose when the cancel button is clicked", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(
      <ApplyConfirmDialog open requiresNote={false} onClose={onClose} onConfirm={() => {}} />,
    );
    await user.click(screen.getByTestId(selectors.features.apply.confirmDialog.cancelButton));
    expect(onClose).toHaveBeenCalled();
  });
});
