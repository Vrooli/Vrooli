import { renderWithProviders as render } from "../test-utils";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { ReturnToLivePill } from "../components/ReturnToLivePill";

describe("ReturnToLivePill", () => {
  it("renders nothing while the pane is live", () => {
    render(<ReturnToLivePill visible={false} onReturnToLive={vi.fn()} />);
    expect(screen.queryByTestId("return-to-live-pill")).toBeNull();
  });

  it("shows the recovery control while scrolled back and invokes the handler on tap", () => {
    const onReturnToLive = vi.fn();
    render(<ReturnToLivePill visible onReturnToLive={onReturnToLive} />);

    expect(screen.getByTestId("return-to-live-pill")).toBeTruthy();
    // The action must be tappable — this is the escape hatch for a pane the
    // keystroke path can no longer reach.
    fireEvent.click(screen.getByTestId("return-to-live-action"));
    expect(onReturnToLive).toHaveBeenCalledTimes(1);
  });
});
