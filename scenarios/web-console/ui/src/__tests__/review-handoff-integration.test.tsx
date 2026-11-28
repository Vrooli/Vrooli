import { useState } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ConversationEvent } from "../api/conversation";
import MessageExportDrawer from "../components/MessageExportDrawer";
import HandoffComposer from "../components/handoff/HandoffComposer";
import type { HandoffResult } from "../lib/handoff";
import { targetKey, type HandoffTarget, type HandoffTargetSection } from "../hooks/useHandoff";
import { renderWithProviders as render } from "../test-utils";

const selectedEvents: ConversationEvent[] = [
  {
    id: "selected-user", sessionId: "source", source: "codex_tailer", role: "user",
    text: "Use opportunity-code: amber-otter-731", speechParagraphs: [], summarized: false,
    createdAt: "2026-11-22T00:00:00Z", sequence: 41, deliveryState: "received", ttsState: "idle", consumptionState: "seen",
  },
  {
    id: "selected-tool", sessionId: "source", source: "codex_tailer", role: "assistant",
    text: "[Tool call: exec]\\n{\\\"command\\\":\\\"read opportunity-code: amber-otter-731\\\"}", speechParagraphs: [], summarized: false,
    createdAt: "2026-11-22T00:00:01Z", sequence: 42, deliveryState: "received", ttsState: "idle", consumptionState: "seen",
  },
];

const target: HandoffTarget = { kind: "session", sessionId: "local-target", label: "Local target", incomingPrompt: "" };
const targetSections: HandoffTargetSection[] = [{ kind: "group", labelKey: "handoff.sections.group", targets: [target] }];

function ReviewThenSubmit({ onSubmit }: { onSubmit: (target: HandoffTarget, payload: string) => void }) {
  const [reviewOpen, setReviewOpen] = useState(true);
  const [composerOpen, setComposerOpen] = useState(false);
  const [payload, setPayload] = useState("");
  return (
    <>
      <MessageExportDrawer
        open={reviewOpen}
        events={selectedEvents}
        onClose={() => setReviewOpen(false)}
        onHandoff={(exactSelection) => {
          setPayload(exactSelection);
          setReviewOpen(false);
          setComposerOpen(true);
        }}
      />
      <HandoffComposer
        open={composerOpen}
        onClose={() => setComposerOpen(false)}
        sourceLabel="Selected conversation"
        sourceSessionId="source"
        payload={payload}
        targets={targetSections}
        onSend={async (targets, textFor): Promise<HandoffResult[]> => {
          for (const selectedTarget of targets) onSubmit(selectedTarget, textFor(selectedTarget));
          return targets.map((selectedTarget) => ({ targetId: targetKey(selectedTarget), label: selectedTarget.label, status: "sent" as const }));
        }}
      />
    </>
  );
}

describe("review selection to local handoff submit", () => {
  it("keeps the selected tool span intact through review and explicit submit", async () => {
    const fakeLocalTargetAdapter = vi.fn();
    render(<ReviewThenSubmit onSubmit={fakeLocalTargetAdapter} />);

    const preview = screen.getByTestId("msg-export-preview").textContent ?? "";
    expect(preview).toContain("amber-otter-731");
    expect(preview).toContain("Tool call: exec");
    expect(fakeLocalTargetAdapter).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("msg-export-handoff"));
    const message = await screen.findByTestId("handoff-message");
    expect(message).toHaveValue(preview);
    expect(fakeLocalTargetAdapter).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("handoff-send"));
    await waitFor(() => expect(fakeLocalTargetAdapter).toHaveBeenCalledTimes(1));
    expect(fakeLocalTargetAdapter).toHaveBeenCalledWith(target, preview);
  });
});
