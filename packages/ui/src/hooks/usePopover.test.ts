// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
/* eslint-disable @typescript-eslint/ban-ts-comment */
import { renderHook } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { act } from "react";
import { usePopover } from "./usePopover.js";

describe("usePopover", () => {
    it("initial state should be null", () => {
        const { result } = renderHook(() => usePopover());
        const [anchorEl] = result.current;

        expect(anchorEl).toBeNull();
    });

    it("opens the popover when the condition is true", () => {
        const { result } = renderHook(() => usePopover());
        const [, openPopover] = result.current;

        act(() => {
            // @ts-ignore: Testing runtime scenario
            openPopover({ currentTarget: document.createElement("button") });
        });

        const [anchorEl] = result.current;
        expect(anchorEl).toBeInstanceOf(HTMLElement);
        expect(anchorEl).not.toBeNull();
    });

    it("does not open the popover when the condition is false", () => {
        const { result } = renderHook(() => usePopover());
        const [, openPopover] = result.current;

        act(() => {
            // @ts-ignore: Testing runtime scenario
            openPopover({ currentTarget: document.createElement("button") }, false);
        });

        const [anchorEl] = result.current;
        expect(anchorEl).toBeNull();
    });

    it("closes the popover correctly", () => {
        const { result } = renderHook(() => usePopover());
        const [, openPopover, closePopover] = result.current;

        // Open the popover first
        act(() => {
            // @ts-ignore: Testing runtime scenario
            openPopover({ currentTarget: document.createElement("button") });
        });

        // Then close it
        act(() => {
            closePopover();
        });

        const [anchorEl] = result.current;
        expect(anchorEl).toBeNull();
    });

    it("isPopoverOpen returns the correct state", () => {
        const { result } = renderHook(() => usePopover());
        const [, openPopover, , isPopoverOpen] = result.current;

        expect(isPopoverOpen).toBe(false);

        act(() => {
            // @ts-ignore: Testing runtime scenario
            openPopover({ currentTarget: document.createElement("button") });
        });

        expect(result.current[3]).toBe(true);

        act(() => {
            result.current[2](); // closePopover
        });

        expect(result.current[3]).toBe(false);
    });
});
