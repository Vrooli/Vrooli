import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { DesktopPane } from "./DesktopPane";

const ready = { state: 6, selectedDisplayId: "display-1" } as unknown as DesktopReadiness;
const connectedViewer = {
  connectionState: "connected",
  controller: false,
  leaseId: "lease-1",
  leaseEpoch: 1n,
  selectedDisplayId: "display-1",
  codec: "VP8",
} as unknown as DesktopSession;

describe("DesktopPane", () => {
  it("offers explicit controller and viewer entry points once readiness is proven", () => {
    const onOpen = vi.fn();
    const onOpenViewer = vi.fn();
    render(<DesktopPane readiness={ready} onOpen={onOpen} onOpenViewer={onOpenViewer} />);

    fireEvent.click(screen.getByRole("button", { name: "Open with control" }));
    fireEvent.click(screen.getByRole("button", { name: "View only" }));
    expect(onOpen).toHaveBeenCalledOnce();
    expect(onOpenViewer).toHaveBeenCalledOnce();
  });

  it("refreshes blocked readiness instead of attempting to open a session", () => {
    const onRefresh = vi.fn();
    const onOpen = vi.fn();
    render(<DesktopPane readiness={{ state: 2, recovery: "Sign in to the desktop session." } as unknown as DesktopReadiness} onRefresh={onRefresh} onOpen={onOpen} />);

    fireEvent.click(screen.getByRole("button", { name: "Check readiness" }));
    expect(onRefresh).toHaveBeenCalledOnce();
    expect(onOpen).not.toHaveBeenCalled();
  });

	it("keeps mutating controls disabled for a viewer session", () => {
    const onInput = vi.fn();
    render(<DesktopPane readiness={ready} session={connectedViewer} onInput={onInput} onClipboardToDesktop={vi.fn()} onClipboardFromDesktop={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Send text" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Read text" })).toBeDisabled();
    expect(screen.getByText(/Viewer mode/)).toBeInTheDocument();
    fireEvent.pointerDown(screen.getByLabelText("Selected desktop display"));
    expect(onInput).not.toHaveBeenCalled();
	});

	it("exposes counter-only transport statistics", () => {
		render(<DesktopPane readiness={ready} session={{ ...connectedViewer, transportStats: { framesSent: 2n, framesDropped: 1n, bytesSent: 1024n, controlMessagesRejected: 1n, dataMessagesRejected: 2n } } as unknown as DesktopSession} />);
		expect(screen.getByLabelText("Desktop transport statistics")).toHaveTextContent("Frames 2 · dropped 1 · bytes 1024 · rejected 3");
	});

  it("wires pointer, wheel, and key release events to the typed input callback", () => {
    const onInput = vi.fn();
    const controller = { ...connectedViewer, controller: true } as DesktopSession;
    render(<DesktopPane readiness={ready} session={controller} onInput={onInput} />);
    const video = screen.getByLabelText("Selected desktop display");

    fireEvent.pointerDown(video, { clientX: 10, clientY: 20, button: 0 });
    fireEvent.wheel(video, { deltaY: 100, clientX: 10, clientY: 20 });
    fireEvent.keyUp(video, { key: "Shift" });
    expect(onInput).toHaveBeenCalledTimes(3);
  });
});
