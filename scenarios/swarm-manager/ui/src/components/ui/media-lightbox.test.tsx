import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MediaLightbox } from "./media-lightbox";

describe("MediaLightbox", () => {
  it("renders nothing when isOpen is false", () => {
    const { container } = render(
      <MediaLightbox isOpen={false} onClose={vi.fn()} src="/video.webm" type="video" />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders video element when type is video", () => {
    render(
      <MediaLightbox isOpen onClose={vi.fn()} src="/video.webm" type="video" label="Test video" />,
    );
    const video = screen.getByTestId("media-lightbox").querySelector("video");
    expect(video).toBeTruthy();
    expect(video!.getAttribute("src")).toBe("/video.webm");
    expect(video!.hasAttribute("controls")).toBe(true);
  });

  it("renders img element when type is image", () => {
    render(
      <MediaLightbox isOpen onClose={vi.fn()} src="/shot.png" type="image" label="Screenshot" />,
    );
    const img = screen.getByAltText("Screenshot");
    expect(img).toBeTruthy();
    expect(img.getAttribute("src")).toBe("/shot.png");
  });

  it("renders label in the top bar", () => {
    render(
      <MediaLightbox isOpen onClose={vi.fn()} src="/shot.png" type="image" label="My label" />,
    );
    expect(screen.getByText("My label")).toBeTruthy();
  });

  it("calls onClose when Escape key is pressed", () => {
    const onClose = vi.fn();
    render(
      <MediaLightbox isOpen onClose={onClose} src="/video.webm" type="video" />,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose when backdrop is clicked", () => {
    const onClose = vi.fn();
    render(
      <MediaLightbox isOpen onClose={onClose} src="/video.webm" type="video" />,
    );
    fireEvent.click(screen.getByTestId("media-lightbox"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does not call onClose when media content is clicked", () => {
    const onClose = vi.fn();
    render(
      <MediaLightbox isOpen onClose={onClose} src="/shot.png" type="image" label="Shot" />,
    );
    fireEvent.click(screen.getByAltText("Shot"));
    expect(onClose).not.toHaveBeenCalled();
  });
});
