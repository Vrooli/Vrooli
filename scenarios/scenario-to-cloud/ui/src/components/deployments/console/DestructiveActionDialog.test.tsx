import "@testing-library/jest-dom";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { DestructiveActionDialog } from "./DestructiveActionDialog";

// provider-free-exception: the dialog accepts all state via props.

function Host({ onConfirm, remote }: { onConfirm: () => void; remote: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div>
      <button type="button" onClick={() => setOpen(true)}>
        Open dialog
      </button>
      <span data-testid="remote">{remote}</span>
      {open && (
        <DestructiveActionDialog
          title="Restore?"
          description="Replaces data."
          target="machine:m-1"
          affectedData={["postgres:main"]}
          confirmText="abc12345"
          confirmLabel="Restore"
          isPending={false}
          onConfirm={onConfirm}
          onCancel={() => setOpen(false)}
        />
      )}
    </div>
  );
}

describe("DestructiveActionDialog", () => {
  // [REQ:STC-P0-039] Keyboard-only: focus lands in the dialog, Tab cycles inside it,
  // Escape closes it and focus returns to the opener.
  it("traps focus, closes on Escape and restores focus to the opener", async () => {
    const user = userEvent.setup();
    render(<Host onConfirm={vi.fn()} remote="v1" />);
    const opener = screen.getByRole("button", { name: "Open dialog" });
    opener.focus();
    await user.keyboard("{Enter}");
    const input = screen.getByTestId("console-destructive-confirm-input");
    expect(input).toHaveFocus();
    await user.tab();
    expect(screen.getByTestId("console-destructive-cancel")).toHaveFocus();
    await user.tab();
    // The confirm button is disabled until the text matches, so Tab wraps to the input.
    expect(input).toHaveFocus();
    await user.tab({ shift: true });
    expect(screen.getByTestId("console-destructive-cancel")).toHaveFocus();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(opener).toHaveFocus();
  });

  // [REQ:STC-P0-039] Typed confirmation is local state and survives a remote refresh.
  it("keeps the typed confirmation when the surrounding observation re-renders and confirms only on an exact match", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const { rerender } = render(<Host onConfirm={onConfirm} remote="v1" />);
    await user.click(screen.getByRole("button", { name: "Open dialog" }));
    await user.type(screen.getByTestId("console-destructive-confirm-input"), "abc1");
    rerender(<Host onConfirm={onConfirm} remote="v2" />);
    expect(screen.getByTestId("remote")).toHaveTextContent("v2");
    expect(screen.getByTestId("console-destructive-confirm-input")).toHaveValue("abc1");
    expect(screen.getByTestId("console-destructive-confirm")).toBeDisabled();
    await user.type(screen.getByTestId("console-destructive-confirm-input"), "2345");
    expect(screen.getByTestId("console-destructive-confirm")).toBeEnabled();
    await user.keyboard("{Enter}");
    expect(onConfirm).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("console-destructive-confirm"));
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it("names the target and data, lists extra details, and blocks closing while pending", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(
      <DestructiveActionDialog
        title="Apply?"
        description="Applies."
        target="host:203.0.113.10"
        affectedData={[]}
        details={[{ label: "Downtime", value: "5s" }]}
        confirmText="x"
        confirmLabel="Apply"
        isPending
        error="last attempt refused"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />,
    );
    expect(screen.getByRole("dialog")).toHaveAccessibleName("Apply?");
    expect(screen.getByRole("dialog")).toHaveAccessibleDescription("Applies.");
    expect(screen.getByTestId("console-destructive-target")).toHaveTextContent("host:203.0.113.10");
    expect(screen.getByTestId("console-destructive-data")).toHaveTextContent("No declared data bindings");
    expect(screen.getByText("5s")).toBeInTheDocument();
    expect(screen.getByTestId("console-destructive-error")).toHaveTextContent("last attempt refused");
    expect(screen.getByTestId("console-destructive-cancel")).toBeDisabled();
    await user.keyboard("{Escape}");
    expect(onCancel).not.toHaveBeenCalled();
  });
});
