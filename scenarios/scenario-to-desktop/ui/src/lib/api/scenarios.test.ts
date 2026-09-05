import { describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  loadScenarioState: vi.fn(),
  saveScenarioState: vi.fn(),
  deleteScenarioState: vi.fn(),
  checkScenarioState: vi.fn(),
  getScenarioStateLog: vi.fn(),
  invalidateScenarioState: vi.fn(),
  listDesktopScenarioStatus: vi.fn(),
  listTemplates: vi.fn(),
}));

vi.mock("./connect", () => ({
  stateConnectClient: clients,
  operationsConnectClient: clients,
  systemConnectClient: clients,
}));

import {
  checkStateStaleness,
  deleteScenarioState,
  fetchScenarioDesktopStatus,
  fetchScenarioState,
  fetchTemplates,
  getScenarioLogs,
  invalidateScenarioStage,
  saveScenarioState,
} from "./scenarios";

const state = {
  scenario_name: "calculator",
  schema_version: 1,
  created_at: "2026-07-27T00:00:00Z",
  updated_at: "2026-07-27T00:00:00Z",
  form_state: { app_display_name: "Calculator" },
};

describe("scenario state Connect client", () => {
  it("loads, saves, deletes, and checks the typed persisted build state", async () => {
    clients.loadScenarioState.mockResolvedValue({
      payload: { state, found: true },
    });
    clients.saveScenarioState.mockResolvedValue({
      payload: { success: true, updated_at: "2026-07-27T01:00:00Z" },
    });
    clients.checkScenarioState.mockResolvedValue({
      payload: { valid: false, changed: true, affected_stages: ["build"] },
    });

    await expect(
      fetchScenarioState("calculator", {
        includeLogs: true,
        validateManifest: true,
        manifestPath: "/tmp/manifest.json",
      }),
    ).resolves.toMatchObject({ found: true, state });
    await expect(
      saveScenarioState(
        "calculator",
        { app_display_name: "Calculator", platforms: { linux: true } },
        {
          manifestPath: "/tmp/manifest.json",
          computeHash: true,
          expectedHash: "previous-hash",
        },
      ),
    ).resolves.toMatchObject({ success: true });
    await deleteScenarioState("calculator");
    await expect(
      checkStateStaleness("calculator", { manifest_hash: "new-hash" }),
    ).resolves.toMatchObject({ valid: false, affected_stages: ["build"] });

    expect(clients.loadScenarioState).toHaveBeenCalledWith({
      scenarioName: "calculator",
      includeLogs: true,
      validateManifest: true,
      manifestPath: "/tmp/manifest.json",
    });
    expect(clients.saveScenarioState).toHaveBeenCalledWith({
      scenarioName: "calculator",
      payload: {
        form_state: {
          app_display_name: "Calculator",
          platforms: { linux: true },
        },
        manifest_path: "/tmp/manifest.json",
        compute_hash: true,
        expected_hash: "previous-hash",
      },
    });
    expect(clients.deleteScenarioState).toHaveBeenCalledWith({
      scenarioName: "calculator",
    });
    expect(clients.checkScenarioState).toHaveBeenCalledWith({
      scenarioName: "calculator",
      currentConfig: { manifest_hash: "new-hash" },
    });
  });

  it("rejects malformed or missing state payloads instead of accepting untyped data", async () => {
    clients.saveScenarioState.mockResolvedValue({});
    clients.checkScenarioState.mockResolvedValue({ payload: { valid: "yes" } });

    await expect(saveScenarioState("calculator", {})).rejects.toThrow(
      "StateService returned an empty payload",
    );
    await expect(checkStateStaleness("calculator", {})).rejects.toThrow();
  });

  it("returns logs only when the generated response declares them present", async () => {
    clients.getScenarioStateLog
      .mockResolvedValueOnce({ found: false })
      .mockResolvedValueOnce({
        found: true,
        payload: {
          service_id: "api",
          content: "started",
          lines: 1,
          captured_at: "2026-07-27T00:00:00Z",
        },
      });

    await expect(getScenarioLogs("calculator", "api")).resolves.toBeNull();
    await expect(getScenarioLogs("calculator", "api")).resolves.toMatchObject({
      content: "started",
    });
    expect(clients.getScenarioStateLog).toHaveBeenCalledWith({
      scenarioName: "calculator",
      serviceId: "api",
    });
  });

  it("maps generated desktop inventory and preserves typed template responses", async () => {
    clients.invalidateScenarioState.mockResolvedValue({
      payload: {
        scenario_name: "calculator",
        overall_status: "stale",
        stages: { build: { status: "stale" } },
      },
    });
    clients.listDesktopScenarioStatus.mockResolvedValue({
      scenarios: [
        {
          name: "calculator",
          displayName: "Calculator",
          hasDesktop: true,
          packageSize: 42n,
          platforms: ["linux"],
          buildArtifacts: [{ fileName: "Calculator.AppImage", sizeBytes: 42n }],
          connectionConfig: {
            mode: "proxy",
            endpoint: "http://localhost:8080",
          },
        },
      ],
      stats: { total: 1, withDesktop: 1, built: 1, webOnly: 0 },
    });
    const templates = { templates: [{ id: "react", displayName: "React" }] };
    clients.listTemplates.mockResolvedValue(templates);

    await expect(
      invalidateScenarioStage("calculator", "build", "manifest changed"),
    ).resolves.toMatchObject({ overall_status: "stale" });
    await expect(fetchScenarioDesktopStatus()).resolves.toMatchObject({
      scenarios: [
        {
          name: "calculator",
          package_size: 42,
          connection_config: { deployment_mode: "proxy" },
          build_artifacts: [
            { file_name: "Calculator.AppImage", size_bytes: 42 },
          ],
        },
      ],
      stats: { built: 1 },
    });
    await expect(fetchTemplates()).resolves.toEqual(templates);
    expect(clients.invalidateScenarioState).toHaveBeenCalledWith({
      scenarioName: "calculator",
      fromStage: "build",
      reason: "manifest changed",
    });
    expect(clients.listDesktopScenarioStatus).toHaveBeenCalledWith({});
    expect(clients.listTemplates).toHaveBeenCalledWith({});
  });
});
