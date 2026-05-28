// Render tests for the swarm-manager VoiceMicButton. Mirrors the
// web-console suite to ensure the ring + phase rendering remain in sync
// across consumer scenarios. Drives the component with synthetic props
// (no useVoiceCore), so it's a fast pure-render test.

import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

import VoiceMicButton from "./VoiceMicButton";
import { _resetServerVadStateForTesting, setServerVadState } from "../../audio-integration";

describe("VoiceMicButton", () => {
  const onStart = vi.fn();
  const onStop = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    _resetServerVadStateForTesting();
  });

  const defaults = {
    supported: true,
    isPreparing: false,
    isRecording: false,
    isTranscribing: false,
    error: null,
    onStart,
    onStop,
  };

  it("returns null when not supported", () => {
    const { container } = render(<VoiceMicButton {...defaults} supported={false} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders mic icon in idle state", () => {
    render(<VoiceMicButton {...defaults} />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn).toBeTruthy();
    expect(btn.className).toContain("border-slate-700");
  });

  it("shows red styling when recording", () => {
    render(<VoiceMicButton {...defaults} isRecording />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-red-500");
  });

  it("shows blue styling and spinner when transcribing", () => {
    render(<VoiceMicButton {...defaults} isTranscribing />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-blue-500");
    expect(btn.querySelector(".animate-spin")).toBeTruthy();
  });

  it("allows the idle error tooltip to be dismissed", () => {
    const { rerender } = render(<VoiceMicButton {...defaults} error="Test error" />);
    rerender(<VoiceMicButton {...defaults} error="Test error" />);

    expect(screen.getByText("Test error")).toBeTruthy();
    fireEvent.click(screen.getByLabelText("Dismiss voice input error"));
    expect(screen.queryByText("Test error")).toBeNull();
  });

  it("renders the auto-stop ring when recording and in silence phase", () => {
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "silence",
          audioLevel: 0,
          rms: 0,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 600,
          silenceTimeoutMs: 1200,
          autoStopProgress: 0.5,
          autoStopVisible: true,
        }}
      />,
    );
    expect(screen.getByTestId("voice-auto-stop-ring")).toBeTruthy();
  });

  it("sizes the auto-stop ring to cover the button like web-console", () => {
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "silence",
          audioLevel: 0,
          rms: 0,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 600,
          silenceTimeoutMs: 1200,
          autoStopProgress: 0.5,
          autoStopVisible: true,
        }}
      />,
    );

    const ring = screen.getByTestId("voice-auto-stop-ring");
    expect(ring.getAttribute("class")).toContain("h-[calc(100%-4px)]");
    expect(ring.getAttribute("class")).toContain("max-h-8");
  });

  it("hides the auto-stop ring before the visual grace period elapses", () => {
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "silence",
          audioLevel: 0,
          rms: 0,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 100,
          silenceTimeoutMs: 1200,
          autoStopProgress: 0.1,
          autoStopVisible: false,
        }}
      />,
    );
    expect(screen.queryByTestId("voice-auto-stop-ring")).toBeNull();
  });

  it("hides the auto-stop ring when silenceTimeoutMs is zero (server-config not hydrated)", () => {
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "silence",
          audioLevel: 0,
          rms: 0,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 500,
          silenceTimeoutMs: 0,
          autoStopProgress: 0,
          autoStopVisible: true,
        }}
      />,
    );
    expect(screen.queryByTestId("voice-auto-stop-ring")).toBeNull();
  });

  it("ring stroke-dashoffset reflects autoStopProgress (server-driven)", () => {
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "silence",
          audioLevel: 0,
          rms: 0,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 1200,
          silenceTimeoutMs: 1200,
          autoStopProgress: 1,
          autoStopVisible: true,
        }}
      />,
    );
    const ring = screen.getByTestId("voice-auto-stop-ring");
    const circle = ring.querySelector("circle");
    expect(circle?.getAttribute("stroke-dashoffset")).toBe("0");
  });

  it("shows audio-level fill when recording", () => {
    render(<VoiceMicButton {...defaults} isRecording audioLevel={0.5} />);
    const btn = screen.getByTestId("voice-mic-btn");
    const fill = btn.querySelector("span");
    expect(fill?.style.height).toBe("50%");
  });

  // ── Server-driven ring (StreamVadState contract) ──
  // See plan: server-driven-mic-ring-streamvadstate-event.md §9 item 5.

  it("prefers fresh serverVad prop over client voiceActivity for the ring", () => {
    // Stale (or absent) voiceActivity: ring would not render with autoStopVisible=false.
    // Fresh serverVad with 50% elapsed → ring renders, dashoffset reflects ~0.5.
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "speech",
          audioLevel: 0.2,
          rms: 0.1,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 0,
          silenceTimeoutMs: 1500,
          autoStopProgress: 0,
          autoStopVisible: false,
        }}
        serverVad={{
          voiced: false,
          silenceElapsedMs: 750,
          silenceTimeoutMs: 1500,
          receivedAt: performance.now(),
          tickSeq: 3,
          silenceTimedOut: false,
        }}
      />,
    );
    const ring = screen.queryByTestId("voice-auto-stop-ring");
    expect(ring).toBeTruthy();
    // Circumference is 2π*18 ≈ 113.097. At ~0.5 progress, dashoffset is ~56.5
    // (allow loose tolerance for the +(now-receivedAt) interpolation drift).
    const offset = Number(ring!.querySelector("circle")?.getAttribute("stroke-dashoffset"));
    expect(offset).toBeGreaterThan(0);
    expect(offset).toBeLessThan(113.097);
  });

  it("falls back to client voiceActivity when serverVad is stale (>250 ms)", () => {
    const staleAt = performance.now() - 1000;
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "silence",
          audioLevel: 0,
          rms: 0,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 600,
          silenceTimeoutMs: 1200,
          autoStopProgress: 0.5,
          autoStopVisible: true,
        }}
        serverVad={{
          voiced: false,
          silenceElapsedMs: 1100,
          silenceTimeoutMs: 1200,
          receivedAt: staleAt,
          tickSeq: 7,
          silenceTimedOut: false,
        }}
      />,
    );
    const ring = screen.getByTestId("voice-auto-stop-ring");
    const offset = Number(ring.querySelector("circle")?.getAttribute("stroke-dashoffset"));
    // Client value: progress 0.5 → dashoffset ≈ 0.5 * 113.097 ≈ 56.5
    expect(offset).toBeGreaterThan(50);
    expect(offset).toBeLessThan(65);
  });

  it("hides the ring when serverVad is fresh but silenceTimeoutMs is 0", () => {
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "silence",
          audioLevel: 0,
          rms: 0,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 500,
          silenceTimeoutMs: 0,
          autoStopProgress: 0,
          autoStopVisible: true,
        }}
        serverVad={{
          voiced: false,
          silenceElapsedMs: 500,
          silenceTimeoutMs: 0,
          receivedAt: performance.now(),
          tickSeq: 1,
          silenceTimedOut: false,
        }}
      />,
    );
    expect(screen.queryByTestId("voice-auto-stop-ring")).toBeNull();
  });

  it("subscribes to useServerVadStateStore when no serverVad prop is provided", () => {
    setServerVadState({
      voiced: false,
      silenceElapsedMs: 900,
      silenceTimeoutMs: 1500,
      tickSeq: 12,
    });
    render(
      <VoiceMicButton
        {...defaults}
        isRecording
        voiceActivity={{
          phase: "speech",
          audioLevel: 0.2,
          rms: 0.1,
          speechThreshold: 0.06,
          silenceThreshold: 0.02,
          silenceElapsedMs: 0,
          silenceTimeoutMs: 1500,
          autoStopProgress: 0,
          autoStopVisible: false,
        }}
      />,
    );
    // The store-driven server snapshot should override the speech-phase
    // voiceActivity and render the ring.
    expect(screen.queryByTestId("voice-auto-stop-ring")).toBeTruthy();
  });
});
