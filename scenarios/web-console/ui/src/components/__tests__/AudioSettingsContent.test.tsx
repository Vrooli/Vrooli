import { renderWithProviders as render } from "../../test-utils";
import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { strings } from "../../consts/strings";
import { AudioSettingsContent } from "../tts/AudioSettingsContent";
import type { TTSPlaybackCapabilities } from "../../audio-integration";

const fullCapabilities: TTSPlaybackCapabilities = {
  canPause: true,
  canSeek: true,
  canAdjustSpeed: true,
  canAdjustVolume: true,
};

function renderContent(overrides?: Partial<Parameters<typeof AudioSettingsContent>[0]>) {
  return render(
    <AudioSettingsContent
      testIdPrefix="x"
      volume={0.8}
      isMuted={false}
      playbackRate={1}
      isSummarized={false}
      capabilities={fullCapabilities}
      onVolumeChange={vi.fn()}
      onSetMuted={vi.fn()}
      onSetPlaybackRate={vi.fn()}
      {...overrides}
    />,
  );
}

describe("AudioSettingsContent", () => {
  it("renders volume slider when canAdjustVolume is true", () => {
    renderContent();
    expect(screen.getByTestId("x-volume-slider")).toBeInTheDocument();
  });

  it("hides volume slider when canAdjustVolume is false", () => {
    renderContent({ capabilities: { ...fullCapabilities, canAdjustVolume: false } });
    expect(screen.queryByTestId("x-volume-slider")).toBeNull();
  });

  it("renders all six speed presets when canAdjustSpeed is true", () => {
    renderContent();
    for (const rate of [0.5, 0.75, 1, 1.25, 1.5, 2]) {
      expect(screen.getByTestId(`x-speed-preset-${rate}`)).toBeInTheDocument();
    }
  });

  it("hides speed presets when canAdjustSpeed is false", () => {
    renderContent({ capabilities: { ...fullCapabilities, canAdjustSpeed: false } });
    expect(screen.queryByTestId("x-speed-preset-1")).toBeNull();
  });

  it("highlights the active speed preset", () => {
    renderContent({ playbackRate: 1.5 });
    const active = screen.getByTestId("x-speed-preset-1.5");
    expect(active.className).toMatch(/bg-wc-accent/);
  });

  it("clicking a speed preset calls onSetPlaybackRate", () => {
    const onSetPlaybackRate = vi.fn();
    renderContent({ onSetPlaybackRate });
    fireEvent.click(screen.getByTestId("x-speed-preset-0.5"));
    expect(onSetPlaybackRate).toHaveBeenCalledWith(0.5);
  });

  it("changing volume slider calls onVolumeChange", () => {
    const onVolumeChange = vi.fn();
    renderContent({ onVolumeChange });
    fireEvent.change(screen.getByTestId("x-volume-slider"), { target: { value: "0.3" } });
    expect(onVolumeChange).toHaveBeenCalledWith(0.3);
  });

  it("uses amber accent on volume slider when summarized", () => {
    renderContent({ isSummarized: true });
    const slider = screen.getByTestId("x-volume-slider");
    expect(slider.className).toMatch(/accent-amber-400/);
  });

  it("does NOT render a summarized/original toggle (that moved to PlaybackModeControl)", () => {
    renderContent({ isSummarized: true });
    expect(screen.queryByTestId("x-play-summarized")).toBeNull();
    expect(screen.queryByTestId("x-play-original")).toBeNull();
  });

  // --- Mute toggle ---

  it("renders mute toggle reflecting unmuted state", () => {
    renderContent({ isMuted: false });
    const toggle = screen.getByTestId("x-mute-toggle");
    expect(toggle.getAttribute("aria-pressed")).toBe("false");
    expect(toggle.getAttribute("aria-label")).toBe(strings.audioSettings.muteAria);
    expect(toggle.textContent).toContain(strings.audioSettings.mute);
  });

  it("renders mute toggle reflecting muted state", () => {
    renderContent({ isMuted: true });
    const toggle = screen.getByTestId("x-mute-toggle");
    expect(toggle.getAttribute("aria-pressed")).toBe("true");
    expect(toggle.getAttribute("aria-label")).toBe(strings.audioSettings.unmuteAria);
    expect(toggle.textContent).toContain(strings.audioSettings.muted);
  });

  it("clicking the mute toggle calls onSetMuted with the negation", () => {
    const onSetMuted = vi.fn();
    renderContent({ isMuted: false, onSetMuted });
    fireEvent.click(screen.getByTestId("x-mute-toggle"));
    expect(onSetMuted).toHaveBeenCalledWith(true);
  });

  it("clicking the mute toggle while muted calls onSetMuted(false)", () => {
    const onSetMuted = vi.fn();
    renderContent({ isMuted: true, onSetMuted });
    fireEvent.click(screen.getByTestId("x-mute-toggle"));
    expect(onSetMuted).toHaveBeenCalledWith(false);
  });

  it("dragging the slider while muted calls onSetMuted(false) BEFORE onVolumeChange", () => {
    const calls: Array<{ name: string; value: unknown }> = [];
    renderContent({
      isMuted: true,
      onSetMuted: (v) => calls.push({ name: "onSetMuted", value: v }),
      onVolumeChange: (v) => calls.push({ name: "onVolumeChange", value: v }),
    });
    fireEvent.change(screen.getByTestId("x-volume-slider"), { target: { value: "0.4" } });
    expect(calls).toEqual([
      { name: "onSetMuted", value: false },
      { name: "onVolumeChange", value: 0.4 },
    ]);
  });

  it("dragging the slider while NOT muted calls only onVolumeChange", () => {
    const onSetMuted = vi.fn();
    const onVolumeChange = vi.fn();
    renderContent({ isMuted: false, onSetMuted, onVolumeChange });
    fireEvent.change(screen.getByTestId("x-volume-slider"), { target: { value: "0.4" } });
    expect(onSetMuted).not.toHaveBeenCalled();
    expect(onVolumeChange).toHaveBeenCalledWith(0.4);
  });

  it("slider has dimmed opacity class when muted", () => {
    renderContent({ isMuted: true });
    expect(screen.getByTestId("x-volume-slider").className).toMatch(/opacity-50/);
  });

  it("slider does not have dimmed opacity class when unmuted", () => {
    renderContent({ isMuted: false });
    expect(screen.getByTestId("x-volume-slider").className).not.toMatch(/opacity-50/);
  });
});
