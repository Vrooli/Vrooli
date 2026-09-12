import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ChatMessageBubble } from "./ChatMessageBubble";
import type { ChatMessageView } from "./chat-types";
import type { AgentSessionContextItem } from "../../types";

function ctxItem(overrides: Partial<AgentSessionContextItem>): AgentSessionContextItem {
  return {
    type: "goal",
    ref: "ship-cockpit",
    title: "Ship the cockpit",
    summary: "",
    nodeId: "goal/ship-cockpit",
    metadataJson: "",
    selectedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  } as AgentSessionContextItem;
}

function message(content: string, context?: AgentSessionContextItem[]): ChatMessageView {
  return { id: "m1", role: "assistant", content, context };
}

describe("ChatMessageBubble entity references", () => {
  it("renders a resolved reference as an anchor to its detail path", () => {
    render(
      <ChatMessageBubble
        message={message("Start `goal:ship-cockpit` next.", [ctxItem({})])}
      />,
    );
    const anchor = document.querySelector('a[data-entity-ref="true"]');
    expect(anchor).not.toBeNull();
    expect(anchor?.getAttribute("href")).toBe("/goals/ship-cockpit");
    expect(anchor?.textContent).toBe("goal:ship-cockpit");
  });

  it("does not linkify a typed span that has no resolved context entry", () => {
    render(<ChatMessageBubble message={message("Maybe `goal:ghost`.", [])} />);
    expect(document.querySelector('a[data-entity-ref="true"]')).toBeNull();
    expect(document.querySelector("code")?.textContent).toBe("goal:ghost");
  });

  it("intercepts reference clicks for client-side navigation", () => {
    const onReferenceNavigate = vi.fn();
    render(
      <ChatMessageBubble
        message={message("Open `goal:ship-cockpit`.", [ctxItem({})])}
        onReferenceNavigate={onReferenceNavigate}
      />,
    );
    const anchor = screen.getByText("goal:ship-cockpit");
    fireEvent.click(anchor);
    expect(onReferenceNavigate).toHaveBeenCalledWith("/goals/ship-cockpit");
  });

  it("maps backlog context items through the kind/name route", () => {
    render(
      <ChatMessageBubble
        message={message("See `backlog:execute/wire-snapshot`.", [
          ctxItem({ type: "backlog_item", ref: "execute/wire-snapshot", nodeId: "backlog-item/execute/wire-snapshot" }),
        ])}
      />,
    );
    const anchor = document.querySelector('a[data-entity-ref="true"]');
    expect(anchor?.getAttribute("href")).toBe("/backlog/execute/wire-snapshot");
  });
});
