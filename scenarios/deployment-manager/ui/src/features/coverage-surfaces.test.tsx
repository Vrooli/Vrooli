import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Route, Routes } from "react-router-dom";
import { Analyze } from "./dependencies/Analyze";
import { GuidedFlow } from "./deployments/GuidedFlow";
import { MigrationTaskCard } from "./deployments/MigrationTaskCard";
import { NewProfile } from "./profiles/NewProfile";
import { Profiles } from "./profiles/Profiles";
import { ProfileDetail } from "./profiles/ProfileDetail";
import { Dashboard } from "./shared/Dashboard";
import { Layout } from "./shared/Layout";
import { BundleTelemetry } from "./telemetry/BundleTelemetry";
import { TelemetryEntry } from "./telemetry/TelemetryEntry";
import * as api from "../lib/api";
import { renderWithProviders } from "../test-utils/renderWithProviders";

vi.mock("../lib/api");

const profile: api.DeploymentProfile = {
  id: "profile-1",
  name: "Production",
  scenario: "picker-wheel",
  tiers: [2],
  version: 1,
};

const analysis: api.DependencyAnalysisResponse = {
  scenario: "picker-wheel",
  dependencies: { postgres: { version: "16" } },
  circular_dependencies: ["picker-wheel"],
  aggregate_requirements: { memory: "512MB", cpu: "2", gpu: "none", storage: "1GB", network: "low" },
  tiers: {
    "2": { overall: 92, portability: 90, resources: 88, licensing: 95, platform_support: 94 },
  },
};

describe("uncovered operator surfaces", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }));
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("analyzes a scenario and shows scored requirements", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    vi.mocked(api.analyzeDependencies).mockResolvedValue(analysis);
    renderWithProviders(<Analyze />);
    fireEvent.change(screen.getByPlaceholderText("Search or type a scenario name"), { target: { value: "picker-wheel" } });
    fireEvent.click(screen.getByRole("button", { name: "Analyze" }));
    expect(await screen.findByText("Deployment Fitness Scores")).toBeInTheDocument();
    expect(screen.getByText("92")).toBeInTheDocument();
    expect(screen.getByText("Circular Dependencies Detected")).toBeInTheDocument();
    expect(screen.getByText("512MB")).toBeInTheDocument();
  });

  it("walks the guided flow from manual scenario to tier selection", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    vi.mocked(api.analyzeDependencies).mockResolvedValue(analysis);
    const onClose = vi.fn();
    renderWithProviders(<GuidedFlow open onClose={onClose} />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("e.g., picker-wheel"), { target: { value: "picker-wheel" } });
    const [guidedScenario] = screen.getAllByText("picker-wheel");
    if (guidedScenario) fireEvent.click(guidedScenario);
    fireEvent.click(screen.getByText("Tier 3 · Mobile"));
    fireEvent.click(screen.getByRole("button", { name: "Run analysis" }));
    await waitFor(() => expect(screen.getByText(/Step 2/i)).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /close|cancel/i }));
    expect(onClose).toHaveBeenCalled();
  });

  it("completes the guided profile journey through issue resolution", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    vi.mocked(api.analyzeDependencies).mockResolvedValue(analysis);
    const onClose = vi.fn();
    renderWithProviders(<GuidedFlow open onClose={onClose} />);

    fireEvent.click(screen.getByRole("button", { name: "Manual scenario/tier" }));
    fireEvent.click(screen.getByRole("button", { name: "Select profile" }));
    fireEvent.click(await screen.findByText("Production"));
    fireEvent.click(screen.getByRole("button", { name: "Run analysis" }));
    expect(await screen.findByText("Readiness summary")).toBeInTheDocument();
    expect(await screen.findByText("Overall fitness")).toBeInTheDocument();
    expect(api.analyzeDependencies).toHaveBeenCalledWith("picker-wheel");

    fireEvent.click(screen.getByRole("button", { name: "Continue guided flow" }));
    expect(await screen.findByText("Export & hand-off")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Continue to issue resolution" }));
    expect(await screen.findByText("Resolve issues & ingest telemetry")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Finish flow" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("reports analysis failures and supports starting over", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);
    vi.mocked(api.analyzeDependencies).mockRejectedValue(new Error("analyzer unavailable"));
    renderWithProviders(<GuidedFlow open onClose={vi.fn()} />);

    fireEvent.change(screen.getByPlaceholderText("e.g., picker-wheel"), { target: { value: "demo" } });
    fireEvent.click(screen.getByRole("button", { name: "Run analysis" }));
    expect(await screen.findByText("analyzer unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start over" }));
    expect(await screen.findByText("Pick scenario & target tier")).toBeInTheDocument();
  });

  it("files a migration task only after required dependencies are supplied", async () => {
    vi.mocked(api.reportMigrationTask).mockResolvedValue({ item_id: "item-1", kind: "fix", name: "redis", deep_link: "https://swarm/item-1", status: "queued", queue_position: 2, priority: 3, deduped: false });
    renderWithProviders(<MigrationTaskCard defaultScenario="picker-wheel" />);
    const submit = screen.getByRole("button", { name: /file migration task/i });
    expect(submit).toBeDisabled();
    fireEvent.change(screen.getByLabelText("From dependency"), { target: { value: "redis" } });
    fireEvent.change(screen.getByLabelText("To dependency"), { target: { value: "valkey" } });
    expect(submit).toBeEnabled();
    fireEvent.click(submit);
    expect(await screen.findByText(/queued/i)).toBeInTheDocument();
    expect(api.reportMigrationTask).toHaveBeenCalledWith(expect.objectContaining({ scenario: "picker-wheel", from_dependency: "redis", to_dependency: "valkey" }));
  });

  it("renders a deduplicated completed migration task", async () => {
    vi.mocked(api.reportMigrationTask).mockResolvedValue({ item_id: "item-2", kind: "fix", name: "redis", deep_link: "https://swarm/item-2", status: "completed", queue_position: undefined, priority: 1, deduped: true });
    renderWithProviders(<MigrationTaskCard defaultScenario="picker-wheel" />);
    fireEvent.change(screen.getByLabelText("From dependency"), { target: { value: "redis" } });
    fireEvent.change(screen.getByLabelText("To dependency"), { target: { value: "valkey" } });
    fireEvent.click(screen.getByRole("button", { name: /file migration task/i }));
    expect(await screen.findByText("Linked to existing backlog item")).toBeInTheDocument();
  });

  it("creates a profile through the three-step form", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);
    vi.mocked(api.createProfile).mockResolvedValue({ id: "new-profile", version: 1 });
    renderWithProviders(<NewProfile />);
    fireEvent.change(screen.getByLabelText("Profile Name"), { target: { value: "Desktop" } });
    fireEvent.change(screen.getByLabelText("Scenario"), { target: { value: "picker-wheel" } });
    const [newProfileScenario] = screen.getAllByText("picker-wheel");
    if (newProfileScenario) fireEvent.click(newProfileScenario);
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByText(/Step 2 of 3/)).toBeInTheDocument();
    fireEvent.click(screen.getByText(/Tier 2: Desktop/));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByText("Review & Create")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Create Profile" }));
    await waitFor(() => expect(api.createProfile).toHaveBeenCalledWith({ name: "Desktop", scenario: "picker-wheel", tiers: [2] }, expect.anything()));
  });

  it("renders profile configuration details and surfaces create failures", async () => {
    vi.mocked(api.getProfile).mockResolvedValue({
      ...profile,
      created_at: "2026-01-01T00:00:00Z",
      swaps: { redis: "valkey" },
      secrets: { API_KEY: "configured" },
      settings: { region: "us-east" },
    });
    vi.mocked(api.getRequiredPlatforms).mockResolvedValue({ profile_id: profile.id, platforms: [] });
    vi.mocked(api.getProfileLPBSConfig).mockResolvedValue({ profile_id: profile.id, lpbs_domain: "", lpbs_remote_profile: "", lpbs_app_key: "", default_channel: "stable", update_url: "" });
    vi.mocked(api.listProfileReleases).mockResolvedValue({ releases: [] });
    vi.mocked(api.deployProfile).mockResolvedValue({ deployment_id: "deployment-1", profile_id: profile.id, status: "queued", logs_url: "/logs/deployment-1" });
    vi.mocked(api.checkReleaseGate).mockRejectedValue(new Error("gate unavailable"));
    renderWithProviders(
      <Routes>
        <Route path="/profiles/:id" element={<ProfileDetail />} />
        <Route path="/deployments/:id" element={<div>deployment destination</div>} />
      </Routes>,
      { route: `/profiles/${profile.id}` },
    );
    expect(await screen.findByText("Dependency Swaps")).toBeInTheDocument();
    expect(screen.getByText("Secret Configuration")).toBeInTheDocument();
    expect(screen.getByText("Additional Settings")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Enter git commit hash..."), { target: { value: "abc123" } });
    expect(await screen.findByText(/Failed to check gate/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Deploy \(stub\)/ }));
    await waitFor(() => expect(api.deployProfile).toHaveBeenCalledWith(profile.id, expect.anything()));

    cleanup();
    vi.mocked(api.createProfile).mockRejectedValue(new Error("profile rejected"));
    renderWithProviders(<NewProfile />);
    fireEvent.change(screen.getByLabelText("Profile Name"), { target: { value: "Broken" } });
    fireEvent.change(screen.getByLabelText("Scenario"), { target: { value: "demo" } });
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByText(/Tier 2: Desktop/));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Create Profile" }));
    expect(await screen.findByText(/Failed to create profile: profile rejected/)).toBeInTheDocument();
  });

  it("shows telemetry aggregates and a failure entry", async () => {
    vi.mocked(api.listTelemetry).mockResolvedValue([{
      scenario: "picker-wheel",
      path: "/tmp/telemetry.jsonl",
      total_events: 3,
      last_event: "ready",
      last_timestamp: "2026-08-04T12:00:00Z",
      failure_counts: { dependency_unreachable: 2 },
      recent_failures: [{ event: "dependency_unreachable", timestamp: "2026-08-04T11:59:00Z", details: { dependency: "postgres" } }],
      recent_events: [{ event: "ready", timestamp: "2026-08-04T12:00:00Z" }],
    }]);
    renderWithProviders(<BundleTelemetry />);
    expect(await screen.findByText("Desktop Bundle Telemetry")).toBeInTheDocument();
    expect(await screen.findByText("dependency unreachable")).toBeInTheDocument();
    expect(screen.getByText("picker-wheel")).toBeInTheDocument();
  });

  it("covers telemetry filtering, help, and successful upload", async () => {
    vi.mocked(api.listTelemetry).mockResolvedValue([
      { scenario: "picker-wheel", path: "/tmp/picker", total_events: 2, failure_counts: { dependency_unreachable: 1 }, recent_events: [] },
      { scenario: "healthy", path: "/tmp/healthy", total_events: 1, failure_counts: {}, recent_events: [] },
    ]);
    vi.mocked(api.uploadTelemetry).mockResolvedValue({ path: "/tmp/uploaded" });
    renderWithProviders(<BundleTelemetry />);
    expect(await screen.findByText("picker-wheel")).toBeInTheDocument();
    const [telemetryRefresh] = screen.getAllByRole("button", { name: "Refresh" });
    if (telemetryRefresh) fireEvent.click(telemetryRefresh);
    fireEvent.click(screen.getByRole("button", { name: "How this works" }));
    expect(screen.getByText("Use after running a packaged app")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Filter by scenario..."), { target: { value: "healthy" } });
    expect(screen.getByText("healthy")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Failures only" }));
    fireEvent.change(screen.getByPlaceholderText("Filter by scenario..."), { target: { value: "picker" } });
    const file = new File(["{}"], "deployment-telemetry.jsonl", { type: "application/json" });
    fireEvent.change(screen.getByLabelText("Telemetry file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Upload" }));
    expect(await screen.findByText(/Uploaded successfully/)).toBeInTheDocument();
  });

  it("covers dashboard guided entry, profile error, and responsive navigation", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    renderWithProviders(<Dashboard />);
    fireEvent.click(screen.getByTestId("start-guided-flow"));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    fireEvent.click(screen.getByRole("button", { name: "Open guided flow" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));

    cleanup();
    vi.mocked(api.listProfiles).mockRejectedValue(new Error("profiles unavailable"));
    renderWithProviders(<Profiles />);
    expect(await screen.findByText(/Failed to load profiles: profiles unavailable/)).toBeInTheDocument();

    cleanup();
    vi.mocked(api.listProfiles).mockResolvedValue([]);
    renderWithProviders(<Profiles />);
    fireEvent.click(screen.getByRole("button", { name: "How to use" }));
    expect(screen.getByText("Profiles 101")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Hide help" }));

    cleanup();
    const layout = renderWithProviders(<Layout><div>content</div></Layout>);
    fireEvent.click(screen.getByRole("button", { name: "Toggle menu" }));
    fireEvent.click(screen.getByText("Profiles"));
    fireEvent.click(screen.getByRole("button", { name: "Toggle menu" }));
    const mobileOverlay = layout.container.querySelector(".fixed.inset-0.z-40");
    if (mobileOverlay) fireEvent.click(mobileOverlay);
    fireEvent.click(screen.getByRole("button", { name: "Toggle menu" }));
    expect(screen.getByText("content")).toBeInTheDocument();
  });

  it("surfaces migration task filing failures", async () => {
    vi.mocked(api.reportMigrationTask).mockRejectedValue(new Error("swarm unavailable"));
    renderWithProviders(<MigrationTaskCard defaultScenario="picker-wheel" />);
    fireEvent.change(screen.getByLabelText("From dependency"), { target: { value: "redis" } });
    fireEvent.change(screen.getByLabelText("To dependency"), { target: { value: "valkey" } });
    fireEvent.click(screen.getByRole("button", { name: /file migration task/i }));
    expect(await screen.findByText("swarm unavailable")).toBeInTheDocument();
  });

  it("covers analyzer query initialization and raw/fullscreen views", async () => {
    const focusedAnalysis = { ...analysis, circular_dependencies: [] };
    vi.mocked(api.listProfiles).mockResolvedValue([profile]);
    vi.mocked(api.analyzeDependencies).mockResolvedValue(focusedAnalysis);
    renderWithProviders(<Analyze />, { route: "/analyze?scenario=picker-wheel&tier=desktop" });
    expect(await screen.findByText("Deployment Fitness Scores")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Full screen" }));
    expect(screen.getByRole("button", { name: "Exit full screen" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Raw Data" }));
    expect(screen.getByRole("tab", { name: "Raw Data" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "How this works" }));
    expect(screen.getByText("How this works")).toBeInTheDocument();
  });

  it("renders clean telemetry without inventing failures", () => {
    const entry: api.TelemetrySummary = { scenario: "demo", path: "/tmp/demo", total_events: 1, failure_counts: {}, recent_events: [] };
    const refresh = vi.fn();
    renderWithProviders(<TelemetryEntry entry={entry} onRefresh={refresh} />);
    expect(screen.getByText("Clean")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /refresh/i }));
    expect(refresh).toHaveBeenCalledTimes(1);
  });
});
