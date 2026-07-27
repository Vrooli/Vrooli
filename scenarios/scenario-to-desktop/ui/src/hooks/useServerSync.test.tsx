import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { SmokeTestStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/operation_results_pb";

const state = vi.hoisted(() => ({
  pipelineId: "pipeline-1",
  runStatus: "completed",
  generateResult: { desktopPath: "/artifacts/calculator.AppImage" },
  smokeTestResult: null as Record<string, unknown> | null,
  clearError: vi.fn(),
  updateFormState: vi.fn(),
  saveStageResult: vi.fn(),
}));

vi.mock("../store", () => ({
  usePipelineStore: (selector: (value: typeof state) => unknown) => selector(state),
}));
vi.mock("./useScenarioState", () => ({
  useScenarioState: () => ({
    hasInitiallyLoaded: true,
    updateFormState: state.updateFormState,
    saveStageResult: state.saveStageResult,
  }),
}));

import { useServerSync } from "./useServerSync";

describe("useServerSync", () => {
  beforeEach(() => {
    state.pipelineId = "pipeline-1";
    state.runStatus = "completed";
    state.generateResult = { desktopPath: "/artifacts/calculator.AppImage" };
    state.smokeTestResult = null;
    vi.clearAllMocks();
  });

  it("persists a completed desktop build once the server state is ready", async () => {
    const { result } = renderHook(() => useServerSync({ scenarioName: "calculator", viewMode: "generator" }));

    expect(result.current.serverStateLoaded).toBe(true);
    await waitFor(() => {
      expect(state.updateFormState).toHaveBeenCalledWith({
        wrapper_build_id: "pipeline-1",
        wrapper_build_status: "ready",
        wrapper_output_path: "/artifacts/calculator.AppImage",
      });
    });
    expect(state.clearError).toHaveBeenCalledTimes(1);
  });

  it("stores an in-progress smoke test update with normalized platform and timestamps", async () => {
    state.runStatus = "running";
    state.smokeTestResult = {
      smokeTestId: "smoke-1",
      platform: Platform.LINUX,
      status: SmokeTestStatus.RUNNING,
      startedAt: { seconds: 3n, nanos: 0 },
      completedAt: undefined,
      logs: "launching",
      telemetryUploaded: false,
    };

    renderHook(() => useServerSync({ scenarioName: "calculator", viewMode: "inventory" }));

    await waitFor(() => {
      expect(state.updateFormState).toHaveBeenCalledWith(expect.objectContaining({
        smoke_test_id: "smoke-1",
        smoke_test_platform: "linux",
        smoke_test_status: "running",
        smoke_test_started_at: "1970-01-01T00:00:03.000Z",
        smoke_test_completed_at: null,
        smoke_test_logs: "launching",
      }));
    });
  });

  it("records terminal smoke test evidence through the durable stage-result path", async () => {
    state.smokeTestResult = {
      smokeTestId: "smoke-2",
      platform: Platform.WIN,
      status: SmokeTestStatus.FAILED,
      logs: "failed",
      error: "application exited",
      telemetryUploaded: true,
    };

    renderHook(() => useServerSync({ scenarioName: "calculator", viewMode: "generator" }));

    await waitFor(() => {
      expect(state.saveStageResult).toHaveBeenCalledWith(
        "smoke_test",
        state.smokeTestResult,
        expect.objectContaining({ smoke_test_platform: "win", smoke_test_status: "failed", smoke_test_error: "application exited" }),
      );
    });
  });
});
