import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { PlanPage } from "./PlanPage";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn(), listCampaigns: vi.fn(), listProjects: vi.fn(), listPublicationIntents: vi.fn(), getDraftCurrentRevision: vi.fn(), setPublicationIntent: vi.fn() }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, getDraftCurrentRevision: mocks.getDraftCurrentRevision } }));
vi.mock("../api/campaigns", () => ({ campaignsClient: { listCampaigns: mocks.listCampaigns } }));
vi.mock("../api/projects", () => ({ projectsClient: { listProjects: mocks.listProjects } }));
vi.mock("../api/intents", () => ({ intentsClient: { listPublicationIntents: mocks.listPublicationIntents, setPublicationIntent: mocks.setPublicationIntent } }));

function renderPlan() {
  return renderWithProviders(<MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}><PlanPage /></MemoryRouter>, { withoutRouter: true });
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
  });

  it("summarizes editorial mix and keeps time reservation and publication authority distinct", async () => {
    renderPlan();
    expect(await screen.findByText(strings.pages.plan.authorityTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.plan.authorityDescription)).toBeInTheDocument();
    expect(screen.getByText("Launch story")).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("content-desk") && content.includes("linkedDrafts"))).toBeInTheDocument();
    expect(screen.getByText("linkedin")).toBeInTheDocument();
    expect(screen.getByText("article")).toBeInTheDocument();
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
    mocks.listPublicationIntents.mockResolvedValueOnce({ intents: [{ id: "intent-1", draftId: "d1", destination: "linkedin", desiredLocalDate: "2026-12-03", daypart: "", version: 3n }] });
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
    expect(await screen.findByRole("status")).toHaveTextContent(strings.pages.plan.intentSaved);
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
