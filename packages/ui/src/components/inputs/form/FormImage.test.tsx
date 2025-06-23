import { act, fireEvent, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormImage } from "./FormImage.js";
import type { FormImageType } from "@vrooli/shared";

// Mock the translation hook
vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key,
    }),
}));

// Mock MUI theme
vi.mock("@mui/material/styles", () => ({
    useTheme: () => ({
        palette: {
            background: { paper: "#ffffff" },
            divider: "#e0e0e0",
            text: { disabled: "#999999" },
            error: { main: "#ff0000" },
        },
    }),
}));

// Mock TextField to properly handle onChange
vi.mock("@mui/material/TextField", () => ({
    default: ({ onChange, value, inputProps, ...props }: any) => {
        const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
            onChange(e);
        };
        return (
            <div data-testid={props["data-testid"]}>
                <input
                    value={value || ""}
                    onChange={handleChange}
                    {...inputProps}
                />
            </div>
        );
    },
}));

// Mock child components
vi.mock("../Dropzone/Dropzone.js", () => ({
    Dropzone: ({ onDrop, dropzoneText, ...props }: any) => (
        <div data-testid={props["data-testid"]} onClick={() => onDrop([new File(["test"], "test.png", { type: "image/png" })])}>
            {dropzoneText}
        </div>
    ),
}));

vi.mock("../LinkInput/LinkInput.js", () => ({
    LinkInput: ({ onChange, value, ...props }: any) => {
        const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
            onChange(e.target.value);
        };
        return (
            <input
                data-testid={props["data-testid"]}
                value={value || ""}
                onChange={handleChange}
                aria-label={props.label}
            />
        );
    },
}));

describe("FormImage", () => {
    const mockElement: FormImageType = {
        id: "test-id",
        label: "Test Image",
        url: "https://example.com/image.jpg",
        type: "image" as any,
    };

    const mockElementNoUrl: FormImageType = {
        id: "test-id-no-url",
        label: "Test Image No URL",
        url: "",
        type: "image" as any,
    };

    describe("Non-editing mode", () => {
        it("renders image when URL exists", () => {
            render(
                <FormImage
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const imageDisplay = screen.getByTestId("form-image-display");
            expect(imageDisplay).toBeDefined();

            const img = screen.getByTestId("form-image-img");
            expect(img).toBeDefined();
            expect(img.getAttribute("src")).toBe("https://example.com/image.jpg");
            expect(img.getAttribute("alt")).toBe("Test Image");
        });

        it("renders nothing when URL is empty", () => {
            const { container } = render(
                <FormImage
                    element={mockElementNoUrl}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(container.firstChild).toBeNull();
            expect(screen.queryByTestId("form-image-display")).toBeNull();
            expect(screen.queryByTestId("form-image-placeholder")).toBeNull();
        });

        it("uses fallback alt text when label is empty", () => {
            const elementNoLabel = { ...mockElement, label: "" };
            render(
                <FormImage
                    element={elementNoLabel}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const img = screen.getByTestId("form-image-img");
            expect(img.getAttribute("alt")).toBe("Image");
        });

        it("does not render any controls in non-editing mode", () => {
            render(
                <FormImage
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.queryByRole("button")).toBeNull();
            expect(screen.queryByTestId("form-image-settings-button")).toBeNull();
            expect(screen.queryByTestId("form-image-delete-button")).toBeNull();
        });
    });

    describe("Editing mode", () => {
        it("renders image with controls when URL exists", () => {
            render(
                <FormImage
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const container = screen.getByTestId("form-image-container");
            expect(container).toBeDefined();

            const imageDisplay = screen.getByTestId("form-image-display");
            expect(imageDisplay).toBeDefined();

            const img = screen.getByTestId("form-image-img");
            expect(img).toBeDefined();
            expect(img.getAttribute("src")).toBe("https://example.com/image.jpg");

            const settingsButton = screen.getByTestId("form-image-settings-button");
            expect(settingsButton).toBeDefined();
            expect(settingsButton.getAttribute("aria-label")).toBe("Toggle image settings");
            expect(settingsButton.getAttribute("aria-expanded")).toBe("false");

            const deleteButton = screen.getByTestId("form-image-delete-button");
            expect(deleteButton).toBeDefined();
            expect(deleteButton.getAttribute("aria-label")).toBe("Delete image");
        });

        it("renders placeholder with controls when URL is empty", () => {
            render(
                <FormImage
                    element={mockElementNoUrl}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const placeholder = screen.getByTestId("form-image-placeholder");
            expect(placeholder).toBeDefined();
            expect(placeholder.textContent).toBe("🖼️");

            expect(screen.queryByTestId("form-image-img")).toBeNull();

            const settingsButton = screen.getByTestId("form-image-settings-button");
            expect(settingsButton).toBeDefined();

            const deleteButton = screen.getByTestId("form-image-delete-button");
            expect(deleteButton).toBeDefined();
        });

        it("handles delete button click", async () => {
            const onDelete = vi.fn();
            const user = userEvent.setup();

            render(
                <FormImage
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={onDelete}
                />,
            );

            const deleteButton = screen.getByTestId("form-image-delete-button");

            await act(async () => {
                await user.click(deleteButton);
            });

            expect(onDelete).toHaveBeenCalledTimes(1);
        });

        it("toggles settings panel", async () => {
            const user = userEvent.setup();

            render(
                <FormImage
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const settingsButton = screen.getByTestId("form-image-settings-button");

            // Initially settings are hidden
            expect(screen.queryByTestId("form-image-settings")).toBeNull();
            expect(settingsButton.getAttribute("aria-expanded")).toBe("false");

            // Click to show settings
            await act(async () => {
                await user.click(settingsButton);
            });

            expect(screen.getByTestId("form-image-settings")).toBeDefined();
            expect(settingsButton.getAttribute("aria-expanded")).toBe("true");

            // Click again to hide settings
            await act(async () => {
                await user.click(settingsButton);
            });

            expect(screen.queryByTestId("form-image-settings")).toBeNull();
            expect(settingsButton.getAttribute("aria-expanded")).toBe("false");
        });

        describe("Settings panel", () => {
            it("shows input mode toggle buttons", async () => {
                const user = userEvent.setup();

                render(
                    <FormImage
                        element={mockElement}
                        isEditing={true}
                        onUpdate={vi.fn()}
                        onDelete={vi.fn()}
                    />,
                );

                // Open settings
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-settings-button"));
                });

                const urlModeButton = screen.getByTestId("form-image-url-mode-button");
                const uploadModeButton = screen.getByTestId("form-image-upload-mode-button");

                expect(urlModeButton).toBeDefined();
                expect(urlModeButton.getAttribute("aria-label")).toBe("Use image URL");
                expect(urlModeButton.getAttribute("aria-pressed")).toBe("true"); // Default mode

                expect(uploadModeButton).toBeDefined();
                expect(uploadModeButton.getAttribute("aria-label")).toBe("Upload image");
                expect(uploadModeButton.getAttribute("aria-pressed")).toBe("false");
            });

            it("switches between URL and upload modes", async () => {
                const user = userEvent.setup();

                render(
                    <FormImage
                        element={mockElement}
                        isEditing={true}
                        onUpdate={vi.fn()}
                        onDelete={vi.fn()}
                    />,
                );

                // Open settings
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-settings-button"));
                });

                // Initially in URL mode
                expect(screen.getByTestId("form-image-url-input")).toBeDefined();
                expect(screen.queryByTestId("form-image-dropzone")).toBeNull();

                // Switch to upload mode
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-upload-mode-button"));
                });

                expect(screen.queryByTestId("form-image-url-input")).toBeNull();
                expect(screen.getByTestId("form-image-dropzone")).toBeDefined();
                expect(screen.getByTestId("form-image-upload-mode-button").getAttribute("aria-pressed")).toBe("true");
                expect(screen.getByTestId("form-image-url-mode-button").getAttribute("aria-pressed")).toBe("false");

                // Switch back to URL mode
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-url-mode-button"));
                });

                expect(screen.getByTestId("form-image-url-input")).toBeDefined();
                expect(screen.queryByTestId("form-image-dropzone")).toBeNull();
            });

            it("handles URL input changes", async () => {
                const onUpdate = vi.fn();
                const user = userEvent.setup();

                render(
                    <FormImage
                        element={mockElement}
                        isEditing={true}
                        onUpdate={onUpdate}
                        onDelete={vi.fn()}
                    />,
                );

                // Open settings
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-settings-button"));
                });

                const urlInput = screen.getByTestId("form-image-url-input") as HTMLInputElement;
                expect(urlInput.value).toBe("https://example.com/image.jpg");

                // Simulate changing the URL using fireEvent
                fireEvent.change(urlInput, { target: { value: "https://newurl.com/image.png" } });

                // Check if called with the new URL
                expect(onUpdate).toHaveBeenCalledWith({ url: "https://newurl.com/image.png" });
            });

            it("handles file upload", async () => {
                const onUpdate = vi.fn();
                const user = userEvent.setup();

                render(
                    <FormImage
                        element={mockElementNoUrl}
                        isEditing={true}
                        onUpdate={onUpdate}
                        onDelete={vi.fn()}
                    />,
                );

                // Open settings
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-settings-button"));
                });

                // Wait for settings to be visible
                expect(screen.getByTestId("form-image-settings")).toBeDefined();

                // Switch to upload mode
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-upload-mode-button"));
                });

                const dropzone = screen.getByTestId("form-image-dropzone");

                // Simulate file drop (mocked to automatically call onDrop)
                await act(async () => {
                    await user.click(dropzone);
                });

                expect(onUpdate).toHaveBeenCalledTimes(1);
                expect(onUpdate).toHaveBeenCalledWith({
                    url: expect.stringMatching(/^blob:/),
                });
            });

            it("handles alt text changes", async () => {
                const onUpdate = vi.fn();
                const user = userEvent.setup();

                render(
                    <FormImage
                        element={mockElement}
                        isEditing={true}
                        onUpdate={onUpdate}
                        onDelete={vi.fn()}
                    />,
                );

                // Open settings
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-settings-button"));
                });

                const altTextContainer = screen.getByTestId("form-image-alt-text");
                expect(altTextContainer).toBeDefined();
                
                // Get the actual input element inside the TextField
                const altTextInput = altTextContainer.querySelector("input") as HTMLInputElement;
                expect(altTextInput).toBeDefined();
                expect(altTextInput.value).toBe("Test Image");
                expect(altTextInput.getAttribute("aria-label")).toBe("Alt text for image");

                // Change alt text using fireEvent
                fireEvent.change(altTextInput, { target: { value: "New alt text" } });

                expect(onUpdate).toHaveBeenCalledWith({ label: "New alt text" });
            });

            it("shows alt text field in both URL and upload modes", async () => {
                const user = userEvent.setup();

                render(
                    <FormImage
                        element={mockElement}
                        isEditing={true}
                        onUpdate={vi.fn()}
                        onDelete={vi.fn()}
                    />,
                );

                // Open settings
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-settings-button"));
                });

                // Check in URL mode
                expect(screen.getByTestId("form-image-alt-text")).toBeDefined();

                // Switch to upload mode
                await act(async () => {
                    await user.click(screen.getByTestId("form-image-upload-mode-button"));
                });

                // Alt text should still be present
                expect(screen.getByTestId("form-image-alt-text")).toBeDefined();
            });
        });
    });

    describe("State transitions", () => {
        it("toggles between editing and non-editing modes", () => {
            const { rerender } = render(
                <FormImage
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Non-editing mode: only image visible
            expect(screen.getByTestId("form-image-display")).toBeDefined();
            expect(screen.queryByTestId("form-image-container")).toBeNull();
            expect(screen.queryByRole("button")).toBeNull();

            // Switch to editing mode
            rerender(
                <FormImage
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Editing mode: container with controls visible
            expect(screen.getByTestId("form-image-container")).toBeDefined();
            expect(screen.getByTestId("form-image-display")).toBeDefined();
            expect(screen.getByTestId("form-image-settings-button")).toBeDefined();
            expect(screen.getByTestId("form-image-delete-button")).toBeDefined();

            // Switch back to non-editing mode
            rerender(
                <FormImage
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.getByTestId("form-image-display")).toBeDefined();
            expect(screen.queryByTestId("form-image-container")).toBeNull();
            expect(screen.queryByRole("button")).toBeNull();
        });

        it("preserves settings panel state when element props change", async () => {
            const user = userEvent.setup();
            const { rerender } = render(
                <FormImage
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings
            await act(async () => {
                await user.click(screen.getByTestId("form-image-settings-button"));
            });

            expect(screen.getByTestId("form-image-settings")).toBeDefined();

            // Update element with new URL
            const updatedElement = { ...mockElement, url: "https://newurl.com/image.jpg" };
            rerender(
                <FormImage
                    element={updatedElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Settings should remain open
            expect(screen.getByTestId("form-image-settings")).toBeDefined();
            expect(screen.getByTestId("form-image-img").getAttribute("src")).toBe("https://newurl.com/image.jpg");
        });

        it("maintains state when switching between editing modes", async () => {
            const user = userEvent.setup();
            const { rerender } = render(
                <FormImage
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings
            await act(async () => {
                await user.click(screen.getByTestId("form-image-settings-button"));
            });

            // Wait for settings to be visible
            expect(screen.getByTestId("form-image-settings")).toBeDefined();

            // Switch to upload mode
            await act(async () => {
                await user.click(screen.getByTestId("form-image-upload-mode-button"));
            });

            expect(screen.getByTestId("form-image-settings")).toBeDefined();
            expect(screen.getByTestId("form-image-dropzone")).toBeDefined();

            // Switch to non-editing mode
            rerender(
                <FormImage
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Switch back to editing mode
            rerender(
                <FormImage
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Note: The component maintains its state when switching modes
            // This is expected behavior as React preserves component state
            // unless the component is unmounted
            expect(screen.getByTestId("form-image-settings")).toBeDefined();
            expect(screen.getByTestId("form-image-dropzone")).toBeDefined();
        });
    });
});
