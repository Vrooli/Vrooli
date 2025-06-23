import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { FormStructureType } from "@vrooli/shared";
import { FormQrCode } from "./FormQrCode";

describe("FormQrCode", () => {
    const mockElement = {
        id: "qr-1",
        fieldName: "qrCode",
        type: FormStructureType.QrCode,
        data: "https://example.com",
        description: "Sample QR Code",
        label: "QR Code",
    };

    const mockOnUpdate = vi.fn();
    const mockOnDelete = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Non-editing mode", () => {
        it("renders QR code display when not editing", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Check that display container is present
            expect(screen.getByTestId("qr-code-display")).toBeDefined();

            // Check that QR code image container is present
            expect(screen.getByTestId("qr-code-image")).toBeDefined();

            // Check that description is displayed
            expect(screen.getByTestId("qr-code-description")).toBeDefined();
            expect(screen.getByTestId("qr-code-description").textContent).toBe("Sample QR Code");

            // Check that editing controls are not present
            expect(screen.queryByTestId("qr-code-edit-button")).toBeNull();
            expect(screen.queryByTestId("qr-code-delete-button")).toBeNull();
        });

        it("renders without description when description is not provided", () => {
            const elementWithoutDescription = {
                ...mockElement,
                description: undefined,
            };

            render(
                <FormQrCode
                    element={elementWithoutDescription}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Check that QR code is displayed
            expect(screen.getByTestId("qr-code-image")).toBeDefined();

            // Check that description is not present
            expect(screen.queryByTestId("qr-code-description")).toBeNull();
        });

        it("does not render QR code when data is empty", () => {
            const elementWithoutData = {
                ...mockElement,
                data: "",
            };

            render(
                <FormQrCode
                    element={elementWithoutData}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Check that display container is present
            expect(screen.getByTestId("qr-code-display")).toBeDefined();

            // Check that QR code image is not present
            expect(screen.queryByTestId("qr-code-image")).toBeNull();
        });
    });

    describe("Editing mode (not actively editing)", () => {
        it("renders QR code with edit controls when in editing mode", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Check that edit container is present
            expect(screen.getByTestId("qr-code-edit-container")).toBeDefined();

            // Check that QR code is displayed
            expect(screen.getByTestId("qr-code-image")).toBeDefined();

            // Check that description is displayed
            expect(screen.getByTestId("qr-code-description")).toBeDefined();

            // Check that edit and delete buttons are present
            expect(screen.getByTestId("qr-code-edit-button")).toBeDefined();
            expect(screen.getByTestId("qr-code-delete-button")).toBeDefined();

            // Check accessibility
            expect(screen.getByTestId("qr-code-edit-button").getAttribute("aria-label")).toBe("Edit QR code");
            expect(screen.getByTestId("qr-code-delete-button").getAttribute("aria-label")).toBe("Delete QR code");
        });

        it("calls onDelete when delete button is clicked", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            const deleteButton = screen.getByTestId("qr-code-delete-button");
            fireEvent.click(deleteButton);

            expect(mockOnDelete).toHaveBeenCalledTimes(1);
        });

        it("switches to editing state when edit button is clicked", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            const editButton = screen.getByTestId("qr-code-edit-button");
            fireEvent.click(editButton);

            // Check that we're now in editing mode
            expect(screen.getByTestId("qr-code-editing")).toBeDefined();
            expect(screen.queryByTestId("qr-code-edit-container")).toBeNull();

            // Check that input fields are present
            expect(screen.getByTestId("qr-code-data-input")).toBeDefined();
            expect(screen.getByTestId("qr-code-description-input")).toBeDefined();
        });
    });

    describe("Actively editing mode", () => {
        it("renders editing form when edit button is clicked", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Check that editing container is present
            expect(screen.getByTestId("qr-code-editing")).toBeDefined();

            // Check that input fields are present with correct values
            const dataInputContainer = screen.getByTestId("qr-code-data-input");
            const descriptionInputContainer = screen.getByTestId("qr-code-description-input");
            
            // Get the actual input elements within the TextField containers
            const dataInput = dataInputContainer.querySelector("textarea");
            const descriptionInput = descriptionInputContainer.querySelector("input");

            expect(dataInput).toBeDefined();
            expect(dataInput?.value).toBe("https://example.com");

            expect(descriptionInput).toBeDefined();
            expect(descriptionInput?.value).toBe("Sample QR Code");

            // Check that buttons are present
            expect(screen.getByTestId("qr-code-cancel-button")).toBeDefined();
            expect(screen.getByTestId("qr-code-save-button")).toBeDefined();

            // Check accessibility
            expect(dataInput.getAttribute("aria-label")).toBe("QR code data");
            expect(descriptionInput.getAttribute("aria-label")).toBe("QR code description");
        });

        it("shows preview when data is entered", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Check that preview is shown
            expect(screen.getByTestId("qr-code-preview")).toBeDefined();
        });

        it("hides preview when data is cleared", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Clear the data input
            const dataInputContainer = screen.getByTestId("qr-code-data-input");
            const dataInput = dataInputContainer.querySelector("textarea");
            fireEvent.change(dataInput!, { target: { value: "" } });

            // Check that preview is not shown
            expect(screen.queryByTestId("qr-code-preview")).toBeNull();
        });

        it("updates input values when typing", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Update data input
            const dataInputContainer = screen.getByTestId("qr-code-data-input");
            const dataInput = dataInputContainer.querySelector("textarea");
            fireEvent.change(dataInput!, { target: { value: "https://newurl.com" } });
            expect(dataInput?.value).toBe("https://newurl.com");

            // Update description input
            const descriptionInputContainer = screen.getByTestId("qr-code-description-input");
            const descriptionInput = descriptionInputContainer.querySelector("input");
            fireEvent.change(descriptionInput!, { target: { value: "New description" } });
            expect(descriptionInput?.value).toBe("New description");
        });

        it("saves changes when save button is clicked", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Update inputs
            const dataInput = screen.getByTestId("qr-code-data-input").querySelector("textarea");
            const descriptionInput = screen.getByTestId("qr-code-description-input").querySelector("input");
            fireEvent.change(dataInput!, {
                target: { value: "https://newurl.com" },
            });
            fireEvent.change(descriptionInput!, {
                target: { value: "New description" },
            });

            // Click save
            fireEvent.click(screen.getByTestId("qr-code-save-button"));

            // Check that onUpdate was called with correct data
            expect(mockOnUpdate).toHaveBeenCalledTimes(1);
            expect(mockOnUpdate).toHaveBeenCalledWith({
                data: "https://newurl.com",
                description: "New description",
            });

            // Check that we're back to non-editing state
            expect(screen.getByTestId("qr-code-edit-container")).toBeDefined();
            expect(screen.queryByTestId("qr-code-editing")).toBeNull();
        });

        it("cancels changes when cancel button is clicked", () => {
            render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Update inputs
            const dataInput = screen.getByTestId("qr-code-data-input").querySelector("textarea");
            const descriptionInput = screen.getByTestId("qr-code-description-input").querySelector("input");
            fireEvent.change(dataInput!, {
                target: { value: "https://newurl.com" },
            });
            fireEvent.change(descriptionInput!, {
                target: { value: "New description" },
            });

            // Click cancel
            fireEvent.click(screen.getByTestId("qr-code-cancel-button"));

            // Check that onUpdate was not called
            expect(mockOnUpdate).not.toHaveBeenCalled();

            // Check that we're back to non-editing state
            expect(screen.getByTestId("qr-code-edit-container")).toBeDefined();
            expect(screen.queryByTestId("qr-code-editing")).toBeNull();

            // Check that original values are displayed
            expect(screen.getByTestId("qr-code-description").textContent).toBe("Sample QR Code");
        });

        it("handles empty description correctly", () => {
            const elementWithoutDescription = {
                ...mockElement,
                description: undefined,
            };

            render(
                <FormQrCode
                    element={elementWithoutDescription}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Check that description input is empty
            const descriptionInputContainer = screen.getByTestId("qr-code-description-input");
            const descriptionInput = descriptionInputContainer.querySelector("input");
            expect(descriptionInput?.value).toBe("");
        });
    });

    describe("State transitions", () => {
        it("transitions from non-editing to editing mode", () => {
            const { rerender } = render(
                <FormQrCode
                    element={mockElement}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Initially in non-editing mode
            expect(screen.getByTestId("qr-code-display")).toBeDefined();
            expect(screen.queryByTestId("qr-code-edit-container")).toBeNull();

            // Switch to editing mode
            rerender(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Now in editing mode
            expect(screen.queryByTestId("qr-code-display")).toBeNull();
            expect(screen.getByTestId("qr-code-edit-container")).toBeDefined();
        });

        it("transitions from editing to non-editing mode", () => {
            const { rerender } = render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Initially in editing mode
            expect(screen.getByTestId("qr-code-edit-container")).toBeDefined();
            expect(screen.queryByTestId("qr-code-display")).toBeNull();

            // Switch to non-editing mode
            rerender(
                <FormQrCode
                    element={mockElement}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Now in non-editing mode
            expect(screen.queryByTestId("qr-code-edit-container")).toBeNull();
            expect(screen.getByTestId("qr-code-display")).toBeDefined();
        });

        it("updates when element prop changes", () => {
            const { rerender } = render(
                <FormQrCode
                    element={mockElement}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Check initial description
            expect(screen.getByTestId("qr-code-description").textContent).toBe("Sample QR Code");

            // Update element
            const updatedElement = {
                ...mockElement,
                data: "https://updated.com",
                description: "Updated QR Code",
            };

            rerender(
                <FormQrCode
                    element={updatedElement}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Check updated description
            expect(screen.getByTestId("qr-code-description").textContent).toBe("Updated QR Code");
        });

        it("resets editing state when switching from editing to non-editing mode while actively editing", () => {
            const { rerender } = render(
                <FormQrCode
                    element={mockElement}
                    isEditing={true}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Click edit button to enter editing state
            fireEvent.click(screen.getByTestId("qr-code-edit-button"));

            // Verify we're in editing state
            expect(screen.getByTestId("qr-code-editing")).toBeDefined();

            // Change some values
            const dataInput = screen.getByTestId("qr-code-data-input").querySelector("textarea");
            fireEvent.change(dataInput!, {
                target: { value: "https://newurl.com" },
            });

            // Switch to non-editing mode
            rerender(
                <FormQrCode
                    element={mockElement}
                    isEditing={false}
                    onUpdate={mockOnUpdate}
                    onDelete={mockOnDelete}
                />
            );

            // Should be in display mode
            expect(screen.getByTestId("qr-code-display")).toBeDefined();
            expect(screen.queryByTestId("qr-code-editing")).toBeNull();

            // Original values should be displayed
            expect(screen.getByTestId("qr-code-description").textContent).toBe("Sample QR Code");
        });
    });
});