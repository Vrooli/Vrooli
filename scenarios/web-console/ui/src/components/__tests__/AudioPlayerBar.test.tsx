import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import AudioPlayerBar from "../AudioPlayerBar";
import type { AudioPlayerBarProps } from "../AudioPlayerBar";
import type { TTSPlaybackCapabilities } from "../../hooks/tts/types";

const fullCapabilities: TTSPlaybackCapabilities = {
  canPause: true,
  canSeek: true,
  canAdjustSpeed: true,
  canAdjustVolume: true,
};

const limitedCapabilities: TTSPlaybackCapabilities = {
  canPause: true,
  canSeek: false,
  canAdjustSpeed: false,
  canAdjustVolume: false,
};

function makeProps(overrides?: Partial<AudioPlayerBarProps>): AudioPlayerBarProps {
  return {
    isPaused: false,
    currentTime: 42,
    duration: 83,
    playbackRate: 1,
    volume: 1,
    capabilities: fullCapabilities,
    onPause: vi.fn(),
    onResume: vi.fn(),
    onSeek: vi.fn(),
    onSetPlaybackRate: vi.fn(),
    onSetVolume: vi.fn(),
    onStop: vi.fn(),
    ...overrides,
  };
}

describe("AudioPlayerBar", () => {
  it("renders pause icon when not paused", () => {
    render(<AudioPlayerBar {...makeProps({ isPaused: false })} />);
    const btn = screen.getByTestId("tts-play-pause");
    expect(btn.getAttribute("title")).toBe("Pause");
  });

  it("renders play icon when paused", () => {
    render(<AudioPlayerBar {...makeProps({ isPaused: true })} />);
    const btn = screen.getByTestId("tts-play-pause");
    expect(btn.getAttribute("title")).toBe("Resume");
  });

  it("clicking play/pause toggle calls onPause when playing", () => {
    const props = makeProps({ isPaused: false });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-play-pause"));
    expect(props.onPause).toHaveBeenCalledTimes(1);
  });

  it("clicking play/pause toggle calls onResume when paused", () => {
    const props = makeProps({ isPaused: true });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-play-pause"));
    expect(props.onResume).toHaveBeenCalledTimes(1);
  });

  it("stop button always visible and calls onStop", () => {
    const props = makeProps();
    render(<AudioPlayerBar {...props} />);
    const stopBtn = screen.getByTestId("tts-stop");
    expect(stopBtn).toBeInTheDocument();
    fireEvent.click(stopBtn);
    expect(props.onStop).toHaveBeenCalledTimes(1);
  });

  it("scrub bar reflects currentTime and changing it calls onSeek", () => {
    const props = makeProps({ currentTime: 10, duration: 60 });
    render(<AudioPlayerBar {...props} />);
    const scrub = screen.getByTestId("tts-scrub") as HTMLInputElement;
    expect(scrub.value).toBe("10");
    fireEvent.change(scrub, { target: { value: "30" } });
    expect(props.onSeek).toHaveBeenCalledWith(30);
  });

  it("time display formats correctly (0:42 / 1:23)", () => {
    render(<AudioPlayerBar {...makeProps({ currentTime: 42, duration: 83 })} />);
    expect(screen.getByTestId("tts-time").textContent).toBe("0:42 / 1:23");
  });

  it("time display shows --:-- when duration is null", () => {
    render(<AudioPlayerBar {...makeProps({ currentTime: 0, duration: null })} />);
    expect(screen.getByTestId("tts-time").textContent).toBe("0:00 / --:--");
  });

  it("speed button cycles through presets on click", () => {
    const props = makeProps({ playbackRate: 1 });
    render(<AudioPlayerBar {...props} />);
    const speedBtn = screen.getByTestId("tts-speed");
    expect(speedBtn.textContent).toBe("1x");
    fireEvent.click(speedBtn);
    expect(props.onSetPlaybackRate).toHaveBeenCalledWith(1.25);
  });

  it("hides scrub bar when canSeek is false", () => {
    render(<AudioPlayerBar {...makeProps({ capabilities: limitedCapabilities })} />);
    expect(screen.queryByTestId("tts-scrub")).toBeNull();
  });

  it("hides scrub bar when duration is null", () => {
    render(<AudioPlayerBar {...makeProps({ duration: null })} />);
    expect(screen.queryByTestId("tts-scrub")).toBeNull();
  });

  it("hides speed button when canAdjustSpeed is false", () => {
    render(<AudioPlayerBar {...makeProps({ capabilities: limitedCapabilities })} />);
    expect(screen.queryByTestId("tts-speed")).toBeNull();
  });

  it("hides volume controls when canAdjustVolume is false", () => {
    render(<AudioPlayerBar {...makeProps({ capabilities: limitedCapabilities })} />);
    expect(screen.queryByTestId("tts-volume-toggle")).toBeNull();
  });

  it("volume toggle mutes when volume > 0", () => {
    const props = makeProps({ volume: 0.8 });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-volume-toggle"));
    expect(props.onSetVolume).toHaveBeenCalledWith(0);
  });

  it("volume toggle unmutes to 1 when volume is 0", () => {
    const props = makeProps({ volume: 0 });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-volume-toggle"));
    expect(props.onSetVolume).toHaveBeenCalledWith(1);
  });

  it("disables play/pause when canPause is false", () => {
    const noPause = { ...fullCapabilities, canPause: false };
    render(<AudioPlayerBar {...makeProps({ capabilities: noPause })} />);
    const btn = screen.getByTestId("tts-play-pause");
    expect(btn).toBeDisabled();
  });
});
