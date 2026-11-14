import { renderWithProviders as render } from "../../test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import CameraCaptureDialog from "./CameraCaptureDialog";

const originalMediaDevices = Object.getOwnPropertyDescriptor(navigator, "mediaDevices");

function setMediaDevices(value: unknown) {
  Object.defineProperty(navigator, "mediaDevices", { configurable: true, value });
}

afterEach(() => {
  if (originalMediaDevices) Object.defineProperty(navigator, "mediaDevices", originalMediaDevices);
  else Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: undefined });
  Reflect.deleteProperty(globalThis, "MediaRecorder");
  vi.restoreAllMocks();
});

describe("CameraCaptureDialog", () => {
  it("degrades to the file picker when the camera is unavailable", async () => {
    setMediaDevices(undefined);
    const onUseFilePicker = vi.fn();
    render(
      <CameraCaptureDialog open onClose={() => {}} onCapture={() => {}} onUseFilePicker={onUseFilePicker} />,
    );
    await waitFor(() => expect(screen.getByTestId("camera-capture-error")).toBeTruthy());
    fireEvent.click(screen.getByTestId("camera-use-file-picker"));
    expect(onUseFilePicker).toHaveBeenCalled();
  });

  it("captures a photo to a PNG file", async () => {
    const stop = vi.fn();
    const getUserMedia = vi.fn(() => Promise.resolve({ getTracks: () => [{ stop }] }));
    setMediaDevices({ getUserMedia });

    vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
    Object.defineProperty(HTMLVideoElement.prototype, "videoWidth", { configurable: true, get: () => 640 });
    Object.defineProperty(HTMLVideoElement.prototype, "videoHeight", { configurable: true, get: () => 480 });
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({ drawImage: vi.fn() } as unknown as CanvasRenderingContext2D);
    vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((cb: BlobCallback) => {
      cb(new Blob(["pixels"], { type: "image/png" }));
    });

    const onCapture = vi.fn();
    const onClose = vi.fn();
    render(<CameraCaptureDialog open onClose={onClose} onCapture={onCapture} onUseFilePicker={() => {}} />);

    await screen.findByTestId("camera-capture-video");
    await waitFor(() => expect(getUserMedia).toHaveBeenCalled());
    fireEvent.click(screen.getByTestId("camera-capture-photo"));

    await waitFor(() => expect(onCapture).toHaveBeenCalledTimes(1));
    const captured = onCapture.mock.calls[0]?.[0] as File;
    expect(captured.type).toBe("image/png");
    expect(captured.name.endsWith(".png")).toBe(true);
    expect(onClose).toHaveBeenCalled();
  });

  it("records a video clip", async () => {
    const stop = vi.fn();
    const getUserMedia = vi.fn(() => Promise.resolve({ getTracks: () => [{ stop }] }));
    setMediaDevices({ getUserMedia });
    vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);

    class FakeMediaRecorder {
      static isTypeSupported = () => true;
      state: "inactive" | "recording" = "inactive";
      mimeType: string;
      ondataavailable: ((event: { data: Blob }) => void) | null = null;
      onstop: (() => void) | null = null;
      constructor(_stream: unknown, options?: { mimeType?: string }) {
        this.mimeType = options?.mimeType ?? "video/webm";
      }
      start() {
        this.state = "recording";
      }
      stop() {
        this.state = "inactive";
        this.ondataavailable?.({ data: new Blob(["clip"], { type: this.mimeType }) });
        this.onstop?.();
      }
    }
    (globalThis as unknown as { MediaRecorder: unknown }).MediaRecorder = FakeMediaRecorder;

    const onCapture = vi.fn();
    render(<CameraCaptureDialog open onClose={() => {}} onCapture={onCapture} onUseFilePicker={() => {}} />);

    await screen.findByTestId("camera-capture-video");
    await waitFor(() => expect(getUserMedia).toHaveBeenCalled());
    fireEvent.click(screen.getByTestId("camera-mode-video"));
    fireEvent.click(screen.getByTestId("camera-start-recording"));
    expect(screen.getByTestId("camera-stop-recording")).toBeTruthy();
    fireEvent.click(screen.getByTestId("camera-stop-recording"));

    await waitFor(() => expect(onCapture).toHaveBeenCalledTimes(1));
    const captured = onCapture.mock.calls[0]?.[0] as File;
    expect(captured.type.startsWith("video/")).toBe(true);
  });
});
