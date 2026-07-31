import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DashboardPage } from "./DashboardPage";
import { renderWithProviders } from "../test-utils";
import { assignAutomation, configurePortfolio, createIdentity, enqueueAction, overview, previewRelease, retireIdentity, updateIdentity } from "../api/channelManager";

vi.mock("../api/channelManager", () => ({
  createIdentity: vi.fn().mockResolvedValue({}),
  overview: vi.fn().mockResolvedValue({ identities: {}, actions: {}, platforms: { x: { id: "x" } } }),
  startProgram: vi.fn().mockResolvedValue({ status: "warming" }),
  enqueueAction: vi.fn().mockResolvedValue({ id: "action-1" }),
  completeAction: vi.fn().mockResolvedValue({ status: "succeeded" }),
  recordObservation: vi.fn().mockResolvedValue({ flag: null }),
	previewRelease: vi.fn().mockResolvedValue({ caption: "A preview", caption_truncated: false, disclosure_required: true, release_allowed: true, blocking_errors: [], first_comment: "" }),
	assignAutomation: vi.fn().mockResolvedValue({ identity_id: "x-1" }),
	dispatchBrowserAction: vi.fn().mockResolvedValue({ execution_id: "bas-execution-1" }),
	configurePortfolio: vi.fn().mockResolvedValue({}),
	retireIdentity: vi.fn().mockResolvedValue({ status: "retired" }),
	updateIdentity: vi.fn().mockResolvedValue({}),
}));

describe("DashboardPage", () => {
  // [REQ:CHANMGR-P0-015] [REQ:CHANMGR-P0-016] [REQ:CHANMGR-P0-019]
	it("guides a credential-free keyboard manual completion and observation", async () => {
		vi.mocked(overview).mockResolvedValueOnce({ identities: {}, actions: {}, platforms: { x: { id: "x" } }, programs: { "x-conservative": { id: "x-conservative", platform_id: "x", provenance: { confidence: "speculative", source_kind: "operator", revisit_trigger: "five runs" } } } });
    renderWithProviders(<DashboardPage />);
		await screen.findByText(/No identities yet/i);
    fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "x-1" } });
    fireEvent.change(screen.getByLabelText("Vault credential reference"), { target: { value: "vault://channel/x-1" } });
    fireEvent.click(screen.getByRole("button", { name: "Create identity and start warming" }));
    await waitFor(() => expect(screen.getByTestId("operator-identity-ready")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Queue manual engagement" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Record manual completion" })).not.toBeDisabled());
    fireEvent.change(screen.getByLabelText("Completion evidence"), { target: { value: "https://example.test/proof" } });
    fireEvent.click(screen.getByRole("button", { name: "Record manual completion" }));
    await waitFor(() => expect(screen.getAllByRole("status").some((node) => node.textContent?.includes("evidence"))).toBe(true));
    fireEvent.change(screen.getByLabelText("Reach or impressions"), { target: { value: "120" } });
    fireEvent.click(screen.getByRole("button", { name: "Record observation" }));
    await waitFor(() => expect(screen.getByText(/evidence, not a claim/i)).toBeInTheDocument());
  });

	it("allows a manual-only identity before a Vault reference is needed", async () => {
		vi.mocked(overview).mockResolvedValueOnce({ identities: {}, actions: {}, platforms: { x: { id: "x" } }, programs: { "x-conservative": { id: "x-conservative", platform_id: "x", provenance: { confidence: "speculative", source_kind: "operator", revisit_trigger: "five runs" } } } });
		renderWithProviders(<DashboardPage />);
		await screen.findByText(/No identities yet/i);
		fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "manual-only" } });
		fireEvent.click(screen.getByRole("button", { name: "Create identity and start warming" }));
		await waitFor(() => expect(createIdentity).toHaveBeenCalledWith(expect.objectContaining({ id: "manual-only", vault_ref: "" })));
	});

  // [REQ:CHANMGR-P0-018] [REQ:CHANMGR-P0-019]
  it("renders roster, due work, provenance, flags, and a purposeful empty state", async () => {
    vi.mocked(overview).mockResolvedValueOnce({
      identities: { "x-brand": { id: "x-brand", platform_id: "x", purpose: "brand", environment_ref: "env", vault_ref: "vault://ref", status: "warming", lane_grants: ["main"] } },
      actions: { "a-1": { id: "a-1", identity_id: "x-brand", kind: "engage", window: "2026-07-28T12:00:00Z", status: "scheduled", rolled_count: 2 } },
      programs: { "x-conservative": { id: "x-conservative", platform_id: "x", provenance: { confidence: "speculative", source_kind: "operator-practice", revisit_trigger: "five completed runs" } } },
      program_support: { "x-conservative": 3 },
      flags: { "x-brand": [{ message: "Reach measurement needs review" }] },
    });
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByLabelText("x-brand: warming")).toBeInTheDocument();
    expect(screen.getByText(/rolled count 2/i)).toBeInTheDocument();
    expect(screen.getByText(/operator-practice/i)).toBeInTheDocument();
    expect(screen.getByText(/3 completed run\(s\) support this program/i)).toBeInTheDocument();
    expect(screen.getByText(/Reach measurement needs review/i)).toBeInTheDocument();
  });

  it("[CHANMGR-P1-007] shows Content Desk delivery state and partial completion", async () => {
	vi.mocked(overview).mockResolvedValueOnce({ identities: {}, actions: {}, releases: { "release-1": { id: "release-1", draft_id: "draft-1", action_id: "action-1", status: "partial", platform_post_id: "post-1", published_url: "https://example.test/post-1", first_comment_status: "failed", delivery_status: "pending", delivery_error: "content desk unavailable" } }, metric_samples: { "sample-1": { id: "sample-1", release_id: "release-1", draft_id: "draft-1", metric: "impressions", value: 12, delivery_status: "pending" } } });
	renderWithProviders(<DashboardPage />);
	expect(await screen.findByText(/Content Desk pending/i)).toBeInTheDocument();
	expect(screen.getByText(/first comment failed/i)).toBeInTheDocument();
	expect(screen.getByRole("alert")).toHaveTextContent("content desk unavailable");
	expect(screen.getByText(/impressions 12/i)).toBeInTheDocument();
  });

	it("retires an identity without deleting its roster record", async () => {
		vi.mocked(overview).mockResolvedValueOnce({ identities: { "x-brand": { id: "x-brand", platform_id: "x", purpose: "brand", environment_ref: "env", vault_ref: "vault://ref", status: "active" } }, actions: {} });
		renderWithProviders(<DashboardPage />);
		fireEvent.click(await screen.findByRole("button", { name: "Retire" }));
		await waitFor(() => expect(retireIdentity).toHaveBeenCalledWith("x-brand"));
	});

	it("updates existing identity metadata through the operator console", async () => {
		vi.mocked(overview).mockResolvedValueOnce({ identities: { "x-brand": { id: "x-brand", platform_id: "x", purpose: "brand", environment_ref: "env", vault_ref: "vault://ref", status: "active" } }, actions: {} });
		renderWithProviders(<DashboardPage />);
		await screen.findByLabelText("x-brand: active");
		fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "x-brand" } });
		fireEvent.change(screen.getByLabelText("Identity display label"), { target: { value: "Brand account" } });
		fireEvent.click(screen.getByRole("button", { name: "Save identity metadata" }));
		await waitFor(() => expect(updateIdentity).toHaveBeenCalledWith("x-brand", expect.objectContaining({ display_label: "Brand account" })));
	});

	it("[CHANMGR-P1-009] renders a descriptor-driven preview before release", async () => {
		vi.mocked(overview).mockResolvedValueOnce({ identities: {}, actions: {}, platforms: { x: { id: "x" }, tiktok: { id: "tiktok" } } });
		renderWithProviders(<DashboardPage />);
		await screen.findAllByRole("option", { name: "tiktok" });
		fireEvent.change(screen.getByLabelText("Preview platform"), { target: { value: "tiktok" } });
		fireEvent.change(screen.getByLabelText("Caption"), { target: { value: "A preview" } });
		fireEvent.click(screen.getByLabelText(/Disclosure visible/i));
		fireEvent.click(screen.getByRole("button", { name: "Render platform preview" }));
		await waitFor(() => expect(previewRelease).toHaveBeenCalledWith(expect.objectContaining({ platform_id: "tiktok", caption: "A preview", disclosure_visible: true })));
		expect(screen.getByText("Ready for release")).toBeInTheDocument();
	});

	it("[CHANMGR-P1-001] requires an operator decision before saving a BAS profile reference", async () => {
		renderWithProviders(<DashboardPage />);
		fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "x-1" } });
		fireEvent.change(screen.getByLabelText("Declared BAS profile key"), { target: { value: "operator-account" } });
		fireEvent.change(screen.getByLabelText("BAS session profile reference"), { target: { value: "profile-1" } });
		fireEvent.change(screen.getByLabelText("BAS workflow reference"), { target: { value: "workflow-1" } });
		fireEvent.change(screen.getByLabelText("Automation acceptance note"), { target: { value: "approved sanctioned synthetic test" } });
		fireEvent.click(screen.getByRole("button", { name: "Save browser automation gate" }));
		await waitFor(() => expect(assignAutomation).toHaveBeenCalledWith("x-1", expect.objectContaining({ consumer_profile_key: "operator-account", session_profile_ref: "profile-1", workflow_ref: "workflow-1", enabled_action_kinds: ["engage"] })));
	});

	it("[CHANMGR-P1-002] saves the explicit cross-identity portfolio policy", async () => {
		renderWithProviders(<DashboardPage />);
		fireEvent.change(screen.getByLabelText("Minimum cross-identity posting gap in minutes"), { target: { value: "45" } });
		fireEvent.change(screen.getByLabelText("Portfolio rolling window in minutes"), { target: { value: "120" } });
		fireEvent.change(screen.getByLabelText("Maximum publishes per rolling window"), { target: { value: "2" } });
		fireEvent.click(screen.getByRole("button", { name: "Save portfolio policy" }));
		await waitFor(() => expect(configurePortfolio).toHaveBeenCalledWith({ minimum_post_gap_minutes: 45, window_minutes: 120, max_posts_per_window: 2 }));
	});

	it("keeps the console recoverable when the saved work list temporarily fails", async () => {
		vi.mocked(overview).mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({ identities: {}, actions: {} });
		renderWithProviders(<DashboardPage />);
		expect(await screen.findByRole("alert")).toHaveTextContent("could not load");
		fireEvent.click(screen.getByRole("button", { name: "Retry work list" }));
		await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
	});

	it("shows a safe operator message when a preview cannot be rendered", async () => {
		vi.mocked(previewRelease).mockRejectedValueOnce(new Error("descriptor unavailable"));
		renderWithProviders(<DashboardPage />);
		fireEvent.click(screen.getByRole("button", { name: "Render platform preview" }));
		expect(await screen.findByTestId("operator-status")).toHaveTextContent("Could not render the platform preview");
	});

	it("does not invent a second identity when creation and resume both fail", async () => {
		vi.mocked(overview).mockResolvedValueOnce({ identities: {}, actions: {}, platforms: { x: { id: "x" } } }).mockResolvedValueOnce({ identities: {}, actions: {}, platforms: { x: { id: "x" } } });
		vi.mocked(createIdentity).mockRejectedValueOnce(new Error("unavailable"));
		renderWithProviders(<DashboardPage />);
		await screen.findAllByRole("option", { name: "x" });
		fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "x-missing" } });
		fireEvent.change(screen.getByLabelText("Vault credential reference"), { target: { value: "vault://channel/x-missing" } });
		fireEvent.click(screen.getByRole("button", { name: "Create identity and start warming" }));
		await waitFor(() => expect(createIdentity).toHaveBeenCalled());
		expect(await screen.findByTestId("operator-status")).toHaveTextContent("Could not create or resume the identity");
	});

	it("recovers an existing identity and its already-scheduled action after a transient submit failure", async () => {
		vi.mocked(overview)
			.mockResolvedValueOnce({ identities: { "x-existing": { id: "x-existing", platform_id: "x", purpose: "brand", environment_ref: "env", status: "warming" } }, actions: { "action-existing": { id: "action-existing", identity_id: "x-existing", kind: "engage", window: "2026-07-28T12:00:00Z", status: "scheduled", rolled_count: 0 } }, platforms: { x: { id: "x" } } })
			.mockResolvedValueOnce({ identities: { "x-existing": { id: "x-existing", platform_id: "x", purpose: "brand", environment_ref: "env", status: "warming" } }, actions: { "action-existing": { id: "action-existing", identity_id: "x-existing", kind: "engage", window: "2026-07-28T12:00:00Z", status: "scheduled", rolled_count: 0 } }, platforms: { x: { id: "x" } } })
			.mockResolvedValueOnce({ identities: { "x-existing": { id: "x-existing", platform_id: "x", purpose: "brand", environment_ref: "env", status: "warming" } }, actions: { "action-existing": { id: "action-existing", identity_id: "x-existing", kind: "engage", window: "2026-07-28T12:00:00Z", status: "scheduled", rolled_count: 0 } }, platforms: { x: { id: "x" } } });
		vi.mocked(createIdentity).mockRejectedValueOnce(new Error("already exists"));
		vi.mocked(enqueueAction).mockRejectedValueOnce(new Error("already queued"));
		renderWithProviders(<DashboardPage />);
		await screen.findByLabelText("x-existing: warming");
		fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "x-existing" } });
		fireEvent.click(screen.getByRole("button", { name: "Create identity and start warming" }));
		await screen.findByText(/Existing identity resumed/i);
		fireEvent.click(screen.getByRole("button", { name: "Queue manual engagement" }));
		await screen.findByText(/Existing manual action resumed/i);
	});

	it("explains when a rejected queue request has no durable action to resume", async () => {
		vi.mocked(overview)
			.mockResolvedValueOnce({ identities: {}, actions: {}, platforms: { x: { id: "x" } }, programs: { "x-warm": { id: "x-warm", platform_id: "x" } } })
			.mockResolvedValueOnce({ identities: {}, actions: {} });
		vi.mocked(enqueueAction).mockRejectedValueOnce(new Error("cadence"));
		renderWithProviders(<DashboardPage />);
		await screen.findByText(/No identities yet/i);
		fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "x-new" } });
		fireEvent.click(screen.getByRole("button", { name: "Create identity and start warming" }));
		await screen.findByText(/Identity created and warming program started/i);
		fireEvent.click(screen.getByRole("button", { name: "Queue manual engagement" }));
		expect(await screen.findByTestId("operator-status")).toHaveTextContent("Could not queue or resume this action");
	});
});
