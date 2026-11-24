import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { ReviewPage } from "./ReviewPage";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn(), getDraftCurrentRevision: vi.fn(), transitionDraft: vi.fn(), createEditorialPackage: vi.fn(), listRequiredReviewModes: vi.fn(), recordReviewRun: vi.fn() }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, getDraftCurrentRevision: mocks.getDraftCurrentRevision, transitionDraft: mocks.transitionDraft } }));
vi.mock("../api/review", () => ({ reviewClient: { createEditorialPackage: mocks.createEditorialPackage, listRequiredReviewModes: mocks.listRequiredReviewModes, recordReviewRun: mocks.recordReviewRun } }));
vi.mock("./DashboardPage", () => ({ DashboardPage: ({ initialSelectedDraftId, preserveEmptySelection }: { initialSelectedDraftId?: string; preserveEmptySelection?: boolean }) => <div data-testid="existing-review-workbench" data-selected-draft-id={initialSelectedDraftId ?? ""} data-preserve-empty={String(Boolean(preserveEmptySelection))} /> }));
function Location() { const location = useLocation(); return <output data-testid="location">{location.pathname}{location.search}</output>; }
function renderReview(initialEntry = "/review") {
  return renderWithProviders(<MemoryRouter initialEntries={[initialEntry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}><Routes><Route path="/review" element={<ReviewPage />} /><Route path="/content" element={<div data-testid="returned-to-content" />} /></Routes><Location /></MemoryRouter>, { withoutRouter: true });
}

describe("ReviewPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getDraftCurrentRevision.mockResolvedValue({ currentRevision: { revisionId: "rev-current", body: "Evidence-backed note" } });
    mocks.transitionDraft.mockResolvedValue({});
    mocks.listRequiredReviewModes.mockResolvedValue({ modes: ["evidence_grounding", "fabricated_testimonial"] });
    mocks.recordReviewRun.mockResolvedValue({ reviewRun: { outcome: "passed", sourceRevisionId: "rev-current" } });
    mocks.createEditorialPackage.mockResolvedValue({ editorialPackage: { id: "package-1", draftId: "draft-needs-review", sourceRevisionId: "rev-current", scope: "destination_unbound_wording", body: "Evidence-backed note", bodySha256: "body-hash", packageSha256: "package-hash", manifestJson: JSON.stringify({ context: { channel: "linkedin", format: "article", project_id: "project-a", campaign_id: "campaign-1", post_type_id: "dev-log", lane: "engineering", sku: "starter", scenario_name: "content-desk", revision_authoring: { actor_kind: "agent", capacity: "author", created_at: "2026-11-21T01:00:00Z" }, attachments: [{ id: "a1", asset_id: "asset-1", role: "hero", aspect_ratio: "16:9", alt_text: "diagram", position: 0 }] } }), createdAt: "2026-11-21T00:00:00Z" } });
    mocks.listDrafts.mockResolvedValue({ drafts: [
      { id: "draft-needs-review", status: "draft", revisionId: "r1", body: "Evidence-backed note", postTypeId: "dev-log" },
      { id: "draft-approved", status: "approved", revisionId: "r2", body: "Approved snapshot" },
      { id: "draft-published", status: "published", revisionId: "r3", body: "Published copy" },
    ] });
  });

  it("shows exact native draft snapshots in the needs-review queue by default", async () => {
    renderReview();
    expect(await screen.findByText("draft-needs-review")).toBeInTheDocument();
    expect(screen.getByText("Evidence-backed note")).toBeInTheDocument();
    expect(screen.queryByText("draft-approved")).not.toBeInTheDocument();
    expect(screen.getByTestId("existing-review-workbench")).toBeInTheDocument();
    expect(screen.getByTestId("existing-review-workbench")).toHaveAttribute("data-preserve-empty", "true");
    expect(mocks.listDrafts).toHaveBeenCalledWith({});
  });

  it("preserves the selected queue in browser navigation while opening an exact draft", async () => {
    renderReview();
    await screen.findByText("draft-needs-review");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.filters.approved }));
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.open }));
    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/review?queue=approved&draft=draft-approved"));
    expect(screen.getByText("draft-approved")).toBeInTheDocument();
    expect(screen.getByTestId("existing-review-workbench")).toHaveAttribute("data-selected-draft-id", "draft-approved");
  });

  it("supports the all-status queue and ignores an unsafe return path", async () => {
    renderReview("/review?queue=all&return=https%3A%2F%2Fexample.invalid");
    expect(await screen.findByText("draft-needs-review")).toBeInTheDocument();
    expect(screen.getByText("draft-approved")).toBeInTheDocument();
    expect(screen.getByText("draft-published")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.pages.review.returnToContent })).not.toBeInTheDocument();
  });

  it("creates and displays a digest-bound, destination-unbound wording package", async () => {
    renderReview("/review?draft=draft-needs-review");
    await screen.findByText("draft-needs-review");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.packageCreate }));
    expect(await screen.findByText(strings.pages.review.packageLimit)).toBeInTheDocument();
    expect(screen.getAllByText("Evidence-backed note")).toHaveLength(2);
    expect(screen.getByText(strings.pages.review.packageReceipt)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.review.packageContextTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.review.packageChannelFormat)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.review.packageNativeContext)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.review.packageRevisionProvenance)).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("asset-1") && content.includes("hero") && content.includes("diagram"))).toBeInTheDocument();
    expect(mocks.getDraftCurrentRevision).toHaveBeenCalledWith({ id: "draft-needs-review" });
    expect(mocks.createEditorialPackage).toHaveBeenCalledWith({ draftId: "draft-needs-review", expectedRevisionId: "rev-current", expectedBody: "Evidence-backed note", idempotencyKey: expect.any(String) });
  });

  it("refuses to package wording that changed after the queue loaded", async () => {
    mocks.getDraftCurrentRevision.mockResolvedValueOnce({ currentRevision: { revisionId: "rev-current", body: "Evidence-backed note" } });
    mocks.getDraftCurrentRevision.mockResolvedValueOnce({ currentRevision: { revisionId: "rev-new", body: "Changed after queue load" } });
    renderReview("/review?draft=draft-needs-review");
    await screen.findByText("draft-needs-review");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.packageCreate }));
    expect(await screen.findByRole("alert")).toHaveTextContent(strings.pages.review.staleWording);
    expect(mocks.createEditorialPackage).not.toHaveBeenCalled();
  });

  it("keeps the exact wording receipt readable when optional manifest JSON is malformed", async () => {
    mocks.createEditorialPackage.mockResolvedValueOnce({ editorialPackage: { id: "package-bad-context", draftId: "draft-needs-review", sourceRevisionId: "rev-current", scope: "destination_unbound_wording", body: "Evidence-backed note", bodySha256: "body-hash", packageSha256: "package-hash", manifestJson: "{not-json", createdAt: "2026-11-21T00:00:00Z" } });
    renderReview("/review?draft=draft-needs-review");
    await screen.findByText("draft-needs-review");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.packageCreate }));
    expect(await screen.findByText(strings.pages.review.packageLimit)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.review.packageReceipt)).toBeInTheDocument();
    expect(screen.queryByText(strings.pages.review.packageContextTitle)).not.toBeInTheDocument();
  });

  it("records explicit body-bound evidence for a campaign-free draft without a stored revision", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-needs-review", status: "checking", body: "Evidence-backed note", postTypeId: "dev-log" }] });
    mocks.getDraftCurrentRevision.mockResolvedValueOnce({ currentRevision: { body: "Evidence-backed note" } });
    mocks.getDraftCurrentRevision.mockResolvedValueOnce({ currentRevision: { body: "Evidence-backed note" } });
    renderReview("/review?draft=draft-needs-review");
    expect(await screen.findByRole("button", { name: strings.pages.review.recordReview })).toBeInTheDocument();
    const firstResult = screen.getByLabelText(`evidence_grounding ${strings.pages.review.reviewResult}`);
    const firstEvidence = screen.getByLabelText(`evidence_grounding ${strings.pages.review.reviewEvidence}`);
    const secondResult = screen.getByLabelText(`fabricated_testimonial ${strings.pages.review.reviewResult}`);
    const secondEvidence = screen.getByLabelText(`fabricated_testimonial ${strings.pages.review.reviewEvidence}`);
    fireEvent.change(firstResult, { target: { value: "pass" } });
    fireEvent.change(firstEvidence, { target: { value: "Claim links to the local test receipt." } });
    fireEvent.change(secondResult, { target: { value: "pass" } });
    fireEvent.change(secondEvidence, { target: { value: "No testimonial appears in this copy." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.recordReview }));
    expect(await screen.findByText(strings.pages.review.reviewedReceipt)).toBeInTheDocument();
    expect(mocks.recordReviewRun).toHaveBeenCalledWith({ draftId: "draft-needs-review", expectedRevisionId: "", expectedBody: "Evidence-backed note", verdicts: [
      { mode: "evidence_grounding", passed: true, evidence: "Claim links to the local test receipt.", finding: "" },
      { mode: "fabricated_testimonial", passed: true, evidence: "No testimonial appears in this copy.", finding: "" },
    ] });
    expect(mocks.transitionDraft).toHaveBeenCalledWith({ id: "draft-needs-review", event: "review_pass" });
  });

  it("shows a review receipt loaded from the exact current server basis", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-needs-review", status: "reviewed", body: "Evidence-backed note", postTypeId: "dev-log" }] });
    mocks.getDraftCurrentRevision.mockResolvedValue({ currentRevision: { body: "Evidence-backed note", revisionId: "rev-current", reviewRunId: "run-persisted", hasApplicableReview: true } });
    renderReview("/review?draft=draft-needs-review");
    const receipt = await screen.findByTestId("review-basis-receipt");
    expect(receipt).toHaveTextContent(strings.pages.review.reviewRunReceipt);
    expect(receipt).toHaveAttribute("data-run-id", "run-persisted");
    expect(receipt).toHaveAttribute("data-revision-id", "rev-current");
  });

  it("prepares a requested campaign-free draft by applying each allowed lifecycle event", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-needs-review", status: "requested", body: "Evidence-backed note", postTypeId: "dev-log" }] });
    renderReview("/review?draft=draft-needs-review");
    fireEvent.click(await screen.findByRole("button", { name: strings.pages.review.prepareReview }));
    await waitFor(() => expect(mocks.transitionDraft).toHaveBeenCalledTimes(3));
    expect(mocks.transitionDraft.mock.calls.map(([request]) => request.event)).toEqual(["begin", "complete", "check"]);
  });

  it("returns to the exact content filters passed by the collection", async () => {
    renderReview("/review?draft=draft-needs-review&return=%2Fcontent%3Fstatus%3Dactive%26project%3Dproject-a");
    await screen.findByText("draft-needs-review");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.returnToContent }));
    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/content?status=active&project=project-a"));
  });

  it("distinguishes an empty queue from a filter with no matches and retries failures", async () => {
    mocks.listDrafts.mockResolvedValueOnce({ drafts: [] });
    const first = renderReview();
    expect(await screen.findByText(strings.pages.review.empty)).toBeInTheDocument();
    first.unmount();
    mocks.listDrafts.mockRejectedValueOnce(new Error("draft service offline")).mockResolvedValueOnce({ drafts: [] });
    renderReview();
    expect(await screen.findByRole("alert")).toHaveTextContent("draft service offline");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.review.retry }));
    await waitFor(() => expect(screen.getByText(strings.pages.review.empty)).toBeInTheDocument());
  });
});
