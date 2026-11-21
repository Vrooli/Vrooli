import { beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "@testing-library/react";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { EvidenceCaptureSchema } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";

const capturesApi = vi.hoisted(() => ({
  listCaptures: vi.fn(),
  getCapturesSummary: vi.fn(),
  deleteCapture: vi.fn(),
  deleteAllCaptures: vi.fn(),
  buildCapturesDownloadUrl: vi.fn(),
}));

vi.mock("../lib/api/captures", () => capturesApi);

import { useCapturesStore } from "./capturesStore";

const capture = (init: MessageInitShape<typeof EvidenceCaptureSchema>) =>
  create(EvidenceCaptureSchema, init);

function resetStore() {
  useCapturesStore.setState({
    isOpen: false,
    scenarioName: null,
    captures: [],
    summary: null,
    selectedIds: new Set(),
    loading: false,
    error: null,
  });
}

beforeEach(() => {
  resetStore();
  vi.clearAllMocks();
  capturesApi.listCaptures.mockResolvedValue([
    capture({ captureId: "cap-1" }),
    capture({ captureId: "cap-2" }),
  ]);
  capturesApi.getCapturesSummary.mockResolvedValue({
    count: 2,
    totalBytes: 84n,
  });
  capturesApi.buildCapturesDownloadUrl.mockReturnValue("/captures/download");
  vi.stubGlobal("open", vi.fn());
});

describe("capturesStore", () => {
  it("opens a scenario and loads captures plus summary through the typed API", async () => {
    act(() => {
      useCapturesStore.getState().open("calculator");
    });

    await vi.waitFor(() => {
      expect(useCapturesStore.getState().loading).toBe(false);
    });

    expect(useCapturesStore.getState()).toMatchObject({
      isOpen: true,
      scenarioName: "calculator",
      summary: { count: 2, totalBytes: 84n },
    });
    expect(useCapturesStore.getState().captures).toHaveLength(2);
    expect(capturesApi.listCaptures).toHaveBeenCalledWith("calculator");
    expect(capturesApi.getCapturesSummary).toHaveBeenCalledWith("calculator");
  });

  it("preserves a useful load error and clears it when a later load succeeds", async () => {
    capturesApi.listCaptures.mockRejectedValueOnce(
      new Error("evidence service unavailable"),
    );
    useCapturesStore.setState({ scenarioName: "calculator" });

    await act(async () => useCapturesStore.getState().fetchCaptures());
    expect(useCapturesStore.getState()).toMatchObject({
      loading: false,
      error: "evidence service unavailable",
    });

    await act(async () => useCapturesStore.getState().fetchCaptures());
    expect(useCapturesStore.getState().error).toBeNull();
    expect(useCapturesStore.getState().captures).toHaveLength(2);
  });

  it("selects captures and opens only the selected evidence download", () => {
    useCapturesStore.setState({
      scenarioName: "calculator",
      captures: [
        capture({ captureId: "cap-1" }),
        capture({ captureId: "cap-2" }),
      ],
    });

    act(() => {
      useCapturesStore.getState().selectAll();
    });
    act(() => {
      useCapturesStore.getState().toggleSelect("cap-2");
    });
    act(() => {
      useCapturesStore.getState().downloadSelected();
    });

    expect(capturesApi.buildCapturesDownloadUrl).toHaveBeenCalledWith(
      "calculator",
      ["cap-1"],
    );
    expect(window.open).toHaveBeenCalledWith("/captures/download", "_blank");

    act(() => {
      useCapturesStore.getState().deselectAll();
    });
    act(() => {
      useCapturesStore.getState().downloadSelected();
    });
    expect(window.open).toHaveBeenCalledTimes(1);
  });

  it("deletes selected capture state only after the backend deletion succeeds", async () => {
    useCapturesStore.setState({
      scenarioName: "calculator",
      captures: [capture({ captureId: "cap-1" })],
      selectedIds: new Set(["cap-1"]),
    });

    await act(async () => useCapturesStore.getState().deleteCapture("cap-1"));

    expect(capturesApi.deleteCapture).toHaveBeenCalledWith(
      "calculator",
      "cap-1",
    );
    expect(useCapturesStore.getState().selectedIds).toEqual(new Set());
    expect(capturesApi.listCaptures).toHaveBeenCalledWith("calculator");
  });
});
