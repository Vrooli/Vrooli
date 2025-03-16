/* eslint-disable @typescript-eslint/ban-ts-comment */
import { renderHook } from "@testing-library/react";
import { expect } from "chai";
import { act } from "react";
import { usePopover } from "./usePopover.js";

describe("usePopover", () => {
    it("initial state should be null", () => {
        const { result } = renderHook(() => usePopover());
        const [anchorEl] = result.current;

        expect(anchorEl).to.be.null;
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
        expect(anchorEl).not.to.be.null;
    });

    it("does not open the popover when the condition is false", () => {
        const { result } = renderHook(() => usePopover());
        const [, openPopover] = result.current;

        act(() => {
            // @ts-ignore: Testing runtime scenario
            openPopover({ currentTarget: document.createElement("button") }, false);
        });

        const [anchorEl] = result.current;
        expect(anchorEl).to.be.null;
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
        expect(anchorEl).to.be.null;
    });

    it("isPopoverOpen returns the correct state", () => {
        const { result } = renderHook(() => usePopover());
        const [, openPopover, , isPopoverOpen] = result.current;

        expect(isPopoverOpen).to.not.be.ok;

        act(() => {
            // @ts-ignore: Testing runtime scenario
            openPopover({ currentTarget: document.createElement("button") });
        });

        expect(result.current[3]).to.be.ok;

        act(() => {
            result.current[2](); // closePopover
        });

        expect(result.current[3]).to.not.be.ok;
    });
});
