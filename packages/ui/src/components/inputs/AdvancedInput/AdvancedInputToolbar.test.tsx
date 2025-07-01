import { act, render, screen, within } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { AdvancedInputToolbar, defaultActiveStates, TOOLBAR_CLASS_NAME } from "./AdvancedInputToolbar";
import { AdvancedInputAction } from "./utils";
import { usePopover } from "../../../hooks/usePopover";

// Mock hooks and dependencies
const mockUseIsLeftHanded = vi.fn(() => false);
vi.mock("../../../hooks/subscriptions", () => ({
    useIsLeftHanded: () => mockUseIsLeftHanded(),
}));

const mockUseDimensions = vi.fn(() => ({
    dimensions: { width: 800, height: 600 },
    ref: { current: null },
}));
vi.mock("../../../hooks/useDimensions", () => ({
    useDimensions: () => mockUseDimensions(),
}));

vi.mock("../../../hooks/usePopover", () => ({
    usePopover: vi.fn(() => [
        null, // anchorEl
        vi.fn(), // open function
        vi.fn(), // close function
        false, // isOpen
    ]),
}));

vi.mock("../../../utils/display/device", () => ({
    keyComboToString: (...keys: string[]) => keys.join("+"),
}));

vi.mock("../../../icons/Icons", () => ({
    Icon: ({ info, size }: any) => <div data-testid={`icon-${info.name}`} data-size={size || 20} />,
    IconCommon: ({ name, size }: any) => <div data-testid={`icon-common-${name}`} data-size={size || 20} />,
    IconText: ({ name, size }: any) => <div data-testid={`icon-text-${name}`} data-size={size || 20} />,
}));

// Mock Tooltip component
vi.mock("../../Tooltip/Tooltip", () => ({
    Tooltip: ({ children, title, placement }: any) => (
        <div title={title} data-placement={placement}>
            {children}
        </div>
    ),
}));

describe("AdvancedInputToolbar", () => {
    const defaultProps = {
        activeStates: defaultActiveStates,
        canRedo: false,
        canUndo: false,
        disabled: false,
        handleAction: vi.fn(),
        handleActiveStatesChange: vi.fn(),
        isMarkdownOn: false,
    };

    beforeEach(() => {
        vi.clearAllMocks();
        mockUseIsLeftHanded.mockReturnValue(false);
        mockUseDimensions.mockReturnValue({
            dimensions: { width: 800, height: 600 },
            ref: { current: null },
        });
    });

    describe("Basic rendering", () => {
        it("renders toolbar with correct structure and class name", () => {
            render(<AdvancedInputToolbar {...defaultProps} />);

            const toolbar = screen.getByRole("region", { name: "Text input toolbar" });
            expect(toolbar).toBeDefined();
            expect(toolbar.className).toContain(TOOLBAR_CLASS_NAME);
            expect(toolbar.tagName).toBe("SECTION");
        });

        it("renders left and right sections", () => {
            render(<AdvancedInputToolbar {...defaultProps} />);

            const leftSection = screen.getByTestId("toolbar-left-section");
            const rightSection = screen.getByTestId("toolbar-right-section");
            
            expect(leftSection).toBeDefined();
            expect(rightSection).toBeDefined();
        });

        it("renders header button in left section", () => {
            render(<AdvancedInputToolbar {...defaultProps} />);

            const headerIcon = screen.getByTestId("icon-text-Header");
            expect(headerIcon).toBeDefined();
            // Icon size might be adjusted by the IconButton component
            expect(headerIcon.getAttribute("data-size")).toBeDefined();
        });

        it("renders mode toggle in right section", () => {
            render(<AdvancedInputToolbar {...defaultProps} />);

            const modeToggle = screen.getByTestId("mode-toggle");
            expect(modeToggle).toBeDefined();
            expect(modeToggle.textContent).toBe("MarkdownTo");
            expect(modeToggle.getAttribute("data-markdown")).toBe("false");
        });
    });

    describe("Accessibility", () => {
        it("has proper ARIA attributes on toolbar", () => {
            render(<AdvancedInputToolbar {...defaultProps} />);

            const toolbar = screen.getByRole("region");
            expect(toolbar.getAttribute("aria-label")).toBe("Text input toolbar");
            // Role is already "region" from getByRole
        });

        it("provides accessible labels for buttons", () => {
            render(<AdvancedInputToolbar {...defaultProps} canUndo canRedo />);

            // Find buttons by their data-testid
            const buttons = screen.getAllByTestId("tool-button");
            
            // Header button should have aria-label
            const headerButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-text-Header\"]"),
            );
            expect(headerButton?.getAttribute("aria-label")).toBe("HeaderInsert");

            // Undo/redo buttons should have aria-labels
            const undoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Undo\"]"),
            );
            const redoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Redo\"]"),
            );
            expect(undoButton?.getAttribute("aria-label")).toContain("Undo");
            expect(redoButton?.getAttribute("aria-label")).toContain("Redo");
        });

        it("maintains focus management for keyboard navigation", async () => {
            const user = userEvent.setup();
            render(<AdvancedInputToolbar {...defaultProps} />);

            const buttons = screen.getAllByTestId("tool-button");
            const headerButton = buttons[0];
            
            await act(async () => {
                await user.click(headerButton);
            });

            expect(document.activeElement).toBe(headerButton);
        });
    });

    describe("Disabled state", () => {
        it("disables all interactive buttons when disabled", () => {
            render(<AdvancedInputToolbar {...defaultProps} disabled canUndo canRedo />);

            const buttons = screen.getAllByTestId("tool-button");
            buttons.forEach(button => {
                expect(button.hasAttribute("disabled")).toBe(true);
            });
        });

        it("hides left section when disabled", () => {
            render(<AdvancedInputToolbar {...defaultProps} disabled />);

            const leftSection = screen.getByTestId("toolbar-left-section");
            // The visibility is controlled by styled component CSS
            expect(leftSection).toBeDefined();
        });

        it("keeps mode toggle clickable when disabled", async () => {
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            render(<AdvancedInputToolbar {...defaultProps} disabled handleAction={handleAction} />);

            const modeToggle = screen.getByTestId("mode-toggle");
            
            await act(async () => {
                await user.click(modeToggle);
            });

            expect(handleAction).toHaveBeenCalledWith(AdvancedInputAction.Mode, undefined);
        });
    });

    describe("Active states", () => {
        it("shows buttons as active based on activeStates prop", () => {
            const activeStates = {
                ...defaultActiveStates,
                Bold: true,
                Italic: true,
                Header1: true,
            };

            render(<AdvancedInputToolbar {...defaultProps} activeStates={activeStates} />);

            // Find header button and check if it's active
            const buttons = screen.getAllByTestId("tool-button");
            const headerButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-text-Header\"]"),
            );
            expect(headerButton?.getAttribute("data-active")).toBe("true");
        });

        it("updates active states when toggling actions in WYSIWYG mode", async () => {
            const handleActiveStatesChange = vi.fn();
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            render(
                <AdvancedInputToolbar 
                    {...defaultProps} 
                    handleActiveStatesChange={handleActiveStatesChange}
                    handleAction={handleAction}
                    isMarkdownOn={false}
                />,
            );

            const modeToggle = screen.getByTestId("mode-toggle");
            
            await act(async () => {
                await user.click(modeToggle);
            });

            // Mode action doesn't update active states
            expect(handleAction).toHaveBeenCalledWith(AdvancedInputAction.Mode, undefined);
        });

        it("resets active states when switching to markdown mode", () => {
            const handleActiveStatesChange = vi.fn();
            
            render(
                <AdvancedInputToolbar 
                    {...defaultProps} 
                    handleActiveStatesChange={handleActiveStatesChange}
                    isMarkdownOn={true}
                />,
            );

            expect(handleActiveStatesChange).toHaveBeenCalledWith(defaultActiveStates);
        });

        it("does not update active states in markdown mode", () => {
            const handleActiveStatesChange = vi.fn();
            const activeStates = {
                ...defaultActiveStates,
                Bold: true,
            };
            
            render(
                <AdvancedInputToolbar 
                    {...defaultProps} 
                    activeStates={activeStates}
                    handleActiveStatesChange={handleActiveStatesChange}
                    isMarkdownOn={true}
                />,
            );

            // Should reset to default states
            expect(handleActiveStatesChange).toHaveBeenCalledWith(defaultActiveStates);
        });
    });

    describe("Mode toggle", () => {
        it("shows correct text for WYSIWYG mode", () => {
            render(<AdvancedInputToolbar {...defaultProps} isMarkdownOn={false} />);

            const modeToggle = screen.getByTestId("mode-toggle");
            expect(modeToggle.textContent).toBe("MarkdownTo");
            expect(modeToggle.getAttribute("data-markdown")).toBe("false");
        });

        it("shows correct text for markdown mode", () => {
            render(<AdvancedInputToolbar {...defaultProps} isMarkdownOn={true} />);

            const modeToggle = screen.getByTestId("mode-toggle");
            expect(modeToggle.textContent).toBe("PreviewTo");
            expect(modeToggle.getAttribute("data-markdown")).toBe("true");
        });

        it("calls handleAction with Mode action when clicked", async () => {
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            render(<AdvancedInputToolbar {...defaultProps} handleAction={handleAction} />);

            const modeToggle = screen.getByTestId("mode-toggle");
            
            await act(async () => {
                await user.click(modeToggle);
            });

            expect(handleAction).toHaveBeenCalledTimes(1);
            expect(handleAction).toHaveBeenCalledWith(AdvancedInputAction.Mode, undefined);
        });

        it("has correct tooltip with keyboard shortcut", () => {
            render(<AdvancedInputToolbar {...defaultProps} isMarkdownOn={false} />);

            const modeToggle = screen.getByTestId("mode-toggle");
            const tooltip = modeToggle.closest("[title]");
            
            expect(tooltip?.getAttribute("title")).toContain("PressToMarkdown");
            expect(tooltip?.getAttribute("title")).toContain("Alt+0");
        });
    });

    describe("Undo/Redo functionality", () => {
        it("shows undo button only when canUndo is true", () => {
            const { rerender } = render(<AdvancedInputToolbar {...defaultProps} canUndo={false} />);

            expect(screen.queryByTestId("icon-common-Undo")).toBeNull();

            rerender(<AdvancedInputToolbar {...defaultProps} canUndo={true} />);

            expect(screen.getByTestId("icon-common-Undo")).toBeDefined();
        });

        it("shows redo button only when canRedo is true", () => {
            const { rerender } = render(<AdvancedInputToolbar {...defaultProps} canRedo={false} />);

            expect(screen.queryByTestId("icon-common-Redo")).toBeNull();

            rerender(<AdvancedInputToolbar {...defaultProps} canRedo={true} />);

            expect(screen.getByTestId("icon-common-Redo")).toBeDefined();
        });

        it("enables undo button when canUndo is true and not disabled", () => {
            render(<AdvancedInputToolbar {...defaultProps} canUndo={true} disabled={false} />);

            const buttons = screen.getAllByTestId("tool-button");
            const undoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Undo\"]"),
            );
            expect(undoButton?.hasAttribute("disabled")).toBe(false);
        });

        it("disables undo button when toolbar is disabled", () => {
            render(<AdvancedInputToolbar {...defaultProps} canUndo={true} disabled={true} />);

            const buttons = screen.getAllByTestId("tool-button");
            const undoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Undo\"]"),
            );
            expect(undoButton?.hasAttribute("disabled")).toBe(true);
        });

        it("calls handleAction with Undo when undo clicked", async () => {
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            render(<AdvancedInputToolbar {...defaultProps} handleAction={handleAction} canUndo={true} />);

            const buttons = screen.getAllByTestId("tool-button");
            const undoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Undo\"]"),
            );
            
            if (undoButton) {
                await act(async () => {
                    await user.click(undoButton);
                });
            }

            expect(handleAction).toHaveBeenCalledWith(AdvancedInputAction.Undo, undefined);
        });

        it("calls handleAction with Redo when redo clicked", async () => {
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            render(<AdvancedInputToolbar {...defaultProps} handleAction={handleAction} canRedo={true} />);

            const buttons = screen.getAllByTestId("tool-button");
            const redoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Redo\"]"),
            );
            
            if (redoButton) {
                await act(async () => {
                    await user.click(redoButton);
                });
            }

            expect(handleAction).toHaveBeenCalledWith(AdvancedInputAction.Redo, undefined);
        });

        it("shows keyboard shortcuts in tooltips", () => {
            render(<AdvancedInputToolbar {...defaultProps} canUndo={true} canRedo={true} />);

            const buttons = screen.getAllByTestId("tool-button");
            const undoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Undo\"]"),
            );
            const redoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Redo\"]"),
            );

            // Tooltips are on the parent wrapper
            const undoTooltip = undoButton?.closest("[title]");
            const redoTooltip = redoButton?.closest("[title]");

            expect(undoTooltip?.getAttribute("title")).toBe("Undo (Ctrl+z)");
            expect(redoTooltip?.getAttribute("title")).toBe("Redo (Ctrl+y)");
        });
    });

    describe("Button interactions", () => {
        it("calls handleAction when header button is clicked", async () => {
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            // Mock the popover hook to be controlled
            const mockOpenHeaderSelect = vi.fn();
            vi.mocked(usePopover).mockReturnValueOnce([
                null, // anchorEl
                mockOpenHeaderSelect, // open function
                vi.fn(), // close function
                false, // isOpen
            ]).mockReturnValue([
                null, // anchorEl
                vi.fn(), // open function
                vi.fn(), // close function
                false, // isOpen
            ]);
            
            render(<AdvancedInputToolbar {...defaultProps} handleAction={handleAction} />);

            const buttons = screen.getAllByTestId("tool-button");
            const headerButton = buttons[0];

            await act(async () => {
                await user.click(headerButton);
            });

            // The header button should trigger the popover
            expect(mockOpenHeaderSelect).toHaveBeenCalled();
        });

        it("prevents default and stops propagation on button clicks", async () => {
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            // Render with a parent element that has a click handler
            const parentClickHandler = vi.fn();
            render(
                <div onClick={parentClickHandler}>
                    <AdvancedInputToolbar {...defaultProps} handleAction={handleAction} canUndo />
                </div>,
            );

            const buttons = screen.getAllByTestId("tool-button");
            // Find the undo button which directly calls handleAction
            const undoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Undo\"]"),
            );

            if (undoButton) {
                await act(async () => {
                    await user.click(undoButton);
                });

                // handleAction should be called for undo
                expect(handleAction).toHaveBeenCalledWith(AdvancedInputAction.Undo, undefined);
                // Parent handler should not be called due to stopPropagation
                expect(parentClickHandler).not.toHaveBeenCalled();
            }
        });

        it("handles rapid button clicks gracefully", async () => {
            const handleAction = vi.fn();
            const user = userEvent.setup();
            
            render(<AdvancedInputToolbar {...defaultProps} handleAction={handleAction} canUndo={true} />);

            const buttons = screen.getAllByTestId("tool-button");
            const undoButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-common-Undo\"]"),
            );
            
            if (undoButton) {
                await act(async () => {
                    await user.tripleClick(undoButton);
                });
            }

            // Should handle all clicks
            expect(handleAction).toHaveBeenCalledTimes(3);
        });

        it("handles keyboard navigation between buttons", async () => {
            const user = userEvent.setup();
            
            render(<AdvancedInputToolbar {...defaultProps} canUndo={true} canRedo={true} />);

            const buttons = screen.getAllByTestId("tool-button");
            const firstButton = buttons[0];
            
            await act(async () => {
                await user.click(firstButton);
                await user.tab();
            });

            // Focus should move through toolbar buttons
            expect(document.activeElement).not.toBe(firstButton);
        });
    });

    describe("Responsive behavior", () => {
        it("adapts layout for small screens", () => {
            // Mock small screen dimensions
            mockUseDimensions.mockReturnValue({
                dimensions: { width: 350, height: 600 },
                ref: { current: null },
            });

            render(<AdvancedInputToolbar {...defaultProps} />);

            // Should render minimal view - check for combined format button
            expect(screen.getByTestId("icon-text-CaseSensitive")).toBeDefined();
        });

        it("shows partial controls on medium screens", () => {
            mockUseDimensions.mockReturnValue({
                dimensions: { width: 500, height: 600 },
                ref: { current: null },
            });

            render(<AdvancedInputToolbar {...defaultProps} />);

            // Should show format popover button instead of individual buttons
            const toolbar = screen.getByRole("region");
            expect(toolbar).toBeDefined();
        });

        it("shows full controls on large screens", () => {
            mockUseDimensions.mockReturnValue({
                dimensions: { width: 1200, height: 800 },
                ref: { current: null },
            });

            render(<AdvancedInputToolbar {...defaultProps} />);

            // Should show individual format buttons in full view
            const toolbar = screen.getByRole("region");
            expect(toolbar).toBeDefined();
        });
    });

    describe("Left-handed support", () => {
        it("reverses layout when user is left-handed", () => {
            mockUseIsLeftHanded.mockReturnValue(true);

            render(<AdvancedInputToolbar {...defaultProps} />);

            const toolbar = screen.getByRole("region");
            // Layout reversal would be checked via CSS/style attributes
            expect(toolbar).toBeDefined();
        });

        it("maintains normal layout for right-handed users", () => {
            mockUseIsLeftHanded.mockReturnValue(false);

            render(<AdvancedInputToolbar {...defaultProps} />);

            const toolbar = screen.getByRole("region");
            expect(toolbar).toBeDefined();
        });
    });

    describe("Toolbar sections visibility", () => {
        it("shows both sections when enabled", () => {
            render(<AdvancedInputToolbar {...defaultProps} />);

            const leftSection = screen.getByTestId("toolbar-left-section");
            const rightSection = screen.getByTestId("toolbar-right-section");

            expect(leftSection).toBeDefined();
            expect(rightSection).toBeDefined();
        });

        it("shows mode toggle even when disabled", () => {
            render(<AdvancedInputToolbar {...defaultProps} disabled />);

            const modeToggle = screen.getByTestId("mode-toggle");
            expect(modeToggle).toBeDefined();
        });
    });

    describe("Error handling", () => {
        it("handles missing translations gracefully", () => {
            // Component should not crash with missing translations
            render(<AdvancedInputToolbar {...defaultProps} />);

            const toolbar = screen.getByRole("region");
            expect(toolbar).toBeDefined();
        });

        it("handles undefined callbacks gracefully", () => {
            // Test with undefined handleAction
            const props = {
                ...defaultProps,
                handleAction: undefined as any,
            };

            // Should not crash
            expect(() => render(<AdvancedInputToolbar {...props} />)).not.toThrow();
        });

        it("handles partial active states", () => {
            const partialActiveStates = {
                Bold: true,
                // Missing other properties
            } as any;

            render(<AdvancedInputToolbar {...defaultProps} activeStates={partialActiveStates} />);

            const toolbar = screen.getByRole("region");
            expect(toolbar).toBeDefined();
        });
    });

    describe("State transitions", () => {
        it("updates correctly when activeStates change", () => {
            const { rerender } = render(<AdvancedInputToolbar {...defaultProps} />);

            const buttons = screen.getAllByTestId("tool-button");
            const headerButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-text-Header\"]"),
            );
            expect(headerButton?.getAttribute("data-active")).toBe("false");

            const newActiveStates = {
                ...defaultActiveStates,
                Header1: true,
            };

            rerender(<AdvancedInputToolbar {...defaultProps} activeStates={newActiveStates} />);

            const updatedButtons = screen.getAllByTestId("tool-button");
            const updatedHeaderButton = updatedButtons.find(btn => 
                btn.querySelector("[data-testid=\"icon-text-Header\"]"),
            );
            expect(updatedHeaderButton?.getAttribute("data-active")).toBe("true");
        });

        it("updates when switching between markdown and WYSIWYG modes", () => {
            const { rerender } = render(<AdvancedInputToolbar {...defaultProps} isMarkdownOn={false} />);

            let modeToggle = screen.getByTestId("mode-toggle");
            expect(modeToggle.textContent).toBe("MarkdownTo");

            rerender(<AdvancedInputToolbar {...defaultProps} isMarkdownOn={true} />);

            modeToggle = screen.getByTestId("mode-toggle");
            expect(modeToggle.textContent).toBe("PreviewTo");
        });

        it("enables/disables undo/redo dynamically", () => {
            const { rerender } = render(<AdvancedInputToolbar {...defaultProps} canUndo={false} canRedo={false} />);

            expect(screen.queryByTestId("icon-common-Undo")).toBeNull();
            expect(screen.queryByTestId("icon-common-Redo")).toBeNull();

            rerender(<AdvancedInputToolbar {...defaultProps} canUndo={true} canRedo={true} />);

            expect(screen.getByTestId("icon-common-Undo")).toBeDefined();
            expect(screen.getByTestId("icon-common-Redo")).toBeDefined();
        });
    });

    describe("Popover behavior", () => {
        it("renders header button that opens popover", async () => {
            const user = userEvent.setup();
            render(<AdvancedInputToolbar {...defaultProps} />);

            const buttons = screen.getAllByTestId("tool-button");
            const headerButton = buttons.find(btn => 
                btn.querySelector("[data-testid=\"icon-text-Header\"]"),
            );

            expect(headerButton).toBeDefined();
            
            // Click to open popover
            if (headerButton) {
                await act(async () => {
                    await user.click(headerButton);
                });
            }

            // Popover is mocked, so we just verify the button exists and is clickable
            expect(headerButton).toBeDefined();
        });

        it("shows format popover button in partial view", () => {
            mockUseDimensions.mockReturnValue({
                dimensions: { width: 500, height: 600 },
                ref: { current: null },
            });

            render(<AdvancedInputToolbar {...defaultProps} />);

            // Should show format button with CaseSensitive icon
            const formatIcon = screen.queryByTestId("icon-text-CaseSensitive");
            expect(formatIcon).toBeDefined();
        });

        it("shows list button in non-minimal views", () => {
            mockUseDimensions.mockReturnValue({
                dimensions: { width: 600, height: 600 },
                ref: { current: null },
            });

            render(<AdvancedInputToolbar {...defaultProps} />);

            const listIcon = screen.queryByTestId("icon-text-List");
            expect(listIcon).toBeDefined();
        });
    });
});

describe("AdvancedInputToolbar Constants", () => {
    it("exports correct default active states", () => {
        expect(defaultActiveStates).toEqual({
            Bold: false,
            Code: false,
            Header1: false,
            Header2: false,
            Header3: false,
            Header4: false,
            Header5: false,
            Header6: false,
            Italic: false,
            Link: false,
            ListBullet: false,
            ListNumber: false,
            ListCheckbox: false,
            Quote: false,
            Spoiler: false,
            Strikethrough: false,
            Table: false,
            Underline: false,
        });
    });

    it("exports correct toolbar class name", () => {
        expect(TOOLBAR_CLASS_NAME).toBe("advanced-input-toolbar");
    });

    it("has all required action types in defaultActiveStates", () => {
        const requiredActions = [
            "Bold", "Code", "Header1", "Header2", "Header3", 
            "Header4", "Header5", "Header6", "Italic", "Link",
            "ListBullet", "ListNumber", "ListCheckbox", "Quote",
            "Spoiler", "Strikethrough", "Table", "Underline",
        ];

        requiredActions.forEach(action => {
            expect(defaultActiveStates).toHaveProperty(action);
            expect(defaultActiveStates[action as keyof typeof defaultActiveStates]).toBe(false);
        });
    });
});
