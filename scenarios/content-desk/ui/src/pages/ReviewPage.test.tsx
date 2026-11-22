import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { ReviewPage } from "./ReviewPage";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn(), getDraftCurrentRevision: vi.fn(), createEditorialPackage: vi.fn() }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, getDraftCurrentRevision: mocks.getDraftCurrentRevision } }));
vi.mock("../api/review", () => ({ reviewClient: { createEditorialPackage: mocks.createEditorialPackage } }));
vi.mock("./DashboardPage", () => ({ DashboardPage: ({ initialSelectedDraftId, preserveEmptySelection }: { initialSelectedDraftId?: string; preserveEmptySelection?: boolean }) => <div data-testid="existing-review-workbench" data-selected-draft-id={initialSelectedDraftId ?? ""} data-preserve-empty={String(Boolean(preserveEmptySelection))} /> }));
function Location() { const location = useLocation(); return <output data-testid="location">{location.pathname}{location.search}</output>; }
function renderReview(initialEntry = "/review") {
  return renderWithProviders(<MemoryRouter initialEntries={[initialEntry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}><Routes><Route path="/review" element={<ReviewPage />} /><Route path="/content" element={<div data-testid="returned-to-content" />} /></Routes><Location /></MemoryRouter>, { withoutRouter: true });
}

describe("ReviewPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getDraftCurrentRevision.mockResolvedValue({ currentRevision: { revisionId: "rev-current", body: "Evidence-backed note" } });
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
