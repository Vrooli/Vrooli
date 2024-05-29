import { stringifySearchParams } from "@local/shared";
import { openLink } from "./openLink";

describe("openLink", () => {
    let setLocationMock;
    let originalStringifySearchParams;

    beforeEach(() => {
        setLocationMock = jest.fn();
        global.window.open = jest.fn();
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
        Object.assign(stringifySearchParams, jest.fn());
    });

    afterEach(() => {
        // Restore the original stringifySearchParams function
        Object.assign(stringifySearchParams, originalStringifySearchParams);

        jest.clearAllMocks();
    });

    test("should open external link in a new tab without search params", () => {
        const link = "https://example.com";

        openLink(setLocationMock, link);

        expect(window.open).toHaveBeenCalledWith(link, "_blank", "noopener,noreferrer");
        expect(setLocationMock).not.toHaveBeenCalled();
    });

    test("should open external link in a new tab with search params", () => {
        const link = "https://example.com";
        const searchParams = { foo: "bar" };
        const linkWithParams = `${link}?foo=%22bar%22`;

        openLink(setLocationMock, link, searchParams);

        expect(window.open).toHaveBeenCalledWith(linkWithParams, "_blank", "noopener,noreferrer");
        expect(setLocationMock).not.toHaveBeenCalled();
    });

    test("should push to history for internal link without search params", () => {
        const link = "/internal-page";

        openLink(setLocationMock, link);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams: undefined });
    });

    test("should push to history for internal link with search params", () => {
        const link = "/internal-page";
        const searchParams = { foo: "bar" };

        openLink(setLocationMock, link, searchParams);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams });
    });

    test("should handle internal link with origin in link", () => {
        const link = `${window.location.origin}/internal-page`;

        openLink(setLocationMock, link);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams: undefined });
    });

    test("should handle internal link with origin in link and search params", () => {
        const link = `${window.location.origin}/internal-page`;
        const searchParams = { foo: "bar" };

        openLink(setLocationMock, link, searchParams);

        expect(window.open).not.toHaveBeenCalled();
        expect(setLocationMock).toHaveBeenCalledWith(link, { searchParams });
    });
});
