import { fireEvent, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useGlobalKeydown } from "./useGlobalKeydown";

describe("useGlobalKeydown", () => {
  it("uses the latest handler without changing the global target", () => {
    const firstHandler = vi.fn();
    const latestHandler = vi.fn();
    const { rerender } = renderHook(({ handler }) => useGlobalKeydown(handler), {
      initialProps: { handler: firstHandler },
    });

    rerender({ handler: latestHandler });
    fireEvent.keyDown(window, { key: "k" });

    expect(firstHandler).not.toHaveBeenCalled();
    expect(latestHandler).toHaveBeenCalledOnce();
  });

  it("supports document listeners and disables them when requested", () => {
    const handler = vi.fn();
    const { rerender } = renderHook(
      ({ enabled }) => useGlobalKeydown(handler, { target: "document", enabled }),
      { initialProps: { enabled: true } },
    );

    fireEvent.keyDown(document, { key: "Escape" });
    rerender({ enabled: false });
    fireEvent.keyDown(document, { key: "Escape" });

    expect(handler).toHaveBeenCalledOnce();
  });
});
