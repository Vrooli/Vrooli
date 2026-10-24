import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SnippetDTO } from "../api/snippets";

const api = vi.hoisted(() => ({
  listSnippets: vi.fn(),
  upsertSnippet: vi.fn(),
  deleteSnippet: vi.fn(),
  touchSnippet: vi.fn(),
}));

vi.mock("../api/snippets", () => api);

import { resetSnippetsCacheForTests, useSnippets } from "./useSnippets";

function snippet(overrides: Partial<SnippetDTO> & Pick<SnippetDTO, "id">): SnippetDTO {
  return {
    name: overrides.id,
    body: "",
    color: "",
    pinned: false,
    use_count: 0,
    last_used_at: "",
    sort_order: 0,
    created_at: "",
    updated_at: "",
    ...overrides,
    id: overrides.id,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((ok, fail) => { resolve = ok; reject = fail; });
  return { promise, resolve, reject };
}

describe("useSnippets", () => {
  beforeEach(() => {
    resetSnippetsCacheForTests();
    vi.clearAllMocks();
  });

  it("orders equal recency by use count descending", async () => {
    api.listSnippets.mockResolvedValue([
      snippet({ id: "low", last_used_at: "2026-01-01T00:00:00Z", use_count: 1 }),
      snippet({ id: "high", last_used_at: "2026-01-01T00:00:00Z", use_count: 8 }),
    ]);
    const { result } = renderHook(() => useSnippets());
    await waitFor(() => expect(result.current.status).toBe("ready"));
    expect(result.current.snippets.map((item) => item.id)).toEqual(["high", "low"]);
  });

  it("orders a never-used snippet after one with written recency", async () => {
    api.listSnippets.mockResolvedValue([
      snippet({ id: "never", use_count: 100 }),
      snippet({ id: "used", last_used_at: "2026-01-01T00:00:00Z" }),
    ]);
    const { result } = renderHook(() => useSnippets());
    await waitFor(() => expect(result.current.status).toBe("ready"));
    expect(result.current.snippets.map((item) => item.id)).toEqual(["used", "never"]);
  });

  it("reorders optimistically before touch resolves", async () => {
    api.listSnippets.mockResolvedValue([
      snippet({ id: "old", last_used_at: "2026-01-01T00:00:00Z" }),
      snippet({ id: "touch-me" }),
    ]);
    const pending = deferred<SnippetDTO>();
    api.touchSnippet.mockReturnValue(pending.promise);
    const { result } = renderHook(() => useSnippets());
    await waitFor(() => expect(result.current.status).toBe("ready"));
    let touching!: Promise<void>;
    act(() => { touching = result.current.touch("touch-me"); });
    expect(result.current.snippets[0]!.id).toBe("touch-me");
    expect(result.current.snippets[0]!.use_count).toBe(1);
    pending.resolve(snippet({ id: "touch-me", use_count: 1, last_used_at: "2026-02-01T00:00:00Z" }));
    await act(() => touching);
  });

  it("rolls a rejected touch back without poisoning status", async () => {
    const original = snippet({ id: "probe", use_count: 3, last_used_at: "2026-01-01T00:00:00Z" });
    api.listSnippets.mockResolvedValue([original]);
    api.touchSnippet.mockRejectedValue(new Error("offline"));
    const { result } = renderHook(() => useSnippets());
    await waitFor(() => expect(result.current.status).toBe("ready"));
    await act(() => result.current.touch("probe"));
    expect(result.current.snippets[0]).toEqual(original);
    expect(result.current.status).toBe("ready");
  });

  it("shares one initial fetch across consumers", async () => {
    const pending = deferred<SnippetDTO[]>();
    api.listSnippets.mockReturnValue(pending.promise);
    const first = renderHook(() => useSnippets());
    const second = renderHook(() => useSnippets());
    expect(api.listSnippets).toHaveBeenCalledTimes(1);
    pending.resolve([]);
    await waitFor(() => expect(first.result.current.status).toBe("ready"));
    expect(second.result.current.status).toBe("ready");
  });
});
