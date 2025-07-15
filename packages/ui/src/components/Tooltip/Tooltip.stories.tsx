import type { Meta, StoryObj } from "@storybook/react";
import { InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../__test/helpers/storybookDecorators.js";
import { IconCommon } from "../../icons/Icons.js";
import { Box } from "../layout/Box.js";
import { Typography } from "../text/Typography.js";
import { Button } from "../buttons/Button.js";
import { IconButton } from "../buttons/IconButton.js";
import { Tooltip } from "./Tooltip.js";
import type { TooltipPlacement } from "./Tooltip.js";

const meta: Meta<typeof Tooltip> = {
    title: "Components/Dialogs/Tooltip",
    component: Tooltip,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Tooltip Playground
export const TooltipShowcase: Story = {
    render: () => {
        // Define the form schema for controls
        const controlsSchema: FormSchema = {
            elements: [
                {
                    id: "placement",
                    type: InputType.Selector,
                    fieldName: "placement",
                    label: "Placement",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Top", value: "top" },
                            { label: "Top Start", value: "top-start" },
                            { label: "Top End", value: "top-end" },
                            { label: "Bottom", value: "bottom" },
                            { label: "Bottom Start", value: "bottom-start" },
                            { label: "Bottom End", value: "bottom-end" },
                            { label: "Left", value: "left" },
                            { label: "Left Start", value: "left-start" },
                            { label: "Left End", value: "left-end" },
                            { label: "Right", value: "right" },
                            { label: "Right Start", value: "right-start" },
                            { label: "Right End", value: "right-end" },
                        ],
                        defaultValue: "top",
                    },
                },
                {
                    id: "arrow",
                    type: InputType.Switch,
                    fieldName: "arrow",
                    label: "Show Arrow",
                    isRequired: false,
                    props: {
                        defaultValue: true,
                    },
                },
                {
                    id: "enterDelay",
                    type: InputType.IntegerInput,
                    fieldName: "enterDelay",
                    label: "Enter Delay (ms)",
                    isRequired: false,
                    props: {
                        defaultValue: 100,
                        min: 0,
                        max: 2000,
                    },
                },
                {
                    id: "leaveDelay",
                    type: InputType.IntegerInput,
                    fieldName: "leaveDelay",
                    label: "Leave Delay (ms)",
                    isRequired: false,
                    props: {
                        defaultValue: 0,
                        min: 0,
                        max: 2000,
                    },
                },
                {
                    id: "controlledState",
                    type: InputType.Radio,
                    fieldName: "controlledState",
                    label: "Controlled State",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Uncontrolled", value: "uncontrolled" },
                            { label: "Always Open", value: "open" },
                            { label: "Always Closed", value: "closed" },
                        ],
                        defaultValue: "uncontrolled",
                        row: true,
                    },
                },
                {
                    id: "tooltipText",
                    type: InputType.Text,
                    fieldName: "tooltipText",
                    label: "Tooltip Text",
                    isRequired: false,
                    props: {
                        defaultValue: "This is a helpful tooltip!",
                        multiline: true,
                        rows: 2,
                    },
                },
            ],
            containers: [{
                totalItems: 6,
            }],
        };

        // Initial values for the form
        const initialValues = {
            placement: "top",
            arrow: true,
            enterDelay: 100,
            leaveDelay: 0,
            controlledState: "uncontrolled",
            tooltipText: "This is a helpful tooltip!",
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "Tooltip",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { placement, arrow, enterDelay, leaveDelay, controlledState, tooltipText } = values;
                
                // Convert controlled state to open prop
                const open = controlledState === "uncontrolled" ? undefined : controlledState === "open";

                return (
                    <Box className="tw-space-y-8">
                        {/* Main Demo */}
                        <Box className="tw-text-center tw-py-16">
                            <Typography variant="h6" className="tw-mb-8 tw-text-gray-600">
                                Interactive Demo
                            </Typography>
                            <Tooltip
                                title={tooltipText}
                                placement={placement as TooltipPlacement}
                                arrow={arrow}
                                enterDelay={enterDelay}
                                leaveDelay={leaveDelay}
                                open={open}
                            >
                                <Button variant="primary" size="lg">
                                    Hover over me!
                                </Button>
                            </Tooltip>
                        </Box>

                        {/* Examples Grid */}
                        <Box>
                            <Typography variant="h6" className="tw-mb-6 tw-text-gray-600">
                                Common Examples
                            </Typography>
                            <Box className="tw-grid tw-grid-cols-1 sm:tw-grid-cols-2 md:tw-grid-cols-3 tw-gap-8">
                                {/* Button with Tooltip */}
                                <Box className="tw-text-center">
                                    <Typography variant="subtitle2" className="tw-mb-4">Button</Typography>
                                    <Tooltip title="Save your changes" placement="top">
                                        <Button variant="primary" startIcon={<IconCommon name="Save" />}>
                                            Save
                                        </Button>
                                    </Tooltip>
                                </Box>

                                {/* Icon Button with Tooltip */}
                                <Box className="tw-text-center">
                                    <Typography variant="subtitle2" className="tw-mb-4">Icon Button</Typography>
                                    <Tooltip title="Delete item" placement="top">
                                        <IconButton variant="outlined" color="error">
                                            <IconCommon name="Delete" />
                                        </IconButton>
                                    </Tooltip>
                                </Box>

                                {/* Text with Tooltip */}
                                <Box className="tw-text-center">
                                    <Typography variant="subtitle2" className="tw-mb-4">Text</Typography>
                                    <Tooltip title="This text has additional information" placement="top">
                                        <Typography 
                                            component="span"
                                            className="tw-underline tw-decoration-dotted tw-cursor-help"
                                        >
                                            Hover for info
                                        </Typography>
                                    </Tooltip>
                                </Box>

                                {/* Disabled Element */}
                                <Box className="tw-text-center">
                                    <Typography variant="subtitle2" className="tw-mb-4">Disabled</Typography>
                                    <Tooltip title="This action is currently unavailable" placement="top">
                                        <span>
                                            <Button variant="secondary" disabled>
                                                Disabled
                                            </Button>
                                        </span>
                                    </Tooltip>
                                </Box>

                                {/* Long Content */}
                                <Box className="tw-text-center">
                                    <Typography variant="subtitle2" className="tw-mb-4">Long Content</Typography>
                                    <Tooltip 
                                        title="This is a longer tooltip with more detailed information that wraps to multiple lines when needed."
                                        placement="top"
                                    >
                                        <Button variant="outline">
                                            Long tooltip
                                        </Button>
                                    </Tooltip>
                                </Box>

                                {/* Custom Delay */}
                                <Box className="tw-text-center">
                                    <Typography variant="subtitle2" className="tw-mb-4">Custom Delay</Typography>
                                    <Tooltip 
                                        title="This appears after 1 second"
                                        placement="top"
                                        enterDelay={1000}
                                        leaveDelay={500}
                                    >
                                        <Button variant="ghost">
                                            Delayed tooltip
                                        </Button>
                                    </Tooltip>
                                </Box>
                            </Box>
                        </Box>

                        {/* Placement Grid */}
                        <Box>
                            <Typography variant="h6" className="tw-mb-6 tw-text-gray-600">
                                All Placements
                            </Typography>
                            <Box className="tw-grid tw-grid-cols-5 tw-grid-rows-5 tw-gap-4 tw-min-h-[400px]">
                                {/* Top row */}
                                <Box className="tw-col-start-2 tw-row-start-1 tw-flex tw-justify-center">
                                    <Tooltip title="top-start" placement="top-start">
                                        <Button variant="outline" size="sm">TS</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-3 tw-row-start-1 tw-flex tw-justify-center">
                                    <Tooltip title="top" placement="top">
                                        <Button variant="outline" size="sm">T</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-4 tw-row-start-1 tw-flex tw-justify-center">
                                    <Tooltip title="top-end" placement="top-end">
                                        <Button variant="outline" size="sm">TE</Button>
                                    </Tooltip>
                                </Box>

                                {/* Left column */}
                                <Box className="tw-col-start-1 tw-row-start-2 tw-flex tw-items-center tw-justify-center">
                                    <Tooltip title="left-start" placement="left-start">
                                        <Button variant="outline" size="sm">LS</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-1 tw-row-start-3 tw-flex tw-items-center tw-justify-center">
                                    <Tooltip title="left" placement="left">
                                        <Button variant="outline" size="sm">L</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-1 tw-row-start-4 tw-flex tw-items-center tw-justify-center">
                                    <Tooltip title="left-end" placement="left-end">
                                        <Button variant="outline" size="sm">LE</Button>
                                    </Tooltip>
                                </Box>

                                {/* Center */}
                                <Box className="tw-col-start-3 tw-row-start-3 tw-flex tw-items-center tw-justify-center">
                                    <Typography variant="h6" className="tw-text-gray-400">
                                        Target
                                    </Typography>
                                </Box>

                                {/* Right column */}
                                <Box className="tw-col-start-5 tw-row-start-2 tw-flex tw-items-center tw-justify-center">
                                    <Tooltip title="right-start" placement="right-start">
                                        <Button variant="outline" size="sm">RS</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-5 tw-row-start-3 tw-flex tw-items-center tw-justify-center">
                                    <Tooltip title="right" placement="right">
                                        <Button variant="outline" size="sm">R</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-5 tw-row-start-4 tw-flex tw-items-center tw-justify-center">
                                    <Tooltip title="right-end" placement="right-end">
                                        <Button variant="outline" size="sm">RE</Button>
                                    </Tooltip>
                                </Box>

                                {/* Bottom row */}
                                <Box className="tw-col-start-2 tw-row-start-5 tw-flex tw-justify-center">
                                    <Tooltip title="bottom-start" placement="bottom-start">
                                        <Button variant="outline" size="sm">BS</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-3 tw-row-start-5 tw-flex tw-justify-center">
                                    <Tooltip title="bottom" placement="bottom">
                                        <Button variant="outline" size="sm">B</Button>
                                    </Tooltip>
                                </Box>
                                <Box className="tw-col-start-4 tw-row-start-5 tw-flex tw-justify-center">
                                    <Tooltip title="bottom-end" placement="bottom-end">
                                        <Button variant="outline" size="sm">BE</Button>
                                    </Tooltip>
                                </Box>
                            </Box>
                        </Box>
                    </Box>
                );
            },
        };

        // Use the showcase decorator
        const ShowcaseComponent = showcaseDecorator(showcaseConfig);
        return <ShowcaseComponent />;
    },
};
