import { renderWithProviders as render } from "../../test-utils";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DeviceFrame } from "../terminal/DeviceFrame";
const rect = { x: 0, y: 0, width: 300, height: 200, fontSize: 12, scale: 1 };
describe("DeviceFrame", () => {
  it.each(["full", "hairline", "strip"] as const)("renders a takeover control at %s", (chromeTier) => {
    const onTakeOver = vi.fn();
    render(<DeviceFrame archetype="phone" chromeTier={chromeTier} rect={rect} leaderDevice="Desktop" gridCols={80} gridRows={24} onTakeOver={onTakeOver} />);
    fireEvent.click(screen.getByRole("button"));
    expect(onTakeOver).toHaveBeenCalledOnce();
  });
});
