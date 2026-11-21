/**
 * Tests for BuildSection component.
 * Tests stage status display, build action, progress, artifact listing, and error states.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { render, screen, fireEvent } from "@testing-library/react";
import { act } from "@testing-library/react";
import { BuildSection } from "./BuildSection";
import { usePipelineStore } from "../../../store";
import {
  createBuildResult,
  createPipelineStatus,
} from "../../../test-utils/mocks";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { GenerateResponseSchema } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";

const createGenerateResult = () =>
  create(GenerateResponseSchema, {
    desktopPath: "/path/to/desktop",
    pipelineId: "pipeline-test",
  });

// Reset store state before each test
beforeEach(() => {
  act(() => {
    usePipelineStore.getState().reset();
    usePipelineStore.setState({ scenarioName: "test-scenario" });
  });
});

describe("BuildSection", () => {
  it("renders pending state when no result", () => {
    render(<BuildSection scenarioName="test-scenario" />);

    expect(screen.getByText("Build")).toBeInTheDocument();
    expect(
      screen.getByText("Compile installers for target platforms"),
    ).toBeInTheDocument();
    expect(screen.getByText("Build Status")).toBeInTheDocument();
    expect(screen.getByText("Waiting for generate stage")).toBeInTheDocument();
  });

  it("renders about section explaining building", () => {
    render(<BuildSection scenarioName="test-scenario" />);

    expect(screen.getByText("About building")).toBeInTheDocument();
    expect(
      screen.getByText(/The build stage packages the Electron wrapper/),
    ).toBeInTheDocument();
  });

  it("shows build button when generate result is available", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: createGenerateResult(),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(screen.getByText("Ready to build")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /build installers/i }),
    ).toBeInTheDocument();
  });

  it("calls runStage when build button is clicked", () => {
    const runStageSpy = vi.fn().mockResolvedValue("pipeline-123");

    act(() => {
      usePipelineStore.setState({
        generateResult: createGenerateResult(),
        runStage: runStageSpy,
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    const buildButton = screen.getByRole("button", {
      name: /build installers/i,
    });
    fireEvent.click(buildButton);

    expect(runStageSpy).toHaveBeenCalledWith(StageName.BUILD);
  });

  it("shows Starting state when submitting", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: createGenerateResult(),
        isSubmitting: true,
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    const startingButton = screen.getByText("Starting...").closest("button");
    expect(startingButton).toBeInTheDocument();
    expect(startingButton).toBeDisabled();
  });

  it("disables build button when another stage is busy", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: createGenerateResult(),
        runStatus: "running",
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    // When running, it should show progress bar instead of button
    // So button should not be present
    expect(
      screen.queryByRole("button", { name: /build installers/i }),
    ).not.toBeInTheDocument();
  });

  it("shows progress bar when running", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: createGenerateResult(),
        runStatus: "running",
        pipelineStatus: createPipelineStatus({
          status: StageStatus.RUNNING,
          currentStage: StageName.BUILD,
          stages: {
            ...createPipelineStatus().stages,
            generate: {
              stage: StageName.GENERATE,
              status: StageStatus.COMPLETED,
            },
            build: {
              stage: StageName.BUILD,
              status: StageStatus.RUNNING,
            },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Running Build stage/)).toBeInTheDocument();
    expect(screen.getByText("Building")).toBeInTheDocument();
  });

  it("shows cancel button when running", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: createGenerateResult(),
        runStatus: "running",
        pipelineStatus: createPipelineStatus({
          status: StageStatus.RUNNING,
          currentStage: StageName.BUILD,
          stages: {
            ...createPipelineStatus().stages,
            build: {
              stage: StageName.BUILD,
              status: StageStatus.RUNNING,
            },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(
      screen.getByRole("button", { name: /cancel build/i }),
    ).toBeInTheDocument();
  });

  it("renders completed state with artifact list", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: createBuildResult(["win", "mac", "linux"], {
          outputPath: "/path/to/output",
          artifacts: {
            win: "/path/to/output/win/installer.exe",
            mac: "/path/to/output/mac/installer.dmg",
            linux: "/path/to/output/linux/installer.AppImage",
          },
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            build: {
              stage: StageName.BUILD,
              status: StageStatus.COMPLETED,
            },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(screen.getByText("Built")).toBeInTheDocument();
    expect(screen.getByText(/3 artifacts built/)).toBeInTheDocument();
    expect(screen.getByText("win")).toBeInTheDocument();
    expect(screen.getByText("mac")).toBeInTheDocument();
    expect(screen.getByText("linux")).toBeInTheDocument();
    expect(screen.getByText("installer.exe")).toBeInTheDocument();
    expect(screen.getByText("installer.dmg")).toBeInTheDocument();
    expect(screen.getByText("installer.AppImage")).toBeInTheDocument();
  });

  it("shows output path when available", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: createBuildResult(["win"], {
          outputPath: "/custom/output/path",
          artifacts: { win: "/custom/output/path/win/app.exe" },
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            build: {
              stage: StageName.BUILD,
              status: StageStatus.COMPLETED,
            },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(screen.getByText("Output Path")).toBeInTheDocument();
    expect(screen.getByText("/custom/output/path")).toBeInTheDocument();
  });

  it("renders failed state with error message", () => {
    act(() => {
      usePipelineStore.setState({
        errorInfo: {
          message: "Build failed: missing dependencies",
          category: "resource",
          suggestions: ["Install required build tools"],
        },
        pipelineStatus: createPipelineStatus({
          status: StageStatus.FAILED,
          stages: {
            ...createPipelineStatus().stages,
            build: { stage: StageName.BUILD, status: StageStatus.FAILED },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(
      screen.getByText("Build failed: missing dependencies"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Install required build tools"),
    ).toBeInTheDocument();
  });

  it("calls resetForRetry when retry clicked", () => {
    const resetForRetrySpy = vi.spyOn(
      usePipelineStore.getState(),
      "resetForRetry",
    );

    act(() => {
      usePipelineStore.setState({
        errorInfo: { message: "Failed", category: "unknown" },
        pipelineStatus: createPipelineStatus({
          status: StageStatus.FAILED,
          stages: {
            ...createPipelineStatus().stages,
            build: { stage: StageName.BUILD, status: StageStatus.FAILED },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    const retryButton = screen.getByRole("button", { name: /^retry$/i });
    fireEvent.click(retryButton);

    expect(resetForRetrySpy).toHaveBeenCalled();
  });

  it("shows 'Rebuild Installers' button when already has result", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: createGenerateResult(),
        buildResult: createBuildResult(["win"], {
          outputPath: "/path",
          artifacts: { win: "/path/win/app.exe" },
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            build: {
              stage: StageName.BUILD,
              status: StageStatus.COMPLETED,
            },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(
      screen.getByRole("button", { name: /rebuild installers/i }),
    ).toBeInTheDocument();
  });

  it("shows placeholder when no scenario selected", () => {
    render(<BuildSection scenarioName="" />);

    expect(
      screen.getByText("Select a scenario to unlock installer builds."),
    ).toBeInTheDocument();
  });

  it("uses singular form for single artifact", () => {
    act(() => {
      usePipelineStore.setState({
        buildResult: createBuildResult(["win"], {
          outputPath: "/path",
          artifacts: { win: "/path/win/app.exe" },
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            build: {
              stage: StageName.BUILD,
              status: StageStatus.COMPLETED,
            },
          },
        }),
      });
    });

    render(<BuildSection scenarioName="test-scenario" />);

    expect(screen.getByText(/1 artifact built/)).toBeInTheDocument();
  });
});
