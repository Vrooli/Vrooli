import { fireEvent, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { vi } from "vitest";

const getDeployment = vi.hoisted(() => vi.fn());
vi.mock("./lib/api", () => ({ getDeployment, API_BASE: "" }));
vi.mock("@vrooli/api-base", () => ({ buildApiUrl: (path: string) => path }));

import { useDeploymentListProgress } from "./hooks/useDeploymentListProgress";
import { useManifestKeyboardShortcuts } from "./hooks/useManifestKeyboardShortcuts";

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider>;
}

describe("deployment list progress polling", () => {
  it("fetches changed ids, ignores individual failures, and handles visibility", async () => {
    getDeployment.mockImplementation(async (id: string) => {
      if (id === "gone") throw new Error("deleted");
      return { deployment: { status: "deploying", progress_step: "setup", progress_percent: 50 } };
    });
    const { result, rerender, unmount } = renderHook(({ ids }) => useDeploymentListProgress(ids), { initialProps: { ids: ["one", "gone"] }, wrapper });
    await waitFor(() => expect(result.current.progressMap.one?.progress_percent).toBe(50));
    Object.defineProperty(document, "hidden", { configurable: true, value: true });
    document.dispatchEvent(new Event("visibilitychange"));
    Object.defineProperty(document, "hidden", { configurable: true, value: false });
    document.dispatchEvent(new Event("visibilitychange"));
    rerender({ ids: [] });
    expect(result.current.isPolling).toBe(false);
    unmount();
  });
});

import { useDocsManifest, useDocContent } from "./hooks/useDocs";

describe("documentation data hooks", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => ({ version: "1", defaultDocument: "a.md", sections: [], content: "# A" }) }));
  });

  it("loads manifest and content, skips null content, and surfaces HTTP failures", async () => {
    const manifest = renderHook(() => useDocsManifest(), { wrapper });
    await waitFor(() => expect(manifest.result.current.data?.version).toBe("1"));
    const empty = renderHook(() => useDocContent(null), { wrapper });
    expect(empty.result.current.data).toBeUndefined();
    const content = renderHook(() => useDocContent("a.md"), { wrapper });
    await waitFor(() => expect(content.result.current.data).toBe("# A"));
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 503 }));
    const failed = renderHook(() => useDocsManifest(), { wrapper });
    await waitFor(() => expect(failed.result.current.error).toBeTruthy());
  });
});

describe("manifest keyboard shortcuts", () => {
  it("ignores normal inputs and handles control/meta undo and redo chords", () => {
    const undo = vi.fn();
    const redo = vi.fn();
    renderHook(() => useManifestKeyboardShortcuts(undo, redo));
    const input = document.createElement("input");
    const textarea = document.createElement("textarea");
    const editor = document.createElement("textarea");
    editor.setAttribute("data-testid", "manifest-input");
    document.body.append(input, textarea, editor);
    fireEvent.keyDown(input, { key: "z", ctrlKey: true });
    fireEvent.keyDown(textarea, { key: "z", ctrlKey: true });
    expect(undo).not.toHaveBeenCalled();
    fireEvent.keyDown(window, { key: "z", metaKey: true });
    fireEvent.keyDown(window, { key: "z", metaKey: true, shiftKey: true });
    fireEvent.keyDown(editor, { key: "y", ctrlKey: true });
    expect(undo).toHaveBeenCalledOnce();
    expect(redo).toHaveBeenCalledTimes(2);
  });
});
