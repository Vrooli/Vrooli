import React from "react";
import Box from "@mui/material/Box";
import { PageContainer } from "../../components/Page/Page.js";
import { ScrollBox } from "../../styles.js";

/**
 * Common Storybook decorators for consistent story presentation
 */

/**
 * Centers content in viewport with padding
 * Used primarily for dialogs, modals, and standalone components
 */
export const centeredDecorator = (Story: React.ComponentType) => (
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

/**
 * Provides a fullscreen container with proper scrolling
 * Used for page views and larger components
 */
export const fullscreenDecorator = (Story: React.ComponentType) => (
    <Box sx={{ 
        p: 2, 
        height: "100vh", 
        overflow: "auto",
        bgcolor: "background.default" 
    }}>
        <Story />
    </Box>
);

/**
 * Provides a page container with scrolling
 * Used for views that need the full page structure
 */
export const pageDecorator = (Story: React.ComponentType) => (
    <PageContainer size="fullSize">
        <ScrollBox>
            <Story />
        </ScrollBox>
    </PageContainer>
);

/**
 * Centers content with a max width container
 * Used for forms and content that shouldn't stretch full width
 */
export const maxWidthDecorator = (maxWidth = 800) => (Story: React.ComponentType) => (
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

/**
 * Provides a controls panel alongside the story
 * Used for interactive stories with configuration options
 */
export const withControlsDecorator = (Story: React.ComponentType) => (
    <Box sx={{ 
        display: "flex", 
        gap: 2, 
        flexDirection: "column",
        p: 2,
        height: "100vh",
        overflow: "auto",
        bgcolor: "background.default",
        maxWidth: 1200, 
        mx: "auto" 
    }}>
        <Story />
    </Box>
);

/**
 * Provides padding without centering
 * Used for components that manage their own layout
 */
export const paddedDecorator = (padding = 2) => (Story: React.ComponentType) => (
    <Box sx={{ p: padding }}>
        <Story />
    </Box>
);

/**
 * Simulates a dark background
 * Useful for testing components on dark themes
 */
export const darkBackgroundDecorator = (Story: React.ComponentType) => (
    <Box sx={{
        minHeight: "100vh",
        bgcolor: "#121212",
        p: 2,
    }}>
        <Story />
    </Box>
);

/**
 * Provides a grid layout for multiple component instances
 * Used for showcasing component variants
 */
export const gridDecorator = (columns = 3, gap = 2) => (Story: React.ComponentType) => (
    <Box sx={{
        display: "grid",
        gridTemplateColumns: { 
            xs: "1fr", 
            sm: `repeat(${Math.min(2, columns)}, 1fr)`, 
            md: `repeat(${columns}, 1fr)` 
        },
        gap,
        p: 2,
    }}>
        <Story />
    </Box>
);

/**
 * Provides a chat-like container at the bottom of viewport
 * Used for input components that appear at the bottom of a chat interface
 */
export const bottomContainerDecorator = (maxWidth = 800) => (Story: React.ComponentType) => (
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
                    padding: "20px",
                    border: "1px solid #ccc",
                    paddingBottom: "100px",
                }}>
                    <Story />
                </Box>
            </Box>
        </ScrollBox>
    </PageContainer>
);

/**
 * Provides a fixed size container for chat components
 * Used for chat message inputs and similar bounded components
 */
export const chatContainerDecorator = (width = 350, height = 400) => (Story: React.ComponentType) => (
    <Box
        height={`${height}px`}
        width={`${width}px`}
        bgcolor="background.paper"
        border="1px solid"
        borderColor="divider"
        borderRadius={2}
        overflow="hidden"
        display="flex"
        flexDirection="column"
    >
        <Box flex={1} />
        <Story />
    </Box>
);

/**
 * Provides a fixed height container with full width
 * Used for components that need a specific height but should fill available width
 */
export const fixedHeightDecorator = (height = 600, padding = 0) => (Story: React.ComponentType) => (
    <Box 
        height={`${height}px`} 
        width="100%" 
        bgcolor="background.paper"
        p={padding}
    >
        <Story />
    </Box>
);

/**
 * Provides a constrained height container with border
 * Used for chat and message components that need visual boundaries
 */
export const borderedContainerDecorator = (maxWidth = 800, customHeight?: string) => (Story: React.ComponentType) => (
    <Box sx={{
        height: customHeight || `calc(100vh - ${72}px)`, // 72px is pagePaddingBottom
        maxWidth,
        padding: "20px",
        border: "1px solid #ccc",
    }}>
        <Story />
    </Box>
);

/**
 * Provides a page container with bottom padding for navigation
 * Used for components that need space for bottom navigation
 */
export const pageWithBottomNavDecorator = (paddingBottom = 120) => (Story: React.ComponentType) => (
    <PageContainer size="fullSize">
        <Box sx={{
            p: 2,
            height: "100%",
            overflow: "auto",
            paddingBottom: `${paddingBottom}px`
        }}>
            <Story />
        </Box>
    </PageContainer>
);

/**
 * Creates a custom decorator with provided wrapper component
 * Used when stories have their own specific wrapper requirements
 */
export const customWrapperDecorator = (Wrapper: React.ComponentType<{ children: React.ReactNode }>) => 
    (Story: React.ComponentType) => (
        <Wrapper>
            <Story />
        </Wrapper>
    );

/**
 * Combines multiple decorators
 * Usage: composeDecorators(centeredDecorator, darkBackgroundDecorator)
 */
export const composeDecorators = (...decorators: Array<(Story: React.ComponentType) => JSX.Element>) => 
    (Story: React.ComponentType) => 
        decorators.reduceRight((acc, decorator) => () => decorator(acc), Story)();