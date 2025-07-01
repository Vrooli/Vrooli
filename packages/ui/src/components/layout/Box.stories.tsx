import type { Meta, StoryObj } from "@storybook/react";
import { Box, BoxFactory } from "./Box.js";

const meta: Meta<typeof Box> = {
    title: "Components/Layout/Box",
    component: Box,
    tags: ["autodocs"],
    argTypes: {
        component: {
            control: { type: "select" },
            options: ["div", "section", "article", "main", "aside", "header", "footer"],
            description: "HTML element type to render",
        },
        variant: {
            control: { type: "select" },
            options: ["default", "paper", "outlined", "elevated", "subtle"],
            description: "Visual style variant of the box",
        },
        padding: {
            control: { type: "select" },
            options: ["none", "xs", "sm", "md", "lg", "xl"],
            description: "Padding size",
        },
        borderRadius: {
            control: { type: "select" },
            options: ["none", "sm", "md", "lg", "xl", "full"],
            description: "Border radius size",
        },
        fullWidth: {
            control: { type: "boolean" },
            description: "Whether the box should take full width",
        },
        fullHeight: {
            control: { type: "boolean" },
            description: "Whether the box should take full height",
        },
        children: {
            control: { type: "text" },
            description: "Content to display inside the box",
        },
    },
    parameters: {
        layout: "padded",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default box with no styling
 */
export const Default: Story = {
    args: {
        children: "This is a default box",
        padding: "md",
    },
};

/**
 * Paper variant with background
 */
export const Paper: Story = {
    args: {
        variant: "paper",
        padding: "md",
        borderRadius: "md",
        children: "This is a paper box with background and shadow",
    },
};

/**
 * Outlined variant with border
 */
export const Outlined: Story = {
    args: {
        variant: "outlined",
        padding: "md",
        borderRadius: "md",
        children: "This is an outlined box with border",
    },
};

/**
 * Elevated variant with shadow
 */
export const Elevated: Story = {
    args: {
        variant: "elevated",
        padding: "lg",
        borderRadius: "lg",
        children: "This is an elevated box with shadow",
    },
};

/**
 * All variants showcase
 */
export const AllVariants: Story = {
    render: () => (
        <div className="tw-space-y-4">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Box Variants</h3>
            
            <div className="tw-space-y-4">
                <Box variant="default" padding="md" borderRadius="md">
                    <strong>Default:</strong> Basic box with no background
                </Box>
                
                <Box variant="paper" padding="md" borderRadius="md">
                    <strong>Paper:</strong> Paper background with subtle shadow
                </Box>
                
                <Box variant="outlined" padding="md" borderRadius="md">
                    <strong>Outlined:</strong> Paper background with border
                </Box>
                
                <Box variant="elevated" padding="md" borderRadius="md">
                    <strong>Elevated:</strong> Paper background with medium shadow
                </Box>
                
                <Box variant="subtle" padding="md" borderRadius="md">
                    <strong>Subtle:</strong> Light gray background
                </Box>
            </div>
        </div>
    ),
};

/**
 * Padding size variations
 */
export const PaddingSizes: Story = {
    render: () => (
        <div className="tw-space-y-4">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Padding Sizes</h3>
            
            <div className="tw-space-y-4">
                <Box variant="outlined" padding="none" borderRadius="md">
                    <span className="tw-bg-blue-200 tw-p-1 tw-rounded">None: No padding</span>
                </Box>
                
                <Box variant="outlined" padding="xs" borderRadius="md">
                    <span className="tw-bg-blue-200 tw-p-1 tw-rounded">XS: Extra small padding</span>
                </Box>
                
                <Box variant="outlined" padding="sm" borderRadius="md">
                    <span className="tw-bg-blue-200 tw-p-1 tw-rounded">SM: Small padding</span>
                </Box>
                
                <Box variant="outlined" padding="md" borderRadius="md">
                    <span className="tw-bg-blue-200 tw-p-1 tw-rounded">MD: Medium padding</span>
                </Box>
                
                <Box variant="outlined" padding="lg" borderRadius="md">
                    <span className="tw-bg-blue-200 tw-p-1 tw-rounded">LG: Large padding</span>
                </Box>
                
                <Box variant="outlined" padding="xl" borderRadius="md">
                    <span className="tw-bg-blue-200 tw-p-1 tw-rounded">XL: Extra large padding</span>
                </Box>
            </div>
        </div>
    ),
};

/**
 * Border radius variations
 */
export const BorderRadiusOptions: Story = {
    render: () => (
        <div className="tw-space-y-4">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Border Radius Options</h3>
            
            <div className="tw-grid tw-grid-cols-2 tw-gap-4 md:tw-grid-cols-3">
                <Box variant="elevated" padding="md" borderRadius="none">
                    <strong>None:</strong> No radius
                </Box>
                
                <Box variant="elevated" padding="md" borderRadius="sm">
                    <strong>SM:</strong> Small radius
                </Box>
                
                <Box variant="elevated" padding="md" borderRadius="md">
                    <strong>MD:</strong> Medium radius
                </Box>
                
                <Box variant="elevated" padding="md" borderRadius="lg">
                    <strong>LG:</strong> Large radius
                </Box>
                
                <Box variant="elevated" padding="md" borderRadius="xl">
                    <strong>XL:</strong> Extra large radius
                </Box>
                
                <Box variant="elevated" padding="md" borderRadius="full" className="tw-aspect-square tw-flex tw-items-center tw-justify-center">
                    <strong>Full:</strong> Complete circle
                </Box>
            </div>
        </div>
    ),
};

/**
 * Different HTML element types
 */
export const ElementTypes: Story = {
    render: () => (
        <div className="tw-space-y-4">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">HTML Element Types</h3>
            
            <div className="tw-space-y-4">
                <Box component="div" variant="paper" padding="md" borderRadius="md">
                    <strong>&lt;div&gt;</strong> Default element type
                </Box>
                
                <Box component="section" variant="paper" padding="md" borderRadius="md">
                    <strong>&lt;section&gt;</strong> Semantic section element
                </Box>
                
                <Box component="article" variant="paper" padding="md" borderRadius="md">
                    <strong>&lt;article&gt;</strong> Article content element
                </Box>
                
                <Box component="aside" variant="paper" padding="md" borderRadius="md">
                    <strong>&lt;aside&gt;</strong> Sidebar element
                </Box>
            </div>
        </div>
    ),
};

/**
 * Layout examples
 */
export const LayoutExamples: Story = {
    render: () => (
        <div className="tw-space-y-6">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Layout Examples</h3>
            
            {/* Full width example */}
            <div>
                <h4 className="tw-text-md tw-font-medium tw-mb-2">Full Width Container</h4>
                <Box variant="elevated" padding="lg" borderRadius="lg" fullWidth>
                    This box takes the full width of its container
                </Box>
            </div>
            
            {/* Card layout */}
            <div>
                <h4 className="tw-text-md tw-font-medium tw-mb-2">Card Layout</h4>
                <div className="tw-grid tw-grid-cols-1 md:tw-grid-cols-2 lg:tw-grid-cols-3 tw-gap-4">
                    <Box variant="elevated" padding="lg" borderRadius="lg">
                        <h5 className="tw-font-semibold tw-mb-2">Card 1</h5>
                        <p>This is card content with elevated styling</p>
                    </Box>
                    <Box variant="elevated" padding="lg" borderRadius="lg">
                        <h5 className="tw-font-semibold tw-mb-2">Card 2</h5>
                        <p>Another card with the same styling</p>
                    </Box>
                    <Box variant="elevated" padding="lg" borderRadius="lg">
                        <h5 className="tw-font-semibold tw-mb-2">Card 3</h5>
                        <p>Third card in the grid layout</p>
                    </Box>
                </div>
            </div>
            
            {/* Nested boxes */}
            <div>
                <h4 className="tw-text-md tw-font-medium tw-mb-2">Nested Boxes</h4>
                <Box variant="outlined" padding="lg" borderRadius="lg">
                    <h5 className="tw-font-semibold tw-mb-2">Outer Container</h5>
                    <Box variant="subtle" padding="md" borderRadius="md">
                        <h6 className="tw-font-medium tw-mb-1">Inner Container</h6>
                        <p>Nested box content</p>
                    </Box>
                </Box>
            </div>
        </div>
    ),
};

/**
 * Factory components showcase
 */
export const FactoryComponents: Story = {
    render: () => (
        <div className="tw-space-y-6">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Factory Components</h3>
            
            <div className="tw-space-y-4">
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">BoxFactory.Paper</h4>
                    <BoxFactory.Paper padding="md" borderRadius="md">
                        Pre-configured paper variant
                    </BoxFactory.Paper>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">BoxFactory.Card</h4>
                    <BoxFactory.Card>
                        Pre-configured card with padding and border radius
                    </BoxFactory.Card>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">BoxFactory.Outlined</h4>
                    <BoxFactory.Outlined padding="md" borderRadius="md">
                        Pre-configured outlined variant
                    </BoxFactory.Outlined>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">BoxFactory.FlexCenter</h4>
                    <BoxFactory.FlexCenter variant="outlined" padding="lg" borderRadius="md" className="tw-h-20">
                        Centered content with flex
                    </BoxFactory.FlexCenter>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">BoxFactory.FlexBetween</h4>
                    <BoxFactory.FlexBetween variant="outlined" padding="md" borderRadius="md">
                        <span>Left content</span>
                        <span>Right content</span>
                    </BoxFactory.FlexBetween>
                </div>
            </div>
        </div>
    ),
};

/**
 * Real-world usage example
 */
export const RealWorldExample: Story = {
    render: () => (
        <div className="tw-max-w-4xl tw-mx-auto tw-space-y-6">
            <h3 className="tw-text-xl tw-font-bold tw-mb-6">Dashboard Layout Example</h3>
            
            {/* Header */}
            <BoxFactory.FlexBetween variant="paper" padding="lg" borderRadius="lg">
                <h1 className="tw-text-2xl tw-font-bold">Dashboard</h1>
                <div className="tw-flex tw-space-x-2">
                    <Box variant="outlined" padding="sm" borderRadius="md" className="tw-text-sm">
                        Settings
                    </Box>
                    <Box variant="elevated" padding="sm" borderRadius="md" className="tw-text-sm">
                        Profile
                    </Box>
                </div>
            </BoxFactory.FlexBetween>
            
            {/* Stats Grid */}
            <div className="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4">
                <BoxFactory.Card>
                    <h3 className="tw-text-lg tw-font-semibold tw-mb-2">Total Users</h3>
                    <p className="tw-text-3xl tw-font-bold tw-text-blue-600">1,234</p>
                    <p className="tw-text-sm tw-text-gray-600">+12% from last month</p>
                </BoxFactory.Card>
                
                <BoxFactory.Card>
                    <h3 className="tw-text-lg tw-font-semibold tw-mb-2">Revenue</h3>
                    <p className="tw-text-3xl tw-font-bold tw-text-green-600">$45,678</p>
                    <p className="tw-text-sm tw-text-gray-600">+8% from last month</p>
                </BoxFactory.Card>
                
                <BoxFactory.Card>
                    <h3 className="tw-text-lg tw-font-semibold tw-mb-2">Orders</h3>
                    <p className="tw-text-3xl tw-font-bold tw-text-purple-600">567</p>
                    <p className="tw-text-sm tw-text-gray-600">+15% from last month</p>
                </BoxFactory.Card>
            </div>
            
            {/* Content Area */}
            <div className="tw-grid tw-grid-cols-1 lg:tw-grid-cols-3 tw-gap-6">
                {/* Main Content */}
                <Box component="main" variant="paper" padding="lg" borderRadius="lg" className="lg:tw-col-span-2">
                    <h2 className="tw-text-xl tw-font-semibold tw-mb-4">Recent Activity</h2>
                    <div className="tw-space-y-3">
                        <Box variant="subtle" padding="md" borderRadius="md">
                            <p className="tw-font-medium">New user registration</p>
                            <p className="tw-text-sm tw-text-gray-600">john.doe@example.com - 2 hours ago</p>
                        </Box>
                        <Box variant="subtle" padding="md" borderRadius="md">
                            <p className="tw-font-medium">Order completed</p>
                            <p className="tw-text-sm tw-text-gray-600">Order #12345 - 4 hours ago</p>
                        </Box>
                        <Box variant="subtle" padding="md" borderRadius="md">
                            <p className="tw-font-medium">Payment processed</p>
                            <p className="tw-text-sm tw-text-gray-600">$299.99 - 6 hours ago</p>
                        </Box>
                    </div>
                </Box>
                
                {/* Sidebar */}
                <Box component="aside" variant="outlined" padding="lg" borderRadius="lg">
                    <h2 className="tw-text-xl tw-font-semibold tw-mb-4">Quick Actions</h2>
                    <div className="tw-space-y-2">
                        <Box variant="subtle" padding="sm" borderRadius="md" className="tw-cursor-pointer hover:tw-bg-gray-200 tw-transition-colors">
                            Add New User
                        </Box>
                        <Box variant="subtle" padding="sm" borderRadius="md" className="tw-cursor-pointer hover:tw-bg-gray-200 tw-transition-colors">
                            Generate Report
                        </Box>
                        <Box variant="subtle" padding="sm" borderRadius="md" className="tw-cursor-pointer hover:tw-bg-gray-200 tw-transition-colors">
                            View Analytics
                        </Box>
                    </div>
                </Box>
            </div>
        </div>
    ),
};
