/**
 * Tests for DistributionSection component.
 * Tests stage status display, upload targets, and error states.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import { act } from "@testing-library/react";
import { DistributionSection } from "./DistributionSection";
import { usePipelineStore } from "../../../store";
import { renderWithProviders } from "../../../test-utils";
import { createPipelineStatus } from "../../../test-utils/mocks";

// Reset store state before each test
beforeEach(() => {
  act(() => {
    usePipelineStore.getState().reset();
    usePipelineStore.setState({ scenarioName: "test-scenario" });
  });
});

// Helper to render with providers
const renderComponent = (props: { scenarioName: string }) => {
  return renderWithProviders(<DistributionSection {...props} />);
};

describe("DistributionSection", () => {
  it("renders pending state when no result", () => {
    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText("Distribution")).toBeInTheDocument();
    expect(screen.getByText("Upload artifacts to cloud storage")).toBeInTheDocument();
    expect(screen.getByText("Distribution Status")).toBeInTheDocument();
    expect(screen.getByText("Waiting for build artifacts")).toBeInTheDocument();
  });

  it("renders about section explaining distribution", () => {
    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText("About distribution")).toBeInTheDocument();
    expect(screen.getByText(/Upload built installers to cloud storage/)).toBeInTheDocument();
  });

  it("shows 'Ready to upload' when build artifacts available and no result", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: {
          output_path: "/path/to/output",
          platforms: ["win"],
          artifacts: { win: "/path/app.exe" },
        },
        distributionResult: null,
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    // The component shows "Ready to upload" when build artifacts are available
    expect(screen.getByText("Ready to upload")).toBeInTheDocument();
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
          current_stage: "distribution",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "running", started_at: Date.now() },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    // Running status label from stage status config
    expect(screen.getByText("Running")).toBeInTheDocument();
  });

  it("renders completed state with upload summary", () => {
    act(() => {
      usePipelineStore.setState({
        distributionResult: {
          version: "1.0.0",
          targets: {
            s3: {
              status: "completed",
              uploads: {
                win: { status: "completed", url: "https://s3.example.com/win" },
                mac: { status: "completed", url: "https://s3.example.com/mac" },
              },
            },
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText("Uploaded")).toBeInTheDocument();
    expect(screen.getByText(/2\/2 artifacts uploaded.*v1\.0\.0/)).toBeInTheDocument();
  });

  it("shows target details with upload status", () => {
    act(() => {
      usePipelineStore.setState({
        distributionResult: {
          version: "1.0.0",
          targets: {
            s3: {
              status: "completed",
              uploads: {
                win: { status: "completed", url: "https://s3.example.com/win" },
              },
            },
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText("s3")).toBeInTheDocument();
    expect(screen.getByText("win")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /view/i })).toBeInTheDocument();
  });

  it("shows multiple targets when configured", () => {
    act(() => {
      usePipelineStore.setState({
        distributionResult: {
          version: "1.0.0",
          targets: {
            s3: {
              status: "completed",
              uploads: {
                win: { status: "completed", url: "https://s3.example.com/win" },
              },
            },
            github: {
              status: "completed",
              uploads: {
                win: { status: "completed", url: "https://github.com/releases/win" },
              },
            },
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText("s3")).toBeInTheDocument();
    expect(screen.getByText("github")).toBeInTheDocument();
  });

  it("shows error for failed upload", () => {
    act(() => {
      usePipelineStore.setState({
        distributionResult: {
          version: "1.0.0",
          targets: {
            s3: {
              status: "failed",
              error: "S3 bucket not accessible",
              uploads: {
                win: { status: "failed" },
              },
            },
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText("S3 bucket not accessible")).toBeInTheDocument();
  });

  it("renders distribution error from result", () => {
    act(() => {
      usePipelineStore.setState({
        distributionResult: {
          version: "1.0.0",
          targets: {},
          error: "No distribution targets configured",
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText(/No distribution targets configured/)).toBeInTheDocument();
  });

  it("renders failed state from stage status", () => {
    act(() => {
      usePipelineStore.setState({
        pipelineStatus: createPipelineStatus({
          status: "failed",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "failed", started_at: Date.now(), error: "Distribution failed" },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText(/Distribution stage failed/)).toBeInTheDocument();
  });

  it("shows placeholder when no scenario selected", () => {
    renderComponent({ scenarioName: "" });

    expect(screen.getByText("Select a scenario to enable distribution.")).toBeInTheDocument();
  });

  it("shows instruction when build artifacts not yet available", () => {
    renderComponent({ scenarioName: "test-scenario" });

    expect(screen.getByText(/Distribution will be available after building installers/)).toBeInTheDocument();
  });

  it("counts only successful uploads", () => {
    act(() => {
      usePipelineStore.setState({
        distributionResult: {
          version: "1.0.0",
          targets: {
            s3: {
              status: "completed",
              uploads: {
                win: { status: "completed", url: "https://s3.example.com/win" },
                mac: { status: "failed" },
                linux: { status: "completed", url: "https://s3.example.com/linux" },
              },
            },
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            distribution: { stage: "distribution", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    renderComponent({ scenarioName: "test-scenario" });

    // 2 successful out of 3 total
    expect(screen.getByText(/2\/3 artifacts uploaded/)).toBeInTheDocument();
  });
});
