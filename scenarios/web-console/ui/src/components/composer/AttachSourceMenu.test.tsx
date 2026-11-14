import { renderWithProviders as render } from "../../test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import AttachSourceMenu from "./AttachSourceMenu";
import type { AttachEnvironment } from "../../lib/attachments";

vi.mock("../../api/filePreview", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/filePreview")>();
  return {
    ...actual,
    resolveFilePreview: vi.fn(() => new Promise(() => { /* the picker stays loading */ })),
  };
});

const originalMediaDevices = Object.getOwnPropertyDescriptor(navigator, "mediaDevices");

afterEach(() => {
  if (originalMediaDevices) Object.defineProperty(navigator, "mediaDevices", originalMediaDevices);
  else Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: undefined });
  vi.restoreAllMocks();
});

const IPHONE: AttachEnvironment = { userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)", hostname: "console.example.dev", maxTouchPoints: 5 };
const IPAD: AttachEnvironment = { userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", hostname: "console.example.dev", maxTouchPoints: 5 };
const REMOTE_DESKTOP: AttachEnvironment = { userAgent: "Mozilla/5.0 (X11; Linux x86_64) Chrome/130.0", hostname: "console.example.dev", maxTouchPoints: 0 };
const ON_HOST: AttachEnvironment = { ...REMOTE_DESKTOP, hostname: "localhost" };

function withCamera(devices: MediaDeviceInfo[] = [{ kind: "videoinput" } as MediaDeviceInfo]) {
  const getUserMedia = vi.fn(() => Promise.resolve({ getTracks: () => [{ stop: vi.fn() }] }));
  Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia, enumerateDevices: vi.fn(() => Promise.resolve(devices)) } });
}

function menuIds(): string[] {
  return screen.getAllByRole("menuitem").map((el) => el.getAttribute("data-testid") ?? "");
}

describe("AttachSourceMenu", () => {
  it("on a phone, offers the phone's own picker and the host — no in-app camera", () => {
    withCamera();
    render(<AttachSourceMenu open onOpenChange={vi.fn()} onFilesPicked={vi.fn()} hostSessionId="s1" onHostPathsPicked={vi.fn()} environment={IPHONE} />);
    expect(menuIds()).toEqual(["composer-attach-device", "composer-attach-host"]);
    expect(screen.getByTestId("composer-attach-device").textContent).toBe("composer.attachDeviceIphone");
  });

  it("opens one plain file input for the phone so the OS shows its full chooser", () => {
    render(<AttachSourceMenu open onOpenChange={vi.fn()} onFilesPicked={vi.fn()} environment={IPHONE} />);
    const input = screen.getByTestId("composer-file-input");
    const click = vi.spyOn(input, "click");
    fireEvent.click(screen.getByTestId("composer-attach-device"));
    expect(click).toHaveBeenCalled();
    // accept/capture would narrow iOS's chooser to one source.
    expect(input.hasAttribute("accept")).toBe(false);
    expect(input.hasAttribute("capture")).toBe(false);
    expect(input.hasAttribute("multiple")).toBe(true);
  });

  it("names an iPad in desktop mode as an iPad", () => {
    render(<AttachSourceMenu open onOpenChange={vi.fn()} onFilesPicked={vi.fn()} environment={IPAD} />);
    expect(screen.getByTestId("composer-attach-device").textContent).toBe("composer.attachDeviceIpad");
  });

  it("on a remote desktop browser, offers upload, the in-app camera, and the host", async () => {
    withCamera();
    render(<AttachSourceMenu open onOpenChange={vi.fn()} onFilesPicked={vi.fn()} hostSessionId="s1" onHostPathsPicked={vi.fn()} environment={REMOTE_DESKTOP} />);
    await waitFor(() => expect(menuIds()).toEqual(["composer-attach-upload", "composer-attach-camera", "composer-attach-host"]));
    expect(screen.getByTestId("composer-attach-host").textContent).toBe("composer.attachHost");
  });

  it("hides the in-app camera when the device has no camera", async () => {
    withCamera([{ kind: "audioinput" } as MediaDeviceInfo]);
    render(<AttachSourceMenu open onOpenChange={vi.fn()} onFilesPicked={vi.fn()} environment={REMOTE_DESKTOP} />);
    await waitFor(() => expect(screen.queryByTestId("composer-attach-camera")).toBeNull());
    expect(menuIds()).toEqual(["composer-attach-upload"]);
  });

  it("on the host itself, puts browse-by-path first and calls the dialog a copy", () => {
    render(<AttachSourceMenu open onOpenChange={vi.fn()} onFilesPicked={vi.fn()} hostSessionId="s1" onHostPathsPicked={vi.fn()} environment={ON_HOST} />);
    expect(menuIds()).toEqual(["composer-attach-host", "composer-attach-upload"]);
    expect(screen.getByTestId("composer-attach-host").textContent).toBe("composer.attachHostBrowse");
    expect(screen.getByTestId("composer-attach-upload").textContent).toBe("composer.attachUploadCopy");
  });

  it("opens the camera dialog when Camera is chosen, even though that closes the menu", async () => {
    withCamera();
    vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
    const onOpenChange = vi.fn();
    render(<AttachSourceMenu open onOpenChange={onOpenChange} onFilesPicked={vi.fn()} environment={REMOTE_DESKTOP} />);

    fireEvent.click(screen.getByTestId("composer-attach-camera"));

    // The menu closes to make room for the camera...
    expect(onOpenChange).toHaveBeenCalledWith(false);
    // ...and the camera dialog must still open.
    await waitFor(() => expect(screen.getByTestId("camera-capture")).toBeTruthy());
  });

  it("opens the host file picker when the host is chosen", async () => {
    const onOpenChange = vi.fn();
    render(<AttachSourceMenu open onOpenChange={onOpenChange} onFilesPicked={vi.fn()} hostSessionId="s1" onHostPathsPicked={vi.fn()} environment={IPHONE} />);
    fireEvent.click(screen.getByTestId("composer-attach-host"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    await waitFor(() => expect(screen.getByTestId("host-file-picker")).toBeTruthy());
  });
});
