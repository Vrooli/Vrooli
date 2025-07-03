import { act, render, screen, waitFor, within } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { FormTip } from "./FormTip";
import type { FormTipType } from "@vrooli/shared";

// Mock dependencies
vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key,
    }),
}));

const mockPublish = vi.fn();
vi.mock("../../../utils/pubsub", () => ({
    PubSub: {
        get: () => ({
            publish: mockPublish,
        }),
    },
}));

vi.mock("../../text/MarkdownDisplay", () => ({
    MarkdownDisplay: ({ content }: { content: string }) => (
        <div data-testid="markdown-display">{content}</div>
    ),
}));

vi.mock("../AdvancedInput/AdvancedInput", () => ({
    AdvancedInputBase: ({ onChange, value, onBlur, disableAssistant, placeholder, "data-testid": dataTestId, ...props }: any) => {
        // Filter out non-DOM props
        const { className, style, disabled, id, name } = props;
        return (
            <input
                className={className}
                style={style}
                placeholder={placeholder}
                disabled={disabled}
                id={id}
                name={name}
                data-testid={dataTestId}
                value={value}
                onChange={(e) => onChange(e.target.value)}
                onBlur={onBlur}
            />
        );
    },
}));

describe("FormTip", () => {
    const defaultElement: FormTipType = {
        type: "Tip" as const,
        id: "tip-1",
        label: "This is a helpful tip",
    };

    const defaultProps = {
        element: defaultElement,
        isEditing: false,
        onDelete: vi.fn(),
        onUpdate: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Non-editing mode", () => {
        it("renders basic tip without link", () => {
            render(<FormTip {...defaultProps} />);

            const tip = screen.getByTestId("form-tip");
            expect(tip).toBeDefined();
            expect(tip.getAttribute("data-editing")).toBe("false");
            expect(tip.getAttribute("data-icon")).toBe("Info");
            expect(tip.getAttribute("data-has-link")).toBe("false");
            expect(tip.getAttribute("data-is-markdown")).toBe("false");
            expect(tip.getAttribute("role")).toBe("note");
            expect(tip.getAttribute("aria-label")).toBe("Tip: This is a helpful tip");

            // Check icon is rendered
            expect(screen.getByTestId("tip-icon")).toBeDefined();

            // Check content is rendered as plain text
            expect(screen.getByText("This is a helpful tip")).toBeDefined();
            expect(screen.queryByTestId("markdown-display")).toBeNull();
        });

        it("renders tip with link", () => {
            const elementWithLink = {
                ...defaultElement,
                link: "https://example.com",
            };

            render(<FormTip {...defaultProps} element={elementWithLink} />);

            const tip = screen.getByTestId("form-tip");
            expect(tip.getAttribute("data-has-link")).toBe("true");

            const link = screen.getByTestId("tip-link");
            expect(link).toBeDefined();
            expect(link.getAttribute("href")).toBe("https://example.com");
            expect(link.getAttribute("target")).toBe("_blank");
            expect(link.getAttribute("rel")).toBe("noopener noreferrer");
            expect(link.getAttribute("aria-label")).toBe("Tip link: This is a helpful tip");
        });

        it("renders markdown content when isMarkdown is true", () => {
            const markdownElement = {
                ...defaultElement,
                isMarkdown: true,
                label: "**Bold** and *italic* text",
            };

            render(<FormTip {...defaultProps} element={markdownElement} />);

            const tip = screen.getByTestId("form-tip");
            expect(tip.getAttribute("data-is-markdown")).toBe("true");

            const markdownDisplay = screen.getByTestId("markdown-display");
            expect(markdownDisplay).toBeDefined();
            expect(markdownDisplay.textContent).toBe("**Bold** and *italic* text");
        });

        it("renders with different icon types", () => {
            const iconTypes = ["Error", "Warning", "Info"] as const;

            iconTypes.forEach((icon) => {
                const { unmount } = render(
                    <FormTip {...defaultProps} element={{ ...defaultElement, icon }} />,
                );

                const tip = screen.getByTestId("form-tip");
                expect(tip.getAttribute("data-icon")).toBe(icon);

                unmount();
            });
        });

        it("handles YouTube link with special behavior", async () => {
            mockPublish.mockClear();

            const youtubeElement = {
                ...defaultElement,
                link: "https://www.youtube.com/watch?v=abc123",
            };

            const user = userEvent.setup();
            render(<FormTip {...defaultProps} element={youtubeElement} />);

            const link = screen.getByTestId("tip-link");

            await act(async () => {
                await user.click(link);
            });

            expect(mockPublish).toHaveBeenCalledTimes(1);
            expect(mockPublish).toHaveBeenCalledWith("popupVideo", { 
                src: "https://www.youtube.com/watch?v=abc123", 
            });
        });

        it("does not show editing controls", () => {
            render(<FormTip {...defaultProps} />);

            expect(screen.queryByTestId("delete-button")).toBeNull();
            expect(screen.queryByTestId("text-input")).toBeNull();
            expect(screen.queryByTestId("link-input")).toBeNull();
            expect(screen.queryByTestId("icon-button")).toBeNull();
            expect(screen.queryByTestId("toggle-markdown-button")).toBeNull();
        });
    });

    describe("Editing mode", () => {
        const editingProps = {
            ...defaultProps,
            isEditing: true,
        };

        it("renders editing interface with all controls", () => {
            render(<FormTip {...editingProps} />);

            const tip = screen.getByTestId("form-tip");
            expect(tip.getAttribute("data-editing")).toBe("true");
            expect(tip.getAttribute("role")).toBe("group");
            expect(tip.getAttribute("aria-label")).toBe("Edit tip");

            // Check all editing controls are present
            expect(screen.getByTestId("delete-button")).toBeDefined();
            expect(screen.getByTestId("text-input")).toBeDefined();
            expect(screen.getByTestId("link-input")).toBeDefined();
            expect(screen.getByTestId("icon-button")).toBeDefined();
            expect(screen.getByTestId("toggle-markdown-button")).toBeDefined();
        });

        it("handles delete button click", async () => {
            const onDelete = vi.fn();
            const user = userEvent.setup();

            render(<FormTip {...editingProps} onDelete={onDelete} />);

            const deleteButton = screen.getByTestId("delete-button");

            await act(async () => {
                await user.click(deleteButton);
            });

            expect(onDelete).toHaveBeenCalledTimes(1);
        });

        it("renders text input with current value", () => {
            render(<FormTip {...editingProps} />);

            const textInputContainer = screen.getByTestId("text-input");
            const textInput = textInputContainer.querySelector("input") as HTMLInputElement;
            expect(textInput).toBeDefined();
            expect(textInput.value).toBe("This is a helpful tip");
            expect(textInput.getAttribute("placeholder")).toBe("Enter tip text...");
        });

        it("renders link input with current value", () => {
            const elementWithLink = {
                ...defaultElement,
                link: "https://example.com",
            };

            render(<FormTip {...editingProps} element={elementWithLink} />);

            const linkInputContainer = screen.getByTestId("link-input");
            const linkInput = linkInputContainer.querySelector("input") as HTMLInputElement;
            expect(linkInput).toBeDefined();
            expect(linkInput.value).toBe("https://example.com");
            expect(linkInput.getAttribute("placeholder")).toBe("Enter URL (optional)");
        });

        it("toggles between text and markdown modes", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();

            render(<FormTip {...editingProps} onUpdate={onUpdate} />);

            const toggleButton = screen.getByTestId("toggle-markdown-button");
            expect(toggleButton.textContent).toBe("Markdown");
            expect(toggleButton.getAttribute("aria-label")).toBe("Switch to markdown mode");

            await act(async () => {
                await user.click(toggleButton);
            });

            expect(onUpdate).toHaveBeenCalledTimes(1);
            expect(onUpdate).toHaveBeenCalledWith({ isMarkdown: true });
        });

        it("renders markdown input when in markdown mode", () => {
            const markdownElement = {
                ...defaultElement,
                isMarkdown: true,
            };

            render(<FormTip {...editingProps} element={markdownElement} />);

            expect(screen.queryByTestId("text-input")).toBeNull();
            expect(screen.getByTestId("markdown-input")).toBeDefined();

            const toggleButton = screen.getByTestId("toggle-markdown-button");
            expect(toggleButton.textContent).toBe("Text");
            expect(toggleButton.getAttribute("aria-label")).toBe("Switch to text mode");
        });

        it("opens icon selection popover", async () => {
            const user = userEvent.setup();
            render(<FormTip {...editingProps} />);

            const iconButton = screen.getByTestId("icon-button");
            expect(iconButton.getAttribute("aria-label")).toBe("Icon type: Default");

            await act(async () => {
                await user.click(iconButton);
            });

            // Check popover is open
            const popover = screen.getByTestId("icon-popover");
            expect(popover).toBeDefined();

            // Check icon options are present
            expect(screen.getByTestId("icon-option-Error")).toBeDefined();
            expect(screen.getByTestId("icon-option-Info")).toBeDefined();
            expect(screen.getByTestId("icon-option-Warning")).toBeDefined();
        });

        it("selects different icon from popover", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();

            render(<FormTip {...editingProps} onUpdate={onUpdate} />);

            // Open popover
            await act(async () => {
                await user.click(screen.getByTestId("icon-button"));
            });

            // Select Warning icon
            const warningOption = screen.getByTestId("icon-option-Warning");
            expect(warningOption.getAttribute("aria-label")).toBe("Select Warning icon");

            await act(async () => {
                await user.click(warningOption);
            });

            expect(onUpdate).toHaveBeenCalledTimes(1);
            expect(onUpdate).toHaveBeenCalledWith({ icon: "Warning" });
        });

        it("displays current icon type in button", () => {
            const elementWithWarning = {
                ...defaultElement,
                icon: "Warning" as const,
            };

            render(<FormTip {...editingProps} element={elementWithWarning} />);

            const iconButton = screen.getByTestId("icon-button");
            expect(iconButton.textContent).toContain("Icon: Warning");
            expect(iconButton.getAttribute("aria-label")).toBe("Icon type: Warning");
        });

        it("does not call onUpdate when not in editing mode", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();

            // Start in non-editing mode
            render(<FormTip {...defaultProps} onUpdate={onUpdate} />);

            // Try to interact with the component (shouldn't have any effect)
            const tip = screen.getByTestId("form-tip");
            await act(async () => {
                await user.click(tip);
            });

            expect(onUpdate).not.toHaveBeenCalled();
        });
    });

    describe("State transitions", () => {
        it("transitions from non-editing to editing mode", () => {
            const { rerender } = render(<FormTip {...defaultProps} />);

            // Check non-editing state
            expect(screen.getByTestId("form-tip").getAttribute("data-editing")).toBe("false");
            expect(screen.queryByTestId("delete-button")).toBeNull();

            // Switch to editing mode
            rerender(<FormTip {...defaultProps} isEditing={true} />);

            // Check editing state
            expect(screen.getByTestId("form-tip").getAttribute("data-editing")).toBe("true");
            expect(screen.getByTestId("delete-button")).toBeDefined();
            expect(screen.getByTestId("text-input")).toBeDefined();
        });

        it("transitions from editing to non-editing mode", () => {
            const { rerender } = render(<FormTip {...defaultProps} isEditing={true} />);

            // Check editing state
            expect(screen.getByTestId("form-tip").getAttribute("data-editing")).toBe("true");
            expect(screen.getByTestId("delete-button")).toBeDefined();

            // Switch to non-editing mode
            rerender(<FormTip {...defaultProps} isEditing={false} />);

            // Check non-editing state
            expect(screen.getByTestId("form-tip").getAttribute("data-editing")).toBe("false");
            expect(screen.queryByTestId("delete-button")).toBeNull();
        });

        it("preserves element data through mode transitions", () => {
            const complexElement: FormTipType = {
                ...defaultElement,
                icon: "Warning",
                link: "https://example.com",
                isMarkdown: true,
            };

            const { rerender } = render(
                <FormTip {...defaultProps} element={complexElement} />,
            );

            // Check initial state
            const tip = screen.getByTestId("form-tip");
            expect(tip.getAttribute("data-icon")).toBe("Warning");
            expect(tip.getAttribute("data-has-link")).toBe("true");
            expect(tip.getAttribute("data-is-markdown")).toBe("true");

            // Switch to editing and back
            rerender(<FormTip {...defaultProps} element={complexElement} isEditing={true} />);
            rerender(<FormTip {...defaultProps} element={complexElement} isEditing={false} />);

            // Check state is preserved
            const tipAfter = screen.getByTestId("form-tip");
            expect(tipAfter.getAttribute("data-icon")).toBe("Warning");
            expect(tipAfter.getAttribute("data-has-link")).toBe("true");
            expect(tipAfter.getAttribute("data-is-markdown")).toBe("true");
        });

        it("updates displayed content when element prop changes", () => {
            const { rerender } = render(<FormTip {...defaultProps} />);

            expect(screen.getByText("This is a helpful tip")).toBeDefined();

            // Update element
            const updatedElement = {
                ...defaultElement,
                label: "Updated tip content",
            };

            rerender(<FormTip {...defaultProps} element={updatedElement} />);

            expect(screen.queryByText("This is a helpful tip")).toBeNull();
            expect(screen.getByText("Updated tip content")).toBeDefined();
        });
    });
});
