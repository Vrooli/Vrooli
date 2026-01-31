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
    // Non-bundled mode shows placeholder
    expect(screen.getByText(/Bundle stage will run when you generate/)).toBeInTheDocument();
  });

  it("renders running state with status badge", () => {
    act(() => {
      usePipelineStore.setState({
        runStatus: "running",
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

    render(<BundleSection scenarioName="test-scenario" isBundled={true} />);

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

    render(<BundleSection scenarioName="test-scenario" isBundled={true} />);

    // Check that at least one "Complete" badge is present
    expect(screen.getAllByText("Complete").length).toBeGreaterThan(0);
    // Check bundle details are shown
    expect(screen.getByText("Bundle directory:")).toBeInTheDocument();
    expect(screen.getByText("/path/to/bundle/dir")).toBeInTheDocument();
    expect(screen.getByText("Manifest:")).toBeInTheDocument();
    expect(screen.getByText("/path/to/manifest.json")).toBeInTheDocument();
    expect(screen.getByText("150 MB")).toBeInTheDocument();
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

    render(<BundleSection scenarioName="test-scenario" isBundled={true} />);

    expect(screen.getByText(/Bundle size exceeds 1 GB/)).toBeInTheDocument();
  });

  it("renders failed state with error", () => {
    act(() => {
      usePipelineStore.setState({
        errorInfo: { message: "Bundle failed" },
        pipelineStatus: createPipelineStatus({
          status: "failed",
          stages: {
            ...createPipelineStatus().stages,
            bundle: { stage: "bundle", status: "failed", started_at: Date.now(), error: "Bundle failed" },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" isBundled={true} />);

    expect(screen.getByText("Bundle failed")).toBeInTheDocument();
  });

  it("shows placeholder when scenario selected but not bundled", () => {
    render(<BundleSection scenarioName="test-scenario" isBundled={false} />);

    expect(screen.getByText(/Bundle stage will run when you generate/)).toBeInTheDocument();
  });

  it("shows different placeholder when no scenario selected", () => {
    render(<BundleSection scenarioName="" isBundled={false} />);

    expect(screen.getByText("Select a scenario to begin.")).toBeInTheDocument();
  });

  it("shows artifact count for single artifact", () => {
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

    render(<BundleSection scenarioName="test-scenario" isBundled={true} />);

    // BundleResultsCard shows "Artifacts: X files"
    expect(screen.getByText("1 files")).toBeInTheDocument();
  });

  it("shows platform builds for runtime binaries", () => {
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

    render(<BundleSection scenarioName="test-scenario" isBundled={true} />);

    // Check that platform builds section is shown
    expect(screen.getByText("Platform builds")).toBeInTheDocument();
    // Each platform appears twice (once in payload JSON, once in UI) so use getAllByText
    expect(screen.getAllByText("linux").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("win").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("mac").length).toBeGreaterThanOrEqual(1);
  });
});
