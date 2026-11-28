import { act, renderHook } from "@/test-utils";
import type { Execution, TimelineFrame } from "../store";
import { useExecutionExport } from "./useExecutionExport";
import { toast } from "react-hot-toast";

const { executeServerExport } = vi.hoisted(() => ({
  executeServerExport: vi.fn(),
}));

vi.mock("../export/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../export/api")>();
  return {
    ...actual,
    getLastOutputDir: () => "/tmp/bas-exports",
    executeServerExport,
  };
});

vi.mock("./useReplaySpec", () => ({
  useReplaySpec: () => ({
    movieSpec: null,
    movieSpecError: null,
    isMovieSpecLoading: false,
    previewMetrics: { capturedFrames: 0, assetCount: 0, totalDurationMs: 0 },
    setPreviewMetrics: vi.fn(),
    activeSpecId: null,
    setActiveSpecId: vi.fn(),
    setMovieSpecError: vi.fn(),
  }),
}));

vi.mock("@/domains/executions/export/hooks", () => ({
  useRecordedVideoStatus: () => ({
    available: false,
    count: 0,
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
  useExportProgress: () => ({ progress: null, reset: vi.fn() }),
}));

vi.mock("./exportPreview", () => ({
  fetchExecutionExportPreview: vi.fn().mockResolvedValue({
    preview: { executionId: "execution-123", specId: "spec-123" },
    status: "ready",
    metrics: { capturedFrames: 1, assetCount: 1, totalDurationMs: 1000 },
    movieSpec: null,
  }),
}));

vi.mock("react-hot-toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

const execution = {
  id: "execution-12345678",
  status: "completed",
  timeline: [],
} as unknown as Execution;

const replayCustomization = {} as Parameters<typeof useExecutionExport>[0]["replayCustomization"];
const createExport = vi.fn();
const params = {
  execution,
  workflowName: "Checkout flow",
  replayCustomization,
  createExport,
};

describe("useExecutionExport", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    executeServerExport.mockResolvedValue({
      export_id: "export-1",
      execution_id: execution.id,
      status: "processing",
    });
  });

  it("does not open the export dialog before replay frames exist", async () => {
    const { result } = renderHook(() =>
      useExecutionExport({ ...params, replayFrames: [] }),
    );
    await act(async () => {
      await new Promise<void>((resolve) => queueMicrotask(resolve));
    });

    act(() => result.current.openExportDialog());

    expect(toast.error).toHaveBeenCalledWith("Replay not ready to export yet");
    expect(result.current.isExportDialogOpen).toBe(false);
    expect(executeServerExport).not.toHaveBeenCalled();
  });

  it("requires an output directory and starts the configured server export", async () => {
    const replayFrames = [{ id: "frame-1", stepType: "click" }] as TimelineFrame[];
    const { result } = renderHook(() =>
      useExecutionExport({ ...params, replayFrames }),
    );

    act(() => result.current.openExportDialog());
    act(() => result.current.exportDialogProps.setOutputDir("   "));
    await act(async () => result.current.confirmExport());

    expect(toast.error).toHaveBeenCalledWith("Please specify an output directory");
    expect(executeServerExport).not.toHaveBeenCalled();

    act(() => result.current.exportDialogProps.setOutputDir("/tmp/my exports"));
    await act(async () => result.current.confirmExport());

    expect(executeServerExport).toHaveBeenCalledWith({
      executionId: execution.id,
      payload: expect.objectContaining({
        format: "mp4",
        file_name: "browser-automation-replay-executio.mp4",
        output_dir: "/tmp/my exports",
      }),
    });
    expect(result.current.isExporting).toBe(true);
    expect(toast.success).toHaveBeenCalledWith("Export started...");
  });
});
