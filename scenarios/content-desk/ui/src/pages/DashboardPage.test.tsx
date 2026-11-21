import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { DashboardPage, editorShapeForPostType } from "./DashboardPage";

const mocks = vi.hoisted(() => ({
  listDrafts: vi.fn(),
  approveDraft: vi.fn(),
	updateDraftBody: vi.fn(),
	attachReleasedAsset: vi.fn(),
	listDraftAttachments: vi.fn(),
	 submitReleaseDraft: vi.fn(),
	commissionAgentWork: vi.fn(),
  listClaims: vi.fn(),
  listDraftClaims: vi.fn(),
  citeClaim: vi.fn(),
  verifyClaim: vi.fn(),
	getClaimCoverage: vi.fn(),
	listClaimProposals: vi.fn(),
	extractClaimProposals: vi.fn(),
	decideClaimProposal: vi.fn(),
  listReviewRuns: vi.fn(),
	listPublishRecords: vi.fn(),
	listRemediations: vi.fn(),
	createRemediation: vi.fn(),
	resolveRemediation: vi.fn(),
}));

vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, approveDraft: mocks.approveDraft, updateDraftBody: mocks.updateDraftBody, submitReleaseDraft: mocks.submitReleaseDraft, attachReleasedAsset: mocks.attachReleasedAsset, listDraftAttachments: mocks.listDraftAttachments, commissionAgentWork: mocks.commissionAgentWork } }));
vi.mock("../api/claims", () => ({ claimsClient: { listClaims: mocks.listClaims, listDraftClaims: mocks.listDraftClaims, citeClaim: mocks.citeClaim, verifyClaim: mocks.verifyClaim, getClaimCoverage: mocks.getClaimCoverage, listClaimProposals: mocks.listClaimProposals, extractClaimProposals: mocks.extractClaimProposals, decideClaimProposal: mocks.decideClaimProposal } }));
vi.mock("../api/review", () => ({ reviewClient: { listReviewRuns: mocks.listReviewRuns } }));
vi.mock("../api/ledger", () => ({ ledgerClient: { listPublishRecords: mocks.listPublishRecords, listRemediations: mocks.listRemediations, createRemediation: mocks.createRemediation, resolveRemediation: mocks.resolveRemediation } }));

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listDrafts.mockResolvedValue({ drafts: [] });
    mocks.approveDraft.mockResolvedValue({});
    mocks.updateDraftBody.mockResolvedValue({});
		mocks.attachReleasedAsset.mockResolvedValue({ attachment: { id: "attachment-1" } });
		mocks.listDraftAttachments.mockResolvedValue({ attachments: [] });
		mocks.submitReleaseDraft.mockResolvedValue({ releaseId: "release-1", actionId: "action-1", releaseStatus: "queued" });
		mocks.commissionAgentWork.mockResolvedValue({ commissionId: "commission-1", taskId: "task-1", runId: "run-1", status: "RUN_STATUS_QUEUED" });
    mocks.listClaims.mockResolvedValue({ claims: [] });
    mocks.listDraftClaims.mockResolvedValue({ claims: [] });
    mocks.citeClaim.mockResolvedValue({});
    mocks.verifyClaim.mockResolvedValue({});
		mocks.getClaimCoverage.mockResolvedValue({ supportedSpans: [], uncoveredSpans: [] });
		mocks.listClaimProposals.mockResolvedValue({ proposals: [] });
		mocks.extractClaimProposals.mockResolvedValue({ proposals: [] });
		mocks.decideClaimProposal.mockResolvedValue({ proposal: {} });
    mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [] });
		mocks.listPublishRecords.mockResolvedValue({ publishRecords: [] });
		mocks.listRemediations.mockResolvedValue({ remediations: [] });
		mocks.createRemediation.mockResolvedValue({ remediation: { id: "remediation-1" } });
		mocks.resolveRemediation.mockResolvedValue({ remediation: { id: "remediation-1", status: "resolved" } });
  });

  // [REQ:CONTENTD-P0-014]
  it("[CONTENTD-P0-014] loads the queue and enables approval only when every displayed gate passes", async () => {
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
    await waitFor(() => expect(mocks.approveDraft).toHaveBeenCalledWith({ id: "draft-approve", identityId: "", lane: "" }));
  });

  it("[CONTENTD-P1-005] sends a targeted approval through the eligibility gate", async () => {
		mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-target", status: "reviewed", campaignId: "campaign-1", body: "Ready" }] });
		mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [{ id: "review-target", draftId: "draft-target", outcome: "passed" }] });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
		await screen.findByText("draft-target");
		fireEvent.change(screen.getByLabelText("Approval target identity"), { target: { value: "identity-1" } });
		fireEvent.change(screen.getByLabelText("Approval target lane"), { target: { value: "main" } });
		fireEvent.click(screen.getByRole("button", { name: "Approve draft" }));
		await waitFor(() => expect(mocks.approveDraft).toHaveBeenCalledWith({ id: "draft-target", identityId: "identity-1", lane: "main" }));
	});

  it("[CONTENTD-P1-006] submits an approved draft to Channel Manager without calling it published", async () => {
		mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-release", status: "approved", campaignId: "campaign-1", body: "Ready" }] });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
		await screen.findByText("draft-release");
		fireEvent.change(screen.getByLabelText("Channel Manager identity"), { target: { value: "identity-1" } });
		fireEvent.change(screen.getByLabelText("Channel Manager lane"), { target: { value: "main" } });
		fireEvent.change(screen.getByLabelText("Release idempotency key"), { target: { value: "release-key" } });
		fireEvent.click(screen.getByRole("button", { name: "Submit release" }));
		await waitFor(() => expect(mocks.submitReleaseDraft).toHaveBeenCalledWith({ id: "draft-release", identityId: "identity-1", lane: "main", idempotencyKey: "release-key", disclosureVisible: false }));
		expect(await screen.findByText(/Publication is recorded only after its outcome returns/i)).toBeInTheDocument();
	});

	it("[CONTENTD-P1-013] records and resolves a contamination remediation", async () => {
		mocks.listRemediations.mockResolvedValue({ remediations: [{ id: "remediation-open", publishRecordId: "publish-1", kind: "retract", note: "stale claim", status: "open" }] });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
		await screen.findByText("stale claim");
		fireEvent.click(screen.getByRole("button", { name: "Resolve remediation" }));
		await waitFor(() => expect(mocks.resolveRemediation).toHaveBeenCalledWith({ id: "remediation-open" }));
		fireEvent.change(screen.getByLabelText("Remediation publish record"), { target: { value: "publish-2" } });
		fireEvent.change(screen.getByLabelText("Remediation note"), { target: { value: "correction link" } });
		fireEvent.click(screen.getByRole("button", { name: "Record remediation" }));
		await waitFor(() => expect(mocks.createRemediation).toHaveBeenCalledWith({ publishRecordId: "publish-2", kind: "correct_in_place", note: "correction link" }));
	});

  it("[CONTENTD-P1-012] presents supported and uncovered spans in accessible text", async () => {
		mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-coverage", status: "drafting", campaignId: "campaign-1", body: "Claim text" }] });
		mocks.getClaimCoverage.mockResolvedValue({ supportedSpans: [{ start: 0, end: 5, claimId: "claim-1", supported: true }], uncoveredSpans: [{ start: 5, end: 10, supported: false }] });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
		await screen.findByText(/1 supported span; 1 uncovered span/i);
		expect(screen.getByText(/Review uncovered assertions before approval/i)).toBeInTheDocument();
		expect(mocks.getClaimCoverage).toHaveBeenCalledWith({ draftId: "draft-coverage", body: "Claim text" });
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

	it("[CONTENTD-P1-007] presents extracted text as review-only proposals", async () => {
		mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-extract", status: "drafting", campaignId: "campaign-1", body: "Vrooli has a typed API." }] });
		mocks.extractClaimProposals.mockResolvedValue({ proposals: [{ id: "proposal-1", statement: "Vrooli has a typed API.", status: "proposed" }] });
		mocks.listClaimProposals.mockResolvedValue({ proposals: [{ id: "proposal-1", statement: "Vrooli has a typed API.", status: "accepted" }] });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
		await screen.findByText("draft-extract");
		fireEvent.click(screen.getByRole("button", { name: "Extract claim proposals" }));
		fireEvent.click(await screen.findByRole("button", { name: "Accept proposal" }));
		await waitFor(() => expect(mocks.decideClaimProposal).toHaveBeenCalledWith({ id: "proposal-1", status: "accepted" }));
	});

	it("[CONTENTD-P1-009] attaches a released Asset Studio reference with accessible metadata", async () => {
		mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-asset", status: "drafting", campaignId: "campaign-1", body: "Image draft" }] });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
		await screen.findByText("draft-asset");
		fireEvent.change(screen.getByLabelText("Released asset id"), { target: { value: "asset-1" } });
		fireEvent.change(screen.getByLabelText("Attachment alt text"), { target: { value: "A descriptive image" } });
		fireEvent.click(screen.getByRole("button", { name: "Attach released asset" }));
		await waitFor(() => expect(mocks.attachReleasedAsset).toHaveBeenCalledWith({ draftId: "draft-asset", assetId: "asset-1", role: "hero", aspectRatio: "16:9", altText: "A descriptive image", position: 0 }));
	});

  // [REQ:CONTENTD-P1-010]
  it("[CONTENTD-P1-010] selects thread and long-form guidance from the draft post type", async () => {
    expect(editorShapeForPostType("thread")).toBe("thread");
    expect(editorShapeForPostType("long-form")).toBe("long-form");
    expect(editorShapeForPostType("dev-log")).toBe("standard");
    mocks.listDrafts.mockResolvedValue({ drafts: [
      { id: "draft-thread", status: "drafting", campaignId: "campaign-1", postTypeId: "thread", body: "First post\n\nSecond post" },
      { id: "draft-long", status: "drafting", campaignId: "campaign-1", postTypeId: "long-form", body: "Essay" },
    ] });
    renderWithProviders(<DashboardPage />);
    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await waitFor(() => expect(screen.getByRole("region", { name: "Thread authoring guidance" })).toHaveTextContent("Post 1: 10/280 characters"));
    fireEvent.click(screen.getByRole("button", { name: /draft-long/i }));
    expect(await screen.findByRole("region", { name: "Long-form authoring guidance" })).toHaveTextContent("Reserve a banner image slot");
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

  it("[CONTENTD-P1-011] commissions only a governed, editable workbench suggestion", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-agent", status: "drafting", campaignId: "campaign-1", body: "Draft" }] });
    renderWithProviders(<DashboardPage />);
    fireEvent.click(screen.getByRole("button", { name: "Refresh desk" }));
    await screen.findByText("draft-agent");
    fireEvent.click(screen.getByRole("button", { name: "Find evidence" }));
    await waitFor(() => expect(mocks.commissionAgentWork).toHaveBeenCalledWith({ draftId: "draft-agent", action: "evidence-hunt" }));
    expect(screen.getByText(/run-1 was commissioned for evidence-hunt/i)).toBeInTheDocument();
  });
});
