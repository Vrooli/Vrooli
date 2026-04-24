import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AudioSettingsContent } from "../tts/AudioSettingsContent";
import type { TTSPlaybackCapabilities } from "../../hooks/tts/types";

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
      playbackRate={1}
      isSummarized={false}
      capabilities={fullCapabilities}
      onVolumeChange={vi.fn()}
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
});
