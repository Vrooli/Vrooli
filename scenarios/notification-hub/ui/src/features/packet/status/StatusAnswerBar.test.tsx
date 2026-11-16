import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { conversationsClient } from "../../../api/notifications";
import { renderWithProviders } from "../../../test-utils";
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

    await user.click(screen.getByRole("button", { name: "Send" }));
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledWith({ askId: "ask-1", answer: "", note: "Use the Wednesday plan.", changes: { pace: { tier: "wednesday" } }, images: [] }));
    expect(onRefresh).toHaveBeenCalledOnce();
  });

  it("marks a status update read without sending an empty answer", async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn();
    renderWithProviders(<StatusAnswerBar ask={makeAsk({ kind: "status", options: [], state: "pending" })} draft={emptyDraft()} onDraftChange={() => undefined} onRefresh={onRefresh} />);

    await user.click(screen.getByRole("button", { name: "Mark read" }));
    await waitFor(() => expect(conversationsClient.markRead).toHaveBeenCalledWith({ askId: "ask-1" }));
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    expect(onRefresh).toHaveBeenCalledOnce();
  });
});
