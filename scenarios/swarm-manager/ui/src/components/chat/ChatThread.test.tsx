import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChatThread } from "./ChatThread";

describe("ChatThread", () => {
  it("renders markdown through the shared markdown seam", () => {
    render(
      <ChatThread
        messages={[{ id: "m1", role: "assistant", content: "**Ready** to plan." }]}
        testId="chat-thread"
      />,
    );

    expect(screen.getByText("Ready").tagName).toBe("STRONG");
  });

  it("renders an empty state and waiting indicator", () => {
    render(<ChatThread messages={[]} isWaiting emptyLabel="Nothing yet." />);

    expect(screen.getByText("Nothing yet.")).toBeInTheDocument();
    expect(screen.getByText("Thinking...")).toBeInTheDocument();
  });

  it("renders attachment/footer slots per message", () => {
    render(
      <ChatThread
        messages={[{ id: "m1", role: "assistant", content: "Evidence attached.", attachmentIds: ["a1"] }]}
        renderAttachmentPreview={(message) => <span>Attachments: {message.attachmentIds?.length ?? 0}</span>}
      />,
    );

    expect(screen.getByText("Attachments: 1")).toBeInTheDocument();
  });
});

describe("ChatThread scrolling", () => {
  it("scrolls its own pane instead of the page when messages arrive", () => {
    // Regression: the thread used scrollIntoView on a bottom sentinel, which
    // walks every scrollable ancestor up to the document. On any layout where
    // the thread is not height-bounded that dragged the whole page down.
    const scrollIntoView = vi.fn();
    const original = Element.prototype.scrollIntoView;
    Element.prototype.scrollIntoView = scrollIntoView;
    try {
      const { rerender } = render(
        <ChatThread messages={[{ id: "m1", role: "assistant", content: "First" }]} testId="thread" />,
      );
      rerender(
        <ChatThread
          messages={[
            { id: "m1", role: "assistant", content: "First" },
            { id: "m2", role: "assistant", content: "Second" },
          ]}
          testId="thread"
        />,
      );

      expect(scrollIntoView).not.toHaveBeenCalled();
      const pane = screen.getByTestId("thread");
      expect(pane.scrollTop).toBe(pane.scrollHeight);
    } finally {
      Element.prototype.scrollIntoView = original;
    }
  });

  it("does not yank a reader who has scrolled up, and offers a way back", async () => {
    const messages = [{ id: "m1", role: "assistant" as const, content: "First" }];
    const { rerender } = render(<ChatThread messages={messages} testId="thread" />);

    const pane = screen.getByTestId("thread");
    // jsdom reports zero dimensions, so describe a pane the reader has
    // scrolled well away from the tail.
    Object.defineProperty(pane, "scrollHeight", { value: 1000, configurable: true });
    Object.defineProperty(pane, "clientHeight", { value: 300, configurable: true });
    pane.scrollTop = 0;
    fireEvent.scroll(pane);

    rerender(
      <ChatThread
        messages={[...messages, { id: "m2", role: "assistant" as const, content: "Second" }]}
        testId="thread"
      />,
    );

    expect(pane.scrollTop).toBe(0);
    expect(screen.getByTestId("chat-jump-to-latest")).toBeInTheDocument();
  });
});

describe("ChatThread density", () => {
  it("renders full-width rows with role rules in compact density", () => {
    render(
      <ChatThread
        density="compact"
        messages={[
          { id: "m1", role: "user", content: "Ask" },
          { id: "m2", role: "assistant", content: "Answer" },
        ]}
        testId="thread"
      />,
    );

    expect(screen.getByTestId("thread")).toHaveAttribute("data-density", "compact");
    const rows = screen.getAllByRole("article");
    // Compact rows carry no alignment wrapper: role is signalled by the rule.
    expect(rows[0]).toHaveAttribute("data-density", "compact");
    expect(rows[0]?.className).toContain("border-l-slate-500/70");
    expect(rows[1]?.className).toContain("border-l-cyan-500/60");
  });

  it("keeps speaker-aligned bubbles in comfortable density", () => {
    render(
      <ChatThread
        density="comfortable"
        messages={[
          { id: "m1", role: "user", content: "Ask" },
          { id: "m2", role: "assistant", content: "Answer" },
        ]}
        testId="thread"
      />,
    );

    const rows = screen.getAllByRole("article");
    expect(rows[0]?.className).toContain("justify-end");
    expect(rows[1]?.className).toContain("justify-start");
  });
});
