import { renderHook, act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useCodeCopy } from "@vrooli/react-component-library/markdown-renderer/0.4.2";

describe("useCodeCopy", () => {
  // Restore the whole `navigator` object, not just mocks: replacing it with a
  // clipboard-only stub would otherwise leak into every later file in the
  // single-threaded test run.
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("reports a copied state after a successful clipboard write", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    const { result } = renderHook(() => useCodeCopy());
    await act(() => result.current.copy("story source"));
    await waitFor(() => expect(result.current.copied).toBe(true));
    expect(writeText).toHaveBeenCalledWith("story source");
  });

  it("leaves copied false when the clipboard rejects", async () => {
    vi.stubGlobal("navigator", {
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) },
    });
    const { result } = renderHook(() => useCodeCopy());
    await expect(result.current.copy("story source")).resolves.toBe(false);
    expect(result.current.copied).toBe(false);
  });
});
