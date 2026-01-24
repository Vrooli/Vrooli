/**
 * Tests for GenerateSection component.
 * Tests stage status display, results, and error states.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { act } from "@testing-library/react";
import { GenerateSection } from "./GenerateSection";
import { usePipelineStore } from "../../../store";
import type { VerbosePipelineStatus } from "../../../lib/api";

// Reset store state before each test
beforeEach(() => {
  act(() => {
    usePipelineStore.getState().reset();
    usePipelineStore.setState({ scenarioName: "test-scenario" });
  });
});

describe("GenerateSection", () => {
  it("renders pending state when no result", () => {
    render(<GenerateSection scenarioName="test-scenario" />);

    expect(screen.getByText("Generate")).toBeInTheDocument();
    expect(screen.getByText("Create desktop wrapper code")).toBeInTheDocument();
    expect(screen.getByText("Generate Status")).toBeInTheDocument();
    expect(screen.getByText("Wrapper not yet generated")).toBeInTheDocument();
  });

  it("renders about section explaining generation", () => {
    render(<GenerateSection scenarioName="test-scenario" />);

    expect(screen.getByText("About generation")).toBeInTheDocument();
    expect(screen.getByText(/The generate stage creates an Electron project scaffold/)).toBeInTheDocument();
  });

  it("renders running state with status badge", () => {
    act(() => {
      usePipelineStore.setState({
        pipelineStatus: {
          pipeline_id: "test",
          status: "running",
          current_stage: "generate",
          stages: {
            generate: { status: "running" },
          },
        } as VerbosePipelineStatus,
      });
    });

    render(<GenerateSection scenarioName="test-scenario" />);

    expect(screen.getByText("Generating")).toBeInTheDocument();
  });

  it("renders completed state with results", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: {
          desktop_path: "/path/to/desktop/app",
          build_id: "build-abc123def456",
        },
        pipelineStatus: {
          pipeline_id: "test",
          status: "completed",
          stages: {
            generate: { status: "completed" },
          },
        } as VerbosePipelineStatus,
      });
    });

    render(<GenerateSection scenarioName="test-scenario" />);

    expect(screen.getByText("Generated")).toBeInTheDocument();
    expect(screen.getByText(/Electron wrapper generated/)).toBeInTheDocument();
    expect(screen.getByText("/path/to/desktop/app")).toBeInTheDocument();
    expect(screen.getByText("Desktop Application Path")).toBeInTheDocument();
  });

  it("shows build ID in description when available", () => {
    act(() => {
      usePipelineStore.setState({
        generateResult: {
          desktop_path: "/path/to/desktop",
          build_id: "build-abc123",
        },
        pipelineStatus: {
          pipeline_id: "test",
          status: "completed",
          stages: {
            generate: { status: "completed" },
          },
        } as VerbosePipelineStatus,
      });
    });

    render(<GenerateSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Build: build-ab/)).toBeInTheDocument();
  });

  it("renders failed state with error message", () => {
    act(() => {
      usePipelineStore.setState({
        errorInfo: {
          message: "Generation failed: invalid template",
          category: "validation",
          suggestions: ["Check your template configuration"],
        },
        pipelineStatus: {
          pipeline_id: "test",
          status: "failed",
          stages: {
            generate: { status: "failed", error: "Generation failed" },
          },
        } as VerbosePipelineStatus,
      });
    });

    render(<GenerateSection scenarioName="test-scenario" />);

    expect(screen.getByText("Generation failed: invalid template")).toBeInTheDocument();
    expect(screen.getByText("Check your template configuration")).toBeInTheDocument();
  });

  it("calls onRetry and resets when retry clicked", () => {
    const onRetry = vi.fn();
    const resetForRetrySpy = vi.spyOn(usePipelineStore.getState(), "resetForRetry");

    act(() => {
      usePipelineStore.setState({
        errorInfo: { message: "Failed", category: "unknown" },
        pipelineStatus: {
          pipeline_id: "test",
          status: "failed",
          stages: {
            generate: { status: "failed" },
          },
        } as VerbosePipelineStatus,
      });
    });

    render(<GenerateSection scenarioName="test-scenario" onRetry={onRetry} />);

    const retryButton = screen.getByRole("button", { name: /^retry$/i });
    fireEvent.click(retryButton);

    expect(resetForRetrySpy).toHaveBeenCalled();
    expect(onRetry).toHaveBeenCalled();
  });

  it("shows placeholder when no scenario selected", () => {
    render(<GenerateSection scenarioName="" />);

    expect(screen.getByText("Select a scenario to begin.")).toBeInTheDocument();
  });

  it("shows instruction when scenario selected but not generated", () => {
    render(<GenerateSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Use the "Generate Desktop Application" button/)).toBeInTheDocument();
  });

  it("clears error when dismiss clicked", () => {
    const clearErrorSpy = vi.spyOn(usePipelineStore.getState(), "clearError");

    act(() => {
      usePipelineStore.setState({
        errorInfo: { message: "Failed", category: "unknown" },
        pipelineStatus: {
          pipeline_id: "test",
          status: "failed",
          stages: {
            generate: { status: "failed" },
          },
        } as VerbosePipelineStatus,
      });
    });

    render(<GenerateSection scenarioName="test-scenario" />);

    const dismissButton = screen.getByRole("button", { name: /^dismiss$/i });
    fireEvent.click(dismissButton);

    expect(clearErrorSpy).toHaveBeenCalled();
  });
});
