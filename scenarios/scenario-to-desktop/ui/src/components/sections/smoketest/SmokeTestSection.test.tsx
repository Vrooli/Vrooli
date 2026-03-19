/**
 * Tests for SmokeTestSection component.
 * Tests stage status display, test results, and telemetry state.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
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

  it("renders running state with status badge and progress indicator", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
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
    expect(screen.getByText("Smoke test in progress")).toBeInTheDocument();
    expect(screen.getByText(/Verifying built artifacts launch correctly/)).toBeInTheDocument();
    // Cancel button should be visible
    expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
  });

  it("shows correct description when running", () => {
    act(() => {
      usePipelineStore.setState({
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

    expect(screen.getByText(/Verifying built artifacts launch correctly/)).toBeInTheDocument();
    // Should NOT show misleading "Waiting for build artifacts" while running
    expect(screen.queryByText("Waiting for build artifacts")).not.toBeInTheDocument();
  });

  it("shows live logs during running state", () => {
    act(() => {
      usePipelineStore.setState({
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

  it("renders failed state with error message", () => {
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

  it("shows placeholder when no scenario selected", () => {
    render(<SmokeTestSection scenarioName="" />);

    expect(screen.getByText("Select a scenario to enable smoke testing.")).toBeInTheDocument();
  });

  it("shows instruction when build artifacts not yet available", () => {
    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Smoke testing will be available after building installers/)).toBeInTheDocument();
  });

  it("shows 'Ready to test' when build artifacts available and pipeline not terminal", () => {
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
    // Should NOT show contradictory "waiting" messages when artifacts exist
    expect(screen.queryByText(/after build/i)).not.toBeInTheDocument();
  });

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
            // smoketest stage absent - pipeline failed before it ran
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

    expect(screen.getByText(/Recording failed: ffmpeg not found/)).toBeInTheDocument();
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
