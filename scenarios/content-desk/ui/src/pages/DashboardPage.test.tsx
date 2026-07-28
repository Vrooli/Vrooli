import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils/renderWithProviders";
import { DashboardPage } from "./DashboardPage";

const mocks = vi.hoisted(() => ({
  listDrafts: vi.fn(),
  approveDraft: vi.fn(),
  updateDraftBody: vi.fn(),
  listClaims: vi.fn(),
  listDraftClaims: vi.fn(),
  citeClaim: vi.fn(),
  verifyClaim: vi.fn(),
  listReviewRuns: vi.fn(),
	listPublishRecords: vi.fn(),
}));

vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, approveDraft: mocks.approveDraft, updateDraftBody: mocks.updateDraftBody } }));
vi.mock("../api/claims", () => ({ claimsClient: { listClaims: mocks.listClaims, listDraftClaims: mocks.listDraftClaims, citeClaim: mocks.citeClaim, verifyClaim: mocks.verifyClaim } }));
vi.mock("../api/review", () => ({ reviewClient: { listReviewRuns: mocks.listReviewRuns } }));
vi.mock("../api/ledger", () => ({ ledgerClient: { listPublishRecords: mocks.listPublishRecords } }));

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listDrafts.mockResolvedValue({ drafts: [] });
    mocks.approveDraft.mockResolvedValue({});
    mocks.updateDraftBody.mockResolvedValue({});
    mocks.listClaims.mockResolvedValue({ claims: [] });
    mocks.listDraftClaims.mockResolvedValue({ claims: [] });
    mocks.citeClaim.mockResolvedValue({});
    mocks.verifyClaim.mockResolvedValue({});
    mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [] });
		mocks.listPublishRecords.mockResolvedValue({ publishRecords: [] });
  });

  it("loads the queue and enables approval only when every displayed gate passes", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-1", status: "reviewed", campaignId: "campaign-1", body: "Approved content" }] });
    mocks.listClaims.mockResolvedValue({ claims: [{ id: "claim-1", statement: "Verified statement", verificationStatus: "verified" }] });
    mocks.listDraftClaims.mockResolvedValue({ claims: [{ id: "claim-1", statement: "Verified statement", verificationStatus: "verified" }] });
    mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [{ id: "review-1", draftId: "draft-1", outcome: "passed" }] });
    renderWithProviders(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));

    await screen.findByText("draft-1");
    expect(screen.getByRole("button", { name: "Approve draft" })).toBeEnabled();
    expect(screen.getByLabelText("Draft body")).toHaveValue("Approved content");
    expect(mocks.listDrafts).toHaveBeenCalledWith({});
  });

  it("submits operator approval and refreshes the ledger queue", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-approve", status: "reviewed", campaignId: "campaign-1", body: "Ready" }] });
    mocks.listClaims.mockResolvedValue({ claims: [] });
    mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [{ id: "review-approve", draftId: "draft-approve", outcome: "passed" }] });
    renderWithProviders(<DashboardPage />);
    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByText("draft-approve");
    fireEvent.click(screen.getByRole("button", { name: "Approve draft" }));
    await waitFor(() => expect(mocks.approveDraft).toHaveBeenCalledWith({ id: "draft-approve" }));
  });

  it("saves an editable operator revision through the artifact API", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-edit", status: "drafting", campaignId: "campaign-1", body: "Before" }] });
    renderWithProviders(<DashboardPage />);
    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByText("draft-edit");
    fireEvent.change(screen.getByLabelText("Draft body"), { target: { value: "After" } });
    fireEvent.click(screen.getByRole("button", { name: "Save revision" }));
    await waitFor(() => expect(mocks.updateDraftBody).toHaveBeenCalledWith({ id: "draft-edit", body: "After" }));
  });

  it("attaches a shared claim to the selected draft's exact body span", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-cite", status: "drafting", campaignId: "campaign-1", body: "Claim text" }] });
    mocks.listClaims.mockResolvedValue({ claims: [{ id: "claim-cite", statement: "Claim text", verificationStatus: "asserted" }] });
    renderWithProviders(<DashboardPage />);
    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByText("draft-cite");
    fireEvent.change(screen.getByLabelText("Claim to cite"), { target: { value: "claim-cite" } });
    fireEvent.change(screen.getByLabelText("Claim span end"), { target: { value: "10" } });
    fireEvent.click(screen.getByRole("button", { name: "Attach claim" }));
    await waitFor(() => expect(mocks.citeClaim).toHaveBeenCalledWith({ draftId: "draft-cite", claimId: "claim-cite", spanStart: 0, spanEnd: 10, body: "Claim text" }));
  });

  it("runs verification for an unverified cited claim", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-verify", status: "reviewed", campaignId: "campaign-1", body: "Claim" }] });
    mocks.listDraftClaims.mockResolvedValue({ claims: [{ id: "claim-verify", statement: "Claim", verificationStatus: "asserted" }] });
    renderWithProviders(<DashboardPage />);
    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByRole("button", { name: "Verify claim" });
    fireEvent.click(screen.getByRole("button", { name: "Verify claim" }));
    await waitFor(() => expect(mocks.verifyClaim).toHaveBeenCalledWith({ id: "claim-verify" }));
  });

  it("explains every approval blocker for an incomplete draft", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-2", status: "editing", campaignId: "campaign-2" }] });
    mocks.listClaims.mockResolvedValue({ claims: [{ id: "claim-2", statement: "Needs review", verificationStatus: "stale" }] });
    mocks.listDraftClaims.mockResolvedValue({ claims: [{ id: "claim-2", statement: "Needs review", verificationStatus: "stale" }] });
    mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [{ id: "review-2", draftId: "draft-2", outcome: "failed" }] });
    renderWithProviders(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));

    await screen.findByText("Claim claim-2 is stale.");
    expect(screen.getByText("Draft must complete review before approval.")).toBeInTheDocument();
    expect(screen.getByText("A review run has blocking verdicts.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve draft" })).toBeDisabled();
  });

  it("renders empty queue and inspector states after a successful refresh", async () => {
    renderWithProviders(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));

    await screen.findByText("No drafts are queued.");
    expect(screen.getByText("Select a draft to inspect it.")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Select claim" })).toBeInTheDocument();
		expect(screen.getByText("No publish records yet.")).toBeInTheDocument();
  });

	it("shows published history in the desk inspector", async () => {
		mocks.listPublishRecords.mockResolvedValue({ publishRecords: [{ id: "record-1", draftId: "draft-published", publishedUrl: "https://example.test/post", platformPostId: "post-1" }] });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
		await screen.findByRole("link", { name: "post-1" });
		expect(screen.getByRole("link", { name: "post-1" })).toHaveAttribute("href", "https://example.test/post");
	});

  it("surfaces a load failure", async () => {
    mocks.listDrafts.mockRejectedValue(new Error("desk unavailable"));
    renderWithProviders(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("desk unavailable"));
  });

  it("surfaces an operator approval refusal instead of silently publishing", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-refused", status: "reviewed", campaignId: "campaign-1", body: "Ready" }] });
    mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [{ id: "review-refused", draftId: "draft-refused", outcome: "passed" }] });
    mocks.approveDraft.mockRejectedValue(new Error("operator approval required"));
    renderWithProviders(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByText("draft-refused");
    fireEvent.click(screen.getByRole("button", { name: "Approve draft" }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("operator approval required"));
  });

  it("retains a visible error when attaching or verifying evidence fails", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-evidence-error", status: "drafting", campaignId: "campaign-1", body: "Claim text" }] });
    mocks.listClaims.mockResolvedValue({ claims: [{ id: "claim-error", statement: "Claim text", verificationStatus: "asserted" }] });
    mocks.listDraftClaims.mockResolvedValue({ claims: [{ id: "claim-error", statement: "Claim text", verificationStatus: "asserted" }] });
    mocks.citeClaim.mockRejectedValue(new Error("citation span is invalid"));
    mocks.verifyClaim.mockRejectedValue(new Error("check runner unavailable"));
    renderWithProviders(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByText("draft-evidence-error");
    fireEvent.change(screen.getByLabelText("Claim to cite"), { target: { value: "claim-error" } });
    fireEvent.change(screen.getByLabelText("Claim span end"), { target: { value: "10" } });
    fireEvent.click(screen.getByRole("button", { name: "Attach claim" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("citation span is invalid"));

    fireEvent.click(screen.getByRole("button", { name: "Verify claim" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("check runner unavailable"));
  });

  it("surfaces a failed revision save and keeps the draft selected", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-save-error", status: "drafting", campaignId: "campaign-1", body: "Before" }] });
    mocks.updateDraftBody.mockRejectedValue(new Error("revision conflict"));
    renderWithProviders(<DashboardPage />);

    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByText("draft-save-error");
    fireEvent.change(screen.getByLabelText("Draft body"), { target: { value: "After" } });
    fireEvent.click(screen.getByRole("button", { name: "Save revision" }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("revision conflict"));
    expect(screen.getByLabelText("Draft body")).toHaveValue("After");
  });
});
