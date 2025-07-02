import Typography from "@mui/material/Typography";
import type { Meta, StoryObj } from "@storybook/react";
import { InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../__test/helpers/storybookDecorators.js";
import { Box } from "../layout/Box.js";
import type { ButtonBorderRadius, ButtonSize, ButtonVariant } from "./Button.js";
import { Button } from "./Button.js";
import { getCustomButtonStyle } from "./buttonStyles.js";

const meta: Meta<typeof Button> = {
    title: "Components/Buttons/Button",
    component: Button,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        // Define the form schema for controls
        const controlsSchema: FormSchema = {
            elements: [
                {
                    id: "size",
                    type: InputType.Radio,
                    fieldName: "size",
                    label: "Size",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Small", value: "sm" },
                            { label: "Medium", value: "md" },
                            { label: "Large", value: "lg" },
                            { label: "Icon", value: "icon" },
                        ],
                        defaultValue: "md",
                        row: true,
                    },
                },
                {
                    id: "variant",
                    type: InputType.Selector,
                    fieldName: "variant",
                    label: "Variant",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Primary", value: "primary" },
                            { label: "Secondary", value: "secondary" },
                            { label: "Outline", value: "outline" },
                            { label: "Ghost", value: "ghost" },
                            { label: "Danger", value: "danger" },
                            { label: "Space", value: "space" },
                            { label: "Custom", value: "custom" },
                            { label: "Neon", value: "neon" },
                        ],
                        defaultValue: "primary",
                    },
                },
                {
                    id: "borderRadius",
                    type: InputType.Radio,
                    fieldName: "borderRadius",
                    label: "Border Radius",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "None", value: "none" },
                            { label: "Minimal", value: "minimal" },
                            { label: "Pill", value: "pill" },
                        ],
                        defaultValue: "minimal",
                        row: true,
                    },
                },
                {
                    id: "disabled",
                    type: InputType.Switch,
                    fieldName: "disabled",
                    label: "Disabled",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "isLoading",
                    type: InputType.Switch,
                    fieldName: "isLoading",
                    label: "Loading",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "loadingIndicator",
                    type: InputType.Radio,
                    fieldName: "loadingIndicator",
                    label: "Loading Indicator",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Circular", value: "circular" },
                            { label: "Orbital", value: "orbital" },
                        ],
                        defaultValue: "circular",
                        row: true,
                    },
                },
                {
                    id: "fullWidth",
                    type: InputType.Switch,
                    fieldName: "fullWidth",
                    label: "Full Width",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "customColor",
                    type: InputType.ColorPicker,
                    fieldName: "customColor",
                    label: "Custom Color",
                    isRequired: false,
                    props: {
                        defaultValue: "#9333EA",
                    },
                },
            ],
            containers: [{
                totalItems: 8,
            }],
        };

        // Initial values for the form
        const initialValues = {
            size: "md",
            variant: "primary",
            borderRadius: "minimal",
            disabled: false,
            isLoading: false,
            loadingIndicator: "circular",
            fullWidth: false,
            customColor: "#9333EA",
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "Button",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { size, variant, borderRadius, disabled, isLoading, loadingIndicator, fullWidth, customColor } = values;

                // Extract style objects to avoid inline creation
                const typographyStyle = (buttonVariant: string) => ({
                    mb: 1,
                    textTransform: "capitalize" as const,
                    color: (buttonVariant === "space" || buttonVariant === "neon") ? "#fff" : "inherit",
                });

                const buttonStyle = (buttonVariant: string) => ({
                    ...(buttonVariant === "custom" ? getCustomButtonStyle(customColor) : {}),
                });

                return (
                    <Box 
                        className="tw-grid tw-grid-cols-1 sm:tw-grid-cols-2 md:tw-grid-cols-3 lg:tw-grid-cols-4 tw-gap-6"
                        style={{ background: "transparent" }}
                    >
                        {["primary", "secondary", "outline", "ghost", "danger", "space", "custom", "neon"].map(buttonVariant => (
                            <Box
                                key={buttonVariant}
                                padding="sm"
                                borderRadius="md"
                                className="tw-text-center"
                                style={{ background: "transparent" }}
                            >
                                <Typography
                                    variant="subtitle2"
                                    sx={typographyStyle(buttonVariant)}
                                >
                                    {buttonVariant}
                                </Typography>
                                <Button
                                    variant={buttonVariant as ButtonVariant}
                                    size={size as ButtonSize}
                                    borderRadius={borderRadius as ButtonBorderRadius}
                                    disabled={disabled}
                                    isLoading={isLoading}
                                    loadingIndicator={loadingIndicator as "circular" | "orbital"}
                                    fullWidth={fullWidth}
                                    style={buttonStyle(buttonVariant)}
                                    className={buttonVariant === "custom" ? "hover:tw-opacity-90" : undefined}
                                >
                                    Button Text
                                </Button>
                            </Box>
                        ))}
                    </Box>
                );
            },
        };

        // Use the showcase decorator
        const ShowcaseComponent = showcaseDecorator(showcaseConfig);
        return <ShowcaseComponent />;
    },
};

