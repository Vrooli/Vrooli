import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { PlanWorkshopPanel } from "./plan-workshop-panel";

const { service } = vi.hoisted(() => ({
  service: (() => {
    const session = {
      id: "pw-1", subject: { kind: "backlog_item", ref: "execute/workshop-a" }, subject_version: "v1", plan_id: "plan-1", packet: {},
      resolutions: [{ response_id: "response-1", state: "candidate_ready", candidate: {
        id: "candidate-1", plan_id: "plan-1", expected_base_content_hash: "sha256:base", quality_status: "pass",
        diff: [{ field: "Purpose", before_json: "\"before\"", after_json: "\"after\"" }],
        diagnostics: [{ severity: "warning", code: "coverage", message: "Add verification", guidance: "Include a test" }],
        impact: { before_grade: "B", after_grade: "A" },
      }}],
    };
    return { open: vi.fn().mockResolvedValue(session), discardCandidate: vi.fn().mockResolvedValue({ session, resolution: session.resolutions[0] }) };
  })(),
}));

vi.mock("../../services/plan-workshop-service", () => ({ planWorkshopService: service }));

describe("PlanWorkshopPanel", () => {
  it("renders the structured candidate review and lets the operator ignore it", async () => {
    renderWithProviders(<PlanWorkshopPanel subject={{ kind: "backlog_item", ref: "execute/workshop-a" }} />);

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Open Plan Workshop" }));
    });
    expect(await screen.findByTestId("plan-workshop-candidate-preview")).toHaveTextContent("Purpose");
    expect(screen.getByTestId("plan-workshop-candidate-preview")).toHaveTextContent("Add verification");
    expect(screen.getByTestId("plan-workshop-candidate-preview")).toHaveTextContent("B → A");

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ignore candidate" }));
    });
    expect(service.discardCandidate).toHaveBeenCalledWith("pw-1", "response-1");
  });
});
