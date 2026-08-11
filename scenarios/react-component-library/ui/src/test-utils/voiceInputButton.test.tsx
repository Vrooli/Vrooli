import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VoiceInputButton } from "../components/VoiceInputButton";
import { renderWithProviders } from "./renderWithProviders";

describe("VoiceInputButton", () => {
  it("renders exactly one control in every state", () => {
    const { rerender } = renderWithProviders(<VoiceInputButton state="idle" />);
    for (const state of [
      "idle",
      "preparing",
      "recording",
      "recovering",
      "transcribing",
      "unavailable",
      "error",
    ] as const) {
      rerender(<VoiceInputButton state={state} />);
      expect(screen.getAllByRole("button")).toHaveLength(1);
      expect(screen.queryByRole("status")).not.toBeInTheDocument();
      expect(screen.queryByText("Transcribe anyway")).not.toBeInTheDocument();
      expect(screen.getByRole("button")).not.toHaveAttribute("title");
    }
  });

  it("does not make unavailable voice input actionable", () => {
    renderWithProviders(<VoiceInputButton state="unavailable" />);
    expect(screen.getByRole("button", { name: "Voice input unavailable" })).toBeDisabled();
  });

  it("keeps web-console mic presentation and pointer semantics without scenario coupling", () => {
    const onStart = vi.fn();
    const onStop = vi.fn();
    const { rerender } = renderWithProviders(
      <VoiceInputButton state="idle" onStart={onStart} onStop={onStop} />,
    );
    const idle = screen.getByRole("button", { name: "Start voice input" });
    expect(idle).toHaveAttribute("data-testid", "voice-input-control");
    fireEvent.pointerDown(idle);
    fireEvent.pointerUp(idle);
    expect(onStart).toHaveBeenCalledOnce();
    expect(onStop).not.toHaveBeenCalled();

    rerender(
      <VoiceInputButton
        state="recording"
        mode="timeout"
        level={0.5}
        timeoutProgress={0.5}
        onStop={onStop}
      />,
    );
    const recording = screen.getByRole("button", { name: "Stop voice input" });
    expect(recording.className).toContain("border-app-danger");
    expect(recording.querySelector("svg")).toBeTruthy();
    fireEvent.pointerDown(recording);
    fireEvent.pointerUp(recording);
    expect(onStop).toHaveBeenCalledOnce();
  });

  it("uses the web-console cyan listening treatment for always-on audio level", () => {
    const { container } = renderWithProviders(
      <VoiceInputButton state="recording" mode="always-on" level={0.8} />,
    );
    const button = screen.getByRole("button", { name: "Stop voice input" });
    expect(button.className).toContain("border-app-info");
    expect(button.className).not.toContain("border-app-danger");
    expect(container.querySelector(".bg-app-primary\\/60")).toHaveStyle({ height: "80%" });
    expect(button.querySelector("circle")).toBeNull();
  });

  it("forwards pointer cancellation without adding a recovery surface", () => {
    const onPointerCancel = vi.fn();
    const { rerender } = renderWithProviders(
      <VoiceInputButton state="idle" onPointerCancel={onPointerCancel} />,
    );
    const button = screen.getByRole("button", { name: "Start voice input" });
    fireEvent.pointerCancel(button);
    expect(onPointerCancel).toHaveBeenCalledOnce();
    rerender(<VoiceInputButton state="preparing" onPrepare={vi.fn()} />);
    fireEvent.focus(screen.getByRole("button", { name: "Preparing microphone" }));
  });
});
