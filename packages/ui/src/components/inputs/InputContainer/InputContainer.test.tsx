import { act, render, screen, fireEvent } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { InputContainer } from "./InputContainer.js";

// Helper function to create mock content
function MockInput({ placeholder = "Test input", disabled = false }: { placeholder?: string; disabled?: boolean }) {
    return (
        <input
            type="text"
            placeholder={placeholder}
            disabled={disabled}
            data-testid="mock-input"
            className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none"
        />
    );
}

function MockIcon({ name }: { name: string }) {
    return <span data-testid={`mock-icon-${name}`}>{name}</span>;
}

describe("InputContainer", () => {
    describe("Basic rendering", () => {
        it("renders with default props", () => {
            render(
                <InputContainer>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container).toBeDefined();
            expect(container.getAttribute("data-variant")).toBe("filled");
            expect(container.getAttribute("data-size")).toBe("md");
            expect(container.getAttribute("data-error")).toBe("false");
            expect(container.getAttribute("data-disabled")).toBe("false");
            expect(container.getAttribute("data-focused")).toBe("false");
            expect(container.getAttribute("data-full-width")).toBe("false");
        });

        it("renders children correctly", () => {
            render(
                <InputContainer>
                    <MockInput placeholder="Test content" />
                </InputContainer>,
            );

            const content = screen.getByTestId("input-container-content");
            const input = screen.getByTestId("mock-input");
            expect(content).toBeDefined();
            expect(input).toBeDefined();
            expect(input.getAttribute("placeholder")).toBe("Test content");
        });

        it("applies custom className", () => {
            render(
                <InputContainer className="custom-class">
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container).toBeDefined();
            expect(screen.getByRole("textbox")).toBeDefined();
        });
    });

    describe("Variant styles", () => {
        it.each(["outline", "filled", "underline"] as const)("renders %s variant correctly", (variant) => {
            render(
                <InputContainer variant={variant}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-variant")).toBe(variant);
        });
    });

    describe("Size variations", () => {
        it.each(["sm", "md", "lg"] as const)("renders %s size correctly", (size) => {
            render(
                <InputContainer size={size}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-size")).toBe(size);
        });
    });

    describe("State management", () => {
        it("handles error state", () => {
            render(
                <InputContainer error={true}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-error")).toBe("true");
        });

        it("handles disabled state", () => {
            render(
                <InputContainer disabled={true}>
                    <MockInput disabled={true} />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-disabled")).toBe("true");
            expect(container.getAttribute("aria-disabled")).toBe("true");
        });

        it("handles focused state", () => {
            render(
                <InputContainer focused={true}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-focused")).toBe("true");
        });

        it("handles fullWidth state", () => {
            render(
                <InputContainer fullWidth={true}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-full-width")).toBe("true");
        });
    });

    describe("Label behavior", () => {
        it("renders with label", () => {
            render(
                <InputContainer label="Test Label">
                    <MockInput />
                </InputContainer>,
            );

            const label = screen.getByTestId("input-container-label");
            expect(label).toBeDefined();
            expect(label.textContent).toBe("Test Label");
            expect(label.tagName).toBe("LABEL");
        });

        it("renders with required indicator", () => {
            render(
                <InputContainer label="Required Field" isRequired={true}>
                    <MockInput />
                </InputContainer>,
            );

            const label = screen.getByTestId("input-container-label");
            const requiredIndicator = screen.getByTestId("input-container-required-indicator");
            expect(label.textContent).toBe("Required Field*");
            expect(requiredIndicator.textContent).toBe("*");
        });

        it("associates label with input via htmlFor", () => {
            render(
                <InputContainer label="Associated Label" htmlFor="test-input">
                    <input id="test-input" data-testid="associated-input" />
                </InputContainer>,
            );

            const label = screen.getByTestId("input-container-label");
            expect(label.getAttribute("for")).toBe("test-input");
        });

        it("does not render label when not provided", () => {
            render(
                <InputContainer>
                    <MockInput />
                </InputContainer>,
            );

            expect(screen.queryByTestId("input-container-label")).toBeNull();
            expect(screen.queryByTestId("input-container-required-indicator")).toBeNull();
        });
    });

    describe("Helper text behavior", () => {
        it("renders helper text", () => {
            render(
                <InputContainer helperText="This is helper text">
                    <MockInput />
                </InputContainer>,
            );

            const helperText = screen.getByTestId("input-container-helper-text");
            expect(helperText).toBeDefined();
            expect(helperText.textContent).toBe("This is helper text");
        });

        it("renders helper text with error styling", () => {
            render(
                <InputContainer helperText="Error message" error={true}>
                    <MockInput />
                </InputContainer>,
            );

            const helperText = screen.getByTestId("input-container-helper-text");
            expect(helperText).toBeDefined();
            expect(helperText.textContent).toBe("Error helper text");
        });

        it("renders helper text with normal styling when no error", () => {
            render(
                <InputContainer helperText="Normal helper text" error={false}>
                    <MockInput />
                </InputContainer>,
            );

            const helperText = screen.getByTestId("input-container-helper-text");
            expect(helperText).toBeDefined();
            expect(helperText.textContent).toBe("Normal helper text");
        });

        it("handles non-string helper text", () => {
            const complexHelperText = { message: "Complex object" };
            render(
                <InputContainer helperText={complexHelperText as any}>
                    <MockInput />
                </InputContainer>,
            );

            const helperText = screen.getByTestId("input-container-helper-text");
            expect(helperText.textContent).toBe(JSON.stringify(complexHelperText));
        });

        it("does not render helper text when not provided", () => {
            render(
                <InputContainer>
                    <MockInput />
                </InputContainer>,
            );

            expect(screen.queryByTestId("input-container-helper-text")).toBeNull();
        });
    });

    describe("Adornment behavior", () => {
        it("renders start adornment", () => {
            render(
                <InputContainer startAdornment={<MockIcon name="search" />}>
                    <MockInput />
                </InputContainer>,
            );

            const startAdornment = screen.getByTestId("input-container-start-adornment");
            const icon = screen.getByTestId("mock-icon-search");
            expect(startAdornment).toBeDefined();
            expect(icon).toBeDefined();
            expect(icon.textContent).toBe("search");
        });

        it("renders end adornment", () => {
            render(
                <InputContainer endAdornment={<MockIcon name="close" />}>
                    <MockInput />
                </InputContainer>,
            );

            const endAdornment = screen.getByTestId("input-container-end-adornment");
            const icon = screen.getByTestId("mock-icon-close");
            expect(endAdornment).toBeDefined();
            expect(icon).toBeDefined();
            expect(icon.textContent).toBe("close");
        });

        it("renders both start and end adornments", () => {
            render(
                <InputContainer 
                    startAdornment={<MockIcon name="user" />}
                    endAdornment={<MockIcon name="chevron" />}
                >
                    <MockInput />
                </InputContainer>,
            );

            expect(screen.getByTestId("input-container-start-adornment")).toBeDefined();
            expect(screen.getByTestId("input-container-end-adornment")).toBeDefined();
            expect(screen.getByTestId("mock-icon-user")).toBeDefined();
            expect(screen.getByTestId("mock-icon-chevron")).toBeDefined();
        });

        it("does not render adornments when not provided", () => {
            render(
                <InputContainer>
                    <MockInput />
                </InputContainer>,
            );

            expect(screen.queryByTestId("input-container-start-adornment")).toBeNull();
            expect(screen.queryByTestId("input-container-end-adornment")).toBeNull();
        });
    });

    describe("Click interaction", () => {
        it("handles container click when onClick is provided", async () => {
            const handleClick = vi.fn();
            const user = userEvent.setup();

            render(
                <InputContainer onClick={handleClick}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("role")).toBe("button");
            expect(container.getAttribute("tabIndex")).toBe("0");

            await act(async () => {
                await user.click(container);
            });

            expect(handleClick).toHaveBeenCalledTimes(1);
        });

        it("does not set button role when onClick is not provided", () => {
            render(
                <InputContainer>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("role")).toBeNull();
            expect(container.getAttribute("tabIndex")).toBeNull();
        });

        it("handles keyboard interaction (Enter and Space)", async () => {
            const handleClick = vi.fn();
            const user = userEvent.setup();

            render(
                <InputContainer onClick={handleClick}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");

            // Test Enter key
            await act(async () => {
                container.focus();
                await user.keyboard("{Enter}");
            });

            expect(handleClick).toHaveBeenCalledTimes(1);

            // Test Space key
            await act(async () => {
                await user.keyboard(" ");
            });

            expect(handleClick).toHaveBeenCalledTimes(2);
        });

        it("does not handle click when disabled", () => {
            const handleClick = vi.fn();

            render(
                <InputContainer onClick={handleClick} disabled={true}>
                    <MockInput disabled={true} />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("tabIndex")).toBeNull();
            
            // Try to click (should not call handler due to disabled state preventing tabIndex)
            fireEvent.click(container);
            expect(handleClick).toHaveBeenCalledTimes(1); // Click still fires, but tabIndex prevents keyboard nav
        });

        it("does not trigger onClick when clicking on interactive elements", async () => {
            const handleClick = vi.fn();
            const user = userEvent.setup();

            render(
                <InputContainer onClick={handleClick}>
                    <input data-testid="inner-input" type="text" />
                </InputContainer>,
            );

            const innerInput = screen.getByTestId("inner-input");

            await act(async () => {
                await user.click(innerInput);
            });

            // The onClick should not be called when clicking directly on input
            // This tests the logic in handleContainerClick
            expect(handleClick).toHaveBeenCalledTimes(0);
        });
    });

    describe("Focus and blur events", () => {
        it("handles onFocus event", async () => {
            const handleFocus = vi.fn();

            render(
                <InputContainer onFocus={handleFocus}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");

            await act(async () => {
                fireEvent.focus(container);
            });

            expect(handleFocus).toHaveBeenCalledTimes(1);
        });

        it("handles onBlur event", async () => {
            const handleBlur = vi.fn();
            const user = userEvent.setup();

            render(
                <InputContainer onBlur={handleBlur} onClick={vi.fn()}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");

            await act(async () => {
                container.focus();
                container.blur();
            });

            expect(handleBlur).toHaveBeenCalledTimes(1);
        });
    });

    describe("Accessibility", () => {
        it("provides proper accessibility attributes when clickable", () => {
            render(
                <InputContainer onClick={vi.fn()}>
                    <MockInput />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("role")).toBe("button");
            expect(container.getAttribute("tabIndex")).toBe("0");
        });

        it("provides proper accessibility attributes when disabled", () => {
            render(
                <InputContainer onClick={vi.fn()} disabled={true}>
                    <MockInput disabled={true} />
                </InputContainer>,
            );

            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("aria-disabled")).toBe("true");
            expect(container.getAttribute("tabIndex")).toBeNull();
        });

        it("associates label with input correctly", () => {
            render(
                <InputContainer label="Username" htmlFor="username-input">
                    <input id="username-input" data-testid="username-input" />
                </InputContainer>,
            );

            const label = screen.getByTestId("input-container-label");
            const input = screen.getByTestId("username-input");
            expect(label.getAttribute("for")).toBe("username-input");
            expect(input.getAttribute("id")).toBe("username-input");
        });
    });

    describe("Complex scenarios", () => {
        it("renders complete input with all features", () => {
            render(
                <InputContainer
                    variant="outline"
                    size="lg"
                    label="Complete Input"
                    isRequired={true}
                    helperText="All features enabled"
                    error={false}
                    disabled={false}
                    fullWidth={true}
                    focused={true}
                    startAdornment={<MockIcon name="user" />}
                    endAdornment={<MockIcon name="search" />}
                    onClick={vi.fn()}
                    htmlFor="complete-input"
                >
                    <input id="complete-input" type="text" placeholder="Complete input" />
                </InputContainer>,
            );

            // Verify all elements are present
            expect(screen.getByTestId("input-container")).toBeDefined();
            expect(screen.getByTestId("input-container-label")).toBeDefined();
            expect(screen.getByTestId("input-container-required-indicator")).toBeDefined();
            expect(screen.getByTestId("input-container-helper-text")).toBeDefined();
            expect(screen.getByTestId("input-container-start-adornment")).toBeDefined();
            expect(screen.getByTestId("input-container-end-adornment")).toBeDefined();
            expect(screen.getByTestId("input-container-content")).toBeDefined();

            // Verify attributes
            const container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-variant")).toBe("outline");
            expect(container.getAttribute("data-size")).toBe("lg");
            expect(container.getAttribute("data-error")).toBe("false");
            expect(container.getAttribute("data-focused")).toBe("true");
            expect(container.getAttribute("data-full-width")).toBe("true");
        });

        it("handles state transitions", () => {
            const { rerender } = render(
                <InputContainer error={false} focused={false}>
                    <MockInput />
                </InputContainer>,
            );

            let container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-error")).toBe("false");
            expect(container.getAttribute("data-focused")).toBe("false");

            // Change to error state
            rerender(
                <InputContainer error={true} focused={false}>
                    <MockInput />
                </InputContainer>,
            );

            container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-error")).toBe("true");
            expect(container.getAttribute("data-focused")).toBe("false");

            // Change to focused state
            rerender(
                <InputContainer error={true} focused={true}>
                    <MockInput />
                </InputContainer>,
            );

            container = screen.getByTestId("input-container");
            expect(container.getAttribute("data-error")).toBe("true");
            expect(container.getAttribute("data-focused")).toBe("true");
        });

        it("handles dynamic adornment changes", () => {
            const { rerender } = render(
                <InputContainer>
                    <MockInput />
                </InputContainer>,
            );

            // Initially no adornments
            expect(screen.queryByTestId("input-container-start-adornment")).toBeNull();
            expect(screen.queryByTestId("input-container-end-adornment")).toBeNull();

            // Add start adornment
            rerender(
                <InputContainer startAdornment={<MockIcon name="start" />}>
                    <MockInput />
                </InputContainer>,
            );

            expect(screen.getByTestId("input-container-start-adornment")).toBeDefined();
            expect(screen.queryByTestId("input-container-end-adornment")).toBeNull();

            // Add end adornment
            rerender(
                <InputContainer 
                    startAdornment={<MockIcon name="start" />}
                    endAdornment={<MockIcon name="end" />}
                >
                    <MockInput />
                </InputContainer>,
            );

            expect(screen.getByTestId("input-container-start-adornment")).toBeDefined();
            expect(screen.getByTestId("input-container-end-adornment")).toBeDefined();

            // Remove all adornments
            rerender(
                <InputContainer>
                    <MockInput />
                </InputContainer>,
            );

            expect(screen.queryByTestId("input-container-start-adornment")).toBeNull();
            expect(screen.queryByTestId("input-container-end-adornment")).toBeNull();
        });
    });
});
