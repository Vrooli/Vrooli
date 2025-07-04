// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-19
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ShareButton } from "./ShareButton.js";

// All react-i18next, MUI styles, and Icons mocks are now centralized in setup.vitest.ts

// Mock dependencies that aren't centralized yet
vi.mock("../Tooltip/Tooltip.js", () => ({
    Tooltip: ({ title, children }: any) => (
        <div data-testid="tooltip" title={title}>
            {children}
        </div>
    ),
}));

vi.mock("./IconButton.js", () => ({
    IconButton: ({ onClick, size, variant, children }: any) => (
        <button
            data-testid="icon-button"
            onClick={onClick}
            data-size={size}
            data-variant={variant}
        >
            {children}
        </button>
    ),
}));

vi.mock("../dialogs/ShareObjectDialog/ShareObjectDialog.js", () => ({
    ShareObjectDialog: ({ object, open, onClose }: any) => (
        <div
            data-testid="share-dialog"
            data-open={open}
            data-object-id={object?.id || ""}
        >
            {open && (
                <button data-testid="close-dialog" onClick={onClose}>
                    Close
                </button>
            )}
        </div>
    ),
}));

describe("ShareButton", () => {
    const mockObject = {
        __typename: "Project" as const,
        id: "test-project-id",
        name: "Test Project",
    };

    it("should render share button with correct props", () => {
        render(<ShareButton object={mockObject} />);

        const iconButton = screen.getByTestId("icon-button");
        expect(iconButton).toBeTruthy();
        expect(iconButton.getAttribute("data-size")).toBe("sm");
        expect(iconButton.getAttribute("data-variant")).toBe("transparent");

        const icon = screen.getByTestId("icon-common");
        expect(icon).toBeTruthy();
        expect(icon.getAttribute("data-icon-name")).toBe("Share");
        expect(icon.getAttribute("fill")).toBe("rgba(0, 0, 0, 0.6)"); // Using centralized theme mock color
        expect(icon.getAttribute("aria-hidden")).toBe("true");
    });

    it("should render tooltip with correct title", () => {
        render(<ShareButton object={mockObject} />);

        const tooltip = screen.getByTestId("tooltip");
        expect(tooltip).toBeTruthy();
        expect(tooltip.getAttribute("title")).toBe("Share");
    });

    it("should render share dialog initially closed", () => {
        render(<ShareButton object={mockObject} />);

        const dialog = screen.getByTestId("share-dialog");
        expect(dialog).toBeTruthy();
        expect(dialog.getAttribute("data-open")).toBe("false");
        expect(dialog.getAttribute("data-object-id")).toBe("test-project-id");
    });

    it("should open dialog when share button is clicked", () => {
        render(<ShareButton object={mockObject} />);

        const iconButton = screen.getByTestId("icon-button");
        fireEvent.click(iconButton);

        const dialog = screen.getByTestId("share-dialog");
        expect(dialog.getAttribute("data-open")).toBe("true");

        const closeButton = screen.getByTestId("close-dialog");
        expect(closeButton).toBeTruthy();
    });

    it("should close dialog when close is triggered", () => {
        render(<ShareButton object={mockObject} />);

        // Open dialog first
        const iconButton = screen.getByTestId("icon-button");
        fireEvent.click(iconButton);

        // Verify dialog is open
        const dialog = screen.getByTestId("share-dialog");
        expect(dialog.getAttribute("data-open")).toBe("true");

        // Close dialog
        const closeButton = screen.getByTestId("close-dialog");
        fireEvent.click(closeButton);

        // Verify dialog is closed
        expect(dialog.getAttribute("data-open")).toBe("false");
        expect(screen.queryByTestId("close-dialog")).toBeNull();
    });

    it("should handle object prop correctly", () => {
        const customObject = {
            __typename: "Api" as const,
            id: "custom-api-id",
            name: "Custom API",
        };

        render(<ShareButton object={customObject} />);

        const dialog = screen.getByTestId("share-dialog");
        expect(dialog.getAttribute("data-object-id")).toBe("custom-api-id");
    });

    it("should handle null object", () => {
        render(<ShareButton object={null} />);

        const dialog = screen.getByTestId("share-dialog");
        expect(dialog.getAttribute("data-object-id")).toBe("");
    });

    it("should handle undefined object", () => {
        render(<ShareButton object={undefined} />);

        const dialog = screen.getByTestId("share-dialog");
        expect(dialog.getAttribute("data-object-id")).toBe("");
    });

    it("should maintain dialog state correctly through multiple interactions", () => {
        render(<ShareButton object={mockObject} />);

        const iconButton = screen.getByTestId("icon-button");
        const dialog = screen.getByTestId("share-dialog");

        // Initially closed
        expect(dialog.getAttribute("data-open")).toBe("false");

        // Open
        fireEvent.click(iconButton);
        expect(dialog.getAttribute("data-open")).toBe("true");

        // Close
        const closeButton = screen.getByTestId("close-dialog");
        fireEvent.click(closeButton);
        expect(dialog.getAttribute("data-open")).toBe("false");

        // Open again
        fireEvent.click(iconButton);
        expect(dialog.getAttribute("data-open")).toBe("true");
    });
});
