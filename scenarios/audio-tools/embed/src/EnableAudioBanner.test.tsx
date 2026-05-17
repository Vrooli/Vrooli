import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { EnableAudioBanner } from "./EnableAudioBanner";

describe("EnableAudioBanner", () => {
  it("renders the default message and action label", () => {
    render(<EnableAudioBanner onEnable={() => {}} />);
    expect(screen.getByText("Audio is muted. Click to enable.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Enable audio" })).toBeInTheDocument();
  });

  it("uses caller-supplied message and actionLabel", () => {
    render(
      <EnableAudioBanner onEnable={() => {}} message="Tap to unmute" actionLabel="Unmute now" />,
    );
    expect(screen.getByText("Tap to unmute")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Unmute now" })).toBeInTheDocument();
  });

  it("fires onEnable exactly once when the button is clicked", async () => {
    const onEnable = vi.fn();
    const user = userEvent.setup();
    render(<EnableAudioBanner onEnable={onEnable} />);
    await user.click(screen.getByRole("button", { name: "Enable audio" }));
    expect(onEnable).toHaveBeenCalledTimes(1);
  });
});
