import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { PacketCover } from "./PacketCover";
import { makeAsk } from "./mocks/askFactory";

describe("PacketCover title", () => {
  it("shows the conversation title while retaining the question", () => {
    const ask = makeAsk({ title: "Opportunity roadmap · 16 Nov", question: "Review the opportunity roadmap" });
    renderWithProviders(<PacketCover ask={ask} links={{ attachments: [], onOpenAttachment: () => undefined }} onOpenAttachment={() => undefined} />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(ask.title);
    expect(screen.getByText(ask.question)).toBeInTheDocument();
    expect(screen.getByText(strings.packet.cover.history)).toBeInTheDocument();
  });
});
