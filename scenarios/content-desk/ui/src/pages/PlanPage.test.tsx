import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import type { InputHTMLAttributes, SelectHTMLAttributes } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { PlanPage } from "./PlanPage";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn(), listCampaigns: vi.fn(), listProjects: vi.fn(), listPublicationIntents: vi.fn(), getDraftCurrentRevision: vi.fn(), setPublicationIntent: vi.fn(), readBoard: vi.fn() }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, getDraftCurrentRevision: mocks.getDraftCurrentRevision } }));
vi.mock("../api/campaigns", () => ({ campaignsClient: { listCampaigns: mocks.listCampaigns } }));
vi.mock("../api/projects", () => ({ projectsClient: { listProjects: mocks.listProjects } }));
vi.mock("../api/intents", () => ({ intentsClient: { listPublicationIntents: mocks.listPublicationIntents, setPublicationIntent: mocks.setPublicationIntent } }));
vi.mock("../api/board", () => ({ readBoard: mocks.readBoard }));
type MockSelectProps = SelectHTMLAttributes<HTMLSelectElement> & { options: Array<{ value: string; label: string }> };
vi.mock("@vrooli/react-component-library/Input/1", () => ({ Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} /> }));
vi.mock("@vrooli/react-component-library/Select/1", () => ({
  Select: ({ options, ...props }: MockSelectProps) => <select {...props}>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>,
}));

function renderPlan(initialEntry = "/plan") {
  return renderWithProviders(<MemoryRouter initialEntries={[initialEntry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}><PlanPage /><Location /></MemoryRouter>, { withoutRouter: true });
}
function Location() { const location = useLocation(); return <div data-testid="location">{location.pathname}{location.search}</div>; }

function renderedPlanText(): string {
  return document.querySelector('[data-testid="page-plan"]')?.textContent ?? "";
}

describe("PlanPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listDrafts.mockResolvedValue({ drafts: [
      { id: "d1", body: "A useful post", channel: "linkedin", format: "post", campaignId: "launch", projectId: "project-a", status: "drafted" },
      { id: "d2", body: "An article", channel: "linkedin", format: "article", campaignId: "", projectId: "unassigned", status: "requested" },
    ] });
    mocks.listCampaigns.mockResolvedValue({ campaigns: [{ id: "launch", name: "Launch story", status: "active", scenarioNames: ["content-desk"] }] });
    mocks.listProjects.mockResolvedValue({ projects: [{ id: "project-a", name: "Presence" }] });
    mocks.listPublicationIntents.mockResolvedValue({ intents: [] });
    mocks.getDraftCurrentRevision.mockResolvedValue({ currentRevision: { body: "A useful post", revisionId: "rev-1" } });
    mocks.setPublicationIntent.mockResolvedValue({ intent: {} });
    mocks.readBoard.mockResolvedValue({ signals: { offer_readiness: { launch_targets: [{ scenario: "offer-desk", node_id: "o14", status: "ready", next_action: "Review offer", blocked_by: [] }] } } });
  });

  it("summarizes editorial mix and keeps time reservation and publication authority distinct", async () => {
    renderPlan();
    expect(await screen.findByText(strings.pages.plan.authorityTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.plan.authorityDescription)).toBeInTheDocument();
    expect(screen.getByText("Launch story")).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("content-desk") && content.includes("linkedDrafts"))).toBeInTheDocument();
    expect(screen.getByText("linkedin")).toBeInTheDocument();
    expect(screen.getByText("article")).toBeInTheDocument();
    await screen.findByText(strings.board.offerTitle);
    expect(renderedPlanText()).toContain("offer-desk");
    expect(renderedPlanText()).toContain("Review offer");
    expect(mocks.listDrafts).toHaveBeenCalledOnce();
    expect(mocks.listCampaigns).toHaveBeenCalledOnce();
    expect(mocks.listProjects).toHaveBeenCalledOnce();
    expect(mocks.listPublicationIntents).toHaveBeenCalledOnce();
  });

  it("shows a clear error and retries the reads", async () => {
    mocks.listCampaigns.mockRejectedValueOnce(new Error("campaign service offline"));
    renderPlan();
    expect(await screen.findByRole("alert")).toHaveTextContent("campaign service offline");
    mocks.listCampaigns.mockResolvedValue({ campaigns: [] });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.plan.retry }));
    await waitFor(() => expect(screen.getByText(strings.pages.plan.noCampaigns)).toBeInTheDocument());
  });

  it("keeps editorial planning available when Offer Desk readiness is unavailable", async () => {
    mocks.readBoard.mockRejectedValueOnce(new Error("offer service offline"));
    renderPlan();
    expect(await screen.findByText(strings.board.offerUnavailable)).toBeInTheDocument();
    expect(renderedPlanText()).toContain("Launch story");
  });

  it("focuses the reviewed draft and preserves a safe return to Review", async () => {
    renderPlan("/plan?draft=d2&return=%2Freview%3Fdraft%3Dd2");
    expect(await screen.findByText(strings.pages.plan.exactDraftPlanTitle)).toBeInTheDocument();
    expect(screen.getByText("d2", { selector: "p" })).toBeInTheDocument();
    expect(screen.getByTestId("draft-intent-d2")).toBeInTheDocument();
    expect(screen.queryByTestId("draft-intent-d1")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.pages.plan.returnToReview }));
    expect(screen.getByTestId("location")).toHaveTextContent("/review?draft=d2");
  });

  it("keeps the Plan and Offer Desk surfaces accessible", async () => {
    const { container } = renderPlan();
    await screen.findByText(strings.board.offerTitle);
    await expectNoA11yViolations(container);
  });

  it("renders a truthful empty workspace with no invented campaign, project or draft", async () => {
    mocks.listDrafts.mockResolvedValueOnce({ drafts: [] });
    mocks.listCampaigns.mockResolvedValueOnce({ campaigns: [] });
    mocks.listProjects.mockResolvedValueOnce({ projects: [] });
    mocks.listPublicationIntents.mockResolvedValueOnce({ intents: [] });
    renderPlan();
    expect(await screen.findByText(strings.pages.plan.noDraftsForIntent)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.plan.noCampaigns)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.plan.noProjects)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.plan.noChannelIntent)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.plan.noFormatIntent)).toBeInTheDocument();
  });

  it("does not invent a message for a non-Error read failure", async () => {
    mocks.listDrafts.mockRejectedValueOnce("transport unavailable");
    renderPlan();
    expect(await screen.findByRole("alert")).toHaveTextContent(strings.pages.plan.loadError);
  });

  it("shows the current day intent and reuses its exact operation key on retry", async () => {
    mocks.listPublicationIntents.mockResolvedValueOnce({ intents: [{ id: "intent-1", draftId: "d1", destination: "linkedin", desiredLocalDate: "2026-12-03", daypart: "", sourceRevisionId: "rev-1", version: 3n }] });
    mocks.setPublicationIntent.mockRejectedValueOnce(new Error("temporary network error"));
    renderPlan();
    const editor = await screen.findByTestId("draft-intent-d1");
    expect(within(editor).getByText(strings.pages.plan.currentIntent)).toBeInTheDocument();
    fireEvent.change(within(editor).getByLabelText(strings.pages.plan.daypart), { target: { value: "morning" } });
    fireEvent.click(within(editor).getByRole("button", { name: strings.pages.plan.saveIntent }));
    expect(await screen.findByRole("alert")).toHaveTextContent("temporary network error");
    fireEvent.click(within(editor).getByRole("button", { name: strings.pages.plan.saveIntent }));
    await screen.findByRole("status");
    await waitFor(() => expect(mocks.setPublicationIntent).toHaveBeenCalledTimes(2));
    const firstKey = mocks.setPublicationIntent.mock.calls[0]![0].idempotencyKey;
    expect(mocks.setPublicationIntent.mock.calls[1]![0].idempotencyKey).toBe(firstKey);
    expect(mocks.setPublicationIntent.mock.calls[0]![0].expectedIntentVersion).toBe(3n);
  });

  it("saves an optional daypart against the exact current draft revision and intent version", async () => {
    renderPlan();
    const editor = await screen.findByTestId("draft-intent-d1");
    fireEvent.change(within(editor).getByLabelText(strings.pages.plan.desiredDate), { target: { value: "2026-12-03" } });
    fireEvent.change(within(editor).getByLabelText(strings.pages.plan.daypart), { target: { value: "afternoon" } });
    fireEvent.click(within(editor).getByRole("button", { name: strings.pages.plan.saveIntent }));
    expect(await within(editor).findByText(strings.pages.plan.intentSaved)).toHaveAttribute("role", "status");
    expect(mocks.getDraftCurrentRevision).toHaveBeenCalledWith({ id: "d1" });
    expect(mocks.setPublicationIntent).toHaveBeenCalledWith(expect.objectContaining({
      draftId: "d1",
      destination: "linkedin",
      desiredLocalDate: "2026-12-03",
      daypart: "afternoon",
      expectedRevisionId: "rev-1",
      expectedBody: "A useful post",
      expectedIntentVersion: 0n,
    }));
  });

  it("marks a saved intention stale when its exact source revision no longer matches", async () => {
    mocks.listPublicationIntents.mockResolvedValueOnce({ intents: [{ id: "intent-old", draftId: "d1", destination: "linkedin", desiredLocalDate: "2026-12-03", daypart: "morning", sourceRevisionId: "rev-before-edit", version: 1n }] });
    renderPlan();
    expect(await screen.findByTestId("intent-basis-d1")).toHaveTextContent(strings.pages.plan.intentStale);
  });

  it("marks a saved intention stale when wording differs even if the revision identifier matches", async () => {
    mocks.listPublicationIntents.mockResolvedValueOnce({ intents: [{ id: "intent-old", draftId: "d1", destination: "linkedin", desiredLocalDate: "2026-12-03", daypart: "morning", sourceRevisionId: "rev-1", version: 1n }] });
    mocks.getDraftCurrentRevision.mockResolvedValueOnce({ currentRevision: { body: "Changed wording", revisionId: "rev-1" } });
    renderPlan();
    expect(await screen.findByTestId("intent-basis-d1")).toHaveTextContent(strings.pages.plan.intentStale);
  });

  it("does not claim an intention is current when its basis read fails", async () => {
    mocks.listPublicationIntents.mockResolvedValueOnce({ intents: [{ id: "intent-old", draftId: "d1", destination: "linkedin", desiredLocalDate: "2026-12-03", daypart: "morning", sourceRevisionId: "rev-1", version: 1n }] });
    mocks.getDraftCurrentRevision.mockRejectedValueOnce(new Error("revision service offline"));
    renderPlan();
    expect(await screen.findByTestId("intent-basis-d1")).toHaveTextContent(strings.pages.plan.intentUnavailable);
  });

  it("does not write an intention when the draft body changed since the plan loaded", async () => {
    mocks.getDraftCurrentRevision.mockResolvedValue({ currentRevision: { body: "New body", revisionId: "rev-2" } });
    renderPlan();
    const editor = await screen.findByTestId("draft-intent-d1");
    fireEvent.change(within(editor).getByLabelText(strings.pages.plan.desiredDate), { target: { value: "2026-12-03" } });
    fireEvent.click(within(editor).getByRole("button", { name: strings.pages.plan.saveIntent }));
    expect(await screen.findByRole("alert")).toHaveTextContent(strings.pages.plan.staleDraft);
    expect(mocks.setPublicationIntent).not.toHaveBeenCalled();
  });
});
