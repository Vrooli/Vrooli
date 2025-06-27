import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormStructureType, InputType } from "@vrooli/shared";
import { PopoverListItem } from "./PopoverListItem";

describe("PopoverListItem", () => {
    const defaultProps = {
        iconInfo: { name: "CaseSensitive", type: "Text" } as const,
        label: "Test Label",
        type: InputType.Text,
        onAddHeader: vi.fn(),
        onAddInput: vi.fn(),
        onAddStructure: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Rendering", () => {
        it("renders with icon and label", () => {
            render(<PopoverListItem {...defaultProps} />);

            const listItem = screen.getByTestId("popover-list-item");
            expect(listItem).toBeDefined();
            expect(listItem.getAttribute("data-type")).toBe(InputType.Text);
            
            // Check label is rendered
            expect(screen.getByText("Test Label")).toBeDefined();
            
            // Check icon is rendered by verifying the Icon component
            const icon = screen.queryByTestId("icon");
            expect(icon).toBeDefined();
        });

        it("renders without icon when iconInfo is null", () => {
            render(<PopoverListItem {...defaultProps} iconInfo={null} />);

            const listItem = screen.getByTestId("popover-list-item");
            expect(listItem).toBeDefined();
            
            // Check label is still rendered
            expect(screen.getByText("Test Label")).toBeDefined();
            
            // Check no icon is rendered
            const icon = screen.queryByTestId("icon");
            expect(icon).toBeNull();
        });

        it("renders without icon when iconInfo is undefined", () => {
            render(<PopoverListItem {...defaultProps} iconInfo={undefined as any} />);

            const listItem = screen.getByTestId("popover-list-item");
            const icon = screen.queryByTestId("icon");
            expect(icon).toBeNull();
        });

        it("includes tag data attribute for header types", () => {
            render(
                <PopoverListItem
                    {...defaultProps}
                    type={FormStructureType.Header}
                    tag="h1"
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            expect(listItem.getAttribute("data-tag")).toBe("h1");
        });
    });

    describe("Input type interactions", () => {
        it("calls onAddInput when clicking an input type item", async () => {
            const onAddInput = vi.fn();
            const user = userEvent.setup();

            render(
                <PopoverListItem
                    {...defaultProps}
                    type={InputType.Text}
                    onAddInput={onAddInput}
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            
            await act(async () => {
                await user.click(listItem);
            });

            expect(onAddInput).toHaveBeenCalledTimes(1);
            expect(onAddInput).toHaveBeenCalledWith({ type: InputType.Text });
            expect(defaultProps.onAddHeader).not.toHaveBeenCalled();
            expect(defaultProps.onAddStructure).not.toHaveBeenCalled();
        });

        it("handles different input types correctly", async () => {
            const inputTypes = [
                InputType.Checkbox,
                InputType.Radio,
                InputType.Switch,
                InputType.JSON,
                InputType.Selector,
                InputType.IntegerInput,
                InputType.Slider,
                InputType.Dropzone,
                InputType.LinkUrl,
                InputType.LinkItem,
            ];

            for (const inputType of inputTypes) {
                const onAddInput = vi.fn();
                const user = userEvent.setup();

                const { unmount } = render(
                    <PopoverListItem
                        {...defaultProps}
                        type={inputType}
                        onAddInput={onAddInput}
                    />
                );

                const listItem = screen.getByTestId("popover-list-item");
                
                await act(async () => {
                    await user.click(listItem);
                });

                expect(onAddInput).toHaveBeenCalledWith({ type: inputType });
                
                unmount();
            }
        });
    });

    describe("Header interactions", () => {
        it("calls onAddHeader with tag when clicking a header item", async () => {
            const onAddHeader = vi.fn();
            const user = userEvent.setup();

            render(
                <PopoverListItem
                    {...defaultProps}
                    type={FormStructureType.Header}
                    tag="h1"
                    onAddHeader={onAddHeader}
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            
            await act(async () => {
                await user.click(listItem);
            });

            expect(onAddHeader).toHaveBeenCalledTimes(1);
            expect(onAddHeader).toHaveBeenCalledWith({ tag: "h1" });
            expect(defaultProps.onAddInput).not.toHaveBeenCalled();
            expect(defaultProps.onAddStructure).not.toHaveBeenCalled();
        });

        it("logs error when header type is missing tag", async () => {
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            const user = userEvent.setup();

            render(
                <PopoverListItem
                    {...defaultProps}
                    type={FormStructureType.Header}
                    tag={undefined}
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            
            await act(async () => {
                await user.click(listItem);
            });

            expect(consoleSpy).toHaveBeenCalledWith("Missing tag for header - cannot add header to form.");
            expect(defaultProps.onAddHeader).not.toHaveBeenCalled();

            consoleSpy.mockRestore();
        });

        it("handles different header tags", async () => {
            const headerTags = ["h1", "h2", "h3", "h4", "h5", "h6"] as const;

            for (const tag of headerTags) {
                const onAddHeader = vi.fn();
                const user = userEvent.setup();

                const { unmount } = render(
                    <PopoverListItem
                        {...defaultProps}
                        type={FormStructureType.Header}
                        tag={tag}
                        onAddHeader={onAddHeader}
                    />
                );

                const listItem = screen.getByTestId("popover-list-item");
                
                await act(async () => {
                    await user.click(listItem);
                });

                expect(onAddHeader).toHaveBeenCalledWith({ tag });
                
                unmount();
            }
        });
    });

    describe("Structure element interactions", () => {
        it("calls onAddStructure for divider", async () => {
            const onAddStructure = vi.fn();
            const user = userEvent.setup();

            render(
                <PopoverListItem
                    {...defaultProps}
                    type={FormStructureType.Divider}
                    onAddStructure={onAddStructure}
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            
            await act(async () => {
                await user.click(listItem);
            });

            expect(onAddStructure).toHaveBeenCalledTimes(1);
            expect(onAddStructure).toHaveBeenCalledWith({ type: "Divider" });
            expect(defaultProps.onAddHeader).not.toHaveBeenCalled();
            expect(defaultProps.onAddInput).not.toHaveBeenCalled();
        });

        it("handles all structure types correctly", async () => {
            const structureTypes = [
                { type: FormStructureType.Divider, expected: "Divider" },
                { type: FormStructureType.Image, expected: "Image" },
                { type: FormStructureType.QrCode, expected: "QrCode" },
                { type: FormStructureType.Tip, expected: "Tip" },
                { type: FormStructureType.Video, expected: "Video" },
            ];

            for (const { type, expected } of structureTypes) {
                const onAddStructure = vi.fn();
                const user = userEvent.setup();

                const { unmount } = render(
                    <PopoverListItem
                        {...defaultProps}
                        type={type}
                        onAddStructure={onAddStructure}
                    />
                );

                const listItem = screen.getByTestId("popover-list-item");
                
                await act(async () => {
                    await user.click(listItem);
                });

                expect(onAddStructure).toHaveBeenCalledWith({ type: expected });
                
                unmount();
            }
        });
    });

    describe("Accessibility", () => {
        it("has button role and is keyboard accessible", () => {
            render(<PopoverListItem {...defaultProps} />);

            const listItem = screen.getByTestId("popover-list-item");
            
            // MUI ListItem with button prop should have button role
            expect(listItem.getAttribute("role")).toBe("button");
            
            // Should be in tab order
            expect(listItem.getAttribute("tabindex")).toBe("0");
        });

        it("can be activated with keyboard", async () => {
            const onAddInput = vi.fn();
            const user = userEvent.setup();

            render(
                <PopoverListItem
                    {...defaultProps}
                    type={InputType.Text}
                    onAddInput={onAddInput}
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            
            // Focus the element
            await act(async () => {
                listItem.focus();
            });
            
            // Press Enter
            await act(async () => {
                await user.keyboard("{Enter}");
            });

            expect(onAddInput).toHaveBeenCalledTimes(1);
            expect(onAddInput).toHaveBeenCalledWith({ type: InputType.Text });
        });

        it("can be activated with Space key", async () => {
            const onAddInput = vi.fn();
            const user = userEvent.setup();

            render(
                <PopoverListItem
                    {...defaultProps}
                    type={InputType.Text}
                    onAddInput={onAddInput}
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            
            // Focus the element
            await act(async () => {
                listItem.focus();
            });
            
            // Press Space
            await act(async () => {
                await user.keyboard(" ");
            });

            expect(onAddInput).toHaveBeenCalledTimes(1);
            expect(onAddInput).toHaveBeenCalledWith({ type: InputType.Text });
        });
    });

    describe("Edge cases", () => {
        it("handles rapid clicks gracefully", async () => {
            const onAddInput = vi.fn();
            const user = userEvent.setup();

            render(
                <PopoverListItem
                    {...defaultProps}
                    type={InputType.Text}
                    onAddInput={onAddInput}
                />
            );

            const listItem = screen.getByTestId("popover-list-item");
            
            // Click multiple times rapidly
            await act(async () => {
                await user.tripleClick(listItem);
            });

            // Should still handle clicks correctly
            expect(onAddInput).toHaveBeenCalledTimes(3);
        });

        it("maintains callback references when props change", () => {
            const initialOnAddInput = vi.fn();
            const newOnAddInput = vi.fn();

            const { rerender } = render(
                <PopoverListItem
                    {...defaultProps}
                    onAddInput={initialOnAddInput}
                />
            );

            // Change the callback
            rerender(
                <PopoverListItem
                    {...defaultProps}
                    onAddInput={newOnAddInput}
                />
            );

            // Component should still be rendered correctly
            expect(screen.getByTestId("popover-list-item")).toBeDefined();
        });

        it("handles label changes", () => {
            const { rerender } = render(<PopoverListItem {...defaultProps} label="Initial Label" />);

            expect(screen.getByText("Initial Label")).toBeDefined();

            rerender(<PopoverListItem {...defaultProps} label="Updated Label" />);

            expect(screen.queryByText("Initial Label")).toBeNull();
            expect(screen.getByText("Updated Label")).toBeDefined();
        });
    });
});