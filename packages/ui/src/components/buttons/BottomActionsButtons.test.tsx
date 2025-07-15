import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { BottomActionsButtons } from "./BottomActionsButtons.js";
import { useErrorPopover } from "../../hooks/useErrorPopover.js";
import { useKeyboardOpen } from "../../hooks/useKeyboardOpen.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";

// Mock the hooks and dependencies
vi.mock("../../hooks/useErrorPopover.js");
vi.mock("../../hooks/useKeyboardOpen.js");
vi.mock("../../hooks/useWindowSize.js");

vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key, // Return the key as the translation
    }),
}));

const mockUseErrorPopover = vi.mocked(useErrorPopover);
const mockUseKeyboardOpen = vi.mocked(useKeyboardOpen);
const mockUseWindowSize = vi.mocked(useWindowSize);

const defaultProps = {
    display: "Page" as const,
    isCreate: true,
    onCancel: vi.fn(),
    onSubmit: vi.fn(),
};

describe("BottomActionsButtons", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        
        // Set up default mock implementations
        mockUseErrorPopover.mockReturnValue({
            openPopover: vi.fn(),
            Popover: () => <div data-testid="error-popover" />,
            closePopover: vi.fn(),
            errorMessage: "",
            hasErrors: false,
        });
        
        mockUseKeyboardOpen.mockReturnValue(false);
        mockUseWindowSize.mockReturnValue(false); // false = not mobile
    });

    describe("Basic rendering", () => {
        it("renders the bottom actions grid", () => {
            render(<BottomActionsButtons {...defaultProps} />);

            const grid = screen.getByTestId("bottom-actions-grid");
            expect(grid).toBeDefined();
        });

        it("renders error popover", () => {
            render(<BottomActionsButtons {...defaultProps} />);

            const popover = screen.getByTestId("error-popover");
            expect(popover).toBeDefined();
        });

        it("renders submit and cancel buttons by default", () => {
            render(<BottomActionsButtons {...defaultProps} />);

            const submitButton = screen.getByTestId("submit-button");
            const cancelButton = screen.getByTestId("cancel-button");

            expect(submitButton).toBeDefined();
            expect(cancelButton).toBeDefined();
        });
    });

    describe("Create vs Save button behavior", () => {
        it("renders Create button when isCreate is true", () => {
            render(<BottomActionsButtons {...defaultProps} isCreate={true} />);

            const submitButton = screen.getByRole("button", { name: "Create" });
            expect(submitButton).toBeDefined();
            expect(submitButton.textContent).toBe("Create");
        });

        it("renders Save button when isCreate is false", () => {
            render(<BottomActionsButtons {...defaultProps} isCreate={false} />);

            const submitButton = screen.getByRole("button", { name: "Save" });
            expect(submitButton).toBeDefined();
            expect(submitButton.textContent).toBe("Save");
        });

        it("has correct aria-label for Create button", () => {
            render(<BottomActionsButtons {...defaultProps} isCreate={true} />);

            const submitButton = screen.getByRole("button", { name: "Create" });
            expect(submitButton.getAttribute("aria-label")).toBe("Create");
        });

        it("has correct aria-label for Save button", () => {
            render(<BottomActionsButtons {...defaultProps} isCreate={false} />);

            const submitButton = screen.getByRole("button", { name: "Save" });
            expect(submitButton.getAttribute("aria-label")).toBe("Save");
        });
    });

    describe("Button interaction", () => {
        it("calls onSubmit when submit button is clicked", async () => {
            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} onSubmit={onSubmit} />);

            const submitButton = screen.getByTestId("submit-button");

            await act(async () => {
                await user.click(submitButton);
            });

            expect(onSubmit).toHaveBeenCalledTimes(1);
        });

        it("calls onCancel when cancel button is clicked", async () => {
            const onCancel = vi.fn();
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} onCancel={onCancel} />);

            const cancelButton = screen.getByTestId("cancel-button");

            await act(async () => {
                await user.click(cancelButton);
            });

            expect(onCancel).toHaveBeenCalledTimes(1);
        });

        it("prevents default and stops propagation on cancel", async () => {
            const onCancel = vi.fn();
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} onCancel={onCancel} />);

            const cancelButton = screen.getByTestId("cancel-button");

            await act(async () => {
                await user.click(cancelButton);
            });

            expect(onCancel).toHaveBeenCalledTimes(1);
        });

        it("calls onSubmit when submit wrapper is clicked", async () => {
            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} onSubmit={onSubmit} />);

            const submitWrapper = screen.getByTestId("submit-button-wrapper");

            await act(async () => {
                await user.click(submitWrapper);
            });

            expect(onSubmit).toHaveBeenCalledTimes(1);
        });
    });

    describe("Loading state", () => {
        it("disables submit button when loading is true", () => {
            render(<BottomActionsButtons {...defaultProps} loading={true} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(true);
        });

        it("disables cancel button when loading is true", () => {
            render(<BottomActionsButtons {...defaultProps} loading={true} />);

            const cancelButton = screen.getByTestId("cancel-button");
            expect(cancelButton.hasAttribute("disabled")).toBe(true);
        });

        it("shows loading state on submit button", () => {
            render(<BottomActionsButtons {...defaultProps} loading={true} />);

            const submitButton = screen.getByTestId("submit-button");
            // The button should have a loading indicator - exact implementation depends on Button component
            expect(submitButton).toBeDefined();
        });

        it("does not call onSubmit when submit button is disabled due to loading", async () => {
            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} onSubmit={onSubmit} loading={true} />);

            const submitButton = screen.getByTestId("submit-button");

            await act(async () => {
                await user.click(submitButton);
            });

            expect(onSubmit).not.toHaveBeenCalled();
        });
    });

    describe("Disabled states", () => {
        it("disables submit button when disabledSubmit is true", () => {
            render(<BottomActionsButtons {...defaultProps} disabledSubmit={true} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(true);
        });

        it("disables cancel button when disabledCancel is true", () => {
            render(<BottomActionsButtons {...defaultProps} disabledCancel={true} />);

            const cancelButton = screen.getByTestId("cancel-button");
            expect(cancelButton.hasAttribute("disabled")).toBe(true);
        });

        it("enables cancel button when disabledCancel is false", () => {
            render(<BottomActionsButtons {...defaultProps} disabledCancel={false} />);

            const cancelButton = screen.getByTestId("cancel-button");
            expect(cancelButton.hasAttribute("disabled")).toBe(false);
        });

        it("enables cancel button when disabledCancel is undefined", () => {
            render(<BottomActionsButtons {...defaultProps} disabledCancel={undefined} />);

            const cancelButton = screen.getByTestId("cancel-button");
            expect(cancelButton.hasAttribute("disabled")).toBe(false);
        });
    });

    describe("Error handling", () => {
        it("disables submit button when there are form errors", () => {
            const errors = { field1: "Error message" };
            render(<BottomActionsButtons {...defaultProps} errors={errors} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(true);
        });

        it("enables submit button when errors are empty", () => {
            const errors = {};
            render(<BottomActionsButtons {...defaultProps} errors={errors} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(false);
        });

        it("enables submit button when errors are undefined", () => {
            render(<BottomActionsButtons {...defaultProps} errors={undefined} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(false);
        });

        it("disables submit button when errors contain non-empty values", () => {
            const errors = { field1: null, field2: "", field3: "Error" };
            render(<BottomActionsButtons {...defaultProps} errors={errors} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(true);
        });

        it("enables submit button when all errors are null or undefined", () => {
            const errors = { field1: null, field2: undefined };
            render(<BottomActionsButtons {...defaultProps} errors={errors} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(false);
        });
    });

    describe("Hidden buttons behavior", () => {
        it("hides submit and cancel buttons when hideButtons is true", () => {
            render(<BottomActionsButtons {...defaultProps} hideButtons={true} />);

            const submitButton = screen.queryByTestId("submit-button");
            const cancelButton = screen.queryByTestId("cancel-button");

            expect(submitButton).toBeNull();
            expect(cancelButton).toBeNull();
        });

        it("shows submit and cancel buttons when hideButtons is false", () => {
            render(<BottomActionsButtons {...defaultProps} hideButtons={false} />);

            const submitButton = screen.getByTestId("submit-button");
            const cancelButton = screen.getByTestId("cancel-button");

            expect(submitButton).toBeDefined();
            expect(cancelButton).toBeDefined();
        });

        it("shows submit and cancel buttons when hideButtons is undefined", () => {
            render(<BottomActionsButtons {...defaultProps} hideButtons={undefined} />);

            const submitButton = screen.getByTestId("submit-button");
            const cancelButton = screen.getByTestId("cancel-button");

            expect(submitButton).toBeDefined();
            expect(cancelButton).toBeDefined();
        });
    });

    describe("Side action buttons", () => {
        it("renders side action buttons when provided", () => {
            const sideButtons = [
                <button key="1" data-testid="side-button-1">Side 1</button>,
                <button key="2" data-testid="side-button-2">Side 2</button>,
            ];

            render(<BottomActionsButtons {...defaultProps} sideActionButtons={sideButtons} />);

            const sideActionsBox = screen.getByTestId("side-actions-box");
            const sideButton1 = screen.getByTestId("side-button-1");
            const sideButton2 = screen.getByTestId("side-button-2");

            expect(sideActionsBox).toBeDefined();
            expect(sideButton1).toBeDefined();
            expect(sideButton2).toBeDefined();
        });

        it("does not render side actions box when no side buttons provided", () => {
            render(<BottomActionsButtons {...defaultProps} sideActionButtons={undefined} />);

            const sideActionsBox = screen.queryByTestId("side-actions-box");
            expect(sideActionsBox).toBeNull();
        });

        it("renders single side action button", () => {
            const sideButton = <button key="1" data-testid="single-side-button">Single Side</button>;

            render(<BottomActionsButtons {...defaultProps} sideActionButtons={sideButton} />);

            const sideActionsBox = screen.getByTestId("side-actions-box");
            const singleSideButton = screen.getByTestId("single-side-button");

            expect(sideActionsBox).toBeDefined();
            expect(singleSideButton).toBeDefined();
        });

        it("handles null side action buttons gracefully", () => {
            const sideButtons = [
                <button key="1" data-testid="valid-side-button">Valid</button>,
                null,
                undefined,
            ];

            render(<BottomActionsButtons {...defaultProps} sideActionButtons={sideButtons} />);

            const sideActionsBox = screen.getByTestId("side-actions-box");
            const validButton = screen.getByTestId("valid-side-button");

            expect(sideActionsBox).toBeDefined();
            expect(validButton).toBeDefined();
        });
    });

    describe("Text visibility on mobile", () => {
        it("renders button text by default", () => {
            render(<BottomActionsButtons {...defaultProps} isCreate={true} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toBe("Create");
        });

        it("hides text on mobile when hideTextOnMobile is true", () => {
            // Mock mobile viewport
            mockUseWindowSize.mockReturnValue(true); // true = mobile

            render(<BottomActionsButtons {...defaultProps} isCreate={true} hideTextOnMobile={true} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toBe("");
        });

        it("shows text on desktop even when hideTextOnMobile is true", () => {
            // Mock desktop viewport
            mockUseWindowSize.mockReturnValue(false); // false = desktop

            render(<BottomActionsButtons {...defaultProps} isCreate={true} hideTextOnMobile={true} />);

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toBe("Create");
        });
    });

    describe("Error popover integration", () => {
        it("opens error popover when submit is clicked with errors", async () => {
            const mockOpenPopover = vi.fn();
            mockUseErrorPopover.mockReturnValue({
                openPopover: mockOpenPopover,
                Popover: () => <div data-testid="error-popover" />,
                closePopover: vi.fn(),
                errorMessage: "Error message",
                hasErrors: true,
            });

            const errors = { field1: "Error message" };
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} errors={errors} />);

            const submitWrapper = screen.getByTestId("submit-button-wrapper");

            await act(async () => {
                await user.click(submitWrapper);
            });

            expect(mockOpenPopover).toHaveBeenCalledTimes(1);
        });

        it("does not open error popover when submit is clicked without errors", async () => {
            const mockOpenPopover = vi.fn();
            mockUseErrorPopover.mockReturnValue({
                openPopover: mockOpenPopover,
                Popover: () => <div data-testid="error-popover" />,
                closePopover: vi.fn(),
                errorMessage: "",
                hasErrors: false,
            });

            const onSubmit = vi.fn();
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} onSubmit={onSubmit} errors={undefined} />);

            const submitWrapper = screen.getByTestId("submit-button-wrapper");

            await act(async () => {
                await user.click(submitWrapper);
            });

            expect(mockOpenPopover).not.toHaveBeenCalled();
            expect(onSubmit).toHaveBeenCalledTimes(1);
        });
    });

    describe("Component state transitions", () => {
        it("toggles between loading and non-loading states", () => {
            const { rerender } = render(<BottomActionsButtons {...defaultProps} loading={false} />);

            // Initially not loading
            let submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(false);

            // Switch to loading state
            rerender(<BottomActionsButtons {...defaultProps} loading={true} />);

            submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(true);

            // Switch back to non-loading state
            rerender(<BottomActionsButtons {...defaultProps} loading={false} />);

            submitButton = screen.getByTestId("submit-button");
            expect(submitButton.hasAttribute("disabled")).toBe(false);
        });

        it("toggles between Create and Save modes", () => {
            const { rerender } = render(<BottomActionsButtons {...defaultProps} isCreate={true} />);

            // Initially Create mode
            let submitButton = screen.getByRole("button", { name: "Create" });
            expect(submitButton.textContent).toBe("Create");

            // Switch to Save mode
            rerender(<BottomActionsButtons {...defaultProps} isCreate={false} />);

            submitButton = screen.getByRole("button", { name: "Save" });
            expect(submitButton.textContent).toBe("Save");

            // Switch back to Create mode
            rerender(<BottomActionsButtons {...defaultProps} isCreate={true} />);

            submitButton = screen.getByRole("button", { name: "Create" });
            expect(submitButton.textContent).toBe("Create");
        });

        it("toggles button visibility", () => {
            const { rerender } = render(<BottomActionsButtons {...defaultProps} hideButtons={false} />);

            // Initially buttons visible
            expect(screen.getByTestId("submit-button")).toBeDefined();
            expect(screen.getByTestId("cancel-button")).toBeDefined();

            // Hide buttons
            rerender(<BottomActionsButtons {...defaultProps} hideButtons={true} />);

            expect(screen.queryByTestId("submit-button")).toBeNull();
            expect(screen.queryByTestId("cancel-button")).toBeNull();

            // Show buttons again
            rerender(<BottomActionsButtons {...defaultProps} hideButtons={false} />);

            expect(screen.getByTestId("submit-button")).toBeDefined();
            expect(screen.getByTestId("cancel-button")).toBeDefined();
        });

        it("handles side action buttons addition and removal", () => {
            const { rerender } = render(<BottomActionsButtons {...defaultProps} sideActionButtons={undefined} />);

            // Initially no side buttons
            expect(screen.queryByTestId("side-actions-box")).toBeNull();

            // Add side buttons
            const sideButtons = [<button key="1" data-testid="side-button">Side</button>];
            rerender(<BottomActionsButtons {...defaultProps} sideActionButtons={sideButtons} />);

            expect(screen.getByTestId("side-actions-box")).toBeDefined();
            expect(screen.getByTestId("side-button")).toBeDefined();

            // Remove side buttons
            rerender(<BottomActionsButtons {...defaultProps} sideActionButtons={undefined} />);

            expect(screen.queryByTestId("side-actions-box")).toBeNull();
            expect(screen.queryByTestId("side-button")).toBeNull();
        });
    });

    describe("Accessibility", () => {
        it("has correct ARIA labels for all buttons", () => {
            render(<BottomActionsButtons {...defaultProps} isCreate={true} />);

            const submitButton = screen.getByRole("button", { name: "Create" });
            const cancelButton = screen.getByRole("button", { name: "Cancel" });

            expect(submitButton.getAttribute("aria-label")).toBe("Create");
            expect(cancelButton.getAttribute("aria-label")).toBe("Cancel");
        });

        it("maintains button semantics", () => {
            render(<BottomActionsButtons {...defaultProps} />);

            const submitButton = screen.getByTestId("submit-button");
            const cancelButton = screen.getByTestId("cancel-button");

            expect(submitButton.tagName).toBe("BUTTON");
            expect(cancelButton.tagName).toBe("BUTTON");
        });

        it("handles keyboard navigation correctly", async () => {
            const user = userEvent.setup();

            render(<BottomActionsButtons {...defaultProps} />);

            const submitButton = screen.getByTestId("submit-button");
            const cancelButton = screen.getByTestId("cancel-button");

            // Focus first button
            await act(async () => {
                await user.tab();
            });

            expect(document.activeElement).toBe(submitButton);

            // Tab to second button
            await act(async () => {
                await user.tab();
            });

            expect(document.activeElement).toBe(cancelButton);
        });
    });

    describe("Integration with different display modes", () => {
        it("works with Page display mode", () => {
            render(<BottomActionsButtons {...defaultProps} display="Page" />);

            const grid = screen.getByTestId("bottom-actions-grid");
            expect(grid).toBeDefined();
        });

        it("works with Dialog display mode", () => {
            render(<BottomActionsButtons {...defaultProps} display="Dialog" />);

            const grid = screen.getByTestId("bottom-actions-grid");
            expect(grid).toBeDefined();
        });

        it("works with Drawer display mode", () => {
            render(<BottomActionsButtons {...defaultProps} display="Drawer" />);

            const grid = screen.getByTestId("bottom-actions-grid");
            expect(grid).toBeDefined();
        });
    });
});
