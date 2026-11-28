import { beforeEach, describe, expect, it } from "vitest";

import {
  clampFilePreviewWidth,
  FILE_PREVIEW_DEFAULT_WIDTH,
  FILE_PREVIEW_MAX_WIDTH,
  fitFilePreviewWidth,
  useFilePreviewStore,
} from "../useFilePreviewStore";

const STORAGE_KEY = "web-console.file-preview.v1";

describe("file preview persistence boundary", () => {
  beforeEach(() => {
    window.localStorage.removeItem(STORAGE_KEY);
    useFilePreviewStore.setState({ request: null, width: FILE_PREVIEW_DEFAULT_WIDTH, recent: [] });
  });

  it("persists layout width only while requests and recent paths stay in memory", () => {
    const request = { sessionId: "opaque-web-session", path: "/workspace/private-plan.md", source: "message_link" as const };
    useFilePreviewStore.getState().setRequest(request);
    useFilePreviewStore.getState().rememberRecent({ sessionId: request.sessionId, path: request.path, basename: "private-plan.md", kind: "markdown" });
    useFilePreviewStore.getState().setWidth(612);

    const raw = window.localStorage.getItem(STORAGE_KEY) ?? "";
    const stored = JSON.parse(raw) as { state?: Record<string, unknown>; version?: number };
    expect(stored.state).toEqual({ width: 612 });
    expect(stored.version).toBe(2);
    for (const secret of [request.sessionId, request.path, "private-plan.md"]) expect(raw).not.toContain(secret);
    expect(useFilePreviewStore.getState().request).toEqual(request);
    expect(useFilePreviewStore.getState().recent).toHaveLength(1);
  });

  it("migrates legacy durable requests and recents out of storage", async () => {
    const legacy = {
      request: { sessionId: "legacy-session-handle", path: "/private/workspace.md", source: "cli" },
      width: 655,
      recent: [{ sessionId: "legacy-session-handle", path: "/private/recent.txt", basename: "recent.txt", kind: "text", openedAt: 1 }],
    };
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: legacy, version: 1 }));
    await useFilePreviewStore.persist.rehydrate();

    expect(useFilePreviewStore.getState().request).toBeNull();
    expect(useFilePreviewStore.getState().recent).toEqual([]);
    expect(useFilePreviewStore.getState().width).toBe(655);
    const raw = window.localStorage.getItem(STORAGE_KEY) ?? "";
    const stored = JSON.parse(raw) as { state?: Record<string, unknown>; version?: number };
    expect(stored).toEqual({ state: { width: 655 }, version: 2 });
    for (const secret of ["legacy-session-handle", "/private/workspace.md", "/private/recent.txt"]) expect(raw).not.toContain(secret);
  });

  it("clears only the open request while preserving the in-memory width and recents", () => {
    useFilePreviewStore.getState().setRequest({ sessionId: "session-1", path: "/workspace/plan.md", source: "cli" });
    useFilePreviewStore.getState().rememberRecent({ sessionId: "session-1", path: "/workspace/recent.md", basename: "recent.md", kind: "markdown" });
    useFilePreviewStore.getState().setWidth(700);
    useFilePreviewStore.getState().clear();
    expect(useFilePreviewStore.getState().request).toBeNull();
    expect(useFilePreviewStore.getState().width).toBe(700);
    expect(useFilePreviewStore.getState().recent).toHaveLength(1);
  });

  it("remembers recent files newest first in memory without duplicates", () => {
    useFilePreviewStore.getState().rememberRecent({ sessionId: "s1", path: "/a.ts", basename: "a.ts", kind: "code" });
    useFilePreviewStore.getState().rememberRecent({ sessionId: "s1", path: "/b.ts", basename: "b.ts", kind: "code" });
    useFilePreviewStore.getState().rememberRecent({ sessionId: "s1", path: "/a.ts", basename: "a.ts", kind: "code" });
    expect(useFilePreviewStore.getState().recent.map((item) => item.path)).toEqual(["/a.ts", "/b.ts"]);
    expect(JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "{}").state).toEqual({ width: FILE_PREVIEW_DEFAULT_WIDTH });
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
