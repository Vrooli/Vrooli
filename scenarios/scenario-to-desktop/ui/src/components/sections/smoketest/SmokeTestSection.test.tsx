/**
 * Tests for SmokeTestSection component.
 * Tests stage status display, smoke test action, progress, results, and error states.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { act } from "@testing-library/react";
import { SmokeTestSection } from "./SmokeTestSection";
import { usePipelineStore } from "../../../store";
import { createPipelineStatus } from "../../../test-utils/mocks";

// Reset store state before each test
beforeEach(() => {
  act(() => {
    usePipelineStore.getState().reset();
    usePipelineStore.setState({ scenarioName: "test-scenario" });
  });
});

describe("SmokeTestSection", () => {
  it("renders pending state when no result", () => {
    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Smoke Test")).toBeInTheDocument();
    expect(screen.getByText("Validate built artifacts")).toBeInTheDocument();
    expect(screen.getByText("Smoke Test Status")).toBeInTheDocument();
    expect(screen.getByText("Waiting for build artifacts")).toBeInTheDocument();
  });

  it("shows placeholder when no scenario selected", () => {
    render(<SmokeTestSection scenarioName="" />);

    expect(screen.getByText("Select a scenario to enable smoke testing.")).toBeInTheDocument();
  });

  it("shows instruction when build artifacts not yet available", () => {
    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Smoke testing will be available after building installers/)).toBeInTheDocument();
  });

  // =========================================================================
  // Run Smoke Test button
  // =========================================================================

  it("shows 'Run Smoke Test' button when build artifacts available", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Ready to test")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run smoke test/i })).toBeInTheDocument();
    // Should NOT show contradictory "waiting" messages when artifacts exist
    expect(screen.queryByText(/after build/i)).not.toBeInTheDocument();
  });

  it("shows 'Run Smoke Test' button when pipeline completed at checkpoint", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stopped_after_stage: "build",
          stages: {
            ...createPipelineStatus().stages,
            build: { stage: "build", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    // Should show "Run Smoke Test", NOT "Skipped" or "Pipeline ended..."
    expect(screen.getByRole("button", { name: /run smoke test/i })).toBeInTheDocument();
    expect(screen.getByText("Ready to test")).toBeInTheDocument();
    expect(screen.queryByText(/skipped/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/pipeline ended/i)).not.toBeInTheDocument();
  });

  it("calls runStage when Run Smoke Test button is clicked", () => {
    const runStageSpy = vi.fn().mockResolvedValue("pipeline-123");

    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        runStage: runStageSpy,
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    fireEvent.click(screen.getByRole("button", { name: /run smoke test/i }));

    expect(runStageSpy).toHaveBeenCalledWith("smoketest");
  });

  it("shows Starting state when submitting", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        isSubmitting: true,
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    const startingButton = screen.getByText("Starting...").closest("button");
    expect(startingButton).toBeInTheDocument();
    expect(startingButton).toBeDisabled();
  });

  it("disables button when another stage is busy", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        runStatus: "running",
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    // When running, shows progress bar instead of button
    expect(screen.queryByRole("button", { name: /run smoke test/i })).not.toBeInTheDocument();
  });

  // =========================================================================
  // Running state
  // =========================================================================

  it("renders running state with progress bar", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        runStatus: "running",
        pipelineStatus: createPipelineStatus({
          status: "running",
          current_stage: "smoketest",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "running", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Testing")).toBeInTheDocument();
    expect(screen.getByText(/Running Smoketest stage/)).toBeInTheDocument();
    expect(screen.getByText(/Verifying built artifacts launch correctly/)).toBeInTheDocument();
  });

  it("shows cancel button when running", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        runStatus: "running",
        pipelineStatus: createPipelineStatus({
          status: "running",
          current_stage: "smoketest",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "running", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByRole("button", { name: /cancel smoke test/i })).toBeInTheDocument();
  });

  it("shows live logs during running state", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        runStatus: "running",
        pipelineStatus: createPipelineStatus({
          status: "running",
          current_stage: "smoketest",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "running", started_at: Date.now() },
          },
        }),
        stageLogs: {
          smoketest: ["Launching installer...", "Waiting for window..."],
        },
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Live Logs/)).toBeInTheDocument();
    expect(screen.getByText(/Launching installer/)).toBeInTheDocument();
    expect(screen.getByText(/Waiting for window/)).toBeInTheDocument();
  });

  // =========================================================================
  // Completed results
  // =========================================================================

  it("renders completed state with passed status", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          status: "passed",
          platform: "linux",
          artifact_path: "/path/to/artifact.AppImage",
          logs: ["Test started", "Test passed"],
          telemetry_uploaded: true,
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Passed")).toBeInTheDocument();
    expect(screen.getByText("Tested on linux")).toBeInTheDocument();
    expect(screen.getByText("Platform")).toBeInTheDocument();
    expect(screen.getByText("linux")).toBeInTheDocument();
  });

  it("shows telemetry upload status", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          status: "completed",
          platform: "linux",
          artifact_path: "/path/to/artifact",
          logs: [],
          telemetry_uploaded: true,
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Telemetry")).toBeInTheDocument();
    expect(screen.getByText("Uploaded")).toBeInTheDocument();
  });

  it("shows 'Not uploaded' when telemetry not sent", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          status: "completed",
          platform: "linux",
          artifact_path: "/path/to/artifact",
          logs: [],
          telemetry_uploaded: false,
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Not uploaded")).toBeInTheDocument();
  });

  it("shows artifact path when available", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          status: "completed",
          platform: "linux",
          artifact_path: "/path/to/tested/artifact.AppImage",
          logs: [],
          telemetry_uploaded: true,
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Tested Artifact")).toBeInTheDocument();
    expect(screen.getByText("/path/to/tested/artifact.AppImage")).toBeInTheDocument();
  });

  it("shows test logs when available", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          status: "completed",
          platform: "linux",
          artifact_path: "/path/to/artifact",
          logs: ["Starting test...", "Test completed successfully"],
          telemetry_uploaded: true,
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Test Logs/)).toBeInTheDocument();
    expect(screen.getByText(/Starting test.../)).toBeInTheDocument();
    expect(screen.getByText(/Test completed successfully/)).toBeInTheDocument();
  });

  // =========================================================================
  // Error states
  // =========================================================================

  it("renders failed state with error message from result", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          status: "failed",
          platform: "linux",
          artifact_path: "/path/to/artifact",
          logs: ["Starting test...", "Test failed!"],
          error: "Application crashed during startup",
        },
        pipelineStatus: createPipelineStatus({
          status: "failed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Application crashed during startup/)).toBeInTheDocument();
  });

  it("renders stage-level failure with error recovery", () => {
    const resetForRetrySpy = vi.spyOn(usePipelineStore.getState(), "resetForRetry");

    act(() => {
      usePipelineStore.setState({
        errorInfo: {
          message: "Smoke test failed: timeout",
          category: "timeout",
          suggestions: ["Increase timeout duration"],
        },
        pipelineStatus: createPipelineStatus({
          status: "failed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "failed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Smoke test failed: timeout")).toBeInTheDocument();
    expect(screen.getByText("Increase timeout duration")).toBeInTheDocument();

    const retryButton = screen.getByRole("button", { name: /^retry$/i });
    fireEvent.click(retryButton);

    expect(resetForRetrySpy).toHaveBeenCalled();
  });

  // =========================================================================
  // Pipeline failure/cancellation before smoketest
  // =========================================================================

  it("shows 'Skipped' status and New Pipeline button when pipeline failed before smoketest", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["linux"],
          artifacts: { linux: "/path/app.AppImage" },
        },
        pipelineStatus: createPipelineStatus({
          status: "failed",
          current_stage: "smoketest",
          stages: {
            ...createPipelineStatus().stages,
            build: { stage: "build", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Skipped")).toBeInTheDocument();
    expect(screen.getByText(/Pipeline ended before smoke testing/)).toBeInTheDocument();
    expect(screen.getByText("Build artifacts available")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /new pipeline/i })).toBeInTheDocument();
    // Should NOT show misleading placeholder messages
    expect(screen.queryByText(/will run automatically/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/available after building/i)).not.toBeInTheDocument();
  });

  it("shows 'Skipped' with cancelled message when pipeline was cancelled", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["linux"],
          artifacts: { linux: "/path/app.AppImage" },
        },
        pipelineStatus: createPipelineStatus({
          status: "cancelled",
          stages: {
            ...createPipelineStatus().stages,
            build: { stage: "build", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText(/was cancelled/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /new pipeline/i })).toBeInTheDocument();
  });

  // =========================================================================
  // Screen recording
  // =========================================================================

  it("renders video player when screen recording is available", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          smoke_test_id: "smoke-test-123",
          status: "passed",
          platform: "linux",
          artifact_path: "/path/to/artifact.AppImage",
          logs: [],
          telemetry_uploaded: true,
          screen_recording: {
            recorded: true,
            video_path: "/data/recordings/smoke-test-123.mp4",
            duration_ms: 15000,
            file_size_bytes: 2621440,
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Screen Recording")).toBeInTheDocument();
    const video = document.querySelector("video");
    expect(video).toBeInTheDocument();
    expect(video?.getAttribute("src")).toContain("/smoketest/smoke-test-123/video");
    expect(screen.getByText(/Duration: 15\.0s/)).toBeInTheDocument();
    expect(screen.getByText(/Size: 2\.5 MB/)).toBeInTheDocument();
  });

  it("shows recording error when recording failed", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          smoke_test_id: "smoke-test-456",
          status: "passed",
          platform: "linux",
          artifact_path: "/path/to/artifact.AppImage",
          logs: [],
          telemetry_uploaded: true,
          screen_recording: {
            recorded: false,
            error: "ffmpeg not found",
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Screen Recording Failed")).toBeInTheDocument();
    expect(screen.getByText("ffmpeg not found")).toBeInTheDocument();
    expect(screen.getByText("FFmpeg is not installed")).toBeInTheDocument();
    expect(screen.getByText(/the smoke test result is not affected/i)).toBeInTheDocument();
    expect(document.querySelector("video")).not.toBeInTheDocument();
  });

  it("does not render video elements when screen_recording is absent", () => {
    act(() => {
      usePipelineStore.setState({
        smokeTestResult: {
          status: "passed",
          platform: "linux",
          artifact_path: "/path/to/artifact.AppImage",
          logs: [],
          telemetry_uploaded: true,
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            smoketest: { stage: "smoketest", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(document.querySelector("video")).not.toBeInTheDocument();
    expect(screen.queryByText("Screen Recording")).not.toBeInTheDocument();
  });
});
