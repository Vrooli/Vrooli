import { act, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import { VideoPreview } from "../renderers/VideoPreview";

const c = strings.artifactPreview;
const a = strings.audioFilePlayer;
const props = previewProps({ model: previewModel({ kind: "video", mimeType: "video/mp4", basename: "clip.mp4" }) });
function open() {
  const result = render(<VideoPreview {...props} />);
  const video = result.container.querySelector("video");
  if (!video) throw new Error("Missing video transport");
  Object.defineProperties(video, { duration: { configurable: true, value: 20 }, videoWidth: { value: 1920 }, videoHeight: { value: 1080 } });
  fireEvent.loadedMetadata(video);
  return { ...result, video, play: vi.spyOn(video, "play"), pause: vi.spyOn(video, "pause") };
}
afterEach(() => { vi.restoreAllMocks(); });

describe("video preview", () => {
  it("uses inline native streaming with custom transport, seek/skip, rate and repeat", async () => {
    const { video, play } = open();
    expect(video).toHaveAttribute("playsinline");
    expect(video).toHaveAttribute("preload", "metadata");
    expect(video).not.toHaveAttribute("autoplay");
    expect(screen.getByText(/1920 × 1080/)).toBeInTheDocument();
    fireEvent.change(screen.getByRole("slider", { name: a.seek }), { target: { value: "5" } });
    fireEvent.click(screen.getByRole("button", { name: a.forward }));
    expect(video.currentTime).toBe(15);
    fireEvent.click(screen.getByRole("button", { name: a.back }));
    expect(video.currentTime).toBe(5);
    await act(async () => { fireEvent.click(screen.getByTestId("artifact-video-play")); await Promise.resolve(); });
    expect(play).toHaveBeenCalled();
    fireEvent.play(video);
    expect(screen.getByTestId("artifact-video-play")).toHaveAttribute("aria-label", a.pause);
    fireEvent.change(screen.getByRole("combobox", { name: a.speed }), { target: { value: "1.5" } });
    expect(video.playbackRate).toBe(1.5);
    fireEvent.click(screen.getByRole("button", { name: strings.artifactPreview.repeatVideo }));
    expect(video.loop).toBe(true);
  });
  it("handles iPhone's native fullscreen entry and preserves system controls", async () => {
    const { video } = open();
    const nativeFullscreen = vi.fn();
    Object.defineProperty(document, "fullscreenEnabled", { configurable: true, value: false });
    video.webkitEnterFullscreen = nativeFullscreen;
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: c.fullscreen })); await Promise.resolve(); });
    expect(nativeFullscreen).toHaveBeenCalled();
    expect(video.controls).toBe(true);
  });
  it("keeps download on codec failure and releases media when the drawer closes", () => {
    const { video, pause, unmount } = open();
    fireEvent.error(video);
    expect(screen.getByRole("alert")).toHaveTextContent(strings.messagesFileViewer.codecHint);
    expect(screen.getByTestId("artifact-video-play")).toBeDisabled();
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/preview/blob");
    unmount();
    expect(pause).toHaveBeenCalled();
    expect(video).not.toHaveAttribute("src");
  });
});
