import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
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
  it("names the running phase and elapsed time instead of a bare spinner", () => {
    render(<SyncButton {...baseProps} isPushing progressLabel="Pushing 1m 12s" />);

    expect(screen.getByTestId("sync-push-button")).toHaveTextContent("Pushing 1m 12s");
  });

  it("opens itself while an operation runs so progress stays visible", () => {
    const { rerender } = render(<SyncButton {...baseProps} />);
    expect(screen.queryByTestId("sync-push-button")).not.toBeInTheDocument();

    rerender(<SyncButton {...baseProps} isPushing progressLabel="Pushing 4s" />);
    expect(screen.getByTestId("sync-push-button")).toBeInTheDocument();
    expect(screen.getByTestId("sync-progress-note")).toBeInTheDocument();
  });

  it("blocks a second remote operation while one is in flight", () => {
    const onPull = vi.fn();
    render(
      <SyncButton {...baseProps} behind={2} isPushing progressLabel="Pushing 4s" onPull={onPull} />
    );

    const pullButton = screen.getByTestId("sync-pull-button");
    expect(pullButton).toBeDisabled();
    fireEvent.click(pullButton);
    expect(onPull).not.toHaveBeenCalled();
  });

  it("closes itself once the operation finishes so the result is read from the toast", () => {
    const { rerender } = render(<SyncButton {...baseProps} isPushing progressLabel="Pushing 4s" />);
    expect(screen.getByTestId("sync-push-button")).toBeInTheDocument();

    rerender(<SyncButton {...baseProps} isPushing={false} />);
    expect(screen.queryByTestId("sync-push-button")).not.toBeInTheDocument();
  });

  it("stays mounted while a push runs even after the ahead count clears", () => {
    render(<SyncButton {...baseProps} ahead={0} behind={0} isPushing progressLabel="Pushing 8s" />);

    expect(screen.getByTestId("sync-button")).toBeInTheDocument();
  });
});
