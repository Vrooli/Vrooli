import { screen } from "@testing-library/react";
import { expect } from "chai";
import React from "react";
import sinon from "sinon";
import { render } from "../__test/testUtils.js";
import { IconFavicon } from "./Icons.js";

describe("<IconFavicon />", () => {
    let consoleErrorStub: sinon.SinonStub;

    beforeEach(() => {
        // Stub console.error to prevent error messages from cluttering test output
        consoleErrorStub = sinon.stub(console, "error");
    });

    afterEach(() => {
        // Restore console.error after each test
        consoleErrorStub.restore();
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
            expect(icon).to.exist;
            expect(icon.tagName.toLowerCase()).to.equal("svg");

            const useElement = icon.querySelector("use");
            expect(useElement).to.exist;
            expect(useElement?.getAttribute("href")).to.equal(
                `http://www.google.com/s2/favicons?domain=${new URL(url).hostname}`,
            );
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
            expect(icon).to.exist;
            expect(icon.tagName.toLowerCase()).to.equal("svg");

            const useElement = icon.querySelector("use");
            expect(useElement).to.exist;
            expect(useElement?.getAttribute("href")).to.contain("#Website");

            // Verify error was logged
            expect(consoleErrorStub.calledOnce).to.be.true;
            expect(consoleErrorStub.firstCall.args[0]).to.include("[IconFavicon] Invalid URL");
        });
    });

    // Test custom props
    it("passes through custom props to svg element", () => {
        const testId = "favicon-icon";
        const customSize = 32;
        const customFill = "#FF0000";
        const customClass = "custom-class";

        render(
            <IconFavicon
                href="https://example.com"
                size={customSize}
                fill={customFill}
                className={customClass}
                data-testid={testId}
            />,
        );

        const icon = screen.getByTestId(testId);
        expect(icon).to.exist;
        expect(icon.getAttribute("width")).to.equal(customSize.toString());
        expect(icon.getAttribute("height")).to.equal(customSize.toString());
        expect(icon.getAttribute("fill")).to.equal(customFill);
        expect(icon.classList.contains(customClass)).to.be.true;
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
        expect(icon).to.exist;
        expect(icon.getAttribute("aria-label")).to.equal("Website Icon");
        expect(icon.getAttribute("aria-hidden")).to.be.null;
    });
}); 
