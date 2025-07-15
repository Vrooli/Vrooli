import { IconButton, useTheme } from "@mui/material";
import type { FormSchema } from "@vrooli/shared";
import { Formik } from "formik";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Box } from "../../components/layout/Box.js";
import { PageContainer } from "../../components/Page/Page.js";
import { Typography } from "../../components/text/Typography.js";
import { FormView } from "../../forms/FormView/FormView.js";
import { IconCommon } from "../../icons/Icons.js";
import { ScrollBox } from "../../styles.js";

/**
 * Common Storybook decorators for consistent story presentation
 */

/**
 * Centers content in viewport with padding
 * Used primarily for dialogs, modals, and standalone components
 */
export function centeredDecorator(Story: React.ComponentType) {
    return (
        <div style={{
            minHeight: "100vh",
            backgroundColor: "transparent",
            padding: "20px",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
        }}>
            <Story />
        </div>
    );
}

/**
 * Provides a fullscreen container with proper scrolling
 * Used for page views and larger components
 */
export function fullscreenDecorator(Story: React.ComponentType) {
    return (
        <Box
            padding="md"
            style={{
                height: "100vh",
                overflow: "auto",
            }}
            className="bg-background"
        >
            <Story />
        </Box>
    );
}

/**
 * Provides a page container with scrolling
 * Used for views that need the full page structure
 */
export function pageDecorator(Story: React.ComponentType) {
    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Story />
            </ScrollBox>
        </PageContainer>
    );
}

/**
 * Centers content with a max width container
 * Used for forms and content that shouldn't stretch full width
 */
const MAX_WIDTH_DEFAULT = 800;
export function maxWidthDecorator(maxWidth = MAX_WIDTH_DEFAULT) {
    return function MaxWidthDecoratorComponent(Story: React.ComponentType) {
        return (
            <Box sx={{
                width: "100%",
                minHeight: "100vh",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                p: 2,
            }}>
                <Box sx={{ width: "100%", maxWidth }}>
                    <Story />
                </Box>
            </Box>
        );
    };
}

/**
 * Provides a controls panel alongside the story
 * Used for interactive stories with configuration options
 */
const MAX_CONTROLS_WIDTH = 1200;
export function withControlsDecorator(Story: React.ComponentType) {
    return (
        <Box sx={{
            display: "flex",
            gap: 2,
            flexDirection: "column",
            p: 2,
            height: "100vh",
            overflow: "auto",
            bgcolor: "background.default",
            maxWidth: MAX_CONTROLS_WIDTH,
            mx: "auto",
        }}>
            <Story />
        </Box>
    );
}

/**
 * Provides padding without centering
 * Used for components that manage their own layout
 */
const DEFAULT_PADDING = 2;
export function paddedDecorator(padding = DEFAULT_PADDING) {
    return function PaddedDecoratorComponent(Story: React.ComponentType) {
        return (
            <Box sx={{ p: padding }}>
                <Story />
            </Box>
        );
    };
}

/**
 * Simulates a dark background
 * Useful for testing components on dark themes
 */
const DARK_BACKGROUND_COLOR = "#121212";
const DARK_PADDING = 2;
export function darkBackgroundDecorator(Story: React.ComponentType) {
    return (
        <Box sx={{
            minHeight: "100vh",
            bgcolor: DARK_BACKGROUND_COLOR,
            p: DARK_PADDING,
        }}>
            <Story />
        </Box>
    );
}

/**
 * Provides a grid layout for multiple component instances
 * Used for showcasing component variants
 */
const DEFAULT_GRID_COLUMNS = 3;
const DEFAULT_GRID_GAP = 2;
const DEFAULT_GRID_PADDING = 2;
const MIN_RESPONSIVE_COLUMNS = 2;
export function gridDecorator(columns = DEFAULT_GRID_COLUMNS, gap = DEFAULT_GRID_GAP) {
    return function GridDecoratorComponent(Story: React.ComponentType) {
        return (
            <Box sx={{
                display: "grid",
                gridTemplateColumns: {
                    xs: "1fr",
                    sm: `repeat(${Math.min(MIN_RESPONSIVE_COLUMNS, columns)}, 1fr)`,
                    md: `repeat(${columns}, 1fr)`,
                },
                gap,
                p: DEFAULT_GRID_PADDING,
            }}>
                <Story />
            </Box>
        );
    };
}

/**
 * Provides a chat-like container at the bottom of viewport
 * Used for input components that appear at the bottom of a chat interface
 */
const DEFAULT_BOTTOM_MAX_WIDTH = 800;
const BOTTOM_PADDING = "20px";
const BOTTOM_PADDING_BOTTOM = "100px";
const BORDER_COLOR = "#ccc";
export function bottomContainerDecorator(maxWidth = DEFAULT_BOTTOM_MAX_WIDTH) {
    return function BottomContainerDecoratorComponent(Story: React.ComponentType) {
        return (
            <PageContainer>
                <ScrollBox>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "column",
                        justifyContent: "flex-end",
                        minHeight: "100vh",
                    }}>
                        <Box sx={{
                            maxWidth,
                            padding: BOTTOM_PADDING,
                            border: `1px solid ${BORDER_COLOR}`,
                            paddingBottom: BOTTOM_PADDING_BOTTOM,
                        }}>
                            <Story />
                        </Box>
                    </Box>
                </ScrollBox>
            </PageContainer>
        );
    };
}

/**
 * Provides a fixed size container for chat components
 * Used for chat message inputs and similar bounded components
 */
const DEFAULT_CHAT_WIDTH = 350;
const DEFAULT_CHAT_HEIGHT = 400;
const CHAT_BORDER_RADIUS = 2;
export function chatContainerDecorator(width = DEFAULT_CHAT_WIDTH, height = DEFAULT_CHAT_HEIGHT) {
    return function ChatContainerDecoratorComponent(Story: React.ComponentType) {
        return (
            <Box
                height={`${height}px`}
                width={`${width}px`}
                bgcolor="background.paper"
                border="1px solid"
                borderColor="divider"
                borderRadius={CHAT_BORDER_RADIUS}
                overflow="hidden"
                display="flex"
                flexDirection="column"
            >
                <Box flex={1} />
                <Story />
            </Box>
        );
    };
}

/**
 * Provides a fixed height container with full width
 * Used for components that need a specific height but should fill available width
 */
const DEFAULT_FIXED_HEIGHT = 600;
const DEFAULT_FIXED_PADDING = 0;
export function fixedHeightDecorator(height = DEFAULT_FIXED_HEIGHT, padding = DEFAULT_FIXED_PADDING) {
    return function FixedHeightDecoratorComponent(Story: React.ComponentType) {
        return (
            <Box
                height={`${height}px`}
                width="100%"
                bgcolor="background.paper"
                p={padding}
            >
                <Story />
            </Box>
        );
    };
}

/**
 * Provides a constrained height container with border
 * Used for chat and message components that need visual boundaries
 */
const DEFAULT_BORDERED_MAX_WIDTH = 800;
const PAGE_PADDING_BOTTOM_PX = 72;
const BORDERED_PADDING = "20px";
export function borderedContainerDecorator(maxWidth = DEFAULT_BORDERED_MAX_WIDTH, customHeight?: string) {
    return function BorderedContainerDecoratorComponent(Story: React.ComponentType) {
        return (
            <Box sx={{
                height: customHeight || `calc(100vh - ${PAGE_PADDING_BOTTOM_PX}px)`,
                maxWidth,
                padding: BORDERED_PADDING,
                border: `1px solid ${BORDER_COLOR}`,
            }}>
                <Story />
            </Box>
        );
    };
}

/**
 * Provides a page container with bottom padding for navigation
 * Used for components that need space for bottom navigation
 */
const DEFAULT_BOTTOM_NAV_PADDING = 120;
const PAGE_PADDING = 2;
export function pageWithBottomNavDecorator(paddingBottom = DEFAULT_BOTTOM_NAV_PADDING) {
    return function PageWithBottomNavDecoratorComponent(Story: React.ComponentType) {
        return (
            <PageContainer size="fullSize">
                <Box sx={{
                    p: PAGE_PADDING,
                    height: "100%",
                    overflow: "auto",
                    paddingBottom: `${paddingBottom}px`,
                }}>
                    <Story />
                </Box>
            </PageContainer>
        );
    };
}

/**
 * Creates a custom decorator with provided wrapper component
 * Used when stories have their own specific wrapper requirements
 */
export function customWrapperDecorator(Wrapper: React.ComponentType<{ children: React.ReactNode }>) {
    return function CustomWrapperDecoratorComponent(Story: React.ComponentType) {
        return (
            <Wrapper>
                <Story />
            </Wrapper>
        );
    };
}

/**
 * Combines multiple decorators
 * Usage: composeDecorators(centeredDecorator, darkBackgroundDecorator)
 */
export function composeDecorators(...decorators: Array<(Story: React.ComponentType) => JSX.Element>) {
    return function ComposeDecoratorsComponent(Story: React.ComponentType) {
        return decorators.reduceRight((acc, decorator) => () => decorator(acc), Story)();
    };
}

/**
 * Configuration for the showcase decorator
 */
export interface ShowcaseDecoratorConfig {
    /** Name of the component being showcased (used in section titles) */
    componentName: string;
    /** Form schema defining the controls */
    controlsSchema: FormSchema;
    /** Initial values for the form controls */
    initialValues: Record<string, any>;
    /** Function that renders the showcase content based on form values */
    renderShowcase: (values: Record<string, any>) => React.ReactNode;
}

/**
 * Helper component to handle Formik values and call hooks properly
 */
interface FormikValuesHandlerProps {
    values: Record<string, any>;
    onValuesChange: (values: Record<string, any>) => void;
    schema: FormSchema;
    onSchemaChange: () => void;
}

function FormikValuesHandler({ values, onValuesChange, schema, onSchemaChange }: FormikValuesHandlerProps) {
    // Update form values when they change
    useEffect(() => {
        onValuesChange(values);
    }, [values, onValuesChange]);

    return (
        <FormView
            disabled={false}
            isEditing={false}
            schema={schema}
            onSchemaChange={onSchemaChange}
        />
    );
}

/**
 * Creates a showcase decorator with form controls and component display
 * Used for interactive stories with configuration options like Button showcase
 */
export function showcaseDecorator(config: ShowcaseDecoratorConfig) {
    return function ShowcaseDecoratorComponent() {
        const theme = useTheme();
        const [formValues, setFormValues] = useState(config.initialValues);
        const [isControlsCollapsed, setIsControlsCollapsed] = useState(false);
        const [showcaseBackgroundColor, setShowcaseBackgroundColor] = useState(theme.palette.background.paper);
        const [colorInputValue, setColorInputValue] = useState(theme.palette.background.paper);
        const colorInputRef = useRef<HTMLInputElement>(null);
        const isColorPickerActiveRef = useRef(false);

        // Update the default color when theme changes
        useEffect(() => {
            setShowcaseBackgroundColor(theme.palette.background.paper);
            setColorInputValue(theme.palette.background.paper);
        }, [theme.palette.background.paper]);

        const handleFormChange = useCallback((values: Record<string, any>) => {
            setFormValues(values);
        }, []);

        const toggleControls = useCallback(() => {
            setIsControlsCollapsed(!isControlsCollapsed);
        }, [isControlsCollapsed]);

        // Only update the input value while dragging
        const handleBackgroundColorInput = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
            setColorInputValue(event.target.value);
        }, []);

        // Update the actual background color when dragging stops
        const handleBackgroundColorChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
            const newColor = event.target.value;
            setShowcaseBackgroundColor(newColor);
        }, []);

        // Reset colors to theme default
        const handleResetColors = useCallback(() => {
            const defaultColor = theme.palette.background.paper;
            setColorInputValue(defaultColor);
            setShowcaseBackgroundColor(defaultColor);
        }, [theme.palette.background.paper]);

        // No-op handlers for form components
        const handleSubmit = useCallback(() => {
            // No-op for showcase purposes
        }, []);

        const handleSchemaChange = useCallback(() => {
            // No-op for showcase purposes
        }, []);

        // Memoized style objects
        const containerStyle = useMemo(() => ({
            height: "100vh",
            overflow: "auto" as const,
        }), []);

        const flexContainerStyle = useMemo(() => ({
            display: "flex",
            gap: "1rem",
            flexDirection: "column" as const,
            maxWidth: "1400px",
            margin: "0 auto",
        }), []);

        const controlsContainerStyle = useMemo(() => ({
            width: "100%",
        }), []);

        const headerStyle = useMemo(() => ({
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: isControlsCollapsed ? "0" : "1rem",
            background: "transparent",
        }), [isControlsCollapsed]);

        const backgroundControlsStyle = useMemo(() => ({
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
        }), []);

        const colorInputStyle = useMemo(() => ({
            width: "32px",
            height: "32px",
            border: "none",
            borderRadius: "4px",
            cursor: "pointer",
        }), []);

        const showcaseBoxStyle = useMemo(() => ({
            width: "100%",
            backgroundColor: showcaseBackgroundColor,
        }), [showcaseBackgroundColor]);

        // Setup event listeners for color picker
        useEffect(() => {
            const colorInput = colorInputRef.current;
            if (!colorInput) return;

            function handleMouseDown() {
                isColorPickerActiveRef.current = true;
            }

            function handleMouseUp() {
                if (isColorPickerActiveRef.current) {
                    isColorPickerActiveRef.current = false;
                    setShowcaseBackgroundColor(colorInputValue);
                }
            }

            colorInput.addEventListener("mousedown", handleMouseDown);
            window.addEventListener("mouseup", handleMouseUp);

            return function cleanup() {
                colorInput.removeEventListener("mousedown", handleMouseDown);
                window.removeEventListener("mouseup", handleMouseUp);
            };
        }, [colorInputValue]);

        return (
            <Box
                padding="md"
                style={containerStyle}
                className="bg-background"
            >
                <Box
                    style={flexContainerStyle}
                >
                    {/* Controls Section */}
                    <Box
                        variant="elevated"
                        padding="lg"
                        borderRadius="lg"
                        style={controlsContainerStyle}
                    >
                        <Box
                            style={headerStyle}
                        >
                            <Typography variant="h5" color="primary">
                                {config.componentName} Controls
                            </Typography>
                            <IconButton
                                onClick={toggleControls}
                                size="small"
                                aria-label={isControlsCollapsed ? "Expand controls" : "Collapse controls"}
                            >
                                {isControlsCollapsed ?
                                    <IconCommon name="ExpandMore" /> :
                                    <IconCommon name="ExpandLess" />
                                }
                            </IconButton>
                        </Box>

                        {!isControlsCollapsed && (
                            <Formik
                                initialValues={config.initialValues}
                                onSubmit={handleSubmit}
                                enableReinitialize
                            >
                                {({ values }) => (
                                    <FormikValuesHandler
                                        values={values}
                                        onValuesChange={handleFormChange}
                                        schema={config.controlsSchema}
                                        onSchemaChange={handleSchemaChange}
                                    />
                                )}
                            </Formik>
                        )}
                    </Box>

                    {/* Showcase Display */}
                    <Box
                        variant="elevated"
                        padding="lg"
                        borderRadius="lg"
                        style={showcaseBoxStyle}
                    >
                        <Box
                            style={headerStyle}
                        >
                            <Typography variant="h5" color="primary">
                                {config.componentName} Showcase
                            </Typography>
                            <Box style={backgroundControlsStyle}>
                                <Typography variant="body2" color="text">
                                    Background:
                                </Typography>
                                <input
                                    ref={colorInputRef}
                                    type="color"
                                    value={colorInputValue}
                                    onInput={handleBackgroundColorInput}
                                    onChange={handleBackgroundColorChange}
                                    style={colorInputStyle}
                                    title="Change showcase background color"
                                />
                                <IconButton
                                    size="small"
                                    onClick={handleResetColors}
                                    title="Reset to theme default"
                                >
                                    <IconCommon name="Refresh" />
                                </IconButton>
                            </Box>
                        </Box>

                        {useMemo(() => config.renderShowcase(formValues), [formValues])}
                    </Box>
                </Box>
            </Box>
        );
    };
}
