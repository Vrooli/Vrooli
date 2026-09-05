import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { defaultRoutes, makeThread, makeThreadDetail, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { ConversationsPage } from "./ConversationsPage";

class FakeWebSocket {
  static readonly OPEN = 1;
  static instances: FakeWebSocket[] = [];
  readonly readyState = FakeWebSocket.OPEN;
  sent: string[] = [];
  onopen?: () => void;
  onclose?: () => void;
  onerror?: () => void;
  onmessage?: (event: MessageEvent<string>) => void;
  constructor(public readonly url: string) {
    FakeWebSocket.instances.push(this);
    queueMicrotask(() => this.onopen?.());
  }
  send(payload: string) {
    this.sent.push(payload);
    queueMicrotask(() => this.onmessage?.({ data: JSON.stringify({ text: "reply" }) } as MessageEvent<string>));
  }
  close() {
    /* closed by the hook on unmount */
  }
}

const renderAt = (path: string) =>
  renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route path="/conversations" element={<ConversationsPage />} />
        <Route path="/conversations/:threadId" element={<ConversationsPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );

describe("ConversationsPage", () => {
  beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
  });
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P0-006]
  it("lists every thread with its channel chip", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt("/conversations");
    expect(await screen.findByTestId("conversations-thread-item")).toBeInTheDocument();
    expect(screen.getByTestId("conversations-channel-chip")).toHaveAttribute("data-channel", "in-app");
    expect(screen.getByTestId("conversations-thread-list-region")).toHaveAttribute("data-experience-state", "ready");
  });

  it("offers to start a conversation when there are no threads", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/threads"] = [];
    stubConsoleFetch(routes);
    renderAt("/conversations");
    expect(await screen.findByTestId("conversations-empty-cta")).toBeInTheDocument();
    expect(screen.getByTestId("conversations-thread-list-region")).toHaveAttribute("data-experience-state", "empty");
  });

  // [REQ:SWBD-P0-014]
  it("opens a thread, marks agent messages, and sends through the in-app socket", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt("/conversations/thread-1");
    expect(await screen.findByTestId("conversations-transcript")).toBeInTheDocument();
    expect(screen.getAllByTestId("conversations-message")).toHaveLength(2);
    expect(screen.getByTestId("conversations-agent-marker")).toBeInTheDocument();
    await waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    expect(FakeWebSocket.instances[0]?.url).toContain("thread_key=5f2c");
    const input = await screen.findByTestId("conversations-composer-input");
    fireEvent.change(input, { target: { value: "hello agent" } });
    fireEvent.click(screen.getByTestId("conversations-send"));
    await waitFor(() => expect(FakeWebSocket.instances[0]?.sent).toHaveLength(1));
    const envelope = JSON.parse(FakeWebSocket.instances[0]?.sent[0] ?? "{}") as { text: string; sender_address: string; channel_id: string };
    expect(envelope).toMatchObject({ text: "hello agent", sender_address: "owner", channel_id: "in-app" });
    // The composer keeps its draft only on failure; success clears it.
    await waitFor(() => expect(screen.getByTestId("conversations-composer-input")).toHaveValue(""));
  });

  it("disables the composer with a reason for threads that live on another channel", async () => {
    const routes = defaultRoutes();
    const thread = makeThread({ id: "thread-2", channel_id: "telegram", channel_display_name: "Telegram", thread_key: "chat-9", is_group: true, participant_count: 3, ceiling_tier: "stranger" });
    routes["/api/v1/threads"] = [thread];
    routes["/api/v1/threads/thread-2"] = makeThreadDetail({
      thread,
      participants: [
        { contact_id: "c-sam", address: "@sam", display_name: "Sam", tier: "known" },
        { contact_id: "c-x", address: "@x", tier: "stranger" },
      ],
    });
    stubConsoleFetch(routes);
    renderAt("/conversations/thread-2");
    expect(await screen.findByTestId("conversations-roster")).toBeInTheDocument();
    expect(screen.getByTestId("conversations-composer")).toHaveAttribute("role", "status");
    expect(screen.getByTestId("conversations-silence")).toBeInTheDocument();
    expect(screen.getByTestId("conversations-roster")).toHaveTextContent("Sam");
    expect(screen.getByTestId("conversations-ceiling")).toHaveAttribute("data-tier", "stranger");
    expect(FakeWebSocket.instances).toHaveLength(0);
  });
});

describe("ConversationsPage edge states", () => {
  beforeEach(() => vi.stubGlobal("WebSocket", FakeWebSocket));
  afterEach(() => vi.unstubAllGlobals());

  it("keeps both declared regions present when no thread is open", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt("/conversations");
    await screen.findByTestId("conversations-thread-item");
    expect(screen.getByTestId("conversations-transcript-region")).toHaveAttribute("data-experience-state", "empty");
  });

  it("shows the phone-width thread strip beside an open thread", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt("/conversations/thread-1");
    expect(await screen.findByTestId("conversations-thread-strip")).toBeInTheDocument();
  });

  it("treats an unknown thread as an empty transcript, not an error", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/threads/ghost"] = new Response("not found", { status: 404 });
    stubConsoleFetch(routes);
    renderAt("/conversations/ghost");
    await waitFor(() => expect(screen.getByTestId("conversations-transcript-region")).toHaveAttribute("data-experience-state", "empty"));
    expect(screen.queryByTestId("conversations-composer")).not.toBeInTheDocument();
  });
});
