import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VoiceInputButton } from "../../../library/components/VoiceInputButton/versions/1.0.0/VoiceInputButton";
import { renderWithProviders } from "./renderWithProviders";

describe("VoiceInputButton", () => {
  it("exposes stateful labels and a keyboard-operable rejection override", () => {
    const onTranscribeAnyway = vi.fn();
    renderWithProviders(<VoiceInputButton state="recording" mode="timeout" timeoutProgress={0.5} rejectionReason="not verified" onTranscribeAnyway={onTranscribeAnyway} />);
    expect(screen.getByRole("button", { name: "Stop voice input" })).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: "Transcribe anyway" }));
    expect(onTranscribeAnyway).toHaveBeenCalledOnce();
  });

  it("does not make unavailable voice input actionable", () => {
    renderWithProviders(<VoiceInputButton state="unavailable" />);
    expect(screen.getByRole("button", { name: "Voice input unavailable" })).toBeDisabled();
  });

  it("keeps web-console mic presentation and pointer semantics without scenario coupling", () => {
    const onStart = vi.fn();
    const onStop = vi.fn();
    const { rerender } = renderWithProviders(<VoiceInputButton state="idle" onStart={onStart} onStop={onStop} />);
    const idle = screen.getByRole("button", { name: "Start voice input" });
    expect(idle.className).toContain("rounded border px-1.5 py-1");
    fireEvent.pointerDown(idle);
    fireEvent.pointerUp(idle);
    expect(onStart).toHaveBeenCalledOnce();
    expect(onStop).not.toHaveBeenCalled();

    rerender(<VoiceInputButton state="recording" mode="timeout" level={0.5} timeoutProgress={0.5} onStop={onStop} />);
    const recording = screen.getByRole("button", { name: "Stop voice input" });
    expect(recording.className).toContain("border-app-danger");
    expect(recording.querySelector("svg")).toBeTruthy();
    fireEvent.pointerDown(recording);
    fireEvent.pointerUp(recording);
    expect(onStop).toHaveBeenCalledOnce();
  });

  it("uses the web-console cyan listening treatment for always-on audio level", () => {
    const { container } = renderWithProviders(<VoiceInputButton state="recording" mode="always-on" level={0.8} />);
    const button = screen.getByRole("button", { name: "Stop voice input" });
    expect(button.className).toContain("border-app-info");
    expect(button.className).not.toContain("border-app-danger");
    expect(container.querySelector(".bg-app-info\\/30")).toHaveStyle({ height: "80%" });
    expect(button.querySelector("circle")).toBeNull();
  });
});
