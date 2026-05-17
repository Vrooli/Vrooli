import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { VoiceInputButton } from "./VoiceInputButton";

describe("VoiceInputButton", () => {
  it("renders with the default aria-label and is reachable via role", () => {
    render(<VoiceInputButton onTranscript={() => {}} />);
    expect(screen.getByRole("button", { name: "Voice input" })).toBeInTheDocument();
  });

  it("respects ariaLabel prop override for screen readers", () => {
    render(<VoiceInputButton onTranscript={() => {}} ariaLabel="Start dictation" />);
    expect(screen.getByRole("button", { name: "Start dictation" })).toBeInTheDocument();
  });

  it("toggles aria-pressed and shows listeningLabel on click", async () => {
    const user = userEvent.setup();
    render(<VoiceInputButton onTranscript={() => {}} listeningLabel="Recording" />);
    const button = screen.getByRole("button", { name: "Voice input" });
    expect(button).toHaveAttribute("aria-pressed", "false");
    await user.click(button);
    expect(button).toHaveAttribute("aria-pressed", "true");
    expect(button).toHaveTextContent("Recording");
  });

  it("accepts the documented optional props without throwing", () => {
    const onTranscript = vi.fn();
    const commandHandler = vi.fn();
    render(
      <VoiceInputButton
        onTranscript={onTranscript}
        commandHandler={commandHandler}
        language="en"
        mode="wake-word"
      />,
    );
    expect(screen.getByRole("button")).toBeInTheDocument();
  });
});
