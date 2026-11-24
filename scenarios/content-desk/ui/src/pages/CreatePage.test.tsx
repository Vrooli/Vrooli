import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { CreatePage } from "./CreatePage";
import { ReviewPage } from "./ReviewPage";
import { PlanPage } from "./PlanPage";

function CurrentLocation() { const location = useLocation(); return <output data-testid="journey-location">{location.pathname}{location.search}</output>; }

const mocks = vi.hoisted(() => ({ listPostTypes: vi.fn(), createDraft: vi.fn(), listProjects: vi.fn(), createProject: vi.fn(), listDrafts: vi.fn(), getDraftCurrentRevision: vi.fn(), transitionDraft: vi.fn(), listRequiredReviewModes: vi.fn(), recordReviewRun: vi.fn(), listCampaigns: vi.fn(), listPublicationIntents: vi.fn(), setPublicationIntent: vi.fn(), readBoard: vi.fn() }));
vi.mock("../api/posttypes", () => ({ posttypesClient: { listPostTypes: mocks.listPostTypes } }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { createDraft: mocks.createDraft, listDrafts: mocks.listDrafts, getDraftCurrentRevision: mocks.getDraftCurrentRevision, transitionDraft: mocks.transitionDraft } }));
vi.mock("../api/projects", () => ({ projectsClient: { listProjects: mocks.listProjects, createProject: mocks.createProject } }));
vi.mock("../api/review", () => ({ reviewClient: { listRequiredReviewModes: mocks.listRequiredReviewModes, recordReviewRun: mocks.recordReviewRun } }));
vi.mock("../api/campaigns", () => ({ campaignsClient: { listCampaigns: mocks.listCampaigns } }));
vi.mock("../api/intents", () => ({ intentsClient: { listPublicationIntents: mocks.listPublicationIntents, setPublicationIntent: mocks.setPublicationIntent } }));
vi.mock("../api/board", () => ({ readBoard: mocks.readBoard }));
vi.mock("./DashboardPage", () => ({ DashboardPage: () => <div data-testid="journey-approval-workbench" /> }));
vi.mock("@vrooli/react-component-library/Select/1", () => ({
  Select: ({ value, onChange, options, required }: { value: string; onChange: (event: React.ChangeEvent<HTMLSelectElement>) => void; options: Array<{ value: string; label: string }>; required?: boolean }) => <select aria-label={strings.pages.create.postType} value={value} onChange={onChange} required={required}>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>,
}));
vi.mock("@vrooli/react-component-library/Textarea/1", () => ({
  Textarea: ({ value, onChange, rows, className }: { value: string; onChange: (event: React.ChangeEvent<HTMLTextAreaElement>) => void; rows?: number; className?: string }) => <textarea aria-label={strings.pages.create.body} value={value} onChange={onChange} rows={rows} className={className} />,
}));

function renderCreate() {
  return renderWithProviders(
    <MemoryRouter initialEntries={["/create"]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <CreatePage />
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("CreatePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listPostTypes.mockResolvedValue({ postTypes: [{ id: "dev-log", status: "active" }, { id: "stub", status: "v0" }] });
    mocks.listProjects.mockResolvedValue({ projects: [{ id: "project-o14", name: "O14" }] });
    mocks.createDraft.mockResolvedValue({ draft: { id: "saved-native-draft" } });
    mocks.createProject.mockResolvedValue({ project: { id: "project-new", name: "New project" } });
    mocks.listDrafts.mockResolvedValue({ drafts: [] });
    let hasReview = false;
    mocks.getDraftCurrentRevision.mockImplementation(() => Promise.resolve({ currentRevision: { revisionId: "rev-campaign-free", body: "A short observation.", hasApplicableReview: hasReview, reviewRunId: hasReview ? "run-campaign-free" : "" } }));
    mocks.transitionDraft.mockResolvedValue({});
    mocks.listRequiredReviewModes.mockResolvedValue({ modes: ["evidence_grounding"] });
    mocks.recordReviewRun.mockImplementation(() => { hasReview = true; return Promise.resolve({ reviewRun: { id: "run-campaign-free", outcome: "passed", sourceRevisionId: "rev-campaign-free" } }); });
    mocks.listCampaigns.mockResolvedValue({ campaigns: [] });
    mocks.listPublicationIntents.mockResolvedValue({ intents: [] });
    mocks.setPublicationIntent.mockResolvedValue({ intent: { id: "intent-campaign-free", sourceRevisionId: "rev-campaign-free", desiredLocalDate: "2026-12-03", daypart: "", version: 1n } });
    mocks.readBoard.mockResolvedValue({ signals: {} });
  });

  it("saves an exact campaign-free native draft without invoking release actions", async () => {
    renderCreate();
    const input = await screen.findByLabelText(strings.pages.create.body);
    fireEvent.change(input, { target: { value: "A short observation." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));

    await waitFor(() => expect(mocks.createDraft).toHaveBeenCalledWith({
      campaignId: "",
      projectId: "",
      postTypeId: "dev-log",
      body: "A short observation.",
      channel: "x-twitter",
      format: "post",
      idempotencyKey: expect.any(String),
    }));
    expect(await screen.findByText(strings.pages.create.savedDetail)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: strings.pages.create.reviewDraft })).toHaveAttribute("href", "/review?draft=saved-native-draft");
    expect(mocks.listPostTypes).toHaveBeenCalledOnce();
  });

  it("does not offer a review route when the create API omits the persisted draft id", async () => {
    mocks.createDraft.mockResolvedValueOnce({ draft: undefined });
    renderCreate();
    fireEvent.change(await screen.findByLabelText(strings.pages.create.body), { target: { value: "A short observation." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    await screen.findByText(strings.pages.create.savedDetail);
    expect(screen.queryByRole("link", { name: strings.pages.create.reviewDraft })).not.toBeInTheDocument();
    const firstKey = mocks.createDraft.mock.calls.at(-1)?.[0].idempotencyKey;
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    await screen.findByRole("link", { name: strings.pages.create.reviewDraft });
    expect(mocks.createDraft.mock.calls.at(-1)?.[0].idempotencyKey).toBe(firstKey);
  });

  it("creates a named project and saves the draft into it", async () => {
    renderCreate();
    await screen.findByLabelText(strings.pages.create.project);
    fireEvent.change(screen.getByLabelText(strings.pages.create.newProjectName), { target: { value: "  New project  " } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.createProject }));
    await waitFor(() => expect(mocks.createProject).toHaveBeenCalledWith({ name: "New project" }));
    fireEvent.change(screen.getByLabelText(strings.pages.create.body), { target: { value: "A project note." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    await waitFor(() => expect(mocks.createDraft).toHaveBeenCalledWith(expect.objectContaining({ projectId: "project-new" })));
  });

  it("does not invent a post type when the active registry is empty", async () => {
    mocks.listPostTypes.mockResolvedValue({ postTypes: [{ id: "stub", status: "v0" }] });
    renderCreate();
    expect(await screen.findByText(strings.pages.create.noTypes)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.pages.create.save })).not.toBeInTheDocument();
    expect(mocks.createDraft).not.toHaveBeenCalled();
  });

  it("keeps an empty simple post from being saved", async () => {
    renderCreate();
    await screen.findByLabelText(strings.pages.create.body);
    expect(screen.getByRole("button", { name: strings.pages.create.save })).toBeDisabled();
    expect(mocks.createDraft).not.toHaveBeenCalled();
  });

  it("keeps Unassigned available when projects cannot be loaded", async () => {
    mocks.listProjects.mockRejectedValueOnce(new Error("project API offline"));
    renderCreate();
    expect(await screen.findByText(strings.pages.create.projectLoadError)).toBeInTheDocument();
    expect(screen.getByLabelText(strings.pages.create.project)).toBeInTheDocument();
  });

  it("shows project creation and draft save errors without implying success", async () => {
    mocks.createProject.mockRejectedValueOnce(new Error("project rejected"));
    mocks.createDraft.mockRejectedValueOnce(new Error("save response unknown")).mockRejectedValueOnce(new Error("save response unknown"));
    renderCreate();
    await screen.findByLabelText(strings.pages.create.project);
    fireEvent.change(screen.getByLabelText(strings.pages.create.newProjectName), { target: { value: "New project" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.createProject }));
    expect(await screen.findByRole("alert")).toHaveTextContent("project rejected");

    fireEvent.change(screen.getByLabelText(strings.pages.create.body), { target: { value: "A short observation." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    await waitFor(() => expect(screen.getAllByRole("alert")).toHaveLength(2));
    expect(screen.getAllByRole("alert")[1]).toHaveTextContent("save response unknown");
    const firstKey = mocks.createDraft.mock.calls.at(-1)?.[0].idempotencyKey;

    fireEvent.change(screen.getByLabelText(strings.pages.create.body), { target: { value: "Different wording." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    await waitFor(() => expect(mocks.createDraft).toHaveBeenCalledTimes(2));
    const secondKey = mocks.createDraft.mock.calls.at(-1)?.[0].idempotencyKey;
    expect(secondKey).not.toBe(firstKey);

    fireEvent.change(screen.getByLabelText(strings.pages.create.body), { target: { value: "A short observation." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    await screen.findByRole("link", { name: strings.pages.create.reviewDraft });
    expect(mocks.createDraft.mock.calls.at(-1)?.[0].idempotencyKey).toBe(firstKey);
  });

  it("shows a post-type load failure", async () => {
    mocks.listPostTypes.mockRejectedValueOnce(new Error("post type service offline"));
    renderCreate();
    expect(await screen.findByRole("alert")).toHaveTextContent("post type service offline");
    expect(mocks.createDraft).not.toHaveBeenCalled();
  });

  it("carries a campaign-free draft from Create through exact review into its focused Plan", async () => {
    const draft: { id: string; status: string; body: string; channel: string; format: string; postTypeId: string } = { id: "saved-native-draft", status: "requested", body: "A short observation.", channel: "x-twitter", format: "post", postTypeId: "dev-log" };
    mocks.createDraft.mockResolvedValueOnce({ draft });
    mocks.listDrafts.mockImplementation(() => Promise.resolve({ drafts: [draft] }));
    mocks.transitionDraft.mockImplementation(({ event }: { event: string }) => {
      const nextStatus: Record<string, string> = { begin: "drafting", complete: "drafted", check: "checking", review_pass: "reviewed" };
      draft.status = nextStatus[event] ?? draft.status;
      return Promise.resolve({});
    });
    renderWithProviders(<MemoryRouter initialEntries={["/create"]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}><Routes>
      <Route path="/create" element={<CreatePage />} />
      <Route path="/review" element={<ReviewPage />} />
      <Route path="/plan" element={<PlanPage />} />
    </Routes><CurrentLocation /></MemoryRouter>, { withoutRouter: true });

    fireEvent.change(await screen.findByLabelText(strings.pages.create.body), { target: { value: "A short observation." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    fireEvent.click(await screen.findByRole("link", { name: strings.pages.create.reviewDraft }));

    fireEvent.click(await screen.findByRole("button", { name: strings.pages.review.prepareReview }));
    await waitFor(() => expect(mocks.transitionDraft).toHaveBeenCalledTimes(3));
    fireEvent.change(await screen.findByTestId("review-result-evidence_grounding"), { target: { value: "pass" } });
    fireEvent.change(screen.getByTestId("review-evidence-evidence_grounding"), { target: { value: "Checked the local evidence receipt." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.recordReview }));
    await screen.findByText(strings.pages.review.reviewedReceipt);
    expect(mocks.recordReviewRun).toHaveBeenCalledWith({ draftId: "saved-native-draft", expectedRevisionId: "rev-campaign-free", expectedBody: "A short observation.", verdicts: [{ mode: "evidence_grounding", passed: true, evidence: "Checked the local evidence receipt.", finding: "" }] });

    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.planExactDraft }));
    expect(await screen.findByText(strings.pages.plan.exactDraftPlanTitle)).toBeInTheDocument();
    const intentEditor = screen.getByTestId("draft-intent-saved-native-draft");
    expect(intentEditor).toBeInTheDocument();
    fireEvent.change(within(intentEditor).getByLabelText(strings.pages.plan.desiredDate), { target: { value: "2026-12-03" } });
    fireEvent.click(within(intentEditor).getByRole("button", { name: strings.pages.plan.saveIntent }));
    expect(await within(intentEditor).findByText(strings.pages.plan.intentSaved)).toHaveAttribute("role", "status");
    expect(mocks.setPublicationIntent).toHaveBeenCalledWith(expect.objectContaining({ draftId: "saved-native-draft", expectedRevisionId: "rev-campaign-free", expectedBody: "A short observation.", desiredLocalDate: "2026-12-03" }));
    fireEvent.click(screen.getByRole("button", { name: strings.pages.plan.returnToReview }));
    expect(await screen.findByTestId("review-basis-receipt")).toHaveAttribute("data-run-id", "run-campaign-free");
    expect(screen.getByTestId("journey-location")).toHaveTextContent("/review?draft=saved-native-draft");
    expect(mocks.listCampaigns).toHaveBeenCalled();
  });
});
