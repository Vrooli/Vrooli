import { screen } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import React from "react";
import { render } from "../__test/testUtils.js";
import { Icon, IconCommon, IconFavicon, IconRoutine, IconService, IconText } from "./Icons.js";

describe("<IconCommon />", () => {
    it("renders with correct sprite reference", () => {
        render(<IconCommon name="User" data-testid="icon-common" />);
        
        const icon = screen.getByTestId("icon-common");
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("svg");
        expect(icon.getAttribute("data-icon-type")).toBe("common");
        expect(icon.getAttribute("data-icon-name")).toBe("User");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#User");
        expect(useElement?.getAttribute("href")).toContain("common-sprite.svg");
    });

    it("passes through props to IconBase", () => {
        render(
            <IconCommon 
                name="User" 
                size={48} 
                fill="blue" 
                decorative="false"
                aria-label="User Icon"
                data-testid="icon-common" 
            />,
        );
        
        const icon = screen.getByTestId("icon-common");
        expect(icon.getAttribute("width")).toBe("48");
        expect(icon.getAttribute("height")).toBe("48");
        expect(icon.getAttribute("fill")).toBe("blue");
        expect(icon.getAttribute("aria-hidden")).toBeNull();
        expect(icon.getAttribute("aria-label")).toBe("User Icon");
    });

    it("forwards ref correctly", () => {
        const ref = React.createRef<SVGSVGElement>();
        render(<IconCommon name="User" ref={ref} data-testid="icon-common" />);
        
        expect(ref.current).toBeDefined();
        expect(ref.current?.tagName.toLowerCase()).toBe("svg");
    });
});

describe("<IconRoutine />", () => {
    it("renders with correct sprite reference", () => {
        render(<IconRoutine name="Routine" data-testid="icon-routine" />);
        
        const icon = screen.getByTestId("icon-routine");
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("svg");
        expect(icon.getAttribute("data-icon-type")).toBe("routine");
        expect(icon.getAttribute("data-icon-name")).toBe("Routine");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#Routine");
        expect(useElement?.getAttribute("href")).toContain("routine-sprite.svg");
    });

    it("passes through props to IconBase", () => {
        render(
            <IconRoutine 
                name="Routine" 
                size={16} 
                fill="green" 
                data-testid="icon-routine" 
            />,
        );
        
        const icon = screen.getByTestId("icon-routine");
        expect(icon.getAttribute("width")).toBe("16");
        expect(icon.getAttribute("height")).toBe("16");
        expect(icon.getAttribute("fill")).toBe("green");
    });

    it("forwards ref correctly", () => {
        const ref = React.createRef<SVGSVGElement>();
        render(<IconRoutine name="Routine" ref={ref} data-testid="icon-routine" />);
        
        expect(ref.current).toBeDefined();
        expect(ref.current?.tagName.toLowerCase()).toBe("svg");
    });
});

describe("<IconService />", () => {
    it("renders with correct sprite reference", () => {
        render(<IconService name="GitHub" data-testid="icon-service" />);
        
        const icon = screen.getByTestId("icon-service");
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("svg");
        expect(icon.getAttribute("data-icon-type")).toBe("service");
        expect(icon.getAttribute("data-icon-name")).toBe("GitHub");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#GitHub");
        expect(useElement?.getAttribute("href")).toContain("service-sprite.svg");
    });

    it("passes through props to IconBase", () => {
        render(
            <IconService 
                name="GitHub" 
                size={20} 
                fill="purple" 
                data-testid="icon-service" 
            />,
        );
        
        const icon = screen.getByTestId("icon-service");
        expect(icon.getAttribute("width")).toBe("20");
        expect(icon.getAttribute("height")).toBe("20");
        expect(icon.getAttribute("fill")).toBe("purple");
    });

    it("forwards ref correctly", () => {
        const ref = React.createRef<SVGSVGElement>();
        render(<IconService name="GitHub" ref={ref} data-testid="icon-service" />);
        
        expect(ref.current).toBeDefined();
        expect(ref.current?.tagName.toLowerCase()).toBe("svg");
    });
});

describe("<IconText />", () => {
    it("renders with correct sprite reference", () => {
        render(<IconText name="Quote" data-testid="icon-text" />);
        
        const icon = screen.getByTestId("icon-text");
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("svg");
        expect(icon.getAttribute("data-icon-type")).toBe("text");
        expect(icon.getAttribute("data-icon-name")).toBe("Quote");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#Quote");
        expect(useElement?.getAttribute("href")).toContain("text-sprite.svg");
    });

    it("passes through props to IconBase", () => {
        render(
            <IconText 
                name="Quote" 
                size={14} 
                fill="orange" 
                data-testid="icon-text" 
            />,
        );
        
        const icon = screen.getByTestId("icon-text");
        expect(icon.getAttribute("width")).toBe("14");
        expect(icon.getAttribute("height")).toBe("14");
        expect(icon.getAttribute("fill")).toBe("orange");
    });

    it("forwards ref correctly", () => {
        const ref = React.createRef<SVGSVGElement>();
        render(<IconText name="Quote" ref={ref} data-testid="icon-text" />);
        
        expect(ref.current).toBeDefined();
        expect(ref.current?.tagName.toLowerCase()).toBe("svg");
    });
});

describe("<Icon />", () => {
    beforeEach(() => {
        vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("renders Common icon correctly", () => {
        render(<Icon info={{ name: "User", type: "Common" }} data-testid="icon" />);
        
        const icon = screen.getByTestId("icon");
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("svg");
        expect(icon.getAttribute("data-icon-type")).toBe("common");
        expect(icon.getAttribute("data-icon-name")).toBe("User");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#User");
        expect(useElement?.getAttribute("href")).toContain("common-sprite.svg");
    });

    it("renders Routine icon correctly", () => {
        render(<Icon info={{ name: "Routine", type: "Routine" }} data-testid="icon" />);
        
        const icon = screen.getByTestId("icon");
        expect(icon).toBeDefined();
        expect(icon.getAttribute("data-icon-type")).toBe("routine");
        expect(icon.getAttribute("data-icon-name")).toBe("Routine");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#Routine");
        expect(useElement?.getAttribute("href")).toContain("routine-sprite.svg");
    });

    it("renders Service icon correctly", () => {
        render(<Icon info={{ name: "GitHub", type: "Service" }} data-testid="icon" />);
        
        const icon = screen.getByTestId("icon");
        expect(icon).toBeDefined();
        expect(icon.getAttribute("data-icon-type")).toBe("service");
        expect(icon.getAttribute("data-icon-name")).toBe("GitHub");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#GitHub");
        expect(useElement?.getAttribute("href")).toContain("service-sprite.svg");
    });

    it("renders Text icon correctly", () => {
        render(<Icon info={{ name: "Quote", type: "Text" }} data-testid="icon" />);
        
        const icon = screen.getByTestId("icon");
        expect(icon).toBeDefined();
        expect(icon.getAttribute("data-icon-type")).toBe("text");
        expect(icon.getAttribute("data-icon-name")).toBe("Quote");
        
        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#Quote");
        expect(useElement?.getAttribute("href")).toContain("text-sprite.svg");
    });

    it("handles invalid icon info gracefully", () => {
        // Test with null
        const { container: container1 } = render(<Icon info={null as any} data-testid="icon" />);
        // The Icon component returns null, but it's wrapped in theme providers
        // Check that no SVG element was rendered
        expect(container1.querySelector("svg")).toBeNull();
        expect(console.error).toHaveBeenCalledWith("Invalid icon info", null, "Stack:", expect.stringContaining("Error"));

        // Test with undefined
        const { container: container2 } = render(<Icon info={undefined as any} data-testid="icon" />);
        expect(container2.querySelector("svg")).toBeNull();

        // Test with invalid object
        const { container: container3 } = render(<Icon info={{ invalid: "object" } as any} data-testid="icon" />);
        expect(container3.querySelector("svg")).toBeNull();
        expect(console.error).toHaveBeenCalledWith("Invalid icon info", { invalid: "object" }, "Stack:", expect.stringContaining("Error"));
    });

    it("passes through props to IconBase", () => {
        render(
            <Icon 
                info={{ name: "User", type: "Common" }}
                size={36} 
                fill="pink" 
                decorative="false"
                aria-label="User Profile"
                data-testid="icon" 
            />,
        );
        
        const icon = screen.getByTestId("icon");
        expect(icon.getAttribute("width")).toBe("36");
        expect(icon.getAttribute("height")).toBe("36");
        expect(icon.getAttribute("fill")).toBe("pink");
        expect(icon.getAttribute("aria-hidden")).toBeNull();
        expect(icon.getAttribute("aria-label")).toBe("User Profile");
    });

    it("forwards ref correctly", () => {
        const ref = React.createRef<SVGSVGElement>();
        render(<Icon info={{ name: "User", type: "Common" }} ref={ref} data-testid="icon" />);
        
        expect(ref.current).toBeDefined();
        expect(ref.current?.tagName.toLowerCase()).toBe("svg");
    });
});

describe("Icon Components - Additional Behavioral Tests", () => {
    beforeEach(() => {
        vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Theme integration", () => {
        it("accepts theme color values through fill prop", () => {
            render(<IconCommon name="User" fill="primary.main" data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            // The resolveThemeColor utility should handle theme colors
            expect(icon.getAttribute("fill")).toBeDefined();
        });

        it("defaults to currentColor when no fill specified", () => {
            render(<IconCommon name="User" data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("fill")).toBe("currentColor");
        });
    });

    describe("Accessibility best practices", () => {
        it("is decorative by default", () => {
            render(<IconCommon name="User" data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("aria-hidden")).toBe("true");
            expect(icon.getAttribute("aria-label")).toBeNull();
        });

        it("becomes accessible when decorative=false", () => {
            render(
                <IconCommon 
                    name="User" 
                    decorative="false" 
                    aria-label="User profile"
                    data-testid="icon" 
                />,
            );
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("aria-hidden")).toBeNull();
            expect(icon.getAttribute("aria-label")).toBe("User profile");
        });

        it("handles mixed accessibility props correctly", () => {
            render(
                <IconCommon 
                    name="User" 
                    decorative="true"
                    aria-hidden={false}
                    aria-label="This should be ignored"
                    data-testid="icon" 
                />,
            );
            
            const icon = screen.getByTestId("icon");
            // When aria-hidden is explicitly false, it should not be hidden
            expect(icon.getAttribute("aria-hidden")).toBeNull();
            // But aria-label should still be null because decorative=true
            expect(icon.getAttribute("aria-label")).toBeNull();
        });
    });

    describe("Size handling", () => {
        it("handles null size gracefully", () => {
            render(<IconCommon name="User" size={null} data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("width")).toBe("24");
            expect(icon.getAttribute("height")).toBe("24");
        });

        it("handles undefined size gracefully", () => {
            render(<IconCommon name="User" size={undefined} data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("width")).toBe("24");
            expect(icon.getAttribute("height")).toBe("24");
        });

        it("accepts size=0", () => {
            render(<IconCommon name="User" size={0} data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("width")).toBe("0");
            expect(icon.getAttribute("height")).toBe("0");
        });
    });

    describe("Fill color handling", () => {
        it("handles null fill gracefully", () => {
            render(<IconCommon name="User" fill={null} data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("fill")).toBe("currentColor");
        });

        it("handles undefined fill gracefully", () => {
            render(<IconCommon name="User" fill={undefined} data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("fill")).toBe("currentColor");
        });

        it("accepts transparent as fill", () => {
            render(<IconCommon name="User" fill="transparent" data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            expect(icon.getAttribute("fill")).toBe("transparent");
        });
    });

    describe("Sprite URL handling", () => {
        it("uses default sprite URLs when global not available", () => {
            render(<IconCommon name="User" data-testid="icon" />);
            
            const icon = screen.getByTestId("icon");
            const useElement = icon.querySelector("use");
            expect(useElement?.getAttribute("href")).toContain("common-sprite.svg#User");
        });
    });

    describe("Component display names", () => {
        it("has correct display names for debugging", () => {
            expect(IconCommon.displayName).toBe("IconCommon");
            expect(IconRoutine.displayName).toBe("IconRoutine");
            expect(IconService.displayName).toBe("IconService");
            expect(IconText.displayName).toBe("IconText");
            expect(Icon.displayName).toBe("Icon");
            expect(IconFavicon.displayName).toBe("IconFavicon");
        });
    });

    describe("Data attributes for testing", () => {
        it("adds consistent data attributes across all icon components", () => {
            // Test that all icon components have the expected data attributes
            render(
                <div>
                    <IconCommon name="User" data-testid="common-icon" />
                    <IconRoutine name="Routine" data-testid="routine-icon" />
                    <IconService name="GitHub" data-testid="service-icon" />
                    <IconText name="Quote" data-testid="text-icon" />
                    <Icon info={{ name: "User", type: "Common" }} data-testid="unified-icon" />
                </div>,
            );

            const commonIcon = screen.getByTestId("common-icon");
            expect(commonIcon.getAttribute("data-icon-type")).toBe("common");
            expect(commonIcon.getAttribute("data-icon-name")).toBe("User");

            const routineIcon = screen.getByTestId("routine-icon");
            expect(routineIcon.getAttribute("data-icon-type")).toBe("routine");
            expect(routineIcon.getAttribute("data-icon-name")).toBe("Routine");

            const serviceIcon = screen.getByTestId("service-icon");
            expect(serviceIcon.getAttribute("data-icon-type")).toBe("service");
            expect(serviceIcon.getAttribute("data-icon-name")).toBe("GitHub");

            const textIcon = screen.getByTestId("text-icon");
            expect(textIcon.getAttribute("data-icon-type")).toBe("text");
            expect(textIcon.getAttribute("data-icon-name")).toBe("Quote");

            const unifiedIcon = screen.getByTestId("unified-icon");
            expect(unifiedIcon.getAttribute("data-icon-type")).toBe("common");
            expect(unifiedIcon.getAttribute("data-icon-name")).toBe("User");
        });

        it("can be selected using data attributes", () => {
            render(
                <div>
                    <IconCommon name="User" />
                    <IconService name="GitHub" />
                    <IconText name="Quote" />
                </div>,
            );

            // Test that we can select icons by their data attributes
            const userIcon = document.querySelector("[data-icon-type=\"common\"][data-icon-name=\"User\"]");
            expect(userIcon).toBeDefined();
            expect(userIcon?.tagName.toLowerCase()).toBe("svg");

            const githubIcon = document.querySelector("[data-icon-type=\"service\"][data-icon-name=\"GitHub\"]");
            expect(githubIcon).toBeDefined();
            expect(githubIcon?.tagName.toLowerCase()).toBe("svg");

            const quoteIcon = document.querySelector("[data-icon-type=\"text\"][data-icon-name=\"Quote\"]");
            expect(quoteIcon).toBeDefined();
            expect(quoteIcon?.tagName.toLowerCase()).toBe("svg");
        });
    });
});

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

    it.each(validUrls)("renders favicon for valid URL: %s", (url) => {
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

    // Test cases for invalid URLs
    const invalidUrls = [
        ["not-a-url", true],
        ["mailto:user@example.com", false],
        ["tel:+1234567890", false],
        ["", true],
    ] as const;

    it.each(invalidUrls)("falls back to default icon for invalid URL: %s", (url, shouldError) => {
        const testId = "favicon-icon";
        render(<IconFavicon href={url} data-testid={testId} />);

        const icon = screen.getByTestId(testId);
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("svg");

        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#Website");

        // Only expect console.error for URLs that actually throw when parsing
        if (shouldError) {
            expect(console.error).toHaveBeenCalled();
            expect(vi.mocked(console.error).mock.calls[0][0]).toContain("[IconFavicon] Invalid URL");
        }
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
        expect(icon.getAttribute("data-icon-type")).toBe("favicon");
        expect(icon.getAttribute("data-favicon-domain")).toBe("example.com");
    });

    // Test accessibility attributes
    it("handles accessibility attributes correctly", () => {
        const testId = "favicon-icon";

        render(
            <IconFavicon
                href="https://example.com"
                decorative="false"
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

    it("handles custom fallback icon", () => {
        const testId = "favicon-icon";
        const fallbackIcon = { name: "User", type: "Common" } as const;

        render(
            <IconFavicon
                href="invalid-url"
                fallbackIcon={fallbackIcon}
                data-testid={testId}
            />,
        );

        const icon = screen.getByTestId(testId);
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("svg");
        expect(icon.getAttribute("data-icon-type")).toBe("favicon-fallback");

        const useElement = icon.querySelector("use");
        expect(useElement).toBeDefined();
        expect(useElement?.getAttribute("href")).toContain("#User");
    });

    it("handles error fallback chain", () => {
        const testId = "favicon-icon";
        
        render(
            <IconFavicon
                href="https://nonexistent-domain.example"
                data-testid={testId}
            />,
        );

        const icon = screen.getByTestId(testId);
        expect(icon).toBeDefined();
        expect(icon.tagName.toLowerCase()).toBe("img");
        
        // Should start with the first favicon source
        const imgElement = icon as HTMLImageElement;
        expect(imgElement.src).toContain("nonexistent-domain.example");
    });

    it("forwards ref correctly for valid URLs", () => {
        const ref = React.createRef<HTMLImageElement>();
        render(
            <IconFavicon 
                href="https://example.com" 
                ref={ref as React.Ref<HTMLElement>}
                data-testid="favicon-icon" 
            />,
        );
        
        expect(ref.current).toBeDefined();
        expect(ref.current?.tagName.toLowerCase()).toBe("img");
    });

    it("forwards ref correctly for invalid URLs (fallback SVG)", () => {
        const ref = React.createRef<SVGSVGElement>();
        render(
            <IconFavicon 
                href="invalid-url" 
                ref={ref as React.Ref<HTMLElement>}
                data-testid="favicon-icon" 
            />,
        );
        
        expect(ref.current).toBeDefined();
        expect(ref.current?.tagName.toLowerCase()).toBe("svg");
    });

    it("handles edge case URL protocols", () => {
        const invalidProtocols = [
            "data:text/plain,hello",
            "javascript:void(0)",
        ];

        invalidProtocols.forEach((url, index) => {
            const testId = `favicon-icon-${index}`;
            render(<IconFavicon href={url} data-testid={testId} />);
            
            const icon = screen.getByTestId(testId);
            expect(icon).toBeDefined();
            // These should fallback to SVG since they don't have valid hostnames for favicons
            expect(icon.tagName.toLowerCase()).toBe("svg");
        });
    });
}); 
