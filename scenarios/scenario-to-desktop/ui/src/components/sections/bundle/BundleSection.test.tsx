/**
 * Tests for BundleSection component.
 * Tests stage status display, bundle details, and warning states.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@/test-utils";
import { act } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { BundleSection } from "./BundleSection";
import { usePipelineStore } from "../../../store";
import { createPipelineStatus } from "../../../test-utils/mocks";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { BundleStageDetailsSchema } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";

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
    expect(
      screen.getByText("Package dependencies for distribution"),
    ).toBeInTheDocument();
    // Non-bundled mode shows placeholder
    expect(
      screen.getByText(/Bundle stage will run when you generate/),
    ).toBeInTheDocument();
  });

  it("renders running state with status badge", () => {
    act(() => {
      usePipelineStore.setState({
        runStatus: "running",
        pipelineStatus: createPipelineStatus({
          status: StageStatus.RUNNING,
          currentStage: StageName.BUNDLE,
          stages: {
            ...createPipelineStatus().stages,
            bundle: {
              stage: StageName.BUNDLE,
              status: StageStatus.RUNNING,
            },
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
        bundleResult: create(BundleStageDetailsSchema, {
          bundleDir: "/path/to/bundle/dir",
          manifestPath: "/path/to/manifest.json",
          totalSizeHuman: "150 MB",
          copiedArtifacts: ["app.js", "index.html", "styles.css"],
          runtimeBinaries: { linux: "/path/to/binary" },
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            bundle: {
              stage: StageName.BUNDLE,
              status: StageStatus.COMPLETED,
            },
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
        bundleResult: create(BundleStageDetailsSchema, {
          bundleDir: "/path/to/bundle",
          manifestPath: "/path/to/manifest.json",
          totalSizeHuman: "2 GB",
          copiedArtifacts: [],
          runtimeBinaries: {},
          sizeWarning: {
            message: "Bundle size exceeds 1 GB. Consider optimizing.",
          },
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            bundle: {
              stage: StageName.BUNDLE,
              status: StageStatus.COMPLETED,
            },
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
          status: StageStatus.FAILED,
          stages: {
            ...createPipelineStatus().stages,
            bundle: {
              stage: StageName.BUNDLE,
              status: StageStatus.FAILED,
              error: "Bundle failed",
            },
          },
        }),
      });
    });

    render(<BundleSection scenarioName="test-scenario" isBundled={true} />);

    expect(screen.getByText("Bundle failed")).toBeInTheDocument();
  });

  it("shows placeholder when scenario selected but not bundled", () => {
    render(<BundleSection scenarioName="test-scenario" isBundled={false} />);

    expect(
      screen.getByText(/Bundle stage will run when you generate/),
    ).toBeInTheDocument();
  });

  it("shows different placeholder when no scenario selected", () => {
    render(<BundleSection scenarioName="" isBundled={false} />);

    expect(screen.getByText("Select a scenario to begin.")).toBeInTheDocument();
  });

  it("shows artifact count for single artifact", () => {
    act(() => {
      usePipelineStore.setState({
        bundleResult: create(BundleStageDetailsSchema, {
          bundleDir: "/path",
          manifestPath: "/path/manifest.json",
          totalSizeHuman: "10 MB",
          copiedArtifacts: ["app.js"],
          runtimeBinaries: {},
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            bundle: {
              stage: StageName.BUNDLE,
              status: StageStatus.COMPLETED,
            },
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
        bundleResult: create(BundleStageDetailsSchema, {
          bundleDir: "/path",
          manifestPath: "/path/manifest.json",
          totalSizeHuman: "50 MB",
          copiedArtifacts: ["app.js"],
          runtimeBinaries: {
            linux: "/path/linux",
            win: "/path/win",
            mac: "/path/mac",
          },
        }),
        pipelineStatus: createPipelineStatus({
          status: StageStatus.COMPLETED,
          stages: {
            ...createPipelineStatus().stages,
            bundle: {
              stage: StageName.BUNDLE,
              status: StageStatus.COMPLETED,
            },
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
