import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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
