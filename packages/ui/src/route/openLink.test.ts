// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { stringifySearchParams } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { openLink } from "./openLink.js";

describe("openLink", () => {
    let setLocationMock;
    let originalStringifySearchParams;

    beforeEach(() => {
        setLocationMock = vi.fn();
        global.window.open = vi.fn();
        // Mock window.location.origin
        Object.defineProperty(window, "location", {
            value: {
                origin: "http://localhost",
            },
            writable: true,
        });

        // Save the original stringifySearchParams function
        originalStringifySearchParams = stringifySearchParams;

        // Mock stringifySearchParams
        Object.assign(stringifySearchParams, vi.fn());
    });

    afterEach(() => {
        // Restore the original stringifySearchParams function
        Object.assign(stringifySearchParams, originalStringifySearchParams);

        vi.clearAllMocks();
    });

    it("should open external link in a new tab without search params", () => {
        const link = "https://example.com";

        openLink(setLocationMock, link);

        expect(window.open).toHaveBeenCalledWith(link, "_blank", "noopener,noreferrer");
        expect(setLocationMock).not.toHaveBeenCalled();
    });

    it("should open external link in a new tab with search params", () => {
        const link = "https://example.com";
        const searchParams = { foo: "bar" };
        const linkWithParams = `${link}?foo=%22bar%22`;

        openLink(setLocationMock, link, searchParams);

        expect(window.open).toHaveBeenCalledWith(linkWithParams, "_blank", "noopener,noreferrer");
        expect(setLocationMock).not.toHaveBeenCalled();
    });

    it("should push to history for internal link without search params", () => {
        const link = "/internal-page";

        openLink(setLocationMock, link);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams: undefined });
    });

    it("should push to history for internal link with search params", () => {
        const link = "/internal-page";
        const searchParams = { foo: "bar" };

        openLink(setLocationMock, link, searchParams);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams });
    });

    it("should handle internal link with origin in link", () => {
        const link = `${window.location.origin}/internal-page`;

        openLink(setLocationMock, link);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams: undefined });
    });

    it("should handle internal link with origin in link and search params", () => {
        const link = `${window.location.origin}/internal-page`;
        const searchParams = { foo: "bar" };

        openLink(setLocationMock, link, searchParams);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams });
    });
});
