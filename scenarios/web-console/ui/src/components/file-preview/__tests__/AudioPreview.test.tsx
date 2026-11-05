import { act, fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { StrictMode } from "react";
import { renderWithProviders as render } from "../../../test-utils";
import { strings } from "../../../consts/strings";
import { AudioPreview } from "../renderers/AudioPreview";
import type { PreviewModel, PreviewRendererProps } from "../types";

const analysis = vi.hoisted(() => ({ peaks: [0.2, 0.8, 0.4, 1], loading: false, title: undefined as string | undefined, artist: undefined as string | undefined }));
vi.mock("../useAudioAnalysis", () => ({ useAudioAnalysis: () => analysis }));
const c = strings.audioFilePlayer;
const model: PreviewModel = {
  previewId: "audio-1", inputPath: "/tmp/song.m4a", resolvedPath: "/tmp/song.m4a", basename: "song.m4a", resolutionBasis: "absolute",
  kind: "audio", mimeType: "audio/mp4", sizeBytes: 1024, canPreview: true, canDownload: true, supportsRange: true,
  textContentAvailable: false, listingAvailable: false, blobUrl: "/audio-1", blobHref: "/audio-1", expiresMs: 0, warnings: [],
};
const props: PreviewRendererProps = { model, text: null, listing: null, loadingMore: false, onNavigate: vi.fn(), onLoadMore: vi.fn(), onListOptionsChange: vi.fn(), onError: vi.fn() };

function open(duration = 120) {
  const result = render(<AudioPreview {...props} />);
  const audio = result.container.querySelector("audio");
  if (!audio) throw new Error("Expected native transport");
  Object.defineProperty(audio, "duration", { configurable: true, value: duration });
  fireEvent.loadedMetadata(audio);
  return { ...result, audio, play: vi.spyOn(audio, "play"), pause: vi.spyOn(audio, "pause") };
}
function at(audio: HTMLAudioElement, time: number) {
  audio.currentTime = time;
  fireEvent.timeUpdate(audio);
}

beforeEach(() => { analysis.peaks = [0.2, 0.8, 0.4, 1]; analysis.loading = false; analysis.title = undefined; analysis.artist = undefined; vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(); });

describe("audio preview listening", () => {
  it("prioritizes track identity and supports seeking, skipping and media-event playback state", async () => {
    const { audio, play } = open();
    expect(screen.getByRole("heading", { name: "song.m4a" })).toBeVisible();
    expect(audio).toHaveAttribute("preload", "metadata");
    expect(audio).not.toHaveAttribute("autoplay");
    fireEvent.change(screen.getByRole("slider", { name: c.seek }), { target: { value: "50" } });
    expect(audio.currentTime).toBe(50);
    fireEvent.click(screen.getByRole("button", { name: c.back }));
    expect(audio.currentTime).toBe(40);
    fireEvent.click(screen.getByRole("button", { name: c.forward }));
    expect(audio.currentTime).toBe(50);
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: c.play })); await Promise.resolve(); });
    expect(play).toHaveBeenCalled();
    fireEvent.play(audio);
    expect(screen.getByRole("button", { name: c.pause })).toBeEnabled();
    fireEvent.waiting(audio);
    expect(screen.getByRole("status")).toHaveTextContent(c.buffering);
    fireEvent.playing(audio);
    expect(screen.getByRole("status")).not.toHaveTextContent(c.buffering);
    fireEvent.pause(audio);
    expect(screen.getByRole("button", { name: c.play })).toBeEnabled();
  });

  it("handles unavailable duration without NaN controls and keeps play possible", () => {
    const { audio } = open(Infinity);
    expect(screen.getByRole("slider", { name: c.seek })).toBeDisabled();
    expect(screen.getByRole("button", { name: c.play })).toBeEnabled();
    Object.defineProperty(audio, "duration", { configurable: true, value: 0 });
    fireEvent.durationChange(audio);
    expect(screen.getByRole("slider", { name: c.seek })).toHaveValue("0");
  });

  it("reports a rejected play request, permits retry, and disables broken media", async () => {
    const { audio } = open();
    vi.spyOn(audio, "play").mockRejectedValueOnce(new DOMException("Blocked", "NotAllowedError"));
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: c.play })); await Promise.resolve(); });
    expect(screen.getByRole("alert")).toHaveTextContent(c.playRejected);
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: c.play })); await Promise.resolve(); });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    fireEvent.error(audio);
    expect(screen.getByRole("button", { name: c.play })).toBeDisabled();
    expect(screen.getByRole("alert")).toHaveTextContent(strings.messagesFileViewer.codecHint);
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/audio-1");
  });

  it("discards a late play rejection after the user cancels the pending request", async () => {
    const { audio } = open();
    let rejectPlay: (error: Error) => void = () => {};
    vi.spyOn(audio, "play").mockImplementationOnce(() => new Promise((_resolve, reject) => { rejectPlay = reject; }));
    fireEvent.click(screen.getByRole("button", { name: c.play }));
    fireEvent.click(screen.getByRole("button", { name: c.pause }));
    await act(async () => { rejectPlay(new Error("Late rejection")); await Promise.resolve(); });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: c.play })).toBeEnabled();
  });

  it("keeps seeking available without waveform data", () => {
    analysis.peaks = [];
    const { audio } = open();
    expect(screen.getByRole("status")).toHaveTextContent(c.seekFallback);
    fireEvent.change(screen.getByRole("slider", { name: c.seek }), { target: { value: "30" } });
    expect(audio.currentTime).toBe(30);
  });

  it("stops and releases the old transport and resets controls for a different file", () => {
    const { audio, pause, rerender, unmount } = open();
    fireEvent.click(screen.getByRole("button", { name: c.repeat }));
    fireEvent.change(screen.getByRole("combobox", { name: c.speed }), { target: { value: "1.5" } });
    expect(audio.loop).toBe(true);
    expect(audio.playbackRate).toBe(1.5);
    rerender(<AudioPreview {...props} model={{ ...model, previewId: "audio-2", blobHref: "/audio-2" }} />);
    expect(pause).toHaveBeenCalled();
    expect(audio).not.toHaveAttribute("src");
    expect(screen.getByRole("button", { name: c.repeat })).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("combobox", { name: c.speed })).toHaveValue("1");
    const next = document.querySelector("audio");
    unmount();
    expect(next).not.toHaveAttribute("src");
  });

  it("offers system controls on demand without replacing the playing element", () => {
    const { audio } = open();
    fireEvent.click(screen.getByText(c.moreActions));
    fireEvent.click(screen.getByRole("button", { name: c.systemControls }));
    expect(audio).toBeVisible();
    expect(audio).toHaveAttribute("controls");
    expect(document.querySelectorAll("audio")).toHaveLength(1);
  });

  it("restores the native source after StrictMode's setup/cleanup replay", () => {
    const { container } = render(<StrictMode><AudioPreview {...props} /></StrictMode>);
    expect(container.querySelector("audio")).toHaveAttribute("src", model.blobHref);
  });
});

describe("audio preview inspection", () => {
  function tools() {
    const result = open();
    fireEvent.click(screen.getByText(c.tools));
    return result;
  }

  it("zooms around the playhead, pans the window and seeks inside that window", () => {
    const { audio } = tools();
    at(audio, 90);
    fireEvent.change(screen.getByRole("combobox", { name: c.zoom }), { target: { value: "4" } });
    const seek = screen.getByRole("slider", { name: c.seek });
    expect(seek).toHaveAttribute("min", "75");
    expect(seek).toHaveAttribute("max", "105");
    fireEvent.change(screen.getByRole("slider", { name: c.timeWindow }), { target: { value: "10" } });
    expect(seek).toHaveAttribute("min", "10");
    fireEvent.change(seek, { target: { value: "20" } });
    expect(audio.currentTime).toBe(20);
    fireEvent.change(screen.getByRole("combobox", { name: c.zoom }), { target: { value: "1" } });
    expect(seek).toHaveAttribute("min", "0");
    expect(seek).toHaveAttribute("max", "120");
  });

  it("requires ordered A/B points, repeats only the passage and restores whole-track repeat", () => {
    const { audio } = tools();
    expect(screen.getByRole("button", { name: c.setB })).toBeDisabled();
    at(audio, 10);
    fireEvent.click(screen.getByRole("button", { name: c.setA }));
    expect(screen.getByRole("button", { name: c.setB })).toBeDisabled();
    at(audio, 20);
    fireEvent.click(screen.getByRole("button", { name: c.setB }));
    expect(audio.currentTime).toBe(10);
    expect(screen.getByRole("button", { name: c.loopSelection })).toHaveAttribute("aria-pressed", "true");
    at(audio, 20.1);
    expect(audio.currentTime).toBe(10);
    fireEvent.click(screen.getByRole("button", { name: c.repeat }));
    expect(audio.loop).toBe(true);
    expect(screen.getByRole("button", { name: c.loopSelection })).toHaveAttribute("aria-pressed", "false");
    at(audio, 30);
    expect(audio.currentTime).toBe(30);
    fireEvent.click(screen.getByRole("button", { name: c.loopSelection }));
    expect(audio.loop).toBe(false);
    expect(audio.currentTime).toBe(10);
    fireEvent.change(screen.getByRole("slider", { name: c.seek }), { target: { value: "50" } });
    expect(audio.currentTime).toBe(50);
    expect(screen.getByRole("button", { name: c.loopSelection })).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(screen.getByRole("button", { name: c.clearLoop }));
    expect(screen.getByRole("button", { name: c.loopSelection })).toBeDisabled();
  });

  it("changes speed and volume, and restores the volume after muting", () => {
    const { audio } = tools();
    fireEvent.change(screen.getByRole("combobox", { name: c.speed }), { target: { value: "0.75" } });
    expect(audio.playbackRate).toBe(0.75);
    fireEvent.change(screen.getByRole("slider", { name: c.volume }), { target: { value: "0.4" } });
    expect(audio.volume).toBe(0.4);
    fireEvent.click(screen.getByRole("button", { name: c.mute }));
    expect(audio.muted).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: c.unmute }));
    expect(audio.muted).toBe(false);
    expect(audio.volume).toBe(0.4);
  });
});
