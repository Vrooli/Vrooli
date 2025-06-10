import { screen } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import React from "react";
import { render } from "../__test/testUtils.js";
import { IconFavicon } from "./Icons.js";

describe("<IconFavicon />", () => {
    beforeEach(() => {
        // Mock console.error to prevent error messages from cluttering test output
        vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    // Test cases for valid URLs
    const validUrls = [
        "https://www.google.com",
        "http://github.com",
        "https://facebook.com/path/to/something",
        "https://subdomain.example.com:8080/path?query=value",
    ];

    validUrls.forEach(url => {
        it(`renders favicon for valid URL: ${url}`, () => {
            const testId = "favicon-icon";
            render(<IconFavicon href={url} data-testid={testId} />);

            const icon = screen.getByTestId(testId);
            expect(icon).toBeDefined();
            expect(icon.tagName.toLowerCase()).toBe("img");

            const imgElement = icon as HTMLImageElement;
            expect(imgElement.src).toBeDefined();
            // The first source should be the apple-touch-icon
            expect(imgElement.src).toContain(new URL(url).hostname);
        });
    });

    // Test cases for invalid URLs
    const invalidUrls = [
        "not-a-url",
        "mailto:user@example.com",
        "tel:+1234567890",
        "",
    ];

    invalidUrls.forEach(url => {
        it(`falls back to default icon for invalid URL: ${url}`, () => {
            const testId = "favicon-icon";
            render(<IconFavicon href={url} data-testid={testId} />);

            const icon = screen.getByTestId(testId);
            expect(icon).toBeDefined();
            expect(icon.tagName.toLowerCase()).toBe("svg");

            const useElement = icon.querySelector("use");
            expect(useElement).toBeDefined();
            expect(useElement?.getAttribute("href")).toContain("#Website");

            // Only expect console.error for URLs that actually throw when parsing
            if (url === "not-a-url" || url === "") {
                expect(console.error).toHaveBeenCalled();
                expect(vi.mocked(console.error).mock.calls[0][0]).toContain("[IconFavicon] Invalid URL");
            }
        });
    });

    // Test custom props
    it("passes through custom props to img element", () => {
        const testId = "favicon-icon";
        const customSize = 32;
        const customClass = "custom-class";

        render(
            <IconFavicon
                href="https://example.com"
                size={customSize}
                className={customClass}
                data-testid={testId}
            />,
        );

        const icon = screen.getByTestId(testId);
        expect(icon).toBeDefined();
        expect(icon.getAttribute("width")).toBe(customSize.toString());
        expect(icon.getAttribute("height")).toBe(customSize.toString());
        expect(icon.classList.contains(customClass)).toBe(true);
    });

    // Test accessibility attributes
    it("handles accessibility attributes correctly", () => {
        const testId = "favicon-icon";

        render(
            <IconFavicon
                href="https://example.com"
                decorative={false}
                aria-label="Website Icon"
                data-testid={testId}
            />,
        );

        const icon = screen.getByTestId(testId);
        expect(icon).toBeDefined();
        // For img elements, alt attribute is used instead of aria-label
        expect(icon.getAttribute("alt")).toBe("Website Icon");
        expect(icon.getAttribute("aria-hidden")).toBeNull();
    });
}); 
