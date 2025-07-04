// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-19
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ReportsLink } from "./ReportsLink.js";

// All react-i18next, MUI styles, and Icons mocks are now centralized in setup.vitest.ts

// Mock dependencies that aren't centralized yet
vi.mock("../Tooltip/Tooltip.js", () => ({
    Tooltip: ({ title, children }: any) => (
        <div data-testid="tooltip" title={title}>
            {children}
        </div>
    ),
}));

vi.mock("@mui/material/Typography", () => ({
    default: ({ variant, children, ...props }: any) => (
        <span data-testid="typography" data-variant={variant} {...props}>
            {children}
        </span>
    ),
}));

vi.mock("@vrooli/shared", () => ({
    getObjectSlug: vi.fn((object: any) => object?.name?.toLowerCase().replace(/\s+/g, "-") || "object-slug"),
    getObjectUrlBase: vi.fn((object: any) => `/${object?.__typename?.toLowerCase() || "object"}`),
}));

vi.mock("./IconButton.js", () => ({
    IconButton: ({ onClick, size, variant, children, ...props }: any) => (
        <button
            data-testid="icon-button"
            onClick={onClick}
            data-size={size}
            data-variant={variant}
            {...props}
        >
            {children}
        </button>
    ),
}));

const mockSetLocation = vi.fn();
vi.mock("../../route/router.js", () => ({
    useLocation: vi.fn(() => ["/current-location", mockSetLocation]),
}));

describe("ReportsLink", () => {
    const mockObject = {
        __typename: "Project" as const,
        id: "test-project-id",
        name: "Test Project",
        reportsCount: 5,
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should render reports link with correct count", () => {
        render(<ReportsLink object={mockObject} />);

        const iconButton = screen.getByTestId("icon-button");
        expect(iconButton).toBeTruthy();
        expect(iconButton.getAttribute("data-size")).toBe("md");
        expect(iconButton.getAttribute("data-variant")).toBe("transparent");
        expect(iconButton.getAttribute("aria-label")).toBe("Report_5"); // Using centralized i18next mock format

        const icon = screen.getByTestId("icon-common");
        expect(icon).toBeTruthy();
        expect(icon.getAttribute("data-icon-name")).toBe("Report");
        expect(icon.getAttribute("fill")).toBe("rgba(0, 0, 0, 0.87)"); // Using centralized theme mock color
        expect(icon.getAttribute("aria-hidden")).toBe("true");

        const countLabel = screen.getByTestId("typography");
        expect(countLabel).toBeTruthy();
        expect(countLabel.getAttribute("data-variant")).toBe("body1");
        expect(countLabel.textContent).toBe("(5)");
    });

    it("should render tooltip with correct title", () => {
        render(<ReportsLink object={mockObject} />);

        const tooltip = screen.getByTestId("tooltip");
        expect(tooltip).toBeTruthy();
        expect(tooltip.getAttribute("title")).toBe("Press to view and repond to reports.");
    });

    it("should navigate when clicked", () => {
        render(<ReportsLink object={mockObject} />);

        const iconButton = screen.getByTestId("icon-button");
        
        fireEvent.click(iconButton);

        expect(mockSetLocation).toHaveBeenCalledWith("/reports/project/test-project");
    });

    it("should not render when object is null", () => {
        const { container } = render(<ReportsLink object={null} />);
        expect(container.firstChild).toBeNull();
    });

    it("should not render when object is undefined", () => {
        const { container } = render(<ReportsLink object={undefined} />);
        expect(container.firstChild).toBeNull();
    });

    it("should not render when reportsCount is 0", () => {
        const objectWithZeroReports = {
            ...mockObject,
            reportsCount: 0,
        };

        const { container } = render(<ReportsLink object={objectWithZeroReports} />);
        expect(container.firstChild).toBeNull();
    });

    it("should not render when reportsCount is negative", () => {
        const objectWithNegativeReports = {
            ...mockObject,
            reportsCount: -1,
        };

        const { container } = render(<ReportsLink object={objectWithNegativeReports} />);
        expect(container.firstChild).toBeNull();
    });

    it("should not render when reportsCount is undefined", () => {
        const objectWithoutReportsCount = {
            ...mockObject,
            reportsCount: undefined,
        };

        const { container } = render(<ReportsLink object={objectWithoutReportsCount} />);
        expect(container.firstChild).toBeNull();
    });

    it("should handle different object types correctly", () => {
        const apiObject = {
            __typename: "Api" as const,
            id: "api-id",
            name: "Test API",
            reportsCount: 3,
        };

        render(<ReportsLink object={apiObject} />);

        const iconButton = screen.getByTestId("icon-button");
        fireEvent.click(iconButton);

        expect(mockSetLocation).toHaveBeenCalledWith("/reports/api/test-api");
    });

    it("should handle objects with special characters in name", () => {
        const objectWithSpecialName = {
            ...mockObject,
            name: "My Special Project!",
            reportsCount: 2,
        };

        render(<ReportsLink object={objectWithSpecialName} />);

        const iconButton = screen.getByTestId("icon-button");
        fireEvent.click(iconButton);

        expect(mockSetLocation).toHaveBeenCalledWith("/reports/project/my-special-project!");
    });

    it("should display correct count for different report numbers", () => {
        const testCases = [1, 10, 99, 100];

        testCases.forEach(count => {
            const objectWithCount = {
                ...mockObject,
                reportsCount: count,
            };

            const { rerender } = render(<ReportsLink object={objectWithCount} />);

            const countLabel = screen.getByTestId("typography");
            expect(countLabel.textContent).toBe(`(${count})`);

            const iconButton = screen.getByTestId("icon-button");
            expect(iconButton.getAttribute("aria-label")).toBe(`Report_${count}`); // Using centralized i18next mock format

            rerender(<div />); // Clear for next iteration
        });
    });

    it("should handle empty object name gracefully", () => {
        const objectWithEmptyName = {
            ...mockObject,
            name: "",
            reportsCount: 1,
        };

        render(<ReportsLink object={objectWithEmptyName} />);

        const iconButton = screen.getByTestId("icon-button");
        fireEvent.click(iconButton);

        // Should still navigate with fallback slug
        expect(mockSetLocation).toHaveBeenCalledWith("/reports/project/object-slug");
    });
});
