import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import type { Meta, StoryObj } from "@storybook/react";
import { InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../__test/helpers/storybookDecorators.js";
import { Box } from "../../components/layout/Box.js";
import { GeneratedGridItem } from "./GeneratedGridItem.js";

const meta: Meta<typeof GeneratedGridItem> = {
    title: "Components/Forms/GeneratedGridItem",
    component: GeneratedGridItem,
    parameters: {
        layout: "fullscreen",
    },
    argTypes: {
        children: {
            control: false,
            description: "The child element to be wrapped",
        },
        fieldsInGrid: {
            control: { type: "range", min: 1, max: 6, step: 1 },
            description: "Number of fields in the grid (affects sizing)",
        },
        isInGrid: {
            control: "boolean",
            description: "Whether to wrap in a Grid item or not",
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Helper component to visualize the grid item
const SampleContent: React.FC<{ label: string }> = ({ label }) => (
    <Card sx={{ height: "100%", minHeight: 100 }}>
        <CardContent>
            <Typography variant="h6">{label}</Typography>
            <Typography variant="body2" color="text.secondary">
                This is a sample form field
            </Typography>
        </CardContent>
    </Card>
);

export const Default: Story = {
    args: {
        children: <SampleContent label="Default Field" />,
        fieldsInGrid: 1,
        isInGrid: true,
    },
};

export const NotInGrid: Story = {
    args: {
        children: <SampleContent label="Not in Grid" />,
        fieldsInGrid: 1,
        isInGrid: false,
    },
};

export const Showcase: Story = {
    render: () => {
        // Define the form schema for controls
        const controlsSchema: FormSchema = {
            elements: [
                {
                    id: "isInGrid",
                    type: InputType.Switch,
                    fieldName: "isInGrid",
                    label: "Wrap in Grid",
                    isRequired: false,
                    props: {
                        defaultValue: true,
                    },
                },
                {
                    id: "fieldsInGrid",
                    type: InputType.Slider,
                    fieldName: "fieldsInGrid",
                    label: "Number of Fields in Grid",
                    isRequired: false,
                    props: {
                        min: 1,
                        max: 6,
                        step: 1,
                        defaultValue: 3,
                        marks: true,
                    },
                },
            ],
            containers: [{
                totalItems: 2,
                spacing: 3,
                xs: 1,
                sm: 1,
                md: 2,
                lg: 2,
            }],
        };

        // Initial values for the form
        const initialValues = {
            isInGrid: true,
            fieldsInGrid: 3,
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "GeneratedGridItem",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { isInGrid, fieldsInGrid } = values;

                return (
                    <Box>
                        <Typography variant="h6" sx={{ mb: 2 }}>
                            Grid Sizing Preview
                        </Typography>
                        
                        {/* Show single item */}
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>
                            Single Item (current settings):
                        </Typography>
                        <Grid container spacing={2} sx={{ mb: 4 }}>
                            <GeneratedGridItem
                                fieldsInGrid={fieldsInGrid}
                                isInGrid={isInGrid}
                            >
                                <SampleContent label={`Field (${fieldsInGrid} items in grid)`} />
                            </GeneratedGridItem>
                        </Grid>

                        {/* Show multiple items in a grid */}
                        {isInGrid && (
                            <>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                                    Multiple Items in Grid (showing {fieldsInGrid} items):
                                </Typography>
                                <Grid container spacing={2} sx={{ mb: 4 }}>
                                    {Array.from({ length: fieldsInGrid }).map((_, index) => (
                                        <GeneratedGridItem
                                            key={index}
                                            fieldsInGrid={fieldsInGrid}
                                            isInGrid={true}
                                        >
                                            <SampleContent label={`Field ${index + 1}`} />
                                        </GeneratedGridItem>
                                    ))}
                                </Grid>
                            </>
                        )}

                        {/* Comparison of different field counts */}
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>
                            Grid Size Comparison:
                        </Typography>
                        {[1, 2, 3, 4].map(count => (
                            <Box key={count} sx={{ mb: 3 }}>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                                    {count} field{count > 1 ? "s" : ""} in grid:
                                </Typography>
                                <Grid container spacing={2}>
                                    {Array.from({ length: count }).map((_, index) => (
                                        <GeneratedGridItem
                                            key={index}
                                            fieldsInGrid={count}
                                            isInGrid={true}
                                        >
                                            <SampleContent label={`${count}-${index + 1}`} />
                                        </GeneratedGridItem>
                                    ))}
                                </Grid>
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

export const GridSizes: Story = {
    render: () => (
        <Box padding="md">
            <Typography variant="h5" sx={{ mb: 3 }}>
                Grid Item Sizing Based on Field Count
            </Typography>
            
            {[1, 2, 3, 4, 5].map(count => (
                <Box key={count} sx={{ mb: 4 }}>
                    <Typography variant="h6" sx={{ mb: 1 }}>
                        {count} Field{count > 1 ? "s" : ""} in Grid
                    </Typography>
                    <Grid container spacing={2}>
                        {Array.from({ length: count }).map((_, index) => (
                            <GeneratedGridItem
                                key={index}
                                fieldsInGrid={count}
                                isInGrid={true}
                            >
                                <SampleContent label={`Field ${index + 1}`} />
                            </GeneratedGridItem>
                        ))}
                    </Grid>
                </Box>
            ))}
        </Box>
    ),
};

export const NotWrappedInGrid: Story = {
    render: () => (
        <Box padding="md">
            <Typography variant="h5" sx={{ mb: 3 }}>
                Items Not Wrapped in Grid
            </Typography>
            
            <Typography variant="body1" sx={{ mb: 2 }}>
                When isInGrid is false, items are rendered as direct children without Grid wrapping:
            </Typography>
            
            <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                {[1, 2, 3].map((num) => (
                    <GeneratedGridItem
                        key={num}
                        fieldsInGrid={3}
                        isInGrid={false}
                    >
                        <SampleContent label={`Non-Grid Field ${num}`} />
                    </GeneratedGridItem>
                ))}
            </Box>
        </Box>
    ),
};