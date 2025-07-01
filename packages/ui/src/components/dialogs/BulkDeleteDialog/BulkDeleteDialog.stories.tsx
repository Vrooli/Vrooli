import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import {
    Box,
    Card,
    CardContent,
    Typography,
    TextField,
    Alert,
    Stack,
} from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { BulkDeleteDialog } from "./BulkDeleteDialog.js";
import type { ListObject } from "@vrooli/shared";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

// Mock data for demonstration
const createMockData = (count: number): ListObject[] => {
    return Array.from({ length: count }, (_, i) => ({
        id: `item-${i + 1}`,
        __typename: i % 3 === 0 ? "Project" : i % 3 === 1 ? "Routine" : "Note",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        name: `Item ${i + 1}`,
        handle: `item-${i + 1}`,
    } as any));
};

const meta: Meta<typeof BulkDeleteDialog> = {
    title: "Components/Dialogs/BulkDeleteDialog",
    component: BulkDeleteDialog,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 700,
            },
        },
    },
    tags: ["autodocs"],
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase story with controls
export const Showcase: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [itemCount, setItemCount] = useState(5);
        const [selectedData, setSelectedData] = useState<ListObject[]>([]);

        const handleOpen = () => {
            setSelectedData(createMockData(itemCount));
            setIsOpen(true);
        };

        const handleClose = (selectedForDelete: ListObject[]) => {
            action("bulk-delete-closed")({
                selectedCount: selectedForDelete.length,
                selectedIds: selectedForDelete.map(item => item.id),
            });
            setIsOpen(false);
        };

        return (
            <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            BulkDeleteDialog Controls
                        </Typography>
                        
                        <TextField
                            label="Number of items"
                            type="number"
                            inputProps={{ min: 1, max: 20 }}
                            value={itemCount}
                            onChange={(e) => setItemCount(parseInt(e.target.value) || 1)}
                            fullWidth
                            sx={{ mb: 2 }}
                        />

                        <Alert severity="info">
                            <Typography variant="body2">
                                <strong>Note:</strong> The dialog will show {itemCount} mock items with mixed types (Project, Routine, Note).
                            </Typography>
                        </Alert>
                    </CardContent>
                </Card>

                {/* Trigger Button */}
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={handleOpen} variant="danger" size="lg">
                        Open Bulk Delete Dialog
                    </Button>
                </Box>

                {/* Feature highlights */}
                <Alert severity="success">
                    <Typography variant="body2">
                        <strong>Dialog Features:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li>Toggle all checkbox for quick selection</li>
                        <li>Individual item checkboxes</li>
                        <li>Random confirmation words (bunny-themed)</li>
                        <li>Delete button disabled until confirmation matches</li>
                        <li>Returns array of selected items on delete</li>
                    </Box>
                </Alert>

                {/* Dialog Component */}
                <BulkDeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    selectedData={selectedData}
                />
            </Stack>
        );
    },
};

// Example with few items
export const FewItems: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const mockData = createMockData(3);

        const handleClose = (selectedForDelete: ListObject[]) => {
            action("delete-closed")({
                deletedCount: selectedForDelete.length,
                deletedItems: selectedForDelete,
            });
            setIsOpen(false);
        };

        return (
            <>
                <Button onClick={() => setIsOpen(true)} variant="danger">
                    Delete 3 Items
                </Button>
                <BulkDeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    selectedData={mockData}
                />
            </>
        );
    },
};

// Example with many items
export const ManyItems: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const mockData = createMockData(15);

        const handleClose = (selectedForDelete: ListObject[]) => {
            action("delete-closed")({
                deletedCount: selectedForDelete.length,
                deletedItems: selectedForDelete,
            });
            setIsOpen(false);
        };

        return (
            <>
                <Button onClick={() => setIsOpen(true)} variant="danger">
                    Delete 15 Items
                </Button>
                <BulkDeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    selectedData={mockData}
                />
            </>
        );
    },
};

// Example with pre-selected items
export const PreSelectedItems: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const mockData = createMockData(8);
        // Pre-select first 3 items
        const preSelectedData = mockData.slice(0, 3);

        const handleClose = (selectedForDelete: ListObject[]) => {
            action("delete-closed")({
                deletedCount: selectedForDelete.length,
                deletedItems: selectedForDelete,
            });
            setIsOpen(false);
        };

        return (
            <Stack spacing={2}>
                <Alert severity="info">
                    <Typography variant="body2">
                        <strong>Pre-selected:</strong> First 3 items will be pre-selected when dialog opens
                    </Typography>
                </Alert>
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={() => setIsOpen(true)} variant="danger">
                        Open with Pre-selected Items
                    </Button>
                </Box>
                <BulkDeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    selectedData={preSelectedData}
                />
            </Stack>
        );
    },
};
