import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { VoiceCommandSuggestion } from "./VoiceCommandSuggestion";

describe("VoiceCommandSuggestion", () => {
  it("renders the suggestion inside a role=dialog with default aria-label", () => {
    render(
      <VoiceCommandSuggestion suggestion="Open files" onAccept={() => {}} onDismiss={() => {}} />,
    );
    const dialog = screen.getByRole("dialog", { name: "Voice command suggestion" });
    expect(dialog).toBeInTheDocument();
    expect(screen.getByText("Open files")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Accept" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
  });

  it("fires onAccept once when the Accept button is clicked", async () => {
    const onAccept = vi.fn();
    const onDismiss = vi.fn();
    const user = userEvent.setup();
    render(
      <VoiceCommandSuggestion suggestion="Open files" onAccept={onAccept} onDismiss={onDismiss} />,
    );
    await user.click(screen.getByRole("button", { name: "Accept" }));
    expect(onAccept).toHaveBeenCalledTimes(1);
    expect(onDismiss).not.toHaveBeenCalled();
  });

  it("fires onDismiss once when the Dismiss button is clicked", async () => {
    const onAccept = vi.fn();
    const onDismiss = vi.fn();
    const user = userEvent.setup();
    render(
      <VoiceCommandSuggestion suggestion="Open files" onAccept={onAccept} onDismiss={onDismiss} />,
    );
    await user.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(onDismiss).toHaveBeenCalledTimes(1);
    expect(onAccept).not.toHaveBeenCalled();
  });

  it("respects caller-supplied labels", () => {
    render(
      <VoiceCommandSuggestion
        suggestion="Test"
        onAccept={() => {}}
        onDismiss={() => {}}
        ariaLabel="Command preview"
        acceptLabel="Confirm"
        dismissLabel="Cancel"
      />,
    );
    expect(screen.getByRole("dialog", { name: "Command preview" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });
});
