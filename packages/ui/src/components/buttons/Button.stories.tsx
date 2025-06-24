import Typography from "@mui/material/Typography";
import type { Meta, StoryObj } from "@storybook/react";
import { InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../__test/helpers/storybookDecorators.js";
import { Box } from "../layout/Box.js";
import type { ButtonSize, ButtonVariant } from "./Button.js";
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
                totalItems: 4,
                spacing: 3,
                xs: 1,
                sm: 2,
                md: 2,
                lg: 4,
            }],
        };

        // Initial values for the form
        const initialValues = {
            size: "md",
            variant: "primary",
            disabled: false,
            customColor: "#9333EA",
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "Button",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { size, variant, disabled, customColor } = values;

                return (
                    <Box className="tw-grid tw-grid-cols-1 sm:tw-grid-cols-2 md:tw-grid-cols-3 lg:tw-grid-cols-4 tw-gap-6">
                        {["primary", "secondary", "outline", "ghost", "danger", "space", "custom", "neon"].map(buttonVariant => (
                            <Box
                                key={buttonVariant}
                                padding="sm"
                                borderRadius="md"
                                className={`tw-text-center ${(buttonVariant === "space" || buttonVariant === "neon")
                                        ? "tw-bg-slate-900"
                                        : "tw-bg-transparent"
                                    }`}
                            >
                                <Typography
                                    variant="subtitle2"
                                    sx={{
                                        mb: 1,
                                        textTransform: "capitalize",
                                        color: (buttonVariant === "space" || buttonVariant === "neon") ? "#fff" : "inherit",
                                    }}
                                >
                                    {buttonVariant}
                                </Typography>
                                <Button
                                    variant={buttonVariant as ButtonVariant}
                                    size={size as ButtonSize}
                                    disabled={disabled}
                                    style={{
                                        ...(buttonVariant === "custom" ? getCustomButtonStyle(customColor) : {}),
                                    }}
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

