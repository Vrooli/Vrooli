import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import VoiceMicButton from "./VoiceMicButton";

const baseProps = {
  supported: true,
  isPreparing: false,
  isRecording: false,
  isListening: false,
  isPassive: false,
  isTranscribing: false,
  onStart: vi.fn(),
  onStop: vi.fn(),
};

describe("VoiceMicButton", () => {
  it("keeps unavailable voice feedback inside the button", () => {
    render(
      <VoiceMicButton
        {...baseProps}
        supported={false}
        error="Voice input unavailable (discovery_failed)"
        testId="composer-mic"
      />,
    );

    expect(screen.getByRole("button", { name: "Voice input unavailable" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByText("Voice input unavailable (discovery_failed)")).not.toBeInTheDocument();
  });

  it("keeps runtime errors as button state without adding a layout label", () => {
    render(
      <VoiceMicButton
        {...baseProps}
        error="The speech backend is unavailable; retry the turn."
        testId="composer-mic"
      />,
    );

    expect(screen.getByRole("button", { name: "Voice input error" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByText("The speech backend is unavailable; retry the turn.")).not.toBeInTheDocument();
  });
});
