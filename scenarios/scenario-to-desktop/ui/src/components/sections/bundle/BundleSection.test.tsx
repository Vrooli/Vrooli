/**
 * Tests for BundleSection component.
 * Tests stage status display, bundle details, and warning states.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { act } from "@testing-library/react";
import { BundleSection } from "./BundleSection";
import { usePipelineStore } from "../../../store";
import { createPipelineStatus } from "../../../test-utils/mocks";

// Reset store state before each test
beforeEach(() => {
  act(() => {
    usePipelineStore.getState().reset();
    usePipelineStore.setState({ scenarioName: "test-scenario" });
  });
});

describe("BundleSection", () => {
  it("renders pending state when no result", () => {
    render(<BundleSection scenarioName="test-scenario" />);

    expect(screen.getByText("Bundle")).toBeInTheDocument();
    expect(screen.getByText("Package dependencies for distribution")).toBeInTheDocument();
    expect(screen.getByText("Bundle Status")).toBeInTheDocument();
    expect(screen.getByText("No bundle results yet")).toBeInTheDocument();
  });

  it("renders running state with status badge", () => {
    act(() => {
      usePipelineStore.setState({
        pipelineStatus: createPipelineStatus({
          status: "running",
          current_stage: "bundle",
          stages: {
            ...createPipelineStatus().stages,
            bundle: { stage: "bundle", status: "running", started_at: Date.now() },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" />);

    // Check that at least one "Running" badge is present
    expect(screen.getAllByText("Running").length).toBeGreaterThan(0);
  });

  it("renders completed state with bundle details", () => {
    act(() => {
      usePipelineStore.setState({
        bundleResult: {
          bundle_dir: "/path/to/bundle/dir",
          manifest_path: "/path/to/manifest.json",
          total_size_human: "150 MB",
          copied_artifacts: ["app.js", "index.html", "styles.css"],
          runtime_binaries: { linux: "/path/to/binary" },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            bundle: { stage: "bundle", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" />);

    // Check that at least one "Completed" badge is present
    expect(screen.getAllByText("Completed").length).toBeGreaterThan(0);
    expect(screen.getByText(/4 artifacts bundled/)).toBeInTheDocument();
    expect(screen.getByText("Bundle Directory")).toBeInTheDocument();
    expect(screen.getByText("/path/to/bundle/dir")).toBeInTheDocument();
    expect(screen.getByText("Manifest Path")).toBeInTheDocument();
    expect(screen.getByText("/path/to/manifest.json")).toBeInTheDocument();
    expect(screen.getByText(/Total bundle size.*150 MB/)).toBeInTheDocument();
  });

  it("shows size warning when present", () => {
    act(() => {
      usePipelineStore.setState({
        bundleResult: {
          bundle_dir: "/path/to/bundle",
          manifest_path: "/path/to/manifest.json",
          total_size_human: "2 GB",
          copied_artifacts: [],
          runtime_binaries: {},
          size_warning: {
            message: "Bundle size exceeds 1 GB. Consider optimizing.",
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            bundle: { stage: "bundle", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Warning:/)).toBeInTheDocument();
    expect(screen.getByText(/Bundle size exceeds 1 GB/)).toBeInTheDocument();
  });

  it("renders failed state with error", () => {
    act(() => {
      usePipelineStore.setState({
        pipelineStatus: createPipelineStatus({
          status: "failed",
          stages: {
            ...createPipelineStatus().stages,
            bundle: { stage: "bundle", status: "failed", started_at: Date.now(), error: "Bundle failed" },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Bundle stage failed/)).toBeInTheDocument();
  });

  it("shows placeholder when scenario selected but not bundled", () => {
    render(<BundleSection scenarioName="test-scenario" />);

    expect(screen.getByText(/Bundle stage will run when you generate/)).toBeInTheDocument();
  });

  it("shows different placeholder when no scenario selected", () => {
    render(<BundleSection scenarioName="" />);

    expect(screen.getByText("Select a scenario to begin.")).toBeInTheDocument();
  });

  it("uses singular form for single artifact", () => {
    act(() => {
      usePipelineStore.setState({
        bundleResult: {
          bundle_dir: "/path",
          manifest_path: "/path/manifest.json",
          total_size_human: "10 MB",
          copied_artifacts: ["app.js"],
          runtime_binaries: {},
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            bundle: { stage: "bundle", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" />);

    expect(screen.getByText(/1 artifact bundled/)).toBeInTheDocument();
  });

  it("counts runtime binaries in artifact count", () => {
    act(() => {
      usePipelineStore.setState({
        bundleResult: {
          bundle_dir: "/path",
          manifest_path: "/path/manifest.json",
          total_size_human: "50 MB",
          copied_artifacts: ["app.js"],
          runtime_binaries: {
            linux: "/path/linux",
            win: "/path/win",
            mac: "/path/mac",
          },
        },
        pipelineStatus: createPipelineStatus({
          status: "completed",
          stages: {
            ...createPipelineStatus().stages,
            bundle: { stage: "bundle", status: "completed", started_at: Date.now() },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" />);

    // 1 copied artifact + 3 runtime binaries = 4 total
    expect(screen.getByText(/4 artifacts bundled/)).toBeInTheDocument();
  });
});
