import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { strings } from "../../consts/strings";
import AudioPlayerBar from "../AudioPlayerBar";
import type { AudioPlayerBarProps } from "../AudioPlayerBar";
import type { TTSPlaybackCapabilities } from "../../hooks/tts/types";
import type { ConversationEvent } from "../../api/conversation";

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
    isMuted: false,
    capabilities: fullCapabilities,
    onPause: vi.fn(),
    onResume: vi.fn(),
    onSeek: vi.fn(),
    onSetPlaybackRate: vi.fn(),
    onSetVolume: vi.fn(),
    onSetMuted: vi.fn(),
    ...overrides,
  };
}

function makeEvent(id: string, sequence: number): ConversationEvent {
  return {
    id,
    sessionId: "sess-1",
    source: "claude_hook",
    role: "assistant",
    text: `Message ${sequence}`,
    speechParagraphs: [`Message ${sequence}`],
    summarized: false,
    createdAt: new Date().toISOString(),
    sequence,
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
  };
}

describe("AudioPlayerBar", () => {
  it("renders pause icon when not paused", () => {
    render(<AudioPlayerBar {...makeProps({ isPaused: false })} />);
    const btn = screen.getByTestId("tts-play-pause");
    expect(btn.getAttribute("title")).toBe("audioPlayerBar.pause");
  });

  it("renders play icon when paused", () => {
    render(<AudioPlayerBar {...makeProps({ isPaused: true })} />);
    const btn = screen.getByTestId("tts-play-pause");
    expect(btn.getAttribute("title")).toBe("audioPlayerBar.resume");
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

  it("does not render a dismiss control", () => {
    render(<AudioPlayerBar {...makeProps()} />);
    expect(screen.queryByTestId("tts-dismiss")).toBeNull();
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

  it("scrub is always rendered but disabled when canSeek is false", () => {
    render(<AudioPlayerBar {...makeProps({ capabilities: limitedCapabilities })} />);
    const scrub = screen.getByTestId("tts-scrub") as HTMLInputElement;
    expect(scrub).toBeInTheDocument();
    expect(scrub).toBeDisabled();
  });

  it("scrub is always rendered but disabled when duration is null (idle/replay)", () => {
    render(<AudioPlayerBar {...makeProps({ duration: null })} />);
    const scrub = screen.getByTestId("tts-scrub") as HTMLInputElement;
    expect(scrub).toBeInTheDocument();
    expect(scrub).toBeDisabled();
    expect(scrub.value).toBe("0");
  });

  it("hides audio button when canAdjustVolume is false", () => {
    render(<AudioPlayerBar {...makeProps({ capabilities: limitedCapabilities })} />);
    expect(screen.queryByTestId("tts-audio-button")).toBeNull();
  });

  it("clicking audio button opens settings popover with volume slider", () => {
    render(<AudioPlayerBar {...makeProps()} />);
    fireEvent.click(screen.getByTestId("tts-audio-button"));
    expect(screen.getByTestId("audio-popover")).toBeInTheDocument();
    expect(screen.getByTestId("tts-volume-slider")).toBeInTheDocument();
  });

  it("settings popover includes speed presets when canAdjustSpeed is true", () => {
    render(<AudioPlayerBar {...makeProps({ playbackRate: 1 })} />);
    fireEvent.click(screen.getByTestId("tts-audio-button"));
    expect(screen.getByTestId("tts-speed-preset-0.5")).toBeInTheDocument();
    expect(screen.getByTestId("tts-speed-preset-1")).toBeInTheDocument();
    expect(screen.getByTestId("tts-speed-preset-2")).toBeInTheDocument();
  });

  it("clicking a speed preset calls onSetPlaybackRate with that rate", () => {
    const props = makeProps({ playbackRate: 1 });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-audio-button"));
    fireEvent.click(screen.getByTestId("tts-speed-preset-1.5"));
    expect(props.onSetPlaybackRate).toHaveBeenCalledWith(1.5);
  });

  it("volume slider changes call onSetVolume", () => {
    const props = makeProps();
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-audio-button"));
    fireEvent.change(screen.getByTestId("tts-volume-slider"), { target: { value: "0.5" } });
    expect(props.onSetVolume).toHaveBeenCalledWith(0.5);
  });

  it("disables play/pause when canPause is false", () => {
    const noPause = { ...fullCapabilities, canPause: false };
    render(<AudioPlayerBar {...makeProps({ capabilities: noPause })} />);
    const btn = screen.getByTestId("tts-play-pause");
    expect(btn).toBeDisabled();
  });

  // --- De-escalated summarized mode ---

  it("does NOT render a standalone summarized badge", () => {
    render(<AudioPlayerBar {...makeProps({ isSummarized: true, canSummarize: true })} />);
    expect(screen.queryByTestId("tts-summarized-badge")).toBeNull();
  });

  it("does NOT apply an amber background to the bar in summarized mode", () => {
    render(<AudioPlayerBar {...makeProps({ isSummarized: true, canSummarize: true })} />);
    const bar = screen.getByTestId("audio-player-bar");
    expect(bar.className).not.toMatch(/bg-amber/);
  });

  it("scrub retains amber accent in summarized mode (the sole remaining signal)", () => {
    render(<AudioPlayerBar {...makeProps({ isSummarized: true, canSummarize: true })} />);
    const scrub = screen.getByTestId("tts-scrub");
    expect(scrub.className).toMatch(/accent-amber-400/);
  });

  // --- PlaybackModeControl integration ---

  it("renders mode control as the active summary level when isSummarized=true", () => {
    render(<AudioPlayerBar {...makeProps({
      isSummarized: true,
      hasOriginalVersion: true,
      canSummarize: true,
      currentLevel: "heavy",
      onToggleSummarized: vi.fn(),
      onChangeLevel: vi.fn(),
    })} />);
    const ctrl = screen.getByTestId("tts-mode-control");
    expect(ctrl.textContent).toContain(strings.playbackMode.heavy);
  });

  it("renders mode control as 'Original' when not summarized but has original version", () => {
    render(<AudioPlayerBar {...makeProps({
      isSummarized: false,
      hasOriginalVersion: true,
      canSummarize: true,
      onToggleSummarized: vi.fn(),
      onChangeLevel: vi.fn(),
    })} />);
    const ctrl = screen.getByTestId("tts-mode-control");
    expect(ctrl.textContent).toContain(strings.playbackMode.original);
  });

  it("renders mode control as 'Summarize' when no summary exists but canSummarize", () => {
    render(<AudioPlayerBar {...makeProps({
      isSummarized: false,
      hasOriginalVersion: false,
      canSummarize: true,
      onChangeLevel: vi.fn(),
    })} />);
    const ctrl = screen.getByTestId("tts-mode-control");
    expect(ctrl.textContent).toContain(strings.playbackMode.summarize);
  });

  it("hides mode control when !hasOriginal && !canSummarize", () => {
    render(<AudioPlayerBar {...makeProps({
      isSummarized: false,
      hasOriginalVersion: false,
      canSummarize: false,
    })} />);
    expect(screen.queryByTestId("tts-mode-control")).toBeNull();
  });

  it("clicking mode control opens dropdown with level options", () => {
    render(<AudioPlayerBar {...makeProps({
      isSummarized: true,
      hasOriginalVersion: true,
      canSummarize: true,
      currentLevel: "moderate",
      onToggleSummarized: vi.fn(),
      onChangeLevel: vi.fn(),
    })} />);
    fireEvent.click(screen.getByTestId("tts-mode-control"));
    expect(screen.getByTestId("tts-mode-menu")).toBeInTheDocument();
    expect(screen.getByTestId("tts-mode-option-original")).toBeInTheDocument();
    expect(screen.getByTestId("tts-mode-option-light")).toBeInTheDocument();
    expect(screen.getByTestId("tts-mode-option-moderate")).toBeInTheDocument();
    expect(screen.getByTestId("tts-mode-option-heavy")).toBeInTheDocument();
  });

  it("selecting 'Original' from dropdown calls onToggleSummarized(false)", () => {
    const onToggleSummarized = vi.fn();
    render(<AudioPlayerBar {...makeProps({
      isSummarized: true,
      hasOriginalVersion: true,
      canSummarize: true,
      currentLevel: "moderate",
      onToggleSummarized,
      onChangeLevel: vi.fn(),
    })} />);
    fireEvent.click(screen.getByTestId("tts-mode-control"));
    fireEvent.click(screen.getByTestId("tts-mode-option-original"));
    expect(onToggleSummarized).toHaveBeenCalledWith(false);
  });

  it("selecting a different level calls onChangeLevel with that level", () => {
    const onChangeLevel = vi.fn();
    render(<AudioPlayerBar {...makeProps({
      isSummarized: true,
      hasOriginalVersion: true,
      canSummarize: true,
      currentLevel: "moderate",
      onChangeLevel,
      onToggleSummarized: vi.fn(),
    })} />);
    fireEvent.click(screen.getByTestId("tts-mode-control"));
    fireEvent.click(screen.getByTestId("tts-mode-option-heavy"));
    expect(onChangeLevel).toHaveBeenCalledWith("heavy");
  });

  it("selecting the current level is a no-op", () => {
    const onChangeLevel = vi.fn();
    render(<AudioPlayerBar {...makeProps({
      isSummarized: true,
      hasOriginalVersion: true,
      canSummarize: true,
      currentLevel: "moderate",
      onChangeLevel,
      onToggleSummarized: vi.fn(),
    })} />);
    fireEvent.click(screen.getByTestId("tts-mode-control"));
    fireEvent.click(screen.getByTestId("tts-mode-option-moderate"));
    expect(onChangeLevel).not.toHaveBeenCalled();
  });

  it("disables mode control when isSummarizing", () => {
    render(<AudioPlayerBar {...makeProps({
      isSummarized: false,
      canSummarize: true,
      isSummarizing: true,
      onChangeLevel: vi.fn(),
    })} />);
    const ctrl = screen.getByTestId("tts-mode-control");
    expect(ctrl).toBeDisabled();
  });

  // --- Overflow regression: no layout-reserved elements that push buttons off-screen ---

  it("scrub bar has min-w-0 so it can shrink (no overflow)", () => {
    render(<AudioPlayerBar {...makeProps()} />);
    const scrub = screen.getByTestId("tts-scrub");
    expect(scrub.className).toMatch(/min-w-0/);
  });

  it("mode control is disabled in idle/replay state (duration=null)", () => {
    render(<AudioPlayerBar {...makeProps({
      duration: null,
      isSummarized: true,
      hasOriginalVersion: true,
      canSummarize: true,
      onToggleSummarized: vi.fn(),
      onChangeLevel: vi.fn(),
    })} />);
    expect(screen.getByTestId("tts-mode-control")).toBeDisabled();
  });

  it("mode control stays enabled while audio is playing (duration set)", () => {
    render(<AudioPlayerBar {...makeProps({
      duration: 60,
      isSummarized: true,
      hasOriginalVersion: true,
      canSummarize: true,
      onToggleSummarized: vi.fn(),
      onChangeLevel: vi.fn(),
    })} />);
    expect(screen.getByTestId("tts-mode-control")).not.toBeDisabled();
  });

  it("time display is visible at all widths and does not wrap", () => {
    render(<AudioPlayerBar {...makeProps()} />);
    const time = screen.getByTestId("tts-time");
    expect(time.className).not.toMatch(/\bhidden\b/);
    expect(time.className).toMatch(/whitespace-nowrap/);
  });

  it("does NOT render the old standalone speed button on the bar", () => {
    render(<AudioPlayerBar {...makeProps()} />);
    // The speed button used to sit on the bar itself; now it lives in the popover.
    // The only tts-speed testids should be the presets inside the popover (once opened).
    expect(screen.queryByTestId("tts-speed")).toBeNull();
  });

  // --- Mute behavior ---

  it("renders muted icon when isMuted=true", () => {
    render(<AudioPlayerBar {...makeProps({ isMuted: true })} />);
    expect(screen.getByTestId("tts-audio-button").getAttribute("title")).toBe("audioPlayerBar.unmute");
  });

  it("renders unmuted icon when isMuted=false (regardless of volume)", () => {
    render(<AudioPlayerBar {...makeProps({ isMuted: false, volume: 0 })} />);
    expect(screen.getByTestId("tts-audio-button").getAttribute("title")).toBe("audioPlayerBar.audioSettings");
  });

  it("clicking the audio button while muted calls onSetMuted(false) and does NOT open the popover", () => {
    const props = makeProps({ isMuted: true });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-audio-button"));
    expect(props.onSetMuted).toHaveBeenCalledWith(false);
    expect(screen.queryByTestId("audio-popover")).toBeNull();
  });

  it("clicking the audio button while unmuted opens the popover and does NOT call onSetMuted", () => {
    const props = makeProps({ isMuted: false });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-audio-button"));
    expect(screen.getByTestId("audio-popover")).toBeInTheDocument();
    expect(props.onSetMuted).not.toHaveBeenCalled();
  });

  it("popover exposes a mute toggle that calls onSetMuted with the negation", () => {
    const props = makeProps({ isMuted: false });
    render(<AudioPlayerBar {...props} />);
    fireEvent.click(screen.getByTestId("tts-audio-button"));
    fireEvent.click(screen.getByTestId("tts-mute-toggle"));
    expect(props.onSetMuted).toHaveBeenCalledWith(true);
  });

  it("renders the current message affordance and forwards jump clicks", () => {
    const props = makeProps({
      currentMessageLabel: "#12",
      hasQueuedNext: true,
      onJumpToCurrentMessage: vi.fn(),
    });
    render(<AudioPlayerBar {...props} />);
    const button = screen.getByTestId("tts-current-message");
    expect(button.textContent).toContain("#12");
    expect(button.textContent).toContain("next");
    fireEvent.click(button);
    expect(props.onJumpToCurrentMessage).toHaveBeenCalledTimes(1);
  });

  it("opens the shared message selector from the current message affordance", () => {
    const events = [makeEvent("e1", 1), makeEvent("e2", 2)];
    const props = makeProps({
      currentMessageLabel: "1/2",
      currentMessageId: "e1",
      messageSelectorEvents: events,
      onSelectMessage: vi.fn(),
      onJumpToCurrentMessage: vi.fn(),
    });
    render(<AudioPlayerBar {...props} />);

    fireEvent.click(screen.getByTestId("tts-current-message"));

    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("msg-jump-item-e2"));
    expect(props.onSelectMessage).toHaveBeenCalledWith("e2");
    expect(props.onJumpToCurrentMessage).not.toHaveBeenCalled();
  });

});
