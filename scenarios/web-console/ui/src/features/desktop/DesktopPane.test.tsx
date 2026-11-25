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
  it.each([375, 1280])("uses the shared pane and toolbar contract at %i px", (width) => {
    render(<div style={{ width }}><DesktopPane readiness={ready} session={connectedViewer} onReconnect={() => undefined} onTakeControl={() => undefined} /></div>);
    expect(screen.getByTestId("workspace-surface-pane")).toHaveAttribute("data-state", "observe");
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "observe");
    const toolbar = screen.getByRole("toolbar", { name: "Desktop controls" });
    expect(toolbar).toHaveAttribute("data-pane-state", "observe");
    const controls = Array.from(toolbar.querySelectorAll<HTMLButtonElement>("button:not(:disabled)"));
    expect(controls[0]).toHaveAttribute("tabindex", "0");
    controls.slice(1).forEach((control) => { expect(control).toHaveAttribute("tabindex", "-1"); });
    expect(screen.getByRole("status")).toHaveTextContent(/Remote desktop: Observe mode/);
  });

  it("offers explicit controller and viewer entry points once readiness is proven", () => {
    const onOpen = vi.fn();
    const onOpenViewer = vi.fn();
    render(<DesktopPane readiness={ready} onOpen={onOpen} onOpenViewer={onOpenViewer} />);

    fireEvent.click(screen.getByRole("button", { name: "Open with control" }));
    fireEvent.click(screen.getByRole("button", { name: "View only" }));
    expect(onOpen).toHaveBeenCalledOnce();
    expect(onOpenViewer).toHaveBeenCalledOnce();
    expect(screen.getByTestId("workspace-surface-pane")).toHaveClass("min-w-0", "w-full");
    expect(screen.getByTestId("workspace-surface-pane")).toHaveAttribute("data-state", "ready");
  });

  it("offers owner-backed release control only to the controller", () => {
    const onReleaseControl = vi.fn();
    const controller = { ...connectedViewer, controller: true } as DesktopSession;
    render(<DesktopPane readiness={ready} session={controller} onReleaseControl={onReleaseControl} />);
    fireEvent.click(screen.getByRole("button", { name: "Release control" }));
    expect(onReleaseControl).toHaveBeenCalledOnce();
    expect(screen.queryByRole("button", { name: "Take control" })).not.toBeInTheDocument();
  });

  it("refreshes blocked readiness instead of attempting to open a session", () => {
    const onRefresh = vi.fn();
    const onOpen = vi.fn();
    render(<DesktopPane readiness={{ state: 2, recovery: "Sign in to the desktop session." } as unknown as DesktopReadiness} onRefresh={onRefresh} onOpen={onOpen} />);

    fireEvent.click(screen.getByRole("button", { name: "Check readiness" }));
    expect(onRefresh).toHaveBeenCalledOnce();
    expect(onOpen).not.toHaveBeenCalled();
  });

  it("keeps a close-session escape after controller revocation", () => {
    const onClose = vi.fn();
    const revoked = { ...connectedViewer, connectionState: "revoked", controller: true } as unknown as DesktopSession;
    render(<DesktopPane readiness={ready} session={revoked} onClose={onClose} />);
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "revoked");
    expect(screen.getByText(/controller lease was revoked/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close session" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

	it("keeps mutating controls disabled for a viewer session", () => {
    const onInput = vi.fn();
    render(<DesktopPane readiness={ready} session={connectedViewer} onInput={onInput} onClipboardToDesktop={vi.fn()} onClipboardFromDesktop={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Send text" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Read text" })).toBeDisabled();
    expect(screen.getByText(/Viewer mode/)).toBeInTheDocument();
    expect(screen.getByTestId("workspace-surface-pane")).toHaveAttribute("data-state", "observe");
    expect(screen.getByRole("toolbar", { name: "Desktop controls" })).toHaveAttribute("data-pane-state", "observe");
    fireEvent.pointerDown(screen.getByLabelText("Selected desktop display"));
    expect(onInput).not.toHaveBeenCalled();
	});

	it("exposes counter-only transport statistics", () => {
		render(<DesktopPane readiness={ready} session={{ ...connectedViewer, transportStats: { framesSent: 2n, framesDropped: 1n, bytesSent: 1024n, controlMessagesRejected: 1n, dataMessagesRejected: 2n } } as unknown as DesktopSession} />);
		expect(screen.getByLabelText("Desktop transport statistics")).toHaveTextContent("Frames 2 · dropped 1 · bytes 1024 · rejected 3");
	});

  it("wires pointer, wheel, and key release events to the typed input callback", () => {
    const onInput = vi.fn();
    const onReleaseInput = vi.fn();
    const controller = { ...connectedViewer, controller: true } as DesktopSession;
    render(<DesktopPane readiness={ready} session={controller} onInput={onInput} onReleaseInput={onReleaseInput} />);
    const video = screen.getByLabelText("Selected desktop display");
    const setPointerCapture = vi.fn();
    Object.defineProperty(video, "setPointerCapture", { configurable: true, value: setPointerCapture });

    fireEvent.pointerDown(video, { pointerId: 8, clientX: 10, clientY: 20, button: 0 });
    fireEvent.pointerCancel(video, { pointerId: 8, clientX: 10, clientY: 20, button: -1 });
    fireEvent.wheel(video, { deltaY: 100, clientX: 10, clientY: 20 });
    fireEvent.keyUp(video, { key: "Shift" });
    fireEvent.blur(video);
    expect(setPointerCapture).toHaveBeenCalledWith(8);
    expect(onInput).toHaveBeenCalledTimes(4);
    expect(onReleaseInput).toHaveBeenCalledOnce();
  });

  it("keeps Tab available to leave the controlled device surface while reserving other keys for the device", () => {
    const onInput = vi.fn();
    render(<DesktopPane readiness={ready} session={{ ...connectedViewer, controller: true } as DesktopSession} onInput={onInput} />);
    const video = screen.getByLabelText("Selected desktop display");
    const documentKeydown = vi.fn();
    document.addEventListener("keydown", documentKeydown);
    try {
      const tab = new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true });
      video.dispatchEvent(tab);
      expect(tab.defaultPrevented).toBe(false);

      const arrow = new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true });
      video.dispatchEvent(arrow);
      expect(arrow.defaultPrevented).toBe(true);
      expect(documentKeydown).toHaveBeenCalledTimes(1);
    } finally { document.removeEventListener("keydown", documentKeydown); }
    expect(onInput).toHaveBeenCalledOnce();
    expect(onInput.mock.calls[0]?.[0]).toMatchObject({ key: "ArrowDown" });
  });
});
