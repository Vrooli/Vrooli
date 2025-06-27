import type { Meta, StoryObj } from "@storybook/react";
import React from "react";
import Box from "@mui/material/Box";
import List from "@mui/material/List";
import Typography from "@mui/material/Typography";
import Divider from "@mui/material/Divider";
import ListSubheader from "@mui/material/ListSubheader";
import { FormStructureType, InputType } from "@vrooli/shared";
import { PopoverListItem } from "./PopoverListItem.js";
import type { PopoverListItemProps } from "./FormView.types.js";

const meta: Meta<typeof PopoverListItem> = {
    title: "Forms/FormView/PopoverListItem",
    component: PopoverListItem,
    parameters: {
        layout: "centered",
        docs: {
            description: {
                component: "A list item component used in the form builder to add various input types and structural elements to forms.",
            },
        },
    },
    argTypes: {
        iconInfo: {
            control: "object",
            description: "Icon information for the list item",
        },
        label: {
            control: "text",
            description: "Display label for the list item",
        },
        type: {
            control: "select",
            options: [...Object.values(InputType), ...Object.values(FormStructureType)],
            description: "Type of form element this item will create",
        },
        tag: {
            control: "select",
            options: ["h1", "h2", "h3", "h4", "h5", "h6"],
            description: "HTML tag for header elements",
        },
        onAddHeader: {
            action: "onAddHeader",
            description: "Callback when adding a header element",
        },
        onAddInput: {
            action: "onAddInput",
            description: "Callback when adding an input element",
        },
        onAddStructure: {
            action: "onAddStructure",
            description: "Callback when adding a structural element",
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default story with a text input item
export const Default: Story = {
    args: {
        iconInfo: { name: "CaseSensitive", type: "Text" } as const,
        label: "Text Input",
        type: InputType.Text,
    },
};

// Story showing an item without an icon
export const WithoutIcon: Story = {
    args: {
        iconInfo: null,
        label: "Plain Text Input",
        type: InputType.Text,
    },
};

// Story showing a header item
export const HeaderItem: Story = {
    args: {
        iconInfo: { name: "Header", type: "Text" } as const,
        label: "Large Header",
        type: FormStructureType.Header,
        tag: "h1",
    },
};

// Story showing a structural element
export const StructuralElement: Story = {
    args: {
        iconInfo: { name: "Minus", type: "Common" } as const,
        label: "Divider",
        type: FormStructureType.Divider,
    },
};

// Story showing multiple items in a list context
export const InListContext: Story = {
    render: (args) => {
        const inputItems = [
            {
                category: "Text Inputs",
                items: [
                    { type: InputType.Text, iconInfo: { name: "CaseSensitive", type: "Text" } as const, label: "Text" },
                    { type: InputType.JSON, iconInfo: { name: "Object", type: "Common" } as const, label: "JSON" },
                ],
            },
            {
                category: "Selection Inputs",
                items: [
                    { type: InputType.Checkbox, iconInfo: { name: "ListCheck", type: "Text" } as const, label: "Checkbox" },
                    { type: InputType.Radio, iconInfo: { name: "ListBullet", type: "Text" } as const, label: "Radio" },
                    { type: InputType.Switch, iconInfo: { name: "Switch", type: "Common" } as const, label: "Switch" },
                ],
            },
            {
                category: "Headers",
                items: [
                    { type: FormStructureType.Header, tag: "h1", iconInfo: { name: "Header", type: "Text" } as const, label: "Large (H1)" },
                    { type: FormStructureType.Header, tag: "h2", iconInfo: { name: "Header", type: "Text" } as const, label: "Medium (H2)" },
                    { type: FormStructureType.Header, tag: "h3", iconInfo: { name: "Header", type: "Text" } as const, label: "Small (H3)" },
                ],
            },
            {
                category: "Page Elements",
                items: [
                    { type: FormStructureType.Divider, iconInfo: { name: "Minus", type: "Common" } as const, label: "Divider" },
                    { type: FormStructureType.Tip, iconInfo: { name: "Help", type: "Common" } as const, label: "Tip" },
                    { type: FormStructureType.Image, iconInfo: { name: "Image", type: "Common" } as const, label: "Image" },
                ],
            },
        ];

        return (
            <Box sx={{ width: 300, bgcolor: "background.paper", borderRadius: 2, boxShadow: 2 }}>
                <Typography variant="h6" sx={{ p: 2 }}>
                    Form Elements
                </Typography>
                <List sx={{ pt: 0 }}>
                    {inputItems.map(({ category, items }) => (
                        <React.Fragment key={category}>
                            <ListSubheader>{category}</ListSubheader>
                            {items.map((item, index) => (
                                <PopoverListItem
                                    key={`${category}-${item.type}-${item.tag || ""}-${index}`}
                                    iconInfo={item.iconInfo}
                                    label={item.label}
                                    type={item.type}
                                    tag={item.tag as PopoverListItemProps["tag"]}
                                    onAddHeader={args.onAddHeader}
                                    onAddInput={args.onAddInput}
                                    onAddStructure={args.onAddStructure}
                                />
                            ))}
                            <Divider />
                        </React.Fragment>
                    ))}
                </List>
            </Box>
        );
    },
};

// Interactive story demonstrating all possible types
export const Interactive: Story = {
    render: (args) => {
        const [lastAction, setLastAction] = React.useState<string>("");

        const handleAddHeader = (data: any) => {
            setLastAction(`Added header: ${JSON.stringify(data)}`);
            args.onAddHeader?.(data);
        };

        const handleAddInput = (data: any) => {
            setLastAction(`Added input: ${JSON.stringify(data)}`);
            args.onAddInput?.(data);
        };

        const handleAddStructure = (data: any) => {
            setLastAction(`Added structure: ${JSON.stringify(data)}`);
            args.onAddStructure?.(data);
        };

        return (
            <Box sx={{ width: 400 }}>
                <Typography variant="h6" sx={{ mb: 2 }}>
                    Click an item to see the action
                </Typography>
                
                <Box sx={{ bgcolor: "background.paper", borderRadius: 1, p: 2, mb: 2 }}>
                    <Typography variant="body2" color="text.secondary">
                        Last action:
                    </Typography>
                    <Typography variant="body1" sx={{ fontFamily: "monospace", mt: 1 }}>
                        {lastAction || "None"}
                    </Typography>
                </Box>

                <List sx={{ bgcolor: "background.paper", borderRadius: 1 }}>
                    <PopoverListItem
                        iconInfo={{ name: "CaseSensitive", type: "Text" } as const}
                        label="Add Text Input"
                        type={InputType.Text}
                        onAddHeader={handleAddHeader}
                        onAddInput={handleAddInput}
                        onAddStructure={handleAddStructure}
                    />
                    <PopoverListItem
                        iconInfo={{ name: "Header", type: "Text" } as const}
                        label="Add Large Header"
                        type={FormStructureType.Header}
                        tag="h1"
                        onAddHeader={handleAddHeader}
                        onAddInput={handleAddInput}
                        onAddStructure={handleAddStructure}
                    />
                    <PopoverListItem
                        iconInfo={{ name: "Help", type: "Common" } as const}
                        label="Add Tip"
                        type={FormStructureType.Tip}
                        onAddHeader={handleAddHeader}
                        onAddInput={handleAddInput}
                        onAddStructure={handleAddStructure}
                    />
                </List>
            </Box>
        );
    },
};