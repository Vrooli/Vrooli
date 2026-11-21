import { act, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/renderWithProviders";
import { TerminalTab } from "./TerminalTab";

class FakeWebSocket {
  static OPEN = 1;
  readyState = 0;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  send = vi.fn();
  close = vi.fn(() => {
    this.readyState = 3;
    this.onclose?.({} as CloseEvent);
  });

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this);
  }

  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.({} as Event);
  }

  message(data: string) {
    this.onmessage?.({ data } as MessageEvent);
  }

  static instances: FakeWebSocket[] = [];
}

describe("TerminalTab", () => {
  afterEach(() => {
    FakeWebSocket.instances = [];
    vi.unstubAllGlobals();
  });

  it("connects, renders protocol output, handles terminal keys, and disconnects", () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    renderWithProviders(<TerminalTab deploymentId="deployment-1" />);

    expect(screen.getByText(/Click "Connect"/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    const socket = FakeWebSocket.instances[0]!;
    expect(socket.url).toContain("/deployments/deployment-1/terminal");
    expect(screen.getByText("Connecting...")).toBeInTheDocument();
    act(() => socket.open());
    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(screen.getByText(/Connected to VPS terminal/)).toBeInTheDocument();

    act(() => {
      socket.message(JSON.stringify({ type: "data", data: "hello\u001b[31m red\u001b[0m" }));
      socket.message(JSON.stringify({ type: "error", data: "remote failure" }));
      socket.message("raw terminal output");
      socket.message(JSON.stringify({ type: "ignored", data: "not shown" }));
    });
    expect(screen.getByText(/hello red/)).toBeInTheDocument();
    expect(screen.getByText(/Error: remote failure/)).toBeInTheDocument();
    expect(screen.getByText(/raw terminal output/)).toBeInTheDocument();
    expect(screen.queryByText(/not shown/)).not.toBeInTheDocument();

    const input = document.querySelector("textarea");
    expect(input).not.toBeNull();
    fireEvent.change(input!, { target: { value: "ls" } });
    expect(socket.send).toHaveBeenLastCalledWith(JSON.stringify({ type: "data", data: "ls" }));

    const sendKey = (key: string, options: { ctrlKey?: boolean } = {}) => fireEvent.keyDown(input!, { key, ...options });
    sendKey("Enter");
    sendKey("Backspace");
    sendKey("ArrowUp");
    sendKey("ArrowDown");
    sendKey("ArrowRight");
    sendKey("ArrowLeft");
    sendKey("Tab");
    sendKey("c", { ctrlKey: true });
    sendKey("d", { ctrlKey: true });
    sendKey("l", { ctrlKey: true });
    sendKey("a", { ctrlKey: true });
    sendKey("e", { ctrlKey: true });
    sendKey("u", { ctrlKey: true });
    sendKey("k", { ctrlKey: true });
    sendKey("w", { ctrlKey: true });
    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ type: "data", data: "\x1b[A" }));
    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ type: "data", data: "\x17" }));
    expect(screen.queryByText(/hello red/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Disconnect" }));
    expect(socket.close).toHaveBeenCalledOnce();
    expect(screen.getByText("Disconnected")).toBeInTheDocument();
    expect(screen.getByText(/Disconnected from VPS/)).toBeInTheDocument();
  });

  it("shows websocket errors and constructor failures without throwing", () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    renderWithProviders(<TerminalTab deploymentId="deployment-2" />);
    fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    act(() => FakeWebSocket.instances[0]!.onerror?.({} as Event));
    expect(screen.getByText("WebSocket connection error")).toBeInTheDocument();
    expect(screen.getByText("Error")).toBeInTheDocument();

    vi.stubGlobal("WebSocket", class {
      constructor() { throw "invalid socket"; }
    });
    fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    expect(screen.getByText("Failed to connect")).toBeInTheDocument();
  });
});
