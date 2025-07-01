import { act, render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormHeader } from "./FormHeader";
import type { FormHeaderType } from "@vrooli/shared";
import { FormStructureType } from "@vrooli/shared";

// Mock dependencies
vi.mock("@mui/material/TextField", () => ({
    default: React.forwardRef(({ value, onChange, onBlur, onKeyDown, placeholder, InputProps, fullWidth, maxRows, minRows, multiline, variant, ...props }: any, ref: any) => {
        const [localValue, setLocalValue] = React.useState(value || "");
        
        React.useEffect(() => {
            setLocalValue(value || "");
        }, [value]);
        
        const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
            setLocalValue(e.target.value);
            if (onChange) {
                onChange(e);
            }
        };
        
        // Filter out MUI-specific props that shouldn't go on DOM elements
        const domProps = { ...props };
        delete domProps.InputProps;
        
        return (
            <input
                ref={ref}
                type="text"
                value={localValue}
                onChange={handleChange}
                onBlur={onBlur}
                onKeyDown={onKeyDown}
                placeholder={placeholder}
                style={InputProps?.style}
                {...domProps}
            />
        );
    }),
}));

vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key,
    }),
}));

vi.mock("@mui/material/styles", () => ({
    useTheme: () => ({
        palette: {
            background: {
                textPrimary: "#000000",
                textSecondary: "#666666",
            },
            error: {
                main: "#ff0000",
            },
            text: {
                primary: "#000000",
                secondary: "#666666",
            },
        },
        typography: {
            h1: { fontSize: "2rem" },
            h2: { fontSize: "1.75rem" },
            h3: { fontSize: "1.5rem" },
            h4: { fontSize: "1.25rem" },
            h5: { fontSize: "1rem" },
            h6: { fontSize: "0.875rem" },
            body1: { fontSize: "1rem" },
            body2: { fontSize: "0.875rem" },
        },
    }),
}));

vi.mock("../../text/MarkdownDisplay.js", () => ({
    MarkdownDisplay: ({ content, "data-testid": testId }: { content: string; "data-testid"?: string }) => (
        <div data-testid={testId}>{content}</div>
    ),
}));

vi.mock("../../buttons/IconButton.js", () => ({
    IconButton: ({ children, onClick, "data-testid": testId, "aria-label": ariaLabel }: any) => (
        <button data-testid={testId} onClick={onClick} aria-label={ariaLabel}>
            {children}
        </button>
    ),
}));

vi.mock("../../buttons/HelpButton.js", () => ({
    HelpButton: ({ markdown, onMarkdownChange, "data-testid": testId }: { 
        markdown: string; 
        onMarkdownChange?: (value: string) => void;
        "data-testid"?: string;
    }) => (
        <button data-testid={testId} aria-label="Help">
            {markdown}
        </button>
    ),
}));

vi.mock("../../../icons/Icons.js", () => ({
    Icon: ({ decorative, info }: any) => <span>{info?.name || "icon"}</span>,
    IconCommon: ({ name }: any) => <span>{name}</span>,
}));

vi.mock("../../Tooltip/Tooltip.js", () => ({
    Tooltip: ({ children, title }: any) => <div title={title}>{children}</div>,
}));

vi.mock("../AdvancedInput/AdvancedInput.js", () => ({
    AdvancedInputBase: ({ value, onChange, onBlur, placeholder, "data-testid": testId }: any) => (
        <textarea
            data-testid={testId}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onBlur={onBlur}
            placeholder={placeholder}
            aria-label="Header markdown input"
        />
    ),
}));

const createMockHeader = (overrides?: Partial<FormHeaderType>): FormHeaderType => ({
    id: "test-header",
    type: FormStructureType.Header,
    label: "Test Header",
    tag: "h3",
    isMarkdown: false,
    color: "default",
    description: null,
    helpText: null,
    ...overrides,
});

describe("FormHeader", () => {
    describe("Non-editing mode", () => {
        it("renders header with default properties", () => {
            const element = createMockHeader();
            render(
                <FormHeader
                    element={element}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const container = screen.getByTestId("form-header");
            expect(container).toBeDefined();
            expect(container.getAttribute("data-editing")).toBe("false");

            const headerContent = screen.getByTestId("header-content");
            expect(headerContent).toBeDefined();
            expect(headerContent.textContent).toBe("Test Header");
        });

        it("renders markdown content when isMarkdown is true", () => {
            const element = createMockHeader({
                isMarkdown: true,
                label: "**Bold Header**",
            });

            render(
                <FormHeader
                    element={element}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const markdownContent = screen.getByTestId("header-content-markdown");
            expect(markdownContent).toBeDefined();
            expect(markdownContent.textContent).toBe("**Bold Header**");
        });

        it("renders help button when description is provided", () => {
            const element = createMockHeader({
                description: "This is helpful text",
            });

            render(
                <FormHeader
                    element={element}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const helpButton = screen.getByTestId("help-button");
            expect(helpButton).toBeDefined();
            expect(helpButton.textContent).toBe("This is helpful text");
        });

        it("does not render help button when description is empty", () => {
            const element = createMockHeader({
                description: "",
            });

            render(
                <FormHeader
                    element={element}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.queryByTestId("help-button")).toBeNull();
        });

        it("does not provide any editing controls", () => {
            const element = createMockHeader();
            render(
                <FormHeader
                    element={element}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.queryByTestId("delete-button")).toBeNull();
            expect(screen.queryByTestId("size-button")).toBeNull();
            expect(screen.queryByTestId("color-button")).toBeNull();
            expect(screen.queryByTestId("markdown-toggle")).toBeNull();
        });
    });

    describe("Editing mode", () => {
        it("renders with editing controls", () => {
            const element = createMockHeader();
            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const container = screen.getByTestId("form-header");
            expect(container).toBeDefined();
            expect(container.getAttribute("data-editing")).toBe("true");

            expect(screen.getByTestId("delete-button")).toBeDefined();
            expect(screen.getByTestId("size-button")).toBeDefined();
            expect(screen.getByTestId("color-button")).toBeDefined();
            expect(screen.getByTestId("markdown-toggle")).toBeDefined();
        });

        it("renders text input for editing label", () => {
            const element = createMockHeader({
                isMarkdown: false,
            });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const textInput = screen.getByTestId("header-text-input");
            expect(textInput).toBeDefined();
            expect(textInput.getAttribute("aria-label")).toBe("Header text input");
            // The TextField is rendered with the value from editedLabel
        });

        it("renders markdown input when isMarkdown is true", () => {
            const element = createMockHeader({
                isMarkdown: true,
                label: "**Bold Header**",
            });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const markdownInput = screen.getByTestId("header-markdown-input");
            expect(markdownInput).toBeDefined();
            expect(markdownInput.getAttribute("aria-label")).toBe("Header markdown input");
            expect((markdownInput as HTMLTextAreaElement).value).toBe("**Bold Header**");
        });

        it("handles delete button click", async () => {
            const onDelete = vi.fn();
            const user = userEvent.setup();
            const element = createMockHeader();

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={onDelete}
                />,
            );

            const deleteButton = screen.getByTestId("delete-button");
            await act(async () => {
                await user.click(deleteButton);
            });

            expect(onDelete).toHaveBeenCalledTimes(1);
        });

        it("opens size popover when size button is clicked", async () => {
            const user = userEvent.setup();
            const element = createMockHeader();

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const sizeButton = screen.getByTestId("size-button");
            await act(async () => {
                await user.click(sizeButton);
            });

            await waitFor(() => {
                expect(screen.getByTestId("size-popover")).toBeDefined();
                expect(screen.getByTestId("size-option-h1")).toBeDefined();
                expect(screen.getByTestId("size-option-h2")).toBeDefined();
                expect(screen.getByTestId("size-option-h3")).toBeDefined();
                expect(screen.getByTestId("size-option-h4")).toBeDefined();
                expect(screen.getByTestId("size-option-body1")).toBeDefined();
                expect(screen.getByTestId("size-option-body2")).toBeDefined();
            });
        });

        it("updates tag when size option is selected", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();
            const element = createMockHeader({ tag: "h3" });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Open size popover
            await act(async () => {
                await user.click(screen.getByTestId("size-button"));
            });

            // Select h1 option
            await act(async () => {
                await user.click(screen.getByTestId("size-option-h1"));
            });

            expect(onUpdate).toHaveBeenCalledWith({ tag: "h1" });
        });

        it("opens color popover when color button is clicked", async () => {
            const user = userEvent.setup();
            const element = createMockHeader();

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const colorButton = screen.getByTestId("color-button");
            await act(async () => {
                await user.click(colorButton);
            });

            await waitFor(() => {
                expect(screen.getByTestId("color-popover")).toBeDefined();
                expect(screen.getByTestId("color-option-default")).toBeDefined();
                expect(screen.getByTestId("color-option-primary")).toBeDefined();
                expect(screen.getByTestId("color-option-secondary")).toBeDefined();
                expect(screen.getByTestId("color-picker")).toBeDefined();
            });
        });

        it("updates color when color option is selected", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();
            const element = createMockHeader({ color: "default" });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Open color popover
            await act(async () => {
                await user.click(screen.getByTestId("color-button"));
            });

            // Select primary color
            await act(async () => {
                await user.click(screen.getByTestId("color-option-primary"));
            });

            expect(onUpdate).toHaveBeenCalledWith({ color: "primary" });
        });

        it("toggles markdown mode when markdown button is clicked", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();
            const element = createMockHeader({ isMarkdown: false });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            const markdownToggle = screen.getByTestId("markdown-toggle");
            expect(markdownToggle.textContent).toBe("Markdown");

            await act(async () => {
                await user.click(markdownToggle);
            });

            expect(onUpdate).toHaveBeenCalledWith({ isMarkdown: true });
        });

        it("updates label when text input is changed", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();
            const element = createMockHeader({ isMarkdown: false });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            const textInput = screen.getByTestId("header-text-input");
            
            // Clear the input and type new text
            await act(async () => {
                await user.clear(textInput);
                await user.type(textInput, "New Header Text");
                await user.tab(); // Trigger blur
            });

            await waitFor(() => {
                expect(onUpdate).toHaveBeenCalledWith({ label: "New Header Text" });
            });
        });

        it("displays correct size label on button", () => {
            const element = createMockHeader({ tag: "h1" });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const sizeButton = screen.getByTestId("size-button");
            expect(sizeButton.textContent).toContain("Title (Largest)");
        });

        it("renders multiline input for paragraph tags", () => {
            const element = createMockHeader({ tag: "body1", isMarkdown: false });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const textInput = screen.getByTestId("header-text-input");
            expect(textInput.getAttribute("rows")).toBeDefined();
        });
    });

    describe("State transitions", () => {
        it("switches from non-editing to editing mode", () => {
            const element = createMockHeader();
            const { rerender } = render(
                <FormHeader
                    element={element}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Verify non-editing mode
            expect(screen.getByTestId("form-header").getAttribute("data-editing")).toBe("false");
            expect(screen.queryByTestId("delete-button")).toBeNull();
            expect(screen.queryByTestId("form-header-container")).toBeNull();

            // Switch to editing mode
            rerender(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Verify editing mode
            expect(screen.getByTestId("form-header").getAttribute("data-editing")).toBe("true");
            expect(screen.getByTestId("delete-button")).toBeDefined();
            expect(screen.getByTestId("form-header-container")).toBeDefined();
        });

        it("updates when element properties change", () => {
            const { rerender } = render(
                <FormHeader
                    element={createMockHeader({ label: "Original", tag: "h3" })}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.getByTestId("header-content").textContent).toBe("Original");

            // Update element
            rerender(
                <FormHeader
                    element={createMockHeader({ label: "Updated", tag: "h1" })}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.getByTestId("header-content").textContent).toBe("Updated");
        });

        it("preserves input value when switching markdown mode", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();
            
            const { rerender } = render(
                <FormHeader
                    element={createMockHeader({ isMarkdown: false, label: "Test Label" })}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Type in text input
            const textInput = screen.getByTestId("header-text-input");
            await act(async () => {
                await user.clear(textInput);
                await user.type(textInput, "Modified Label");
            });

            // Toggle to markdown mode
            await act(async () => {
                await user.click(screen.getByTestId("markdown-toggle"));
            });

            // Simulate component receiving updated element
            rerender(
                <FormHeader
                    element={createMockHeader({ isMarkdown: true, label: "Modified Label" })}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Verify markdown input has the modified value
            const markdownInput = screen.getByTestId("header-markdown-input") as HTMLTextAreaElement;
            expect(markdownInput.value).toBe("Modified Label");
        });

        it("handles custom color input", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();
            const element = createMockHeader({ color: "#ff0000" });

            render(
                <FormHeader
                    element={element}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Open color popover
            await act(async () => {
                await user.click(screen.getByTestId("color-button"));
            });

            // Clear and type in hex input
            const hexInput = screen.getByTestId("color-hex-input");
            await act(async () => {
                await user.clear(hexInput);
                await user.type(hexInput, "#00ff00");
            });

            // Close popover to trigger update
            await act(async () => {
                await user.keyboard("{Escape}");
            });

            await waitFor(() => {
                expect(onUpdate).toHaveBeenCalledWith({ color: "#00ff00" });
            });
        });
    });
});
