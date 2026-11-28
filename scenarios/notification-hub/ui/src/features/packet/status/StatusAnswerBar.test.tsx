import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { conversationsClient } from "../../../api/notifications";
import { renderWithProviders } from "../../../test-utils";
import { strings } from "../../../consts/strings";
import { makeAsk } from "../mocks/askFactory";
import { emptyDraft } from "../model/draft";
import { StatusAnswerBar } from "./StatusAnswerBar";

vi.mock("../../../api/notifications", () => ({ conversationsClient: { answer: vi.fn(), markRead: vi.fn() } }));

describe("status update answer bar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    vi.mocked(conversationsClient.markRead).mockResolvedValue({} as never);
  });

  it("sends optional note and page changes as steering", async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn();
    const draft = { ...emptyDraft(), note: "Use the Wednesday plan.", answerChanges: { pace: { label: "Use Wednesday", value: { tier: "wednesday" } } } };
    renderWithProviders(<StatusAnswerBar ask={makeAsk({ kind: "status", options: [], state: "pending" })} draft={draft} onDraftChange={() => undefined} onRefresh={onRefresh} />);

    await user.click(screen.getByRole("button", { name: new RegExp(strings.packet.statusBar.sendCount) }));
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledWith({ askId: "ask-1", answer: "", note: "Use the Wednesday plan.", changes: { pace: { tier: "wednesday" } }, images: [] }));
    expect(onRefresh).toHaveBeenCalledOnce();
  });

  it("sends item notes keyed by report item and shows the note count", async () => {
    const user = userEvent.setup();
    const draft = { ...emptyDraft(), answerChanges: { notes: { label: "Notes · 1", value: { "measure.latency": { label: "Latency", text: "Check this", attachments: [] } } } } };
    renderWithProviders(<StatusAnswerBar ask={makeAsk({ kind: "status", options: [], state: "pending" })} draft={draft} onDraftChange={() => undefined} onRefresh={() => undefined} />);
    expect(screen.getByRole("heading", { name: new RegExp(strings.packet.statusBar.yourChanges) })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: new RegExp(strings.packet.statusBar.sendCount) }));
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledWith({ askId: "ask-1", answer: "", note: "", changes: { notes: { "measure.latency": { label: "Latency", text: "Check this", attachments: [] } } }, images: [] }));
  });

  it("shows the desktop change rail with item notes and one primary send action", () => {
    const draft = { ...emptyDraft(), answerChanges: {
      pace: { label: "Use Wednesday", value: "wednesday" },
      notes: { label: "Notes", value: { "measure.latency": { label: "Latency", text: "Check this", attachments: [] } } },
    } };
    renderWithProviders(<StatusAnswerBar ask={makeAsk({ kind: "status", options: [], state: "pending" })} draft={draft} onDraftChange={() => undefined} onRefresh={() => undefined} />);

    expect(screen.getByRole("heading", { name: new RegExp(strings.packet.statusBar.yourChanges) })).toBeInTheDocument();
    expect(document.body.textContent).toContain("Use Wednesday");
    expect(screen.getByText(strings.packet.statusBar.noteChip)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.packet.statusBar.clear })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: new RegExp(strings.packet.statusBar.sendCount) })).toHaveLength(1);
  });

  it("marks a status update read without sending an empty answer", async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn();
    renderWithProviders(<StatusAnswerBar ask={makeAsk({ kind: "status", options: [], state: "pending" })} draft={emptyDraft()} onDraftChange={() => undefined} onRefresh={onRefresh} />);

    await user.click(screen.getByRole("button", { name: strings.packet.statusBar.markRead }));
    await waitFor(() => expect(conversationsClient.markRead).toHaveBeenCalledWith({ askId: "ask-1" }));
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    expect(onRefresh).toHaveBeenCalledOnce();
  });
});
