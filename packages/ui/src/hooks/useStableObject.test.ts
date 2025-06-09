import { renderHook } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useStableObject } from "./useStableObject.js";

describe("useStableObject", () => {
    it("should maintain the same reference for unchanged objects", () => {
        const initialObject = { a: 1, b: { c: 2 } };
        const { result, rerender } = renderHook(({ obj }) => useStableObject(obj), {
            initialProps: { obj: initialObject },
        });

        const firstRef = result.current;
        rerender({ obj: { a: 1, b: { c: 2 } } }); // Same value, different reference

        expect(result.current).toBe(firstRef);
    });

    it("should update the reference for changed objects", () => {
        const initialObject = { a: 1, b: { c: 2 } };
        const { result, rerender } = renderHook(({ obj }) => useStableObject(obj), {
            initialProps: { obj: initialObject },
        });

        const firstRef = result.current;
        rerender({ obj: { a: 1, b: { c: 3 } } }); // Different value

        expect(result.current).not.toBe(firstRef);
    });

    it("should handle primitives correctly", () => {
        const initialVal = 5;
        const { result, rerender } = renderHook(({ val }) => useStableObject(val), {
            initialProps: { val: initialVal },
        });

        const firstRef = result.current;
        rerender({ val: 5 }); // Same value

        expect(result.current).toBe(firstRef);

        rerender({ val: 6 }); // Different value

        expect(result.current).not.toBe(firstRef);
    });

    it("should update the reference for arrays with changed content", () => {
        const initialArray = [1, 2, { a: 3 }];
        const { result, rerender } = renderHook(({ arr }) => useStableObject(arr), {
            initialProps: { arr: initialArray },
        });

        const firstRef = result.current;
        rerender({ arr: [1, 2, { a: 4 }] }); // Array with different content

        expect(result.current).not.toBe(firstRef);
    });

    it("should maintain the same reference for unchanged nested objects", () => {
        const initialObject = { a: 1, b: { c: 2, d: { e: 3 } } };
        const { result, rerender } = renderHook(({ obj }) => useStableObject(obj), {
            initialProps: { obj: initialObject },
        });

        const firstRef = result.current;
        rerender({ obj: { a: 1, b: { c: 2, d: { e: 3 } } } }); // Same nested object

        expect(result.current).toBe(firstRef);
    });

    it("should handle null and undefined correctly", () => {
        const { result, rerender } = renderHook(({ obj }) => useStableObject(obj), {
            initialProps: { obj: null as null | undefined },
        });

        const firstRef = result.current;
        rerender({ obj: null }); // Still null

        expect(result.current).toBe(firstRef);

        rerender({ obj: undefined }); // Change to undefined

        expect(result.current).not.toBe(firstRef);

        const secondRef = result.current;
        rerender({ obj: undefined }); // Still undefined

        expect(result.current).toBe(secondRef);
    });
});
