import { fireEvent } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { VoiceInputButton } from "../components/VoiceInputButton";
import { renderWithProviders } from "../test-utils";

describe("VoiceInputButton", () => {
  let unmount: (() => void) | undefined;
  const renderVoiceInput = (ui: ReactElement) => {
    const result = renderWithProviders(ui);
    unmount = result.unmount;
    return result;
  };

  afterEach(() => {
    unmount?.();
    unmount = undefined;
  });

  it("renders exactly one control in every state", () => {
    const { getAllByRole, rerender, queryByRole, queryByText, getByRole } = renderVoiceInput(
      <VoiceInputButton state="idle" size="lg" />,
    );
    for (const state of [
      "idle",
      "preparing",
      "recording",
      "recovering",
      "transcribing",
      "unavailable",
      "error",
    ] as const) {
      rerender(<VoiceInputButton state={state} size="lg" />);
      expect(getAllByRole("button")).toHaveLength(1);
      expect(queryByRole("status")).not.toBeInTheDocument();
      expect(queryByText("Transcribe anyway")).not.toBeInTheDocument();
      expect(getByRole("button")).not.toHaveAttribute("title");
    }
  });

  it("does not make unavailable voice input actionable", () => {
    const { getByRole } = renderVoiceInput(<VoiceInputButton state="unavailable" size="lg" />);
    expect(getByRole("button", { name: "Voice input unavailable" })).toBeDisabled();
  });

  it("keeps web-console mic presentation and pointer semantics without scenario coupling", () => {
    const onStart = vi.fn();
    const onStop = vi.fn();
    const { getByRole, rerender } = renderVoiceInput(
      <VoiceInputButton state="idle" size="lg" onStart={onStart} onStop={onStop} />,
    );
    const idle = getByRole("button", { name: "Start voice input" });
    expect(idle).toHaveAttribute("data-testid", "voice-input-control");
    fireEvent.pointerDown(idle);
    fireEvent.pointerUp(idle);
    expect(onStart).toHaveBeenCalledOnce();
    expect(onStop).not.toHaveBeenCalled();

    // A mobile press can last longer than the old 300ms hold threshold. It is
    // still a start gesture, not an implicit stop that submits empty audio.
    vi.spyOn(Date, "now").mockReturnValueOnce(1000).mockReturnValueOnce(1500);
    fireEvent.pointerDown(idle);
    fireEvent.pointerUp(idle);
    expect(onStart).toHaveBeenCalledTimes(2);
    expect(onStop).not.toHaveBeenCalled();
    vi.restoreAllMocks();

    rerender(
      <VoiceInputButton
        state="recording"
        mode="timeout"
        size="lg"
        level={0.5}
        timeoutProgress={0.5}
        onStop={onStop}
      />,
    );
    const recording = getByRole("button", { name: "Stop voice input" });
    expect(recording).toHaveAttribute("data-state", "recording");
    expect(recording).toHaveAttribute("data-mode", "timeout");
    expect(recording.querySelector("svg")).toBeTruthy();
    fireEvent.pointerDown(recording);
    fireEvent.pointerUp(recording);
    expect(onStop).toHaveBeenCalledOnce();
  });

  it("allows a host to request a borderless rounded toolbar surface", () => {
    const { getByRole } = renderVoiceInput(
      <VoiceInputButton state="idle" surface="ghost" shape="rounded" />,
    );
    const button = getByRole("button", { name: "Start voice input" });
    expect(button).toHaveAttribute("data-surface", "ghost");
    expect(button).toHaveAttribute("data-rcl-shape", "rounded");
  });

  it("uses the web-console cyan listening treatment for always-on audio level", () => {
    const { container, getByRole } = renderVoiceInput(
      <VoiceInputButton state="recording" mode="always-on" size="lg" level={0.8} />,
    );
    const button = getByRole("button", { name: "Stop voice input" });
    expect(button).toHaveAttribute("data-state", "recording");
    expect(button).toHaveAttribute("data-mode", "always-on");
    expect(container.querySelector("[data-rcl-voice-level]")).toHaveStyle({ height: "80%" });
    expect(button.querySelector("circle")).toBeNull();
  });

  it("forwards pointer cancellation without adding a recovery surface", () => {
    const onPointerCancel = vi.fn();
    const { getByRole, rerender } = renderVoiceInput(
      <VoiceInputButton state="idle" size="lg" onPointerCancel={onPointerCancel} />,
    );
    const button = getByRole("button", { name: "Start voice input" });
    fireEvent.pointerCancel(button);
    expect(onPointerCancel).toHaveBeenCalledOnce();
    rerender(<VoiceInputButton state="preparing" size="lg" onPrepare={vi.fn()} />);
    fireEvent.focus(getByRole("button", { name: "Preparing microphone" }));
  });
});
