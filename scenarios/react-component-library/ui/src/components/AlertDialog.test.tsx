import { act, fireEvent, render, screen, within } from "@testing-library/react";
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
vi.mock("@vrooli/react-component-library/useOverlaySurface/1", () => ({
  useOverlaySurface: ({ open }: { open: boolean }) => ({
    present: open,
    state: open ? "open" : "closed",
    rootProps: { "data-overlay-root": true },
    surfaceRef: { current: null },
  }),
}));

describe("AlertDialog 2.0.13 draft actions", () => {
  afterEach(() => vi.restoreAllMocks());

  it("keeps one row at phone width, shortens German labels, and focuses Cancel", () => {
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    vi.stubGlobal("cancelAnimationFrame", () => undefined);
    render(
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
    expect(buttons[0]).toHaveAttribute("data-action-id", "confirm");
    expect(buttons[1]).toHaveAttribute("data-action-id", "cancel");
    expect(group).toHaveStyle({ flexWrap: "nowrap", flexDirection: "row-reverse" });
    expect(buttons[0]).toHaveTextContent("…");
    expect(buttons[0]).toHaveAttribute("title", "Arbeitsbereich endgültig entfernen");
    expect(buttons[0].parentElement?.parentElement).toHaveAttribute("data-destructive", "true");
    expect(buttons[1]).toHaveFocus();
  });

  it("preserves explicit cancel and destructive confirm callbacks, and disables both while busy", () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    render(
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
    render(
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
    render(
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
