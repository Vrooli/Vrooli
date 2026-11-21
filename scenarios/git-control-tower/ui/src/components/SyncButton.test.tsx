import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../test-utils/renderWithProviders";
import { SyncButton } from "./SyncButton";

const baseProps = {
  ahead: 3,
  behind: 0,
  canPush: true,
  canPull: true,
  onPush: vi.fn(),
  onPull: vi.fn(),
  isPushing: false,
  isPulling: false
};

describe("SyncButton", () => {
  it("marks the trigger as active with the running phase while a push runs", () => {
    render(<SyncButton {...baseProps} isPushing progressLabel="Pushing 1m 12s" />);

    expect(screen.getByTestId("sync-button-trigger")).toHaveAttribute("title", "Pushing 1m 12s");
  });

  it("does not auto-open the popover during a push — the sync snackbar carries progress", () => {
    const { rerender } = render(<SyncButton {...baseProps} />);
    expect(screen.queryByTestId("sync-push-button")).not.toBeInTheDocument();

    rerender(<SyncButton {...baseProps} isPushing progressLabel="Pushing 4s" />);
    // The popover stays closed so it does not duplicate the snackbar; no redundant note.
    expect(screen.queryByTestId("sync-push-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sync-progress-note")).not.toBeInTheDocument();
  });

  it("names the running phase inside the popover when the user opens it", () => {
    render(<SyncButton {...baseProps} isPushing progressLabel="Pushing 4s" />);

    fireEvent.click(screen.getByTestId("sync-button-trigger"));
    expect(screen.getByTestId("sync-push-button")).toHaveTextContent("Pushing 4s");
  });

  it("blocks a second remote operation while one is in flight", () => {
    const onPull = vi.fn();
    render(
      <SyncButton {...baseProps} behind={2} isPushing progressLabel="Pushing 4s" onPull={onPull} />
    );

    fireEvent.click(screen.getByTestId("sync-button-trigger"));
    const pullButton = screen.getByTestId("sync-pull-button");
    expect(pullButton).toBeDisabled();
    fireEvent.click(pullButton);
    expect(onPull).not.toHaveBeenCalled();
  });

  it("stays mounted while a push runs even after the ahead count clears", () => {
    render(<SyncButton {...baseProps} ahead={0} behind={0} isPushing progressLabel="Pushing 8s" />);

    expect(screen.getByTestId("sync-button")).toBeInTheDocument();
  });
});
