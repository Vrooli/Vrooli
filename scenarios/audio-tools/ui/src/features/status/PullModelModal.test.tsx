import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { PullModelModal } from "./PullModelModal";
import { strings } from "../../consts/strings";

afterEach(() => {
  cleanup();
});

describe("PullModelModal", () => {
  it("returns null when closed", () => {
    renderWithProviders(
      <PullModelModal open={false} onClose={() => {}} onConfirm={() => {}} />,
    );
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("Pull button is disabled until a model name is typed", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <PullModelModal open onClose={() => {}} onConfirm={() => {}} />,
    );
    const pull = screen.getByRole("button", { name: strings.status.pullConfirm });
    expect(pull).toBeDisabled();
    await user.type(screen.getByLabelText(strings.status.pullFieldLabel), "phi3");
    expect(pull).toBeEnabled();
  });

  it("Pull forwards the trimmed name to onConfirm", async () => {
    const onConfirm = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <PullModelModal open onClose={() => {}} onConfirm={onConfirm} />,
    );
    await user.type(screen.getByLabelText(strings.status.pullFieldLabel), "  phi3:mini  ");
    await user.click(screen.getByRole("button", { name: strings.status.pullConfirm }));
    expect(onConfirm).toHaveBeenCalledWith("phi3:mini");
  });

  it("Cancel invokes onClose", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <PullModelModal open onClose={onClose} onConfirm={() => {}} />,
    );
    await user.click(screen.getByRole("button", { name: strings.status.pullCancel }));
    expect(onClose).toHaveBeenCalled();
  });

  it("when pending, button copy switches and is disabled", () => {
    renderWithProviders(
      <PullModelModal open pending onClose={() => {}} onConfirm={() => {}} />,
    );
    const btn = screen.getByRole("button", { name: strings.status.pullPulling });
    expect(btn).toBeDisabled();
  });
});
