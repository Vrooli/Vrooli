import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ExportDialog } from "./ExportDialog";
import { ExportDialogProvider } from "../context/ExportDialogProvider";
import type { ExportDialogContextValue } from "../context/ExportDialogContext";
import { renderWithProviders } from "@/test-utils";

vi.mock("@shared/layout", () => ({
  ResponsiveDialog: ({
    isOpen,
    children,
  }: {
    isOpen: boolean;
    children: React.ReactNode;
  }) => (isOpen ? <div role="dialog">{children}</div> : null),
}));

vi.mock("@/domains/exports/replay/ReplayPlayer", () => ({
  default: () => <div data-testid="replay-player" />,
}));

function makeContext(
  format: "mp4" | "json",
  replayFramesLength: number,
): ExportDialogContextValue {
  const noOp = () => {};
  return {
    titleId: "export-title",
    descriptionId: "export-description",
    isEditMode: false,
    formatState: {
      format,
      setFormat: noOp,
      formats: [format],
      toggleFormat: noOp,
      isBinaryExport: format === "mp4",
      formatOptions: [],
    },
    dimensionState: {
      preset: "720p",
      setPreset: noOp,
      presetOptions: [],
      selectedDimensions: { width: 1280, height: 720 },
      customWidthInput: "1280",
      customHeightInput: "720",
      setCustomWidthInput: noOp,
      setCustomHeightInput: noOp,
    },
    fileState: {
      fileStem: "execution",
      setFileStem: noOp,
      defaultFileStem: "execution",
      finalFileName: `execution.${format}`,
      outputDir: "data/exports",
      setOutputDir: noOp,
    },
    renderSourceState: {
      source: "auto",
      setSource: noOp,
      sourceOptions: [],
      recordedVideoAvailable: false,
      recordedVideoCount: 0,
      recordedVideoLoading: false,
    },
    stylizationState: {
      stylization: "raw",
      setStylization: noOp,
      isStylized: false,
    },
    previewState: {
      movieSpec: null,
      replayFrames: [],
      replayStyle: {} as ExportDialogContextValue["previewState"]["replayStyle"],
      recordedVideoUrl: null,
      firstFramePreviewUrl: null,
      firstFrameLabel: null,
      composerPreviewUrl: "",
      composerRef: { current: null },
      composerWindowRef: { current: null },
      composerOriginRef: { current: null },
      isComposerReady: false,
      setIsComposerReady: noOp,
      composerError: null,
      setComposerError: noOp,
    },
    progressState: {
      isExporting: false,
      isPreviewLoading: false,
      statusMessage: "Ready",
      activeExportId: null,
      exportProgress: null,
    },
    metricsState: {
      replayFramesLength,
      estimatedFrameCount: replayFramesLength,
      estimatedDurationSeconds: null,
      activeSpecId: null,
      previewMetrics: { capturedFrames: replayFramesLength, assetCount: 0, totalDurationMs: 0 },
    },
    actions: { onClose: noOp, onConfirm: noOp },
    formatSeconds: (value) => `${value}s`,
  };
}

function renderDialog(format: "mp4" | "json", replayFramesLength: number) {
  const context = makeContext(format, replayFramesLength);
  renderWithProviders(
    <ExportDialogProvider value={context}>
      <ExportDialog isOpen />
    </ExportDialogProvider>,
  );
}

describe("ExportDialog", () => {
  it("prevents a video export when the execution has no replay frames", () => {
    renderDialog("mp4", 0);

    expect(screen.getByRole("button", { name: "Export replay" })).toBeDisabled();
  });

  it("allows a raw JSON replay package when no video frames are available", () => {
    renderDialog("json", 0);

    expect(screen.getByRole("button", { name: "Export replay" })).toBeEnabled();
  });
});
