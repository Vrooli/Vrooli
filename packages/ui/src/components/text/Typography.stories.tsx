import type { Meta, StoryObj } from "@storybook/react";
import { InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../__test/helpers/storybookDecorators.js";
import { Box } from "../layout/Box.js";
import type { TypographyAlign, TypographyColor, TypographySpacing, TypographyVariant, TypographyWeight } from "./Typography.js";
import { Typography } from "./Typography.js";

const meta: Meta<typeof Typography> = {
    title: "Components/Text/Typography",
    component: Typography,
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
                    id: "variant",
                    type: InputType.Selector,
                    fieldName: "variant",
                    label: "Variant",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Display 1", value: "display1" },
                            { label: "Display 2", value: "display2" },
                            { label: "H1", value: "h1" },
                            { label: "H2", value: "h2" },
                            { label: "H3", value: "h3" },
                            { label: "H4", value: "h4" },
                            { label: "H5", value: "h5" },
                            { label: "H6", value: "h6" },
                            { label: "Subtitle 1", value: "subtitle1" },
                            { label: "Subtitle 2", value: "subtitle2" },
                            { label: "Body 1", value: "body1" },
                            { label: "Body 2", value: "body2" },
                            { label: "Caption", value: "caption" },
                            { label: "Overline", value: "overline" },
                        ],
                        defaultValue: "body1",
                    },
                },
                {
                    id: "align",
                    type: InputType.Radio,
                    fieldName: "align",
                    label: "Alignment",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Left", value: "left" },
                            { label: "Center", value: "center" },
                            { label: "Right", value: "right" },
                            { label: "Justify", value: "justify" },
                        ],
                        defaultValue: "left",
                        row: true,
                    },
                },
                {
                    id: "color",
                    type: InputType.Selector,
                    fieldName: "color",
                    label: "Color",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Text", value: "text" },
                            { label: "Primary", value: "primary" },
                            { label: "Secondary", value: "secondary" },
                            { label: "Success", value: "success" },
                            { label: "Warning", value: "warning" },
                            { label: "Error", value: "error" },
                            { label: "Info", value: "info" },
                            { label: "Inherit", value: "inherit" },
                        ],
                        defaultValue: "text",
                    },
                },
                {
                    id: "weight",
                    type: InputType.Radio,
                    fieldName: "weight",
                    label: "Font Weight",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Light", value: "light" },
                            { label: "Normal", value: "normal" },
                            { label: "Medium", value: "medium" },
                            { label: "Semibold", value: "semibold" },
                            { label: "Bold", value: "bold" },
                        ],
                        defaultValue: "normal",
                        row: true,
                    },
                },
                {
                    id: "uppercase",
                    type: InputType.Switch,
                    fieldName: "uppercase",
                    label: "Uppercase",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "noWrap",
                    type: InputType.Switch,
                    fieldName: "noWrap",
                    label: "No Wrap",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "truncate",
                    type: InputType.Switch,
                    fieldName: "truncate",
                    label: "Truncate",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "spacing",
                    type: InputType.Radio,
                    fieldName: "spacing",
                    label: "Spacing",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Default", value: "" },
                            { label: "None", value: "none" },
                            { label: "Tight", value: "tight" },
                            { label: "Normal", value: "normal" },
                            { label: "Relaxed", value: "relaxed" },
                            { label: "Loose", value: "loose" },
                        ],
                        defaultValue: "",
                        row: true,
                    },
                },
                {
                    id: "sampleText",
                    type: InputType.Text,
                    fieldName: "sampleText",
                    label: "Sample Text",
                    isRequired: false,
                    props: {
                        defaultValue: "The quick brown fox jumps over the lazy dog",
                        multiline: true,
                        rows: 2,
                    },
                },
            ],
            containers: [{
                totalItems: 9,
            }],
        };

        // Initial values for the form
        const initialValues = {
            variant: "body1",
            align: "left",
            color: "text",
            weight: "normal",
            uppercase: false,
            noWrap: false,
            truncate: false,
            spacing: "",
            sampleText: "The quick brown fox jumps over the lazy dog",
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "Typography",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { variant, align, color, weight, uppercase, noWrap, truncate, spacing, sampleText } = values;

                // All typography variants to showcase
                const variants: TypographyVariant[] = [
                    "display1", "display2",
                    "h1", "h2", "h3", "h4", "h5", "h6",
                    "subtitle1", "subtitle2",
                    "body1", "body2",
                    "caption", "overline",
                ];

                return (
                    <Box 
                        className="tw-space-y-6"
                        style={{ background: "transparent" }}
                    >
                        {/* Single Typography Display with Current Settings */}
                        <Box
                            padding="md"
                            borderRadius="md"
                            className="tw-border tw-border-gray-200"
                            style={{ background: "transparent" }}
                        >
                            <Typography
                                variant="overline"
                                color="secondary"
                                className="tw-mb-2"
                            >
                                Current Settings Preview
                            </Typography>
                            <Typography
                                variant={variant as TypographyVariant}
                                align={align as TypographyAlign}
                                color={color as TypographyColor}
                                weight={weight as TypographyWeight}
                                spacing={spacing as TypographySpacing || undefined}
                                uppercase={uppercase}
                                noWrap={noWrap}
                                truncate={truncate}
                            >
                                {sampleText}
                            </Typography>
                        </Box>

                        {/* All Variants Grid */}
                        <Box style={{ background: "transparent" }}>
                            <Typography
                                variant="h5"
                                color="primary"
                                className="tw-mb-4"
                            >
                                All Typography Variants
                            </Typography>
                            <Box 
                                className="tw-grid tw-grid-cols-1 md:tw-grid-cols-2 lg:tw-grid-cols-3 tw-gap-4"
                                style={{ background: "transparent" }}
                            >
                                {variants.map(typographyVariant => (
                                    <Box
                                        key={typographyVariant}
                                        padding="sm"
                                        borderRadius="md"
                                        className="tw-border tw-border-gray-100"
                                        style={{ background: "transparent" }}
                                    >
                                        <Typography
                                            variant="overline"
                                            color="secondary"
                                            className="tw-mb-1"
                                        >
                                            {typographyVariant}
                                        </Typography>
                                        <Typography
                                            variant={typographyVariant}
                                            align={align as TypographyAlign}
                                            color={color as TypographyColor}
                                            weight={weight as TypographyWeight}
                                            spacing={spacing as TypographySpacing || undefined}
                                            uppercase={uppercase}
                                            noWrap={noWrap}
                                            truncate={truncate}
                                        >
                                            {sampleText}
                                        </Typography>
                                    </Box>
                                ))}
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
