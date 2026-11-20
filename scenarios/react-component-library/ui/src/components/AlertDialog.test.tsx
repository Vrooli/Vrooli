import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { AlertDialog } from "../../../library/components/AlertDialog/versions/2.0.13-draft.1/AlertDialog";

vi.mock("@vrooli/react-component-library/OverlayActionBar/1", async () =>
  await import("../../../library/components/OverlayActionBar/versions/1.0.1/OverlayActionBar"),
);
vi.mock("@vrooli/react-component-library/Portal/1", () => ({
  Portal: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@vrooli/react-component-library/StyleSheet/1", () => ({
  useLibraryStyleSheet: () => undefined,
  StyleSheet: () => null,
}));
vi.mock("@vrooli/react-component-library/useOverlaySurface/1", () => {
  // A real overlay hook returns a stable ref object. Creating a new one on
  // every render made the focus effect read a different object than the one
  // the surface wrote, so the dialog never received initial focus.
  const surfaceRef = { current: null as HTMLDivElement | null };
  return {
    useOverlaySurface: ({ open }: { open: boolean }) => ({
      present: open,
      state: open ? "open" : "closed",
      rootProps: { "data-overlay-root": true },
      surfaceRef,
    }),
  };
});

describe("AlertDialog 2.0.13 draft actions", () => {
  afterEach(() => vi.restoreAllMocks());

  it("keeps one row at phone width, shortens German labels, and focuses Cancel", async () => {
    renderWithProviders(
      <AlertDialog
        open
        title="Arbeitsbereich entfernen?"
        description="Diese Änderung kann nicht rückgängig gemacht werden."
        confirmLabel="Arbeitsbereich endgültig entfernen"
        cancelLabel="Änderung doch abbrechen"
        destructive
        onConfirm={() => undefined}
        onCancel={() => undefined}
      />,
    );

    const group = screen.getByRole("group", { name: "Confirmation actions" });
    const buttons = within(group).getAllByRole("button");
    expect(buttons).toHaveLength(2);
    const confirmButton = buttons[0];
    const cancel = buttons[1];
    if (!confirmButton || !cancel) throw new Error("the dialog must render confirm and cancel actions");
    expect(confirmButton).toHaveAttribute("data-action-id", "confirm");
    expect(cancel).toHaveAttribute("data-action-id", "cancel");
    expect(group).toHaveStyle({ flexWrap: "nowrap", flexDirection: "row-reverse" });
    expect(confirmButton).toHaveTextContent("…");
    expect(confirmButton).toHaveAttribute("title", "Arbeitsbereich endgültig entfernen");
    expect(confirmButton.parentElement?.parentElement).toHaveAttribute("data-destructive", "true");
    // Focus is scheduled in requestAnimationFrame so it lands after the
    // surface is committed. Wait for the frame instead of forcing rAF to run
    // synchronously inside the effect, which fires before refs attach.
    await waitFor(() => expect(cancel).toHaveFocus());
  });

  it("preserves explicit cancel and destructive confirm callbacks, and disables both while busy", () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    renderWithProviders(
      <AlertDialog
        open
        title="Delete?"
        description="This cannot be undone."
        destructive
        busy
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    const confirm = screen.getByRole("button", { name: "Working…" });
    const cancel = screen.getByRole("button", { name: "Cancel" });
    expect(confirm).toBeDisabled();
    expect(cancel).toBeDisabled();
    expect(confirm).toHaveAttribute("data-primary", "true");
    expect(confirm.parentElement?.parentElement).toHaveAttribute("data-destructive", "true");
    fireEvent.click(confirm);
    fireEvent.click(cancel);
    expect(onConfirm).not.toHaveBeenCalled();
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("holds both actions disabled while an async confirm is pending", async () => {
    let finishConfirm: (() => void) | undefined;
    const onConfirm = vi.fn(
      () => new Promise<void>((resolve) => { finishConfirm = resolve; }),
    );
    renderWithProviders(
      <AlertDialog
        open
        title="Publish?"
        description="Make the changes visible."
        confirmLabel="Publish"
        onConfirm={onConfirm}
        onCancel={() => undefined}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Publish" }));
    expect(screen.getByRole("button", { name: "Working…" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    await act(() => {
      finishConfirm?.();
      return Promise.resolve();
    });
    expect(screen.getByRole("button", { name: "Publish" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeEnabled();
  });

  it("runs confirm and cancel through their own action handlers", async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    const onCancel = vi.fn();
    renderWithProviders(
      <AlertDialog
        open
        title="Publish?"
        description="Make the changes visible."
        confirmLabel="Publish"
        cancelLabel="Keep editing"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    await act(() => {
      fireEvent.click(screen.getByRole("button", { name: "Publish" }));
      return Promise.resolve();
    });
    fireEvent.click(screen.getByRole("button", { name: "Keep editing" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
