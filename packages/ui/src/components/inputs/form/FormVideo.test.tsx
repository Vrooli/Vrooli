import { act, render, screen, fireEvent } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormVideo } from "./FormVideo";
import type { FormVideoType } from "@vrooli/shared";

// Mock react-i18next
vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key,
    }),
}));

// Mock LinkInput component
vi.mock("../LinkInput/LinkInput", () => ({
    LinkInput: ({ value, onChange, "data-testid": dataTestId, fullWidth, helperText, ...props }: any) => {
        const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
            onChange(e.target.value);
        };
        // Filter out non-DOM props
        const { className, style, placeholder, disabled, id, name } = props;
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
                onChange={handleChange}
            />
        );
    },
}));

// Mock TextField to properly handle onChange
vi.mock("@mui/material/TextField", () => ({
    default: ({ value, onChange, "data-testid": dataTestId, multiline, fullWidth, variant, label, helperText, error, ...props }: any) => {
        const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
            onChange(e);
        };
        const Component = multiline ? "textarea" : "input";
        // Filter out non-DOM props
        const { className, style, placeholder, disabled, id, name, rows } = props;
        return (
            <Component
                className={className}
                style={style}
                placeholder={placeholder || label}
                disabled={disabled}
                id={id}
                name={name}
                rows={multiline ? rows : undefined}
                data-testid={dataTestId}
                value={value}
                onChange={handleChange}
            />
        );
    },
}));

// iframe mocking is handled in global test setup

describe("FormVideo", () => {
    const mockElement: FormVideoType = {
        id: "test-video-1",
        type: "Video" as any,
        label: "Test Video",
        description: "Test video description",
        url: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    };

    const mockElementNoUrl: FormVideoType = {
        id: "test-video-2",
        type: "Video" as any,
        label: "Empty Video",
        description: "",
        url: "",
    };

    describe("Non-editing mode", () => {
        it("renders video iframe when URL exists", () => {
            render(
                <FormVideo
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const wrapper = screen.getByTestId("form-video");
            expect(wrapper).toBeDefined();
            expect(wrapper.getAttribute("data-editing")).toBe("false");

            const container = screen.getByTestId("form-video-container");
            expect(container).toBeDefined();
            expect(container.getAttribute("role")).toBe("region");
            expect(container.getAttribute("aria-label")).toBe("Test Video");

            const iframe = screen.getByTestId("form-video-iframe");
            expect(iframe).toBeDefined();
            expect(iframe.getAttribute("src")).toContain("https://www.youtube.com/embed/dQw4w9WgXcQ");
            expect(iframe.getAttribute("title")).toBe("Test Video");
            expect(iframe.getAttribute("data-autoplay")).toBe("false");
            expect(iframe.getAttribute("data-controls")).toBe("true");
        });

        it("renders nothing when URL is empty", () => {
            const { container } = render(
                <FormVideo
                    element={mockElementNoUrl}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(container.firstChild).toBeNull();
        });

        it("does not provide any interactive elements", () => {
            render(
                <FormVideo
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.queryByRole("button")).toBeNull();
            expect(screen.queryByTestId("form-video-settings-button")).toBeNull();
            expect(screen.queryByTestId("form-video-delete-button")).toBeNull();
        });

        it("extracts video ID from various YouTube URL formats", () => {
            const urls = [
                "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
                "https://youtu.be/dQw4w9WgXcQ",
                "https://www.youtube.com/watch?v=dQw4w9WgXcQ&feature=share",
            ];

            urls.forEach(url => {
                const { rerender } = render(
                    <FormVideo
                        element={{ ...mockElement, url }}
                        isEditing={false}
                        onUpdate={vi.fn()}
                        onDelete={vi.fn()}
                    />,
                );

                const iframe = screen.getByTestId("form-video-iframe");
                expect(iframe.getAttribute("src")).toContain("dQw4w9WgXcQ");

                rerender(<div />); // Clear between tests
            });
        });
    });

    describe("Editing mode", () => {
        it("renders with video and control buttons", () => {
            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const wrapper = screen.getByTestId("form-video");
            expect(wrapper.getAttribute("data-editing")).toBe("true");

            const settingsButton = screen.getByTestId("form-video-settings-button");
            expect(settingsButton).toBeDefined();
            expect(settingsButton.getAttribute("aria-label")).toBe("VideoSettings");
            expect(settingsButton.getAttribute("aria-expanded")).toBe("false");

            const deleteButton = screen.getByTestId("form-video-delete-button");
            expect(deleteButton).toBeDefined();
            expect(deleteButton.getAttribute("aria-label")).toBe("DeleteVideo");
        });

        it("renders placeholder when URL is empty", () => {
            render(
                <FormVideo
                    element={mockElementNoUrl}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const placeholder = screen.getByTestId("form-video-placeholder");
            expect(placeholder).toBeDefined();
            expect(placeholder.getAttribute("role")).toBe("img");
            expect(placeholder.getAttribute("aria-label")).toBe("NoVideoAdded");
            expect(placeholder.textContent).toBe("🎬");
        });

        it("toggles settings panel", async () => {
            const user = userEvent.setup();
            
            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            const settingsButton = screen.getByTestId("form-video-settings-button");
            expect(screen.queryByTestId("form-video-settings")).toBeNull();

            await act(async () => {
                await user.click(settingsButton);
            });

            const settingsPanel = screen.getByTestId("form-video-settings");
            expect(settingsPanel).toBeDefined();
            expect(settingsButton.getAttribute("aria-expanded")).toBe("true");

            // Check all settings inputs are present
            expect(screen.getByTestId("form-video-url-input")).toBeDefined();
            expect(screen.getByTestId("form-video-label-input")).toBeDefined();
            expect(screen.getByTestId("form-video-description-input")).toBeDefined();
            expect(screen.getByTestId("form-video-autoplay-switch")).toBeDefined();
            expect(screen.getByTestId("form-video-controls-switch")).toBeDefined();

            // Toggle settings off
            await act(async () => {
                await user.click(settingsButton);
            });

            expect(screen.queryByTestId("form-video-settings")).toBeNull();
            expect(settingsButton.getAttribute("aria-expanded")).toBe("false");
        });

        it("handles delete interaction", async () => {
            const onDelete = vi.fn();
            const user = userEvent.setup();

            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={onDelete}
                />,
            );

            const deleteButton = screen.getByTestId("form-video-delete-button");

            await act(async () => {
                await user.click(deleteButton);
            });

            expect(onDelete).toHaveBeenCalledTimes(1);
        });

        it("updates video URL", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();

            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings
            await act(async () => {
                await user.click(screen.getByTestId("form-video-settings-button"));
            });

            const urlInput = screen.getByTestId("form-video-url-input") as HTMLInputElement;
            const newUrl = "https://www.youtube.com/watch?v=abc123";
            
            // Use fireEvent for more direct control
            await act(async () => {
                fireEvent.change(urlInput, { target: { value: newUrl } });
            });

            // Check that onUpdate was called with the new URL
            expect(onUpdate).toHaveBeenCalledWith({ 
                url: newUrl, 
            });
        });

        it("updates video label", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();

            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings
            await act(async () => {
                await user.click(screen.getByTestId("form-video-settings-button"));
            });

            const labelInput = screen.getByTestId("form-video-label-input") as HTMLInputElement;
            const newLabel = "New Video Title";
            
            // Use fireEvent for text fields
            await act(async () => {
                fireEvent.change(labelInput, { target: { value: newLabel } });
            });

            // Check that onUpdate was called with the new label
            expect(onUpdate).toHaveBeenCalledWith({ 
                label: newLabel, 
            });
        });

        it("updates video description", async () => {
            const onUpdate = vi.fn();
            const user = userEvent.setup();

            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings
            await act(async () => {
                await user.click(screen.getByTestId("form-video-settings-button"));
            });

            const descriptionInput = screen.getByTestId("form-video-description-input") as HTMLTextAreaElement;
            const newDescription = "New video description";
            
            // Use fireEvent for text areas
            await act(async () => {
                fireEvent.change(descriptionInput, { target: { value: newDescription } });
            });

            // Check that onUpdate was called with the new description
            expect(onUpdate).toHaveBeenCalledWith({ 
                description: newDescription, 
            });
        });

        it("toggles autoplay setting", async () => {
            const user = userEvent.setup();

            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings
            await act(async () => {
                await user.click(screen.getByTestId("form-video-settings-button"));
            });

            const autoplaySwitch = screen.getByTestId("form-video-autoplay-switch") as HTMLInputElement;
            const iframe = screen.getByTestId("form-video-iframe");

            expect(iframe.getAttribute("data-autoplay")).toBe("false");

            await act(async () => {
                await user.click(autoplaySwitch);
            });

            expect(iframe.getAttribute("data-autoplay")).toBe("true");
            expect(iframe.getAttribute("src")).toContain("autoplay=1");
        });

        it("toggles controls setting", async () => {
            const user = userEvent.setup();

            render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings
            await act(async () => {
                await user.click(screen.getByTestId("form-video-settings-button"));
            });

            const controlsSwitch = screen.getByTestId("form-video-controls-switch") as HTMLInputElement;
            const iframe = screen.getByTestId("form-video-iframe");

            expect(iframe.getAttribute("data-controls")).toBe("true");

            await act(async () => {
                await user.click(controlsSwitch);
            });

            expect(iframe.getAttribute("data-controls")).toBe("false");
            expect(iframe.getAttribute("src")).toContain("controls=0");
        });
    });

    describe("State transitions", () => {
        it("toggles between editing and non-editing modes", () => {
            const { rerender } = render(
                <FormVideo
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Start in non-editing mode
            expect(screen.getByTestId("form-video").getAttribute("data-editing")).toBe("false");
            expect(screen.queryByTestId("form-video-settings-button")).toBeNull();
            expect(screen.queryByTestId("form-video-delete-button")).toBeNull();

            // Switch to editing mode
            rerender(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.getByTestId("form-video").getAttribute("data-editing")).toBe("true");
            expect(screen.getByTestId("form-video-settings-button")).toBeDefined();
            expect(screen.getByTestId("form-video-delete-button")).toBeDefined();

            // Switch back to non-editing mode
            rerender(
                <FormVideo
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            expect(screen.getByTestId("form-video").getAttribute("data-editing")).toBe("false");
            expect(screen.queryByTestId("form-video-settings-button")).toBeNull();
            expect(screen.queryByTestId("form-video-delete-button")).toBeNull();
        });

        it("maintains settings panel state during re-renders", async () => {
            const user = userEvent.setup();
            const onUpdate = vi.fn();

            const { rerender } = render(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Open settings panel
            await act(async () => {
                await user.click(screen.getByTestId("form-video-settings-button"));
            });

            expect(screen.getByTestId("form-video-settings")).toBeDefined();

            // Re-render with same props
            rerender(
                <FormVideo
                    element={mockElement}
                    isEditing={true}
                    onUpdate={onUpdate}
                    onDelete={vi.fn()}
                />,
            );

            // Settings panel should remain open
            expect(screen.getByTestId("form-video-settings")).toBeDefined();
            
            // Close settings
            await act(async () => {
                await user.click(screen.getByTestId("form-video-settings-button"));
            });

            expect(screen.queryByTestId("form-video-settings")).toBeNull();
        });

        it("updates iframe when URL changes", () => {
            const { rerender } = render(
                <FormVideo
                    element={mockElement}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            let iframe = screen.getByTestId("form-video-iframe");
            expect(iframe.getAttribute("src")).toContain("dQw4w9WgXcQ");

            // Update with new URL
            rerender(
                <FormVideo
                    element={{
                        ...mockElement,
                        url: "https://www.youtube.com/watch?v=xyz789",
                    }}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            iframe = screen.getByTestId("form-video-iframe");
            expect(iframe.getAttribute("src")).toContain("xyz789");
        });

        it("handles invalid YouTube URLs gracefully", () => {
            render(
                <FormVideo
                    element={{
                        ...mockElement,
                        url: "https://not-youtube.com/video",
                    }}
                    isEditing={false}
                    onUpdate={vi.fn()}
                    onDelete={vi.fn()}
                />,
            );

            // Should show placeholder instead of iframe for invalid URLs
            expect(screen.queryByTestId("form-video-iframe")).toBeNull();
            expect(screen.queryByTestId("form-video-container")).toBeNull();
        });
    });
});
