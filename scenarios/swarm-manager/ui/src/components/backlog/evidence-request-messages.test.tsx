import { describe, it, expect, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { EvidenceRequestMessages } from "./evidence-request-messages";
import type { RequestMessage } from "../../services/review-service";

// jsdom doesn't implement scrollIntoView.
beforeAll(() => {
  Element.prototype.scrollIntoView = () => {};
});

function makeMessage(overrides: Partial<RequestMessage> = {}): RequestMessage {
  return {
    role: "user",
    content: "Need more screenshots",
    timestamp: "2026-04-01T10:00:00Z",
    ...overrides,
  };
}

describe("EvidenceRequestMessages", () => {
  it("renders user messages right-aligned", () => {
    render(
      <EvidenceRequestMessages
        messages={[makeMessage({ role: "user", content: "User msg" })]}
        isWaitingForAgent={false}
      />,
    );

    const container = screen.getByText("User msg").closest("[class*='flex']");
    expect(container?.className).toContain("justify-end");
  });

  it("renders assistant messages left-aligned with violet styling", () => {
    render(
      <EvidenceRequestMessages
        messages={[makeMessage({ role: "assistant", content: "Agent reply" })]}
        isWaitingForAgent={false}
      />,
    );

    const container = screen.getByText("Agent reply").closest("[class*='flex']");
    expect(container?.className).toContain("justify-start");

    const bubble = screen.getByText("Agent reply").closest("[class*='rounded-lg']");
    expect(bubble?.className).toContain("violet");
  });

  it("shows thinking spinner when waiting for agent", () => {
    render(
      <EvidenceRequestMessages messages={[]} isWaitingForAgent={true} />,
    );

    expect(screen.getByText("Thinking...")).toBeTruthy();
  });

  it("does not show spinner when not waiting", () => {
    render(
      <EvidenceRequestMessages messages={[]} isWaitingForAgent={false} />,
    );

    expect(screen.queryByText("Thinking...")).toBeNull();
  });

  it("renders added evidence badge for assistant messages", () => {
    render(
      <EvidenceRequestMessages
        messages={[
          makeMessage({
            role: "assistant",
            content: "Here are the screenshots",
            added_evidence_ids: ["ev-1", "ev-2"],
          }),
        ]}
        isWaitingForAgent={false}
      />,
    );

    expect(screen.getByText("Added 2 evidence items")).toBeTruthy();
  });

  it("renders singular badge text for one evidence item", () => {
    render(
      <EvidenceRequestMessages
        messages={[
          makeMessage({
            role: "assistant",
            content: "Added one",
            added_evidence_ids: ["ev-1"],
          }),
        ]}
        isWaitingForAgent={false}
      />,
    );

    expect(screen.getByText("Added 1 evidence item")).toBeTruthy();
  });
});
