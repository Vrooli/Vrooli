import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormDivider } from "./FormDivider";

describe("FormDivider", () => {
    describe("Non-editing mode", () => {
        it("renders as a non-interactive divider", () => {
            render(<FormDivider isEditing={false} onDelete={vi.fn()} />);

            const divider = screen.getByTestId("form-divider");
            expect(divider).toBeDefined();
            expect(divider.getAttribute("data-editing")).toBe("false");
            expect(divider.getAttribute("role")).toBe("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("horizontal");
        });

        it("does not provide any interactive elements", () => {
            render(<FormDivider isEditing={false} onDelete={vi.fn()} />);

            expect(screen.queryByRole("button")).toBeNull();
            expect(screen.queryByText("Delete")).toBeNull();
        });
    });

    describe("Editing mode", () => {
        it("renders with delete functionality", () => {
            render(<FormDivider isEditing={true} onDelete={vi.fn()} />);

            const divider = screen.getByTestId("form-divider");
            expect(divider.getAttribute("data-editing")).toBe("true");

            const deleteButton = screen.getByRole("button", { name: "Delete divider" });
            expect(deleteButton).toBeDefined();
            expect(deleteButton.textContent).toBe("Delete");
        });

        it("handles delete interaction", async () => {
            const onDelete = vi.fn();
            const user = userEvent.setup();

            render(<FormDivider isEditing={true} onDelete={onDelete} />);

            const deleteButton = screen.getByRole("button", { name: "Delete divider" });

            await act(async () => {
                await user.click(deleteButton);
            });

            expect(onDelete).toHaveBeenCalledTimes(1);
        });

        it("maintains separator semantics while editable", () => {
            render(<FormDivider isEditing={true} onDelete={vi.fn()} />);

            const divider = screen.getByTestId("form-divider");
            expect(divider.getAttribute("role")).toBe("separator");
            expect(divider.getAttribute("aria-orientation")).toBe("horizontal");
        });
    });

    describe("State transitions", () => {
        it("toggles between editing and non-editing modes", () => {
            const { rerender } = render(<FormDivider isEditing={false} onDelete={vi.fn()} />);

            // Start in non-editing mode
            expect(screen.getByTestId("form-divider").getAttribute("data-editing")).toBe("false");
            expect(screen.queryByRole("button")).toBeNull();

            // Switch to editing mode
            rerender(<FormDivider isEditing={true} onDelete={vi.fn()} />);

            expect(screen.getByTestId("form-divider").getAttribute("data-editing")).toBe("true");
            expect(screen.getByRole("button", { name: "Delete divider" })).toBeDefined();

            // Switch back to non-editing mode
            rerender(<FormDivider isEditing={false} onDelete={vi.fn()} />);

            expect(screen.getByTestId("form-divider").getAttribute("data-editing")).toBe("false");
            expect(screen.queryByRole("button")).toBeNull();
        });
    });
});