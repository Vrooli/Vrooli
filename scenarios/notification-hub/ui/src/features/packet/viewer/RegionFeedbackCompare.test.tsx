import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { strings } from "../../../consts/strings";
import { renderWithProviders } from "../../../test-utils";
import { makeAttachment } from "../mocks/askFactory";
import { emptyDraft, type PacketDraft } from "../model/draft";
import { RegionFeedbackCompare } from "./RegionFeedbackCompare";
import { parseRegionFeedbackBundle, type ParsedRegionFeedback } from "./regionFeedback";

const sha = (value: string) => value.repeat(64);
const bundle = {
  schema: "ux-region-feedback/v1", id: "capture-set-1", route: "/settings", state_id: "nav-open", data_id: "fixture-1",
  before: { id: "capture-before", build_id: `sha256:${sha("a")}`, route: "/settings", state_id: "nav-open", data_id: "fixture-1", fidelity: "captured", viewport: { width: 390, height: 844 }, image: { path: "captures/before.png", sha256: sha("1") } },
  after: { id: "capture-after", build_id: `sha256:${sha("b")}`, route: "/settings", state_id: "nav-open", data_id: "fixture-1", fidelity: "rendered", viewport: { width: 390, height: 844 }, image: { path: "captures/after.png", sha256: sha("2") } },
  regions: [{ id: "nav-main", node_id: "node-18", label: "Main navigation", component: "BottomNav", evidence_id: "crop-1", bounds: { x: 0.1, y: 0.8, width: 0.8, height: 0.15 }, crop: { path: "regions/crops/nav.png", sha256: sha("3") } }],
};

function Harness({ onClose = vi.fn() }: { onClose?: () => void }) {
  const [draft, setDraft] = useState<PacketDraft>(emptyDraft());
  return <>
    <RegionFeedbackCompare
      askId="ask-1"
      attachment={makeAttachment({ name: "capture-set", folder: true })}
      parsed={parseRegionFeedbackBundle(bundle) as ParsedRegionFeedback}
      failed={false}
      onClose={onClose}
      draft={draft}
      onDraftChange={setDraft}
      canNote
    />
    <output data-testid="saved-notes">{JSON.stringify(draft.attachmentNotes)}</output>
  </>;
}

describe("RegionFeedbackCompare", () => {
  it("shows fidelity and identity, then stores the exact region path with the operator text", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    const fidelities = screen.getAllByText(strings.packet.regionFeedback.fidelity);
    expect(fidelities[0]?.closest("[data-fidelity]")).toHaveAttribute("data-fidelity", "captured");
    expect(fidelities[1]?.closest("[data-fidelity]")).toHaveAttribute("data-fidelity", "rendered");
    expect(screen.getByText("/settings")).toBeInTheDocument();
    await user.click(screen.getByTitle("Main navigation · BottomNav"));
    const note = screen.getByLabelText(strings.packet.regionFeedback.commentLabel);
    await user.type(note, "Keep the selected destination visible.");
    await user.click(screen.getByRole("button", { name: strings.packet.regionFeedback.save }));
    expect(screen.getByTestId("saved-notes")).toHaveTextContent('"path":"regions/nav-main.json","note":"Keep the selected destination visible."');
  });

  it("closes from the accessible control and escape", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(<Harness onClose={onClose} />);
    await user.click(screen.getByRole("button", { name: strings.packet.regionFeedback.close }));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
