import { renderWithProviders as render } from "../../test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import AttachSourceMenu from "./AttachSourceMenu";

const originalMediaDevices = Object.getOwnPropertyDescriptor(navigator, "mediaDevices");

afterEach(() => {
  if (originalMediaDevices) Object.defineProperty(navigator, "mediaDevices", originalMediaDevices);
  else Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: undefined });
  vi.restoreAllMocks();
});

describe("AttachSourceMenu", () => {
  it("opens the camera dialog when Camera is chosen, even though that closes the menu", async () => {
    const getUserMedia = vi.fn(() => Promise.resolve({ getTracks: () => [{ stop: vi.fn() }] }));
    Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia } });
    vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);

    const onOpenChange = vi.fn();
    render(<AttachSourceMenu open onOpenChange={onOpenChange} onFilesPicked={vi.fn()} />);

    fireEvent.click(screen.getByTestId("composer-attach-camera"));

    // The menu closes to make room for the camera...
    expect(onOpenChange).toHaveBeenCalledWith(false);
    // ...and the camera dialog must still open.
    await waitFor(() => expect(screen.getByTestId("camera-capture")).toBeTruthy());
  });

  it("routes Photos and Files to their hidden inputs", () => {
    render(<AttachSourceMenu open onOpenChange={vi.fn()} onFilesPicked={vi.fn()} />);
    const photos = screen.getByTestId("composer-photos-input");
    const files = screen.getByTestId("composer-file-input");
    const photosClick = vi.spyOn(photos, "click");
    const filesClick = vi.spyOn(files, "click");
    fireEvent.click(screen.getByTestId("composer-attach-photos"));
    fireEvent.click(screen.getByTestId("composer-attach-files"));
    expect(photosClick).toHaveBeenCalled();
    expect(filesClick).toHaveBeenCalled();
  });
});
