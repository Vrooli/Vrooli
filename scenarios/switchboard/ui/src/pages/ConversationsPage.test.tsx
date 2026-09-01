import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { ConversationsPage } from "./ConversationsPage";

class FakeWebSocket {
  static readonly OPEN = 1;
  readonly readyState = FakeWebSocket.OPEN;
  onopen?: () => void;
  onclose?: () => void;
  onerror?: () => void;
  onmessage?: (event: MessageEvent<string>) => void;

  constructor() { queueMicrotask(() => this.onopen?.()); }
  send(payload: string) {
    const message = JSON.parse(payload) as { text: string };
    queueMicrotask(() => this.onmessage?.({ data: JSON.stringify({ text: `reply: ${message.text}` }) } as MessageEvent<string>));
  }
  close() { this.onclose?.(); }
}

describe("ConversationsPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("connects the in-app adapter and sends text through the durable thread", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    renderWithProviders(<ConversationsPage />);
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("console.conversations.connected"));
    const input = screen.getByLabelText(strings.console.conversations.message);
    fireEvent.change(input, { target: { value: "hello agent" } });
    fireEvent.click(screen.getByRole("button", { name: strings.console.conversations.send }));
    expect(screen.getByTestId("conversation-message-human")).toHaveTextContent("hello agent");
    await waitFor(() => expect(screen.getByTestId("conversation-message-agent")).toHaveTextContent("reply: hello agent"));
  });
});
