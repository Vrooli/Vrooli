import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { BrowserPane } from "./BrowserPane";
import type { BrowserSurfaceInput } from "./remoteSurface";

const session = {
  sessionId: "bas-session-17",
  mode: "managed" as const,
  state: "viewer" as const,
  controller: false,
  association: { source: "agent-run" as const, runId: "run-31", threadId: "thread-9", verified: true },
  capabilities: { observe: true, control: true, tabs: true, frames: true },
  viewport: { width: 100, height: 50 },
  frameSize: { width: 200, height: 100 },
  pages: [{ id: "page-1", title: "Docs", url: "https://example.test", active: true }],
};

describe("BrowserPane", () => {
  it.each([375, 1280])("uses the shared pane and toolbar contract at %i px", (width) => {
    render(<div style={{ width }}><BrowserPane session={session} frame={<canvas aria-label="Browser frame" />} onTakeControl={() => undefined} onClose={() => undefined} /></div>);
    expect(screen.getByTestId("workspace-surface-pane")).toHaveAttribute("data-state", "observe");
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "observe");
    const toolbar = screen.getByRole("toolbar", { name: "Browser session controls" });
    expect(toolbar).toHaveAttribute("data-pane-state", "observe");
    const controls = Array.from(toolbar.querySelectorAll<HTMLButtonElement>("button:not(:disabled)"));
    expect(controls[0]).toHaveAttribute("tabindex", "0");
    controls.slice(1).forEach((control) => { expect(control).toHaveAttribute("tabindex", "-1"); });
    expect(screen.getByRole("status")).toHaveTextContent(/Browser: Observe mode/);
  });

  it("shows owner session and verified run/thread context with observer controls", () => {
    const onTakeControl = vi.fn();
    render(<BrowserPane session={session} frame={<canvas aria-label="Browser frame" />} onTakeControl={onTakeControl} />);
    expect(screen.getByText(/Browser session · managed · Verified agent run run-31 · Thread thread-9/)).toBeInTheDocument();
    expect(screen.getByTestId("workspace-surface-pane")).toHaveAttribute("data-state", "observe");
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "observe");
    expect(screen.getByRole("toolbar", { name: "Browser session controls" })).toHaveAttribute("data-pane-state", "observe");
    expect(screen.getByRole("navigation", { name: "Browser tabs" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Take control" }));
    expect(onTakeControl).toHaveBeenCalledOnce();
    expect(screen.getByRole("img", { name: "Browser frame" })).toBeInTheDocument();
  });

  it("announces owner revocation as revoked and keeps recovery explicit", () => {
    render(<BrowserPane session={{ ...session, state: "revoked" }} onClose={() => undefined} />);
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "revoked");
    expect(screen.getByTestId("workspace-pane-state")).toHaveTextContent(/Revoked/);
    expect(screen.getByText(/controller lease was revoked/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Take control" })).not.toBeInTheDocument();
  });

  it("gives an owner-decoded frame a useful accessible image name when the decoder omits one", () => {
    render(<BrowserPane session={session} frame={<canvas className="decoded-frame" style={{ imageRendering: "pixelated" }} />} />);
    const frame = screen.getByRole("img", { name: "Live browser frame" });
    expect(frame).toHaveClass("block", "h-full", "w-full", "decoded-frame");
    expect(frame).toHaveStyle({ width: "100%", height: "100%", objectFit: "contain", imageRendering: "pixelated" });
  });

  it("sizes the live frame region from owner dimensions and bounds it to the viewport height", () => {
    render(<BrowserPane session={session} frame={<canvas />} />);
    expect(screen.getByLabelText("Live browser surface")).toHaveStyle({ aspectRatio: "200 / 100", maxHeight: "70vh" });
  });

  it("lets keyboard users tab out of the remote input surface to its controls", async () => {
    const user = userEvent.setup();
    render(<BrowserPane session={session} frame={<canvas />} onTakeControl={() => undefined} />);
    const surface = screen.getByLabelText("Live browser surface");
    surface.focus();
    await user.tab();
    expect(screen.getByRole("button", { name: "Take control" })).toHaveFocus();
  });

  it("refuses control when the run/thread association is unverified", () => {
    const onTakeControl = vi.fn();
    render(<BrowserPane session={{ ...session, association: { ...session.association, verified: false } }} onTakeControl={onTakeControl} />);
    expect(screen.getByText(/not associated with a verified agent run or conversation thread/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Take control" })).not.toBeInTheDocument();
    expect(onTakeControl).not.toHaveBeenCalled();
  });

  it("requires a verified association before opening a broker-owned session", () => {
    const onOpen = vi.fn();
    const onAttach = vi.fn();
    const { rerender } = render(<BrowserPane onOpen={onOpen} />);
    expect(screen.queryByRole("button", { name: "Open browser session" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Attach an approved browser session" })).not.toBeInTheDocument();
    expect(onOpen).not.toHaveBeenCalled();
    rerender(<BrowserPane launchAssociation={{ source: "conversation", threadId: "thread-9", verified: true }} onOpen={onOpen} onAttach={onAttach} />);
    fireEvent.click(screen.getByRole("button", { name: "Open browser session" }));
    fireEvent.click(screen.getByRole("button", { name: "Attach an approved browser session" }));
    expect(onOpen).toHaveBeenCalledOnce();
    expect(onAttach).toHaveBeenCalledOnce();
  });

  it("offers downloads and console only at the declared owner capability", () => {
    const onShowDownloads = vi.fn();
    const onShowConsole = vi.fn();
    const capable = { ...session, capabilities: { ...session.capabilities, downloads: true, console: true } };
    render(<BrowserPane session={capable} onShowDownloads={onShowDownloads} onShowConsole={onShowConsole} />);
    fireEvent.click(screen.getByRole("button", { name: "Downloads" }));
    fireEvent.click(screen.getByRole("button", { name: "Console" }));
    expect(onShowDownloads).toHaveBeenCalledOnce();
    expect(onShowConsole).toHaveBeenCalledOnce();
    expect(screen.queryByRole("button", { name: "Clipboard" })).not.toBeInTheDocument();
  });

  it("types through the ordered owner input under the controller receipt and clears the local buffer", async () => {
    const onInput = vi.fn(async (_input: BrowserSurfaceInput) => undefined);
    const controlled = { ...session, state: "controller" as const, controller: true, controlReceipt: "opaque", sessionEpoch: 3n };
    render(<BrowserPane session={controlled} onInput={onInput} />);
    fireEvent.click(screen.getByRole("button", { name: "Type text" }));
    const entry = screen.getByRole("textbox", { name: "Text to enter in browser" });
    fireEvent.change(entry, { target: { value: "hello from phone" } });
    fireEvent.click(screen.getByRole("button", { name: "Send to browser" }));
    await waitFor(() => expect(onInput).toHaveBeenCalledOnce());
    expect(onInput).toHaveBeenCalledWith({ type: "text", text: "hello from phone" }, 1, { receipt: "opaque", epoch: 3n });
    expect(screen.getByRole("textbox", { name: "Text to enter in browser" })).toHaveValue("");
  });

  it("isolates the rendered frame from its child click handlers", () => {
    const sendInput = vi.fn();
    const frame = <canvas aria-label="Browser frame" onClick={sendInput} />;
    const { rerender } = render(<BrowserPane session={session} frame={frame} />);
    fireEvent.click(screen.getByLabelText("Browser frame"));
    expect(sendInput).not.toHaveBeenCalled();

    rerender(<BrowserPane session={{ ...session, state: "controller", controller: true, controlReceipt: "opaque", sessionEpoch: 3n }} frame={frame} />);
    fireEvent.click(screen.getByLabelText("Browser frame"));
    expect(sendInput).not.toHaveBeenCalled();
  });

  it("maps HiDPI frame coordinates and queues typed input under the opaque lease epoch", async () => {
    const onInput = vi.fn(async (_input: BrowserSurfaceInput) => undefined);
    const controlled = { ...session, state: "controller" as const, controller: true, controlReceipt: "opaque", sessionEpoch: 3n };
    render(<BrowserPane session={controlled} frame={<canvas aria-label="Browser frame" />} onInput={onInput} />);
    const surface = screen.getByLabelText("Live browser surface");
    vi.spyOn(surface, "getBoundingClientRect").mockReturnValue({ left: 0, top: 0, right: 100, bottom: 100, width: 100, height: 100, x: 0, y: 0, toJSON: () => ({}) });
    fireEvent.pointerDown(surface, { pointerId: 4, clientX: 50, clientY: 30, button: 0 });
    await waitFor(() => expect(onInput).toHaveBeenCalledOnce());
    expect(onInput).toHaveBeenCalledWith({ type: "pointer", phase: "down", button: "left", x: 50, y: 5, modifiers: [] }, 1, { receipt: "opaque", epoch: 3n });
  });

  it("releases held keys and pointer buttons when the live surface loses focus", async () => {
    const onInput = vi.fn(async (_input: BrowserSurfaceInput) => undefined);
    const controlled = { ...session, state: "controller" as const, controller: true, controlReceipt: "opaque", sessionEpoch: 3n };
    render(<BrowserPane session={controlled} frame={<canvas aria-label="Browser frame" />} onInput={onInput} />);
    const surface = screen.getByLabelText("Live browser surface");
    vi.spyOn(surface, "getBoundingClientRect").mockReturnValue({ left: 0, top: 0, right: 100, bottom: 100, width: 100, height: 100, x: 0, y: 0, toJSON: () => ({}) });
    fireEvent.keyDown(surface, { key: "Shift" });
    fireEvent.pointerDown(surface, { pointerId: 9, clientX: 50, clientY: 30, button: 0 });
    fireEvent.blur(surface);
    await waitFor(() => expect(onInput).toHaveBeenCalledTimes(4));
    expect(onInput.mock.calls.map(([input]) => input)).toEqual([
      { type: "key", phase: "down", key: "Shift", modifiers: [] },
      { type: "pointer", phase: "down", button: "left", x: 50, y: 5, modifiers: [] },
      { type: "key", phase: "up", key: "Shift", modifiers: [] },
      { type: "pointer", phase: "up", button: "left", x: 50, y: 5 },
    ]);
  });

  it("sends pointer-up at the clipped content edge when capture cancels outside the frame", async () => {
    const onInput = vi.fn(async (_input: BrowserSurfaceInput) => undefined);
    const controlled = { ...session, state: "controller" as const, controller: true, controlReceipt: "opaque", sessionEpoch: 3n };
    render(<BrowserPane session={controlled} frame={<canvas aria-label="Browser frame" />} onInput={onInput} />);
    const surface = screen.getByLabelText("Live browser surface");
    vi.spyOn(surface, "getBoundingClientRect").mockReturnValue({ left: 0, top: 0, right: 100, bottom: 100, width: 100, height: 100, x: 0, y: 0, toJSON: () => ({}) });
    fireEvent.pointerDown(surface, { pointerId: 4, clientX: 50, clientY: 30, button: 0 });
    fireEvent.pointerCancel(surface, { pointerId: 4, clientX: -20, clientY: 10, button: -1 });
    await waitFor(() => expect(onInput).toHaveBeenCalledTimes(2));
    expect(onInput.mock.calls.map(([input]) => input)).toEqual([
      { type: "pointer", phase: "down", button: "left", x: 50, y: 5, modifiers: [] },
      { type: "pointer", phase: "up", button: "left", x: 0, y: 0, modifiers: [] },
    ]);
  });

  it("bounds and transfers clipboard text only under the current controller grant", async () => {
    const previousClipboard = navigator.clipboard;
    const readText = vi.fn(async () => "private clipboard value");
    const writeText = vi.fn(async () => undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { readText, writeText } });
    try {
      const controlled = { ...session, state: "controller" as const, controller: true, controlReceipt: "opaque", sessionEpoch: 3n, capabilities: { ...session.capabilities, clipboard: true } };
      const onPasteToBrowser = vi.fn(async (_text: string, _order: number, _lease: { receipt: string; epoch: bigint }) => undefined);
      const onCopyFromBrowser = vi.fn(async () => "browser clipboard value");
      render(<BrowserPane session={controlled} onPasteToBrowser={onPasteToBrowser} onCopyFromBrowser={onCopyFromBrowser} />);
      fireEvent.click(screen.getByRole("button", { name: "Send text" }));
      await waitFor(() => expect(onPasteToBrowser).toHaveBeenCalledOnce());
      expect(onPasteToBrowser).toHaveBeenCalledWith("private clipboard value", 1, { receipt: "opaque", epoch: 3n });
      fireEvent.click(screen.getByRole("button", { name: "Copy text" }));
      await waitFor(() => expect(writeText).toHaveBeenCalledWith("browser clipboard value"));
      expect(screen.queryByText(/private clipboard value|browser clipboard value/)).not.toBeInTheDocument();
    } finally {
      Object.defineProperty(navigator, "clipboard", { configurable: true, value: previousClipboard });
    }
  });
});
