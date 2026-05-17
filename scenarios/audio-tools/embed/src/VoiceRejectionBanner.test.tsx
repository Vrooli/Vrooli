import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { VoiceRejectionBanner } from "./VoiceRejectionBanner";

describe("VoiceRejectionBanner", () => {
  it("renders the reason in a role=alert region for screen readers", () => {
    render(<VoiceRejectionBanner reason="No microphone detected" />);
    const alert = screen.getByRole("alert");
    expect(alert).toBeInTheDocument();
    expect(alert).toHaveTextContent("No microphone detected");
  });

  it("does not render a dismiss button when onDismiss is omitted", () => {
    render(<VoiceRejectionBanner reason="Permission denied" />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("renders a dismiss button with the default label when onDismiss is provided", () => {
    render(<VoiceRejectionBanner reason="Permission denied" onDismiss={() => {}} />);
    expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
  });

  it("fires onDismiss exactly once when the dismiss button is clicked", async () => {
    const onDismiss = vi.fn();
    const user = userEvent.setup();
    render(
      <VoiceRejectionBanner reason="Permission denied" onDismiss={onDismiss} dismissLabel="Close" />,
    );
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });
});
