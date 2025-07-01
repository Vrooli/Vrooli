import type { Meta, StoryObj } from "@storybook/react";
import { InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../../__test/helpers/storybookDecorators.js";
import { IconCommon } from "../../../icons/Icons.js";
import { Box, Typography, Paper } from "@mui/material";
import { DialogTitle } from "./DialogTitle.js";

const meta: Meta<typeof DialogTitle> = {
    title: "Components/Dialogs/DialogTitle",
    component: DialogTitle,
    parameters: {
        layout: "fullscreen",
        docs: {
            description: {
                component: "A styled dialog title component with enhanced performance, reduced padding, line-clamped titles, and skeleton loading state. Features improved layout where action buttons stay in the same row, animations, responsive design, and support for custom children content for maximum flexibility.",
            },
        },
    },
    tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        // Define the form schema for controls
        const controlsSchema: FormSchema = {
            elements: [
                {
                    id: "titleText",
                    type: InputType.Text,
                    fieldName: "titleText",
                    label: "Title Text",
                    isRequired: false,
                    props: {
                        defaultValue: "Enhanced Dialog Title",
                        placeholder: "Enter title text",
                    },
                },
                {
                    id: "helpText",
                    type: InputType.Text,
                    fieldName: "helpText",
                    label: "Help Text",
                    isRequired: false,
                    props: {
                        defaultValue: "This title showcases all the enhanced features and improvements",
                        placeholder: "Enter help text",
                    },
                },
                {
                    id: "showStartComponent",
                    type: InputType.Switch,
                    fieldName: "showStartComponent",
                    label: "Show Start Icon",
                    isRequired: false,
                    props: {
                        defaultValue: true,
                    },
                },
                {
                    id: "startComponentType",
                    type: InputType.Radio,
                    fieldName: "startComponentType",
                    label: "Start Icon Type",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Settings", value: "Settings" },
                            { label: "User", value: "User" },
                            { label: "Info", value: "Info" },
                            { label: "Warning", value: "Warning" },
                        ],
                        defaultValue: "Settings",
                        row: true,
                    },
                },
                {
                    id: "showBelowContent",
                    type: InputType.Switch,
                    fieldName: "showBelowContent",
                    label: "Show Below Content",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "belowContentType",
                    type: InputType.Radio,
                    fieldName: "belowContentType",
                    label: "Below Content Type",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Info", value: "info" },
                            { label: "Warning", value: "warning" },
                            { label: "Success", value: "success" },
                        ],
                        defaultValue: "info",
                        row: true,
                    },
                },
                {
                    id: "isLoading",
                    type: InputType.Switch,
                    fieldName: "isLoading",
                    label: "Loading State",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "shadow",
                    type: InputType.Switch,
                    fieldName: "shadow",
                    label: "Shadow",
                    isRequired: false,
                    props: {
                        defaultValue: true,
                    },
                },
                {
                    id: "animate",
                    type: InputType.Switch,
                    fieldName: "animate",
                    label: "Animations",
                    isRequired: false,
                    props: {
                        defaultValue: true,
                    },
                },
                {
                    id: "showCloseButton",
                    type: InputType.Switch,
                    fieldName: "showCloseButton",
                    label: "Show Close Button",
                    isRequired: false,
                    props: {
                        defaultValue: true,
                    },
                },
                {
                    id: "useCustomChildren",
                    type: InputType.Switch,
                    fieldName: "useCustomChildren",
                    label: "Use Custom Children",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "variant",
                    type: InputType.Radio,
                    fieldName: "variant",
                    label: "Variant",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Simple", value: "simple" },
                            { label: "Prominent", value: "prominent" },
                        ],
                        defaultValue: "simple",
                        row: true,
                    },
                },
            ],
            containers: [{
                totalItems: 12,
            }],
        };

        // Initial values for the form
        const initialValues = {
            titleText: "Enhanced Dialog Title",
            helpText: "This title showcases all the enhanced features and improvements",
            showStartComponent: true,
            startComponentType: "Settings",
            showBelowContent: false,
            belowContentType: "info",
            isLoading: false,
            shadow: true,
            animate: true,
            showCloseButton: true,
            useCustomChildren: false,
            variant: "simple",
        };

        // Helper function to get below content
        const getBelowContent = (showBelowContent: boolean, belowContentType: string) => {
            if (!showBelowContent) return undefined;
            
            const contentMap = {
                info: {
                    bgcolor: "info.light",
                    color: "info.contrastText",
                    icon: "Info",
                    text: "This is informational content below the title",
                },
                warning: {
                    bgcolor: "warning.light", 
                    color: "warning.contrastText",
                    icon: "Warning",
                    text: "Warning: This action may have important consequences",
                },
                success: {
                    bgcolor: "success.light",
                    color: "success.contrastText", 
                    icon: "CheckCircle",
                    text: "Success: Operation completed successfully",
                },
            };

            const config = contentMap[belowContentType as keyof typeof contentMap];

            return (
                <Box sx={{ 
                    p: 2, 
                    bgcolor: config.bgcolor,
                    color: config.color,
                    fontSize: "0.875rem",
                    display: "flex",
                    alignItems: "center",
                    gap: 1,
                }}>
                    <IconCommon name={config.icon as any} size={16} />
                    {config.text}
                </Box>
            );
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "DialogTitle",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { 
                    titleText, 
                    helpText, 
                    showStartComponent, 
                    startComponentType, 
                    showBelowContent, 
                    belowContentType,
                    isLoading,
                    shadow,
                    animate,
                    showCloseButton,
                    useCustomChildren,
                    variant,
                } = values;

                return (
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                        {/* Main Preview */}
                        <Box>
                            <Typography variant="h6" sx={{ mb: 2 }}>Preview</Typography>
                            <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                {useCustomChildren ? (
                                    <DialogTitle
                                        id="dialog-title-showcase"
                                        onClose={showCloseButton ? () => {} : undefined}
                                        isLoading={isLoading}
                                        shadow={shadow}
                                        animate={animate}
                                        variant={variant as "simple" | "prominent"}
                                        below={getBelowContent(showBelowContent, belowContentType)}
                                    >
                                        <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                                            <IconCommon name="Success" fill="currentColor" size={24} />
                                            <Box>
                                                <Typography variant="h5" component="h2" sx={{ fontWeight: 600, mb: 0.5 }}>
                                                    {titleText}
                                                </Typography>
                                                <Typography variant="body2" sx={{ opacity: 0.7 }}>
                                                    Custom layout with subtitle
                                                </Typography>
                                            </Box>
                                        </Box>
                                    </DialogTitle>
                                ) : (
                                    <DialogTitle
                                        id="dialog-title-showcase"
                                        title={titleText}
                                        help={helpText || undefined}
                                        startComponent={showStartComponent ? <IconCommon name={startComponentType} fill="currentColor" /> : undefined}
                                        below={getBelowContent(showBelowContent, belowContentType)}
                                        onClose={showCloseButton ? () => {} : undefined}
                                        isLoading={isLoading}
                                        shadow={shadow}
                                        animate={animate}
                                        variant={variant as "simple" | "prominent"}
                                    />
                                )}
                            </Paper>
                        </Box>

                        {/* Feature Examples */}
                        <Box>
                            <Typography variant="h6" sx={{ mb: 2 }}>Feature Examples</Typography>
                            
                            <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" }, gap: 3 }}>
                                {/* Simple Variant Example */}
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Simple Variant (Default)</Typography>
                                    <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                        <DialogTitle
                                            id="simple-example"
                                            title="Simple Dialog Title"
                                            help="Clean and minimal for common dialogs"
                                            onClose={() => {}}
                                            variant="simple"
                                        />
                                    </Paper>
                                </Box>

                                {/* Prominent Variant Example */}
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Prominent Variant</Typography>
                                    <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                        <DialogTitle
                                            id="prominent-example"
                                            title="Important Action"
                                            help="Eye-catching for critical dialogs"
                                            startComponent={<IconCommon name="Warning" fill="currentColor" />}
                                            onClose={() => {}}
                                            variant="prominent"
                                        />
                                    </Paper>
                                </Box>

                                {/* Loading Example */}
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Loading State</Typography>
                                    <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                        <DialogTitle
                                            id="loading-example"
                                            title="Loading Dialog"
                                            startComponent={<IconCommon name="Settings" fill="currentColor" />}
                                            below={<Box sx={{ p: 2, bgcolor: "action.hover" }}>Additional content</Box>}
                                            isLoading={true}
                                            onClose={() => {}}
                                        />
                                    </Paper>
                                </Box>

                                {/* Enhanced Example */}
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Enhanced with Animations</Typography>
                                    <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                        <DialogTitle
                                            id="enhanced-example"
                                            title="Enhanced Dialog Title"
                                            help="With smooth animations and improved layout"
                                            startComponent={<IconCommon name="Settings" fill="currentColor" />}
                                            animate={true}
                                            shadow={true}
                                            onClose={() => {}}
                                        />
                                    </Paper>
                                </Box>

                                {/* Full Featured Example */}
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Full Featured</Typography>
                                    <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                        <DialogTitle
                                            id="full-example"
                                            title="Advanced Configuration"
                                            help="Complete example with all features enabled"
                                            startComponent={<IconCommon name="Warning" fill="currentColor" />}
                                            below={
                                                <Box sx={{ 
                                                    p: 2, 
                                                    bgcolor: "warning.light", 
                                                    color: "warning.contrastText",
                                                    fontSize: "0.875rem",
                                                    display: "flex",
                                                    alignItems: "center",
                                                    gap: 1,
                                                }}>
                                                    <IconCommon name="Warning" size={16} />
                                                    Changes may affect system performance
                                                </Box>
                                            }
                                            animate={true}
                                            shadow={true}
                                            onClose={() => {}}
                                        />
                                    </Paper>
                                </Box>

                                {/* Custom Children Example */}
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Custom Children Content</Typography>
                                    <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                        <DialogTitle
                                            id="children-example"
                                            onClose={() => {}}
                                            shadow={true}
                                        >
                                            <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                                                <IconCommon name="Success" fill="currentColor" size={24} />
                                                <Box>
                                                    <Typography variant="h5" component="h2" sx={{ fontWeight: 600, mb: 0.5 }}>
                                                        Custom Title Layout
                                                    </Typography>
                                                    <Typography variant="body2" sx={{ opacity: 0.7 }}>
                                                        With subtitle and custom styling
                                                    </Typography>
                                                </Box>
                                            </Box>
                                        </DialogTitle>
                                    </Paper>
                                </Box>

                                {/* Complex Custom Children Example */}
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Complex Custom Layout</Typography>
                                    <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                        <DialogTitle
                                            id="complex-children-example"
                                            onClose={() => {}}
                                            animate={true}
                                        >
                                            <Box sx={{ textAlign: "center", width: "100%" }}>
                                                <Typography variant="h5" sx={{ fontWeight: "bold", mb: 1 }}>
                                                    Power Up Your AI Assistant 🚀
                                                </Typography>
                                                <Typography variant="body2" sx={{ opacity: 0.8 }}>
                                                    Get credits to run AI-powered tasks and automations
                                                </Typography>
                                            </Box>
                                        </DialogTitle>
                                    </Paper>
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
