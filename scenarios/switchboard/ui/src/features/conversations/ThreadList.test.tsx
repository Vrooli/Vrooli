import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { makeAgent, makeThread } from "../../test-utils/consoleFixtures";
import { ThreadList } from "./ThreadList";

const threads = [
  makeThread(),
  makeThread({
    id: "thread-2",
    channel_id: "telegram",
    channel_display_name: "Telegram",
    channel_accent: "#2AABEE",
    thread_key: "chat-9",
    is_group: true,
    participant_count: 3,
    pending_gates: 1,
    last_message: { text: "can you book the dentist", author_kind: "human", sender_address: "@sam", display_name: "Sam", received_at: new Date().toISOString() },
    budget: { thread_id: "thread-2", channel_id: "telegram", thread_key: "chat-9", agent_id: "household-planner", turn_budget: 20, used: 20, spend_cap_cents: 0, spent_cents: 0, window_started_at: "", exhausted: true },
  }),
  makeThread({ id: "thread-3", thread_key: "quiet", last_message: null, message_count: 0 }),
];

const renderList = (selectedId?: string) =>
  renderWithProviders(
    <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <ThreadList threads={threads} agents={{ "household-planner": makeAgent() }} selectedId={selectedId} />
    </MemoryRouter>,
    { withoutRouter: true },
  );

describe("ThreadList", () => {
  it("titles threads by the human for external rooms and by the agent for own threads", () => {
    renderList("thread-1");
    const items = screen.getAllByTestId("conversations-thread-item");
    expect(items).toHaveLength(3);
    expect(items[0]).toHaveTextContent("Household Planner");
    expect(items[1]).toHaveTextContent("Sam");
    expect(items[1]).toHaveTextContent("3");
    expect(items[0]?.querySelector("a")).toHaveAttribute("aria-current", "page");
  });

  it("filters by channel and by search text", () => {
    renderList();
    fireEvent.click(screen.getByRole("button", { name: /Telegram/ }));
    expect(screen.getAllByTestId("conversations-thread-item")).toHaveLength(1);
    fireEvent.click(screen.getAllByRole("button", { pressed: false })[0] as HTMLElement);
    fireEvent.change(screen.getByTestId("conversations-search"), { target: { value: "dentist" } });
    expect(screen.getAllByTestId("conversations-thread-item")).toHaveLength(1);
    fireEvent.change(screen.getByTestId("conversations-search"), { target: { value: "zzz" } });
    expect(screen.queryAllByTestId("conversations-thread-item")).toHaveLength(0);
  });
});

describe("ThreadList budget tones", () => {
  it("flags a thread near its limit without an agent record", () => {
    const near = makeThread({ id: "near", thread_key: "near", agent_display_name: undefined, budget: { thread_id: "near", channel_id: "in-app", thread_key: "near", agent_id: "x", turn_budget: 20, used: 15, spend_cap_cents: 0, spent_cents: 0, window_started_at: "", exhausted: false } });
    renderWithProviders(
      <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <ThreadList threads={[near]} agents={{}} />
      </MemoryRouter>,
      { withoutRouter: true },
    );
    expect(screen.getByTestId("conversations-thread-item")).toHaveTextContent("household-planner");
    expect(screen.queryByRole("group")).not.toBeInTheDocument();
  });
});
