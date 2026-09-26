import { act, fireEvent, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "i18next";
import type { ConversationEvent } from "../../api/conversation";
import { publishTransport, type PlaybackTransport } from "../../domains/tts-playback/transport";
import { renderWithProviders as render, setDesktopViewport } from "../../test-utils";
import { PlaybackPill, type PlaybackPillProps } from "./PlaybackPill";

const SESSION = "s-pill";

function todayAt(hours: number, minutes: number): string {
  const at = new Date();
  at.setHours(hours, minutes, 0, 0);
  return at.toISOString();
}

const event: ConversationEvent = {
  id: "e1",
  sessionId: SESSION,
  source: "claude_hook",
  role: "assistant",
  text: "Here is the plan.",
  speechParagraphs: ["Here is the plan."],
  summarized: false,
  sequence: 7,
  createdAt: todayAt(14, 43),
  deliveryState: "delivered",
  ttsState: "idle",
  consumptionState: "seen",
};

function transport(overrides: Partial<PlaybackTransport> = {}): PlaybackTransport {
  return {
    currentTime: 42,
    duration: 130,
    playbackRate: 1,
    volume: 1,
    isMuted: false,
    isPaused: false,
    capabilities: { canPause: true, canSeek: true, canAdjustSpeed: true, canAdjustVolume: true },
    ...overrides,
  };
}

function props(overrides: Partial<PlaybackPillProps> = {}): PlaybackPillProps {
  return {
    sessionId: SESSION,
    event,
    expanded: false,
    onExpandedChange: vi.fn(),
    isSpeaking: true,
    queuedAfter: 2,
    canPrevious: true,
    canNext: true,
    voiceName: "af_heart",
    summarize: { isSummarized: false, hasOriginalVersion: false, canSummarize: true, isSummarizing: false, currentLevel: "moderate" },
    onPause: vi.fn(),
    onResume: vi.fn(),
    onSeek: vi.fn(),
    onPrevious: vi.fn(),
    onNext: vi.fn(),
    onStop: vi.fn(),
    onJumpToMessage: vi.fn(),
    onSetPlaybackRate: vi.fn(),
    onSetVolume: vi.fn(),
    onSetMuted: vi.fn(),
    ...overrides,
  };
}

function swipe(target: HTMLElement, points: Array<[number, number]>) {
  const [first, ...rest] = points;
  if (!first) return;
  fireEvent.pointerDown(target, { clientX: first[0], clientY: first[1], pointerId: 1, pointerType: "touch" });
  for (const [x, y] of rest) fireEvent.pointerMove(target, { clientX: x, clientY: y, pointerId: 1, pointerType: "touch" });
  const last = rest[rest.length - 1] ?? first;
  fireEvent.pointerUp(target, { clientX: last[0], clientY: last[1], pointerId: 1, pointerType: "touch" });
}

describe("PlaybackPill", () => {
  beforeEach(async () => {
    setDesktopViewport();
    await i18n.changeLanguage("en");
    act(() => { publishTransport(SESSION, transport()); });
  });

  afterEach(async () => {
    act(() => { publishTransport(SESSION, null); });
    await i18n.changeLanguage("cimode");
  });

  it("[REQ:P0-017h] collapsed: play/pause, equalizer, speaker · time, elapsed / total, close, progress hairline", () => {
    render(<PlaybackPill {...props()} />);
    expect(screen.getByTestId("playback-pill")).toHaveAttribute("data-state", "collapsed");
    expect(screen.getByTestId("pill-play-pause")).toHaveAccessibleName("Pause");
    expect(screen.getByTestId("pill-equalizer")).toBeInTheDocument();
    expect(screen.getByTestId("pill-label")).toHaveTextContent("Claude · 2:43 PM");
    expect(screen.getByTestId("pill-time")).toHaveTextContent("0:42 / 2:10");
    expect(screen.getByTestId("pill-close")).toBeInTheDocument();
    expect(screen.getByTestId("pill-progress-fill").style.width).toBe("32.3%");
    expect(screen.queryByTestId("pill-scrub")).toBeNull();
  });

  it("follows the provider's transport as it changes", () => {
    render(<PlaybackPill {...props()} />);
    act(() => { publishTransport(SESSION, transport({ currentTime: 65, isPaused: true })); });
    expect(screen.getByTestId("pill-time")).toHaveTextContent("1:05 / 2:10");
    expect(screen.getByTestId("pill-play-pause")).toHaveAccessibleName("Play");
    expect(screen.queryByTestId("pill-equalizer")).toBeNull();
  });

  it("play/pause pauses while speaking and resumes when paused", () => {
    const onPause = vi.fn();
    const onResume = vi.fn();
    render(<PlaybackPill {...props({ onPause, onResume })} />);
    fireEvent.click(screen.getByTestId("pill-play-pause"));
    expect(onPause).toHaveBeenCalledTimes(1);
    act(() => { publishTransport(SESSION, transport({ isPaused: true })); });
    fireEvent.click(screen.getByTestId("pill-play-pause"));
    expect(onResume).toHaveBeenCalledTimes(1);
  });

  it("[REQ:P0-017h] a tap toggles the pill to expanded", () => {
    const onExpandedChange = vi.fn();
    render(<PlaybackPill {...props({ onExpandedChange })} />);
    fireEvent.click(screen.getByTestId("pill-summary"));
    expect(onExpandedChange).toHaveBeenLastCalledWith(true);
    swipe(screen.getByTestId("playback-pill"), [[100, 20], [102, 21]]);
    expect(onExpandedChange).toHaveBeenCalledTimes(2);
  });

  it("[REQ:P0-017h] expanded: scrub, previous, next, rate, summarize, voice, jump to message, settings, close, queue", () => {
    render(<PlaybackPill {...props({ expanded: true })} />);
    expect(screen.getByTestId("playback-pill")).toHaveAttribute("data-state", "expanded");
    expect(screen.getByTestId("pill-scrub")).toBeInTheDocument();
    expect(screen.getByTestId("pill-previous")).toBeInTheDocument();
    expect(screen.getByTestId("pill-next")).toBeInTheDocument();
    expect(within(screen.getByTestId("pill-rate")).getByRole("button")).toHaveTextContent("1×");
    expect(screen.getByTestId("pill-mode-control")).toBeInTheDocument();
    expect(screen.getByTestId("pill-voice")).toHaveTextContent("af_heart");
    expect(within(screen.getByTestId("pill-jump")).getByRole("button")).toHaveTextContent("Jump to message");
    expect(screen.getByTestId("pill-settings")).toBeInTheDocument();
    expect(screen.getByTestId("pill-close")).toBeInTheDocument();
    expect(screen.getByTestId("pill-queued")).toHaveTextContent("2 more queued");
  });

  it("expanded controls call through: previous, next, rate, jump", () => {
    const p = props({ expanded: true });
    render(<PlaybackPill {...p} />);
    fireEvent.click(screen.getByTestId("pill-previous"));
    fireEvent.click(screen.getByTestId("pill-next"));
    fireEvent.click(within(screen.getByTestId("pill-rate")).getByRole("button"));
    fireEvent.click(within(screen.getByTestId("pill-jump")).getByRole("button"));
    expect(p.onPrevious).toHaveBeenCalledTimes(1);
    expect(p.onNext).toHaveBeenCalledTimes(1);
    expect(p.onSetPlaybackRate).toHaveBeenCalledWith(1.25);
    expect(p.onJumpToMessage).toHaveBeenCalledTimes(1);
  });

  it("[REQ:P0-017h] the gear opens the audio settings in a dialog", () => {
    render(<PlaybackPill {...props({ expanded: true })} />);
    expect(screen.queryByTestId("pill-settings-dialog")).toBeNull();
    fireEvent.click(screen.getByTestId("pill-settings"));
    expect(screen.getByTestId("pill-settings-dialog")).toBeInTheDocument();
  });

  it("[REQ:P0-017h] close and a swipe down stop playback", () => {
    const onStop = vi.fn();
    render(<PlaybackPill {...props({ onStop })} />);
    fireEvent.click(screen.getByTestId("pill-close"));
    expect(onStop).toHaveBeenCalledTimes(1);
    swipe(screen.getByTestId("playback-pill"), [[100, 20], [101, 50], [102, 80]]);
    expect(onStop).toHaveBeenCalledTimes(2);
  });

  it("[REQ:P0-017h] on the collapsed pill a swipe left plays the next message and a swipe right the previous", () => {
    const p = props();
    render(<PlaybackPill {...p} />);
    const pill = screen.getByTestId("playback-pill");
    swipe(pill, [[200, 20], [150, 21], [110, 22]]);
    expect(p.onNext).toHaveBeenCalledTimes(1);
    swipe(pill, [[100, 20], [150, 21], [190, 22]]);
    expect(p.onPrevious).toHaveBeenCalledTimes(1);
  });

  it("[REQ:P0-017h] a drag along the collapsed progress zone seeks", () => {
    const onSeek = vi.fn();
    render(<PlaybackPill {...props({ onSeek })} />);
    const zone = screen.getByTestId("pill-progress");
    zone.getBoundingClientRect = () => ({ left: 0, width: 200, top: 0, height: 8, right: 200, bottom: 8, x: 0, y: 0, toJSON: () => ({}) });
    swipe(zone, [[20, 4], [60, 4], [100, 4]]);
    expect(onSeek).toHaveBeenLastCalledWith(65);
  });

  it("[REQ:P0-017h] prefers-reduced-motion removes the slide", () => {
    const { unmount } = render(<PlaybackPill {...props()} />);
    expect(screen.getByTestId("playback-pill").className).toContain("slide-in-from-bottom");
    unmount();
    const matchMedia = window.matchMedia;
    window.matchMedia = vi.fn((query: string) => ({
      matches: query.includes("prefers-reduced-motion") || matchMedia(query).matches,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(() => false),
    }));
    render(<PlaybackPill {...props()} />);
    expect(screen.getByTestId("playback-pill").className).not.toContain("slide-in-from-bottom");
    window.matchMedia = matchMedia;
  });
});
