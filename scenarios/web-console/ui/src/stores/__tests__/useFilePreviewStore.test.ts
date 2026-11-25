import { beforeEach, describe, expect, it } from "vitest";

import {
  clampFilePreviewWidth,
  FILE_PREVIEW_DEFAULT_WIDTH,
  FILE_PREVIEW_MAX_WIDTH,
  fitFilePreviewWidth,
  useFilePreviewStore,
} from "../useFilePreviewStore";

describe("file preview persistence", () => {
  beforeEach(() => {
    useFilePreviewStore.setState({ request: null, width: FILE_PREVIEW_DEFAULT_WIDTH });
  });

  it("keeps the open file and its width across a remount", () => {
    const request = { sessionId: "session-1", path: "/workspace/plan.md", source: "message_link" as const };
    useFilePreviewStore.getState().setRequest(request);
    useFilePreviewStore.getState().setWidth(612);

    const stored = JSON.parse(window.localStorage.getItem("web-console.file-preview.v1") ?? "{}") as { state?: { request?: unknown; width?: unknown } };
    expect(stored.state?.request).toMatchObject(request);
    expect(stored.state?.width).toBe(612);
    expect(useFilePreviewStore.getState().request).toEqual(request);
  });

  it("clears only the open request, keeping the chosen width", () => {
    useFilePreviewStore.getState().setRequest({ sessionId: "session-1", path: "/workspace/plan.md", source: "cli" });
    useFilePreviewStore.getState().setWidth(700);
    useFilePreviewStore.getState().clear();
    expect(useFilePreviewStore.getState().request).toBeNull();
    expect(useFilePreviewStore.getState().width).toBe(700);
  });

  it("clamps a stored or dragged width into the supported range", () => {
    expect(clampFilePreviewWidth(10)).toBe(360);
    expect(clampFilePreviewWidth(5000)).toBe(FILE_PREVIEW_MAX_WIDTH);
    expect(clampFilePreviewWidth(Number.NaN)).toBe(FILE_PREVIEW_DEFAULT_WIDTH);
    expect(fitFilePreviewWidth(900, 1000)).toBe(520);
    expect(fitFilePreviewWidth(900, 2000)).toBe(900);
    expect(fitFilePreviewWidth(200, 2000)).toBe(360);
  });
});
