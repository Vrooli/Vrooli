import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";

import { DirtyStateGuard, type DirtyStateGuardHandle } from "./DirtyStateGuard.tsx";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

describe("DirtyStateGuard", () => {
  it("opens a keyboard-operable prompt and completes each operator choice", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    const onLeave = vi.fn();
    const onDiscard = vi.fn();
    const onSave = vi.fn().mockResolvedValue(undefined);
    const ref = createRef<DirtyStateGuardHandle>();

    renderWithProviders(
      <DirtyStateGuard
        ref={ref}
        isDirty
        defaultOpen
        title="Unsaved offer"
        description="The offer has not been recorded."
        note="Choose what to do."
        onAction={onAction}
        onLeave={onLeave}
        onDiscard={onDiscard}
        onSave={onSave}
      >
        <span>editor</span>
      </DirtyStateGuard>,
      { withoutRouter: true },
    );

    expect(screen.getByRole("alertdialog", { name: /Unsaved offer/ })).toBeVisible();
    await user.click(screen.getByTestId("dirty-state-guard-continue"));
    expect(onAction).toHaveBeenCalledWith("continue");
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();

    act(() => {
      expect(ref.current?.requestLeave()).toBe(false);
    });
    await user.click(screen.getByTestId("dirty-state-guard-discard"));
    expect(onDiscard).toHaveBeenCalledOnce();
    expect(onLeave).toHaveBeenCalledOnce();

    act(() => {
      expect(ref.current?.requestLeave()).toBe(false);
    });
    await user.click(screen.getByTestId("dirty-state-guard-save"));
    await screen.findByText(/editor/);
    expect(onSave).toHaveBeenCalledOnce();
    expect(onAction).toHaveBeenCalledWith("save");
    expect(onLeave).toHaveBeenCalledTimes(2);
  });

  it("handles Escape, protected unload, and a clean leave", () => {
    const onAction = vi.fn();
    const onLeave = vi.fn();
    const ref = createRef<DirtyStateGuardHandle>();

    renderWithProviders(
      <DirtyStateGuard ref={ref} isDirty defaultOpen onAction={onAction} onLeave={onLeave}>
        <span>editor</span>
      </DirtyStateGuard>,
      { withoutRouter: true },
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onAction).toHaveBeenCalledWith("continue");

    const beforeUnload = new Event("beforeunload", { cancelable: true });
    window.dispatchEvent(beforeUnload);
    expect(beforeUnload.defaultPrevented).toBe(true);

    const cleanRef = createRef<DirtyStateGuardHandle>();
    renderWithProviders(
      <DirtyStateGuard ref={cleanRef} isDirty={false} onLeave={onLeave}>
        <span>clean editor</span>
      </DirtyStateGuard>,
      { withoutRouter: true },
    );
    act(() => {
      expect(cleanRef.current?.requestLeave()).toBe(true);
    });
    expect(onLeave).toHaveBeenCalledOnce();
  });
});
