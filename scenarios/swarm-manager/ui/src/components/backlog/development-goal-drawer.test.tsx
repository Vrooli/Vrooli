import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PreviewDevelopmentRequestSchema, PreviewDevelopmentResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import { renderWithProviders } from "../../test-utils";
import type { DevelopmentClient } from "../../services/development-service";
import { DevelopmentGoalDrawer } from "./development-goal-drawer";

function setup() {
  const initial = create(PreviewDevelopmentRequestSchema, { workItem: "execute/example", scenario: "example", acceptanceAllow: ["scenarios/example/**"] });
  const result = create(PreviewDevelopmentResponseSchema, { goalMessage: "Owner-generated goal", proposalDigest: "digest", launchBlockers: ["Owner grants unavailable"] });
  const client = { previewDevelopment: vi.fn().mockResolvedValue(result) };
  const onReviewed = vi.fn();
  const rendered = renderWithProviders(<DevelopmentGoalDrawer initial={initial} client={client as unknown as DevelopmentClient} onClose={vi.fn()} onReviewed={onReviewed} />);
  return { ...rendered, client, result, onReviewed };
}

describe("development goal configuration [REQ:SWM-P0-017]", () => {
  beforeEach(() => localStorage.clear());
  it("reviews budget policy changes without remembering that authority", async () => {
    const { client } = setup();
    const policy = screen.getByRole("combobox", { name: "Token budget policy" });
    expect(policy).toHaveValue("metered-cancellation");
    fireEvent.click(screen.getByRole("button", { name: "Preview goal" }));
    await screen.findByText("Owner-generated goal");
    expect(client.previewDevelopment).toHaveBeenLastCalledWith(expect.objectContaining({ budgetPolicy: "metered-cancellation" }));
    fireEvent.change(policy, { target: { value: "hard-ceiling" } });
    expect(screen.getByRole("button", { name: "Use reviewed configuration" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Preview goal" }));
    await screen.findByText("Owner-generated goal");
    expect(client.previewDevelopment).toHaveBeenLastCalledWith(expect.objectContaining({ budgetPolicy: "hard-ceiling" }));
    fireEvent.click(screen.getByRole("button", { name: "Use reviewed configuration" }));
    expect(localStorage.getItem("swarm.development-guidance.v1")).not.toContain("hard-ceiling");
  });
  it("preserves multiline scope editing and submits protected outcomes", async () => {
    const { client } = setup();
    const paths = screen.getByRole("textbox", { name: "Permitted paths, one per line" });
    fireEvent.change(paths, { target: { value: "scenarios/example/**\n" } });
    expect(paths).toHaveValue("scenarios/example/**\n");
    fireEvent.change(paths, { target: { value: "scenarios/example/**\n packages/audio/** \n" } });
    fireEvent.click(screen.getByRole("button", { name: "Add protected outcome" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Outcome 1 ID" }), { target: { value: "tail" } });
    fireEvent.change(screen.getByRole("textbox", { name: "Outcome 1 criterion" }), { target: { value: "No final transcript tail is lost" } });
    fireEvent.change(screen.getByRole("textbox", { name: "Outcome 1 evidence owner" }), { target: { value: "audio-tools" } });
    fireEvent.click(screen.getByRole("button", { name: "Preview goal" }));
    await screen.findByText("Owner-generated goal");
    expect(client.previewDevelopment).toHaveBeenCalledWith(expect.objectContaining({ acceptanceAllow: ["scenarios/example/**", "packages/audio/**"], outcomes: [expect.objectContaining({ id: "tail", evidenceSource: "audio-tools" })] }));
  });
  it("generates guidance through the owner without adding paths or launching", async () => {
    const { client, onReviewed } = setup();
    fireEvent.change(screen.getByRole("slider", { name: "Investigation effort" }), { target: { value: "2" } });
    fireEvent.click(screen.getByRole("checkbox", { name: /Permit necessary repairs/ }));
    fireEvent.change(screen.getByRole("textbox", { name: "Additional instructions" }), { target: { value: "Preserve intent, not just process" } });
    expect(client.previewDevelopment).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Preview goal" }));
    await screen.findByText("Owner-generated goal");
    expect(client.previewDevelopment).toHaveBeenCalledWith(expect.objectContaining({ acceptanceAllow: ["scenarios/example/**"], guidance: expect.objectContaining({ effort: "thorough", repairRelatedCode: true, additionalInstructions: "Preserve intent, not just process" }) }));
    expect(onReviewed).not.toHaveBeenCalled();
    expect(screen.getByText("Owner grants unavailable")).toBeInTheDocument();
  });
  it("invalidates reviewed guidance on edit and ignores an obsolete response", async () => {
    const { client, result } = setup();
    let resolve!: (value: typeof result) => void;
    client.previewDevelopment.mockImplementationOnce(() => new Promise((done) => { resolve = done; }));
    fireEvent.click(screen.getByRole("button", { name: "Preview goal" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Desired outcome" }), { target: { value: "Changed target" } });
    resolve(result);
    await waitFor(() => expect(screen.getByRole("button", { name: "Preview goal" })).toBeEnabled());
    expect(screen.queryByText("Owner-generated goal")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use reviewed configuration" })).toBeDisabled();
  });
  it("remembers suggestions without persisting budgets, paths, instructions or approval", async () => {
    const { onReviewed } = setup();
    fireEvent.change(screen.getByRole("textbox", { name: "Additional instructions" }), { target: { value: "Private task detail" } });
    fireEvent.click(screen.getByRole("button", { name: "Preview goal" }));
    await screen.findByText("Owner-generated goal");
    fireEvent.click(screen.getByRole("button", { name: "Use reviewed configuration" }));
    expect(onReviewed).toHaveBeenCalledOnce();
    const saved = JSON.parse(localStorage.getItem("swarm.development-guidance.v1") ?? "{}") as Record<string, unknown>;
    expect(Object.keys(saved).sort()).toEqual(["effort", "repairRelatedCode", "startingState", "validation"]);
    expect(JSON.stringify(saved)).not.toContain("Private task detail");
  });
  it("survives corrupt remembered preferences", () => {
    localStorage.setItem("swarm.development-guidance.v1", "not-json");
    setup();
    expect(screen.getByRole("slider", { name: "Investigation effort" })).toHaveAttribute("aria-valuetext", "balanced");
    expect(screen.getByRole("checkbox", { name: /Permit necessary repairs/ })).not.toBeChecked();
  });
});
