import { createRef } from "react";
import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import { useVirtualList } from "../hooks/useVirtualList";
describe("useVirtualList", () => {
    it("keeps a 2500-item list bounded before its viewport is measured", () => {
        const scrollElementRef = createRef();
        const { result } = renderHook(() => useVirtualList({
            count: 2500,
            estimateSize: () => 120,
            overscan: 8,
            scrollElementRef,
        }));
        expect(result.current.viewportHeight).toBe(0);
        expect(result.current.virtualItems.length).toBeLessThanOrEqual(16);
    });
    it("keeps the small-list path unvirtualized", () => {
        const scrollElementRef = createRef();
        const { result } = renderHook(() => useVirtualList({
            count: 12,
            estimateSize: () => 120,
            scrollElementRef,
            enabled: false,
        }));
        expect(result.current.virtualItems).toHaveLength(12);
    });
});
