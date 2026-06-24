import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { DashboardPage } from "./DashboardPage";
import { DevicesPage } from "./DevicesPage";
import { OptimizationPage } from "./OptimizationPage";
import { ResolverPolicyPage } from "./ResolverPolicyPage";
import { SettingsPage } from "./SettingsPage";
import { SnapshotsPage } from "./SnapshotsPage";

const api = vi.hoisted(() => ({
  fetchControlCenterOverview: vi.fn(),
  fetchResolverStatus: vi.fn(),
  fetchDevices: vi.fn(),
  previewPolicyChange: vi.fn(),
  applyPolicyChange: vi.fn(),
  rollbackPolicyChange: vi.fn(),
  fetchPolicyProfiles: vi.fn(),
  upsertPolicyProfile: vi.fn(),
  evaluatePolicySchedule: vi.fn(),
  diagnoseEncryptedDnsBypass: vi.fn(),
  fetchEndpointDohGuidance: vi.fn(),
  previewUpstreams: vi.fn(),
  refreshInventory: vi.fn(),
  updateDeviceGroup: vi.fn(),
  createOptimizationRun: vi.fn(),
  scoreOptimizationRun: vi.fn(),
  approveOptimizationCandidate: vi.fn(),
  fetchPrivacySettings: vi.fn(),
  runSnapshot: vi.fn(),
  exportSnapshotReport: vi.fn(),
}));

vi.mock("../api/network", () => api);

describe("Network Manager control pages", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders dashboard loading, empty, and live overview states", async () => {
    // [REQ:NM-P0-009] Operator workflows expose real service-backed dashboard state.
    api.fetchControlCenterOverview.mockResolvedValueOnce({
      snapshots: [],
      resolverStatus: { backend: "none", status: "unconfigured", filteringEnabled: false, upstreams: [], warnings: [] },
      capabilities: [],
      devices: [],
      retention: { queryLogDays: 0, snapshotDays: 14, experimentDays: 14, profile: "minimal" },
      visibility: { showQueryDomains: false, showDeviceHistory: false, householdMode: true, notes: [] },
    });

    renderWithProviders(<DashboardPage />);

    expect(screen.getByTestId(selectors.network.loading)).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.network.empty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.network.resolverStatus)).toBeInTheDocument();
  });

  it("renders populated dashboard readiness and capability branches", async () => {
    api.fetchControlCenterOverview.mockResolvedValueOnce({
      snapshots: [
        {
          id: "snapshot-1",
          status: "baseline",
          profile: "home",
          summary: "9 healthy",
          createdAt: "2026-06-23T19:00:00Z",
          metrics: [],
          findings: [],
        },
      ],
      resolverStatus: {
        backend: "adguardhome",
        status: "healthy",
        filteringEnabled: true,
        upstreams: ["https://dns.example.test/dns-query"],
        warnings: [],
      },
      capabilities: [
        { adapter: "resolver", action: "status", supported: true, requiresAdmin: false, rollbackSupported: false, reason: "" },
        { adapter: "router", action: "write", supported: false, requiresAdmin: true, rollbackSupported: false, reason: "unsupported" },
      ],
      devices: [],
      retention: { queryLogDays: 0, snapshotDays: 14, experimentDays: 14, profile: "minimal" },
      visibility: { showQueryDomains: false, showDeviceHistory: false, householdMode: false, notes: [] },
    });

    renderWithProviders(<DashboardPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.network.latestSnapshot)).toHaveTextContent("9 healthy"));
    expect(screen.getByTestId(selectors.network.resolverStatus)).toHaveTextContent("network.enabled");
    expect(screen.getByTestId(selectors.network.capabilitySummary)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.network.privacySummary)).toHaveTextContent("network.disabled");
  });

  it("renders dashboard error state", async () => {
    api.fetchControlCenterOverview.mockRejectedValueOnce(new Error("offline"));

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByTestId(selectors.network.error)).toBeInTheDocument();
  });

  it("keeps policy apply disabled until preview returns approval evidence", async () => {
    // [REQ:NM-P0-009] Risky operator actions stay preview-first and approval-gated in the UI.
    api.fetchResolverStatus.mockResolvedValueOnce({
      backend: "adguardhome",
      status: "healthy",
      filteringEnabled: true,
      upstreams: ["https://dns.example.test/dns-query"],
      warnings: [],
    });
    api.fetchPolicyProfiles.mockResolvedValueOnce([]);
    api.previewPolicyChange.mockResolvedValueOnce({
      id: "preview-1",
      target: "all-devices",
      action: "denylist",
      status: "preview",
      effects: ["Would add example.test"],
      rollbackSupported: true,
    });
    api.previewUpstreams.mockResolvedValueOnce(["Would update upstreams"]);
    api.applyPolicyChange.mockResolvedValueOnce({
      id: "change-1",
      target: "all-devices",
      action: "denylist",
      status: "applied",
      effects: ["Applied"],
      rollbackSupported: true,
    });
    api.rollbackPolicyChange.mockResolvedValueOnce({
      id: "change-1",
      target: "all-devices",
      action: "rollback",
      status: "rolled_back",
      effects: ["Rolled back"],
      rollbackSupported: false,
    });

    renderWithProviders(<ResolverPolicyPage />);

    const apply = screen.getByRole("button", { name: "pages.resolver.approveApply" });
    expect(apply).toBeDisabled();
    await userEvent.clear(screen.getByLabelText(strings.pages.resolver.upstreams));
    await userEvent.type(screen.getByLabelText(strings.pages.resolver.upstreams), "https://dns-alt.example.test/dns-query");
    await userEvent.click(screen.getByRole("button", { name: "pages.resolver.previewUpstreams" }));
    await waitFor(() => expect(api.previewUpstreams).toHaveBeenCalledWith(["https://dns-alt.example.test/dns-query"]));
    await userEvent.click(screen.getByRole("button", { name: "pages.resolver.previewPolicy" }));
    expect(await screen.findByTestId(selectors.network.policyPreview)).toBeInTheDocument();
    expect(apply).toBeEnabled();
    await userEvent.click(apply);
    await waitFor(() => expect(api.applyPolicyChange).toHaveBeenCalledWith("preview-1"));
    await userEvent.click(screen.getByRole("button", { name: "pages.resolver.rollback" }));
    await waitFor(() => expect(api.rollbackPolicyChange).toHaveBeenCalledWith("change-1"));
  });

  it("creates household policy profiles and evaluates schedules", async () => {
    // [REQ:NM-P1-001] [REQ:NM-P1-002] Household profiles and schedules are visible operator workflows.
    api.fetchResolverStatus.mockResolvedValueOnce({
      backend: "adguardhome",
      status: "healthy",
      filteringEnabled: true,
      upstreams: [],
      warnings: [],
    });
    api.fetchPolicyProfiles.mockResolvedValueOnce([]);
    api.upsertPolicyProfile.mockResolvedValueOnce({
      id: "profile-kids",
      name: "Kids",
      deviceGroup: "kids",
      filteringStrength: "strict",
      schedule: "daily:20:00-07:00",
      overrideBehavior: "parent_override",
      status: "enabled",
      effects: ["stored"],
      updatedAt: "2026-06-23T20:00:00Z",
    });
    api.fetchPolicyProfiles.mockResolvedValueOnce([
      {
        id: "profile-kids",
        name: "Kids",
        deviceGroup: "kids",
        filteringStrength: "strict",
        schedule: "daily:20:00-07:00",
        overrideBehavior: "parent_override",
        status: "enabled",
        effects: ["stored"],
        updatedAt: "2026-06-23T20:00:00Z",
      },
    ]);
    api.evaluatePolicySchedule.mockResolvedValueOnce({
      profileId: "profile-kids",
      profileName: "Kids",
      target: "group:kids",
      active: true,
      status: "active",
      effects: ["Schedule evaluation is advisory until an approved policy apply is run."],
      nextChangeAt: "2026-06-24T07:00:00Z",
    });

    renderWithProviders(<ResolverPolicyPage />);

    expect(await screen.findByTestId(selectors.network.policyProfiles)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "pages.resolver.saveProfile" }));
    await waitFor(() => expect(api.upsertPolicyProfile).toHaveBeenCalledWith(expect.objectContaining({
      name: "Kids",
      deviceGroup: "kids",
      filteringStrength: "strict",
      schedule: "daily:20:00-07:00",
      overrideBehavior: "parent_override",
    })));
    expect(await screen.findByText(/profile-kids|Kids/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "pages.resolver.evaluateSchedule" }));
    await waitFor(() => expect(api.evaluatePolicySchedule).toHaveBeenCalledWith("profile-kids", "group:kids"));
    expect(await screen.findByText(/active/)).toBeInTheDocument();
  });

  it("renders encrypted DNS and endpoint DoH guidance controls", async () => {
    // [REQ:NM-P1-004] [REQ:NM-P1-008] Operators can generate bypass and endpoint DoH guidance without invasive enforcement.
    api.fetchResolverStatus.mockResolvedValueOnce({
      backend: "adguardhome",
      status: "healthy",
      filteringEnabled: true,
      upstreams: [],
      warnings: [],
    });
    api.fetchPolicyProfiles.mockResolvedValueOnce([]);
    api.diagnoseEncryptedDnsBypass.mockResolvedValueOnce({
      id: "guidance-bypass",
      target: "network",
      profile: "ipv6-encrypted-dns",
      status: "manual_required",
      checks: [{ id: "doh", title: "DNS over HTTPS bypass", status: "guidance_only", evidence: "endpoint policy", recommendations: [] }],
      manualSteps: [],
      adapterActions: [],
      guardrails: ["Do not inspect or log user browsing contents to detect bypasses."],
      generatedAt: "2026-06-23T20:00:00Z",
    });
    api.fetchEndpointDohGuidance.mockResolvedValueOnce({
      id: "guidance-doh",
      target: "windows/chrome",
      profile: "endpoint-doh",
      status: "guidance_only",
      checks: [{ id: "privacy", title: "Privacy boundary", status: "enforced_by_design", evidence: "No TLS interception.", recommendations: [] }],
      manualSteps: [],
      adapterActions: [],
      guardrails: ["No TLS interception."],
      generatedAt: "2026-06-23T20:00:00Z",
    });

    renderWithProviders(<ResolverPolicyPage />);

    expect(await screen.findByTestId(selectors.network.policyGuidance)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "pages.resolver.bypassGuidance" }));
    await waitFor(() => expect(api.diagnoseEncryptedDnsBypass).toHaveBeenCalledWith("network", false));
    expect(await screen.findByText(/ipv6-encrypted-dns/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "pages.resolver.dohGuidance" }));
    await waitFor(() => expect(api.fetchEndpointDohGuidance).toHaveBeenCalledWith({
      platform: "windows",
      browser: "chrome",
      managementMode: "group-policy",
    }));
    expect(await screen.findByText(/endpoint-doh/)).toBeInTheDocument();
  });

  it("runs snapshot actions and renders report output", async () => {
    // [REQ:NM-P0-009] Snapshot controls call the shared operation contract and surface evidence.
    const snapshot = {
      id: "snapshot-1",
      status: "baseline",
      profile: "home",
      summary: "9 healthy",
      createdAt: "2026-06-23T19:00:00Z",
      metrics: [{ name: "dns", value: "12", unit: "ms", status: "healthy" }],
      findings: [],
    };
    api.fetchControlCenterOverview.mockResolvedValueOnce({
      snapshots: [snapshot],
      resolverStatus: undefined,
      capabilities: [],
      devices: [],
    });
    api.runSnapshot.mockResolvedValueOnce({ ...snapshot, id: "snapshot-2" });
    const reportText = "# Report";
    api.exportSnapshotReport.mockResolvedValueOnce(reportText);

    renderWithProviders(<SnapshotsPage />);

    expect(await screen.findByTestId(selectors.network.latestSnapshot)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "pages.snapshots.run" }));
    await waitFor(() => expect(api.runSnapshot).toHaveBeenCalledWith("home"));
    await userEvent.click(screen.getByRole("button", { name: "pages.snapshots.export" }));
    expect(await screen.findByText(reportText)).toBeInTheDocument();
  });

  it("renders devices and updates groups", async () => {
    // [REQ:NM-P0-009] Device workflows expose inventory confidence and group updates.
    api.fetchDevices.mockResolvedValueOnce([
      {
        id: "device-1",
        hostname: "laptop",
        ipAddress: "redacted",
        macAddress: "redacted",
        group: "default",
        identityConfidence: "high",
        notes: ["resolver evidence"],
      },
    ]);
    api.refreshInventory.mockResolvedValueOnce({ devices: [], findings: ["Discovery unavailable"] });
    api.updateDeviceGroup.mockResolvedValueOnce({ id: "device-1", group: "trusted" });

    renderWithProviders(<DevicesPage />);

    expect(await screen.findByTestId(selectors.network.deviceTable)).toBeInTheDocument();
    const table = screen.getByTestId(selectors.network.deviceTable);
    const groupInput = table.querySelector("input");
    expect(groupInput).toBeInstanceOf(HTMLInputElement);
    if (!(groupInput instanceof HTMLInputElement)) {
      throw new Error("device group input missing");
    }
    await userEvent.clear(groupInput);
    await userEvent.type(groupInput, "trusted");
    groupInput.blur();
    await waitFor(() => expect(api.updateDeviceGroup).toHaveBeenCalledWith("device-1", "trusted"));
    await userEvent.click(screen.getByRole("button", { name: "pages.devices.refresh" }));
    await waitFor(() => expect(api.refreshInventory).toHaveBeenCalled());
  });

  it("renders an empty devices table state", async () => {
    api.fetchDevices.mockResolvedValueOnce([]);

    renderWithProviders(<DevicesPage />);

    expect(await screen.findByTestId(selectors.network.empty)).toBeInTheDocument();
  });

  it("renders optimization comparison timeline after a safe run starts", async () => {
    // [REQ:NM-P0-009] Optimization workflows show baseline/candidate/after evidence before approval.
    api.createOptimizationRun.mockResolvedValueOnce({
      id: "run-1",
      status: "scored",
      scoringProfile: "reliability-first",
      recommendation: "Keep current settings",
      candidates: [
        {
          id: "candidate-1",
          description: "Manual DNS upstream review",
          status: "manual_required",
          score: 0.72,
          evidence: ["baseline retained"],
          approvalRequired: true,
        },
      ],
    });
    api.scoreOptimizationRun.mockResolvedValueOnce({
      id: "run-1",
      status: "scored",
      scoringProfile: "reliability-first",
      recommendation: "Keep current settings",
      candidates: [
        {
          id: "candidate-1",
          description: "Manual DNS upstream review",
          status: "manual_required",
          score: 0.72,
          evidence: ["baseline retained"],
          approvalRequired: true,
        },
      ],
    });
    api.approveOptimizationCandidate.mockResolvedValueOnce({
      id: "run-1",
      status: "manual_required",
      scoringProfile: "reliability-first",
      recommendation: "Manual apply required",
      candidates: [],
    });

    renderWithProviders(<OptimizationPage />);

    await userEvent.click(screen.getByRole("button", { name: "pages.optimization.start" }));
    expect(await screen.findByTestId(selectors.network.optimizationTimeline)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText(strings.network.timeline.baseline)).toBeInTheDocument());
    expect(screen.getByText(strings.network.timeline.candidate)).toBeInTheDocument();
    expect(screen.getByText(strings.network.timeline.after)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "pages.optimization.score" }));
    await waitFor(() => expect(api.scoreOptimizationRun).toHaveBeenCalledWith("run-1"));
    await userEvent.click(screen.getByRole("button", { name: "pages.optimization.approve" }));
    await waitFor(() => expect(api.approveOptimizationCandidate).toHaveBeenCalledWith("run-1", "candidate-1"));
  });

  it("renders privacy settings defaults", async () => {
    // [REQ:NM-P0-009] Settings workflows expose persisted privacy defaults.
    api.fetchPrivacySettings.mockResolvedValueOnce({
      retention: { queryLogDays: 0, snapshotDays: 14, experimentDays: 14, profile: "minimal" },
      visibility: { showQueryDomains: false, showDeviceHistory: false, householdMode: true, notes: [] },
    });

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByTestId(selectors.network.privacySummary)).toBeInTheDocument();
  });

  it("renders enabled privacy visibility branches", async () => {
    api.fetchPrivacySettings.mockResolvedValueOnce({
      retention: { queryLogDays: 7, snapshotDays: 30, experimentDays: 30, profile: "audit" },
      visibility: { showQueryDomains: true, showDeviceHistory: true, householdMode: true, notes: [] },
    });

    renderWithProviders(<SettingsPage />);

    const summary = screen.getByTestId(selectors.network.privacySummary);
    await waitFor(() => expect(summary).toHaveTextContent("network.enabled"));
    expect(summary).toHaveTextContent("7");
  });
});
