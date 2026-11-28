import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { conversationsClient } from "../../../api/notifications";
import { strings } from "../../../consts/strings";
import { renderWithProviders } from "../../../test-utils";
import { AnswerAttachments } from "./AnswerAttachments";
import { emptyDraft, type PacketDraft } from "../model/draft";

vi.mock("../../../api/notifications", () => ({ conversationsClient: { putAttachmentBlob: vi.fn() } }));

function Harness() {
  const [draft, setDraft] = useState<PacketDraft>(emptyDraft());
  return <AnswerAttachments images={draft.answerImages} attachments={draft.answerAttachments} disabled={false} onChange={(update) => setDraft(update)} />;
}

describe("AnswerAttachments", () => {
  beforeEach(() => {
    vi.mocked(conversationsClient.putAttachmentBlob).mockResolvedValue({} as never);
    vi.spyOn(crypto.subtle, "digest").mockResolvedValue(new ArrayBuffer(32));
    Object.defineProperty(File.prototype, "arrayBuffer", { configurable: true, value: () => Promise.resolve(new TextEncoder().encode("decision evidence").buffer) });
  });
  afterEach(() => vi.restoreAllMocks());

  it("uploads and keeps a non-image file in the answer draft", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);

    await user.upload(screen.getByLabelText(strings.packet.responseFiles.attach), new File(["decision evidence"], "decision notes.txt", { type: "text/plain" }));

    expect(await screen.findByText("decision-notes.txt")).toBeInTheDocument();
    await waitFor(() => expect(conversationsClient.putAttachmentBlob).toHaveBeenCalledWith(expect.objectContaining({ mediaType: "text/plain" })));
    expect(screen.getByLabelText(strings.packet.responseFiles.listLabel)).toBeInTheDocument();
  });
});
