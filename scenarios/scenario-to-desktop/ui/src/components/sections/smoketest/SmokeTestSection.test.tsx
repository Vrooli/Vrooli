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

  it("shows 'Ready to test' when build artifacts available", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path/to/output",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
      });
    });

    render(<SmokeTestSection scenarioName="test-scenario" />);

    expect(screen.getByText("Ready to test")).toBeInTheDocument();
  });

  it("renders running state with status badge", () => {
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

  it("shows auto-run message when build artifacts are available", () => {
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

    expect(screen.getByText(/Smoke tests will run automatically/)).toBeInTheDocument();
  });
});
