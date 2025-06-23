import type { Meta, StoryObj } from "@storybook/react";
import { Stack, StackFactory } from "./Stack.js";

const meta: Meta<typeof Stack> = {
    title: "Components/Layout/Stack",
    component: Stack,
    tags: ["autodocs"],
    argTypes: {
        component: {
            control: { type: "select" },
            options: ["div", "section", "article", "main", "aside", "header", "footer", "nav"],
            description: "HTML element type to render",
        },
        direction: {
            control: { type: "select" },
            options: ["row", "column", "row-reverse", "column-reverse"],
            description: "Flex direction",
        },
        justify: {
            control: { type: "select" },
            options: ["start", "end", "center", "between", "around", "evenly"],
            description: "Justify content alignment",
        },
        align: {
            control: { type: "select" },
            options: ["start", "end", "center", "baseline", "stretch"],
            description: "Align items alignment",
        },
        wrap: {
            control: { type: "select" },
            options: ["nowrap", "wrap", "wrap-reverse"],
            description: "Flex wrap behavior",
        },
        spacing: {
            control: { type: "select" },
            options: ["none", "xs", "sm", "md", "lg", "xl", "2xl"],
            description: "Spacing between items",
        },
        fullWidth: {
            control: { type: "boolean" },
            description: "Whether the stack should take full width",
        },
        fullHeight: {
            control: { type: "boolean" },
            description: "Whether the stack should take full height",
        },
        divider: {
            control: { type: "boolean" },
            description: "Whether to divide items with a visual separator",
        },
        children: {
            control: { type: "text" },
            description: "Content to display inside the stack",
        },
    },
    parameters: {
        layout: "padded",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default vertical stack
 */
export const Default: Story = {
    args: {
        direction: "column",
        spacing: "md",
        children: (
            <>
                <div className="tw-bg-blue-100 tw-p-3 tw-rounded">Item 1</div>
                <div className="tw-bg-green-100 tw-p-3 tw-rounded">Item 2</div>
                <div className="tw-bg-yellow-100 tw-p-3 tw-rounded">Item 3</div>
            </>
        ),
    },
};

/**
 * Horizontal stack layout
 */
export const Horizontal: Story = {
    args: {
        direction: "row",
        spacing: "md",
        children: (
            <>
                <div className="tw-bg-blue-100 tw-p-3 tw-rounded">Item 1</div>
                <div className="tw-bg-green-100 tw-p-3 tw-rounded">Item 2</div>
                <div className="tw-bg-yellow-100 tw-p-3 tw-rounded">Item 3</div>
            </>
        ),
    },
};

/**
 * Stack with dividers
 */
export const WithDividers: Story = {
    args: {
        direction: "row",
        spacing: "md",
        divider: true,
        children: (
            <>
                <div className="tw-px-3 tw-py-2">Home</div>
                <div className="tw-px-3 tw-py-2">About</div>
                <div className="tw-px-3 tw-py-2">Services</div>
                <div className="tw-px-3 tw-py-2">Contact</div>
            </>
        ),
    },
};

/**
 * All direction variations
 */
export const AllDirections: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Stack Directions</h3>
            
            <div className="tw-space-y-6">
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">Row (default)</h4>
                    <Stack direction="row" spacing="md" className="tw-p-4 tw-border tw-border-gray-200 tw-rounded">
                        <div className="tw-bg-blue-100 tw-p-2 tw-rounded">1</div>
                        <div className="tw-bg-green-100 tw-p-2 tw-rounded">2</div>
                        <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">3</div>
                    </Stack>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">Column</h4>
                    <Stack direction="column" spacing="md" className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-w-32">
                        <div className="tw-bg-blue-100 tw-p-2 tw-rounded">1</div>
                        <div className="tw-bg-green-100 tw-p-2 tw-rounded">2</div>
                        <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">3</div>
                    </Stack>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">Row Reverse</h4>
                    <Stack direction="row-reverse" spacing="md" className="tw-p-4 tw-border tw-border-gray-200 tw-rounded">
                        <div className="tw-bg-blue-100 tw-p-2 tw-rounded">1</div>
                        <div className="tw-bg-green-100 tw-p-2 tw-rounded">2</div>
                        <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">3</div>
                    </Stack>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">Column Reverse</h4>
                    <Stack direction="column-reverse" spacing="md" className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-w-32">
                        <div className="tw-bg-blue-100 tw-p-2 tw-rounded">1</div>
                        <div className="tw-bg-green-100 tw-p-2 tw-rounded">2</div>
                        <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">3</div>
                    </Stack>
                </div>
            </div>
        </div>
    ),
};

/**
 * Justify content variations
 */
export const JustifyContent: Story = {
    render: () => (
        <div className="tw-space-y-6">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Justify Content Options</h3>
            
            <div className="tw-space-y-4">
                {(["start", "end", "center", "between", "around", "evenly"] as const).map((justify) => (
                    <div key={justify}>
                        <h4 className="tw-text-sm tw-font-medium tw-mb-2 tw-capitalize">{justify}</h4>
                        <Stack 
                            direction="row" 
                            justify={justify} 
                            className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-w-full tw-h-16"
                        >
                            <div className="tw-bg-blue-100 tw-p-2 tw-rounded">A</div>
                            <div className="tw-bg-green-100 tw-p-2 tw-rounded">B</div>
                            <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">C</div>
                        </Stack>
                    </div>
                ))}
            </div>
        </div>
    ),
};

/**
 * Align items variations
 */
export const AlignItems: Story = {
    render: () => (
        <div className="tw-space-y-6">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Align Items Options</h3>
            
            <div className="tw-space-y-4">
                {(["start", "end", "center", "baseline", "stretch"] as const).map((align) => (
                    <div key={align}>
                        <h4 className="tw-text-sm tw-font-medium tw-mb-2 tw-capitalize">{align}</h4>
                        <Stack 
                            direction="row" 
                            align={align}
                            spacing="md"
                            className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-w-full tw-h-24"
                        >
                            <div className="tw-bg-blue-100 tw-p-2 tw-rounded tw-h-8">Small</div>
                            <div className="tw-bg-green-100 tw-p-2 tw-rounded tw-h-12">Medium</div>
                            <div className="tw-bg-yellow-100 tw-p-2 tw-rounded tw-h-16">Large</div>
                        </Stack>
                    </div>
                ))}
            </div>
        </div>
    ),
};

/**
 * Spacing size variations
 */
export const SpacingSizes: Story = {
    render: () => (
        <div className="tw-space-y-6">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Spacing Sizes</h3>
            
            <div className="tw-space-y-4">
                {(["none", "xs", "sm", "md", "lg", "xl", "2xl"] as const).map((spacing) => (
                    <div key={spacing}>
                        <h4 className="tw-text-sm tw-font-medium tw-mb-2 tw-uppercase">{spacing}</h4>
                        <Stack 
                            direction="row" 
                            spacing={spacing}
                            className="tw-p-4 tw-border tw-border-gray-200 tw-rounded"
                        >
                            <div className="tw-bg-blue-100 tw-p-2 tw-rounded">Item</div>
                            <div className="tw-bg-green-100 tw-p-2 tw-rounded">Item</div>
                            <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">Item</div>
                        </Stack>
                    </div>
                ))}
            </div>
        </div>
    ),
};

/**
 * Wrap behavior demonstration
 */
export const WrapBehavior: Story = {
    render: () => (
        <div className="tw-space-y-6">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Wrap Behavior</h3>
            
            <div className="tw-space-y-4">
                <div>
                    <h4 className="tw-text-sm tw-font-medium tw-mb-2">No Wrap (default)</h4>
                    <Stack 
                        direction="row" 
                        wrap="nowrap"
                        spacing="md"
                        className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-w-64"
                    >
                        {Array.from({ length: 8 }, (_, i) => (
                            <div key={i} className="tw-bg-blue-100 tw-p-2 tw-rounded tw-whitespace-nowrap">
                                Item {i + 1}
                            </div>
                        ))}
                    </Stack>
                </div>
                
                <div>
                    <h4 className="tw-text-sm tw-font-medium tw-mb-2">Wrap</h4>
                    <Stack 
                        direction="row" 
                        wrap="wrap"
                        spacing="md"
                        className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-w-64"
                    >
                        {Array.from({ length: 8 }, (_, i) => (
                            <div key={i} className="tw-bg-green-100 tw-p-2 tw-rounded tw-whitespace-nowrap">
                                Item {i + 1}
                            </div>
                        ))}
                    </Stack>
                </div>
            </div>
        </div>
    ),
};

/**
 * Factory components showcase
 */
export const FactoryComponents: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Factory Components</h3>
            
            <div className="tw-space-y-6">
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">StackFactory.Vertical</h4>
                    <StackFactory.Vertical className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-w-48">
                        <div className="tw-bg-blue-100 tw-p-2 tw-rounded">Item 1</div>
                        <div className="tw-bg-green-100 tw-p-2 tw-rounded">Item 2</div>
                        <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">Item 3</div>
                    </StackFactory.Vertical>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">StackFactory.Horizontal</h4>
                    <StackFactory.Horizontal className="tw-p-4 tw-border tw-border-gray-200 tw-rounded">
                        <div className="tw-bg-blue-100 tw-p-2 tw-rounded">Item 1</div>
                        <div className="tw-bg-green-100 tw-p-2 tw-rounded">Item 2</div>
                        <div className="tw-bg-yellow-100 tw-p-2 tw-rounded">Item 3</div>
                    </StackFactory.Horizontal>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">StackFactory.Center</h4>
                    <StackFactory.Center className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-h-32">
                        <div className="tw-bg-purple-100 tw-p-4 tw-rounded">Centered Content</div>
                    </StackFactory.Center>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">StackFactory.Between</h4>
                    <StackFactory.Between direction="row" className="tw-p-4 tw-border tw-border-gray-200 tw-rounded">
                        <div className="tw-bg-blue-100 tw-p-2 tw-rounded">Left</div>
                        <div className="tw-bg-green-100 tw-p-2 tw-rounded">Right</div>
                    </StackFactory.Between>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">StackFactory.Navbar</h4>
                    <StackFactory.Navbar className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-bg-gray-50">
                        <div className="tw-font-bold tw-text-lg">Logo</div>
                        <div className="tw-space-x-4">
                            <span className="tw-text-sm">Home</span>
                            <span className="tw-text-sm">About</span>
                            <span className="tw-text-sm">Contact</span>
                        </div>
                    </StackFactory.Navbar>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">StackFactory.ButtonGroup</h4>
                    <StackFactory.ButtonGroup spacing="sm" className="tw-border tw-border-gray-300 tw-rounded tw-overflow-hidden">
                        <button className="tw-px-3 tw-py-2 tw-bg-gray-100 hover:tw-bg-gray-200">Edit</button>
                        <button className="tw-px-3 tw-py-2 tw-bg-gray-100 hover:tw-bg-gray-200">Share</button>
                        <button className="tw-px-3 tw-py-2 tw-bg-gray-100 hover:tw-bg-gray-200">Delete</button>
                    </StackFactory.ButtonGroup>
                </div>
                
                <div>
                    <h4 className="tw-text-md tw-font-medium tw-mb-2">StackFactory.Form</h4>
                    <StackFactory.Form className="tw-p-4 tw-border tw-border-gray-200 tw-rounded tw-max-w-md">
                        <div>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">Name</label>
                            <input type="text" className="tw-w-full tw-px-3 tw-py-2 tw-border tw-border-gray-300 tw-rounded" />
                        </div>
                        <div>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">Email</label>
                            <input type="email" className="tw-w-full tw-px-3 tw-py-2 tw-border tw-border-gray-300 tw-rounded" />
                        </div>
                        <div>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">Message</label>
                            <textarea className="tw-w-full tw-px-3 tw-py-2 tw-border tw-border-gray-300 tw-rounded tw-h-20"></textarea>
                        </div>
                        <button className="tw-w-full tw-px-4 tw-py-2 tw-bg-blue-600 tw-text-white tw-rounded hover:tw-bg-blue-700">
                            Submit
                        </button>
                    </StackFactory.Form>
                </div>
            </div>
        </div>
    ),
};

/**
 * Real-world layout examples
 */
export const RealWorldExamples: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <h3 className="tw-text-xl tw-font-bold tw-mb-6">Real-World Layout Examples</h3>
            
            {/* Header Navigation */}
            <div>
                <h4 className="tw-text-lg tw-font-medium tw-mb-4">Header Navigation</h4>
                <StackFactory.Navbar className="tw-p-4 tw-bg-white tw-border tw-border-gray-200 tw-rounded-lg tw-shadow-sm">
                    <Stack direction="row" align="center" spacing="md">
                        <div className="tw-w-8 tw-h-8 tw-bg-blue-600 tw-rounded"></div>
                        <span className="tw-font-bold tw-text-xl">MyApp</span>
                    </Stack>
                    <Stack direction="row" align="center" spacing="lg">
                        <nav>
                            <Stack direction="row" spacing="lg">
                                <a href="#" className="tw-text-gray-700 hover:tw-text-blue-600">Home</a>
                                <a href="#" className="tw-text-gray-700 hover:tw-text-blue-600">Products</a>
                                <a href="#" className="tw-text-gray-700 hover:tw-text-blue-600">About</a>
                            </Stack>
                        </nav>
                        <Stack direction="row" spacing="sm">
                            <button className="tw-px-3 tw-py-1 tw-text-sm tw-border tw-border-gray-300 tw-rounded">
                                Sign In
                            </button>
                            <button className="tw-px-3 tw-py-1 tw-text-sm tw-bg-blue-600 tw-text-white tw-rounded">
                                Sign Up
                            </button>
                        </Stack>
                    </Stack>
                </StackFactory.Navbar>
            </div>
            
            {/* Card Grid */}
            <div>
                <h4 className="tw-text-lg tw-font-medium tw-mb-4">Product Cards</h4>
                <Stack direction="row" wrap="wrap" spacing="lg" className="tw-p-4 tw-bg-gray-50 tw-rounded-lg">
                    {Array.from({ length: 3 }, (_, i) => (
                        <div key={i} className="tw-bg-white tw-p-4 tw-rounded-lg tw-shadow-sm tw-w-64">
                            <Stack direction="column" spacing="md">
                                <div className="tw-w-full tw-h-32 tw-bg-gray-200 tw-rounded"></div>
                                <Stack direction="column" spacing="sm">
                                    <h5 className="tw-font-semibold">Product {i + 1}</h5>
                                    <p className="tw-text-sm tw-text-gray-600">
                                        Description of the product goes here.
                                    </p>
                                    <Stack direction="row" justify="between" align="center">
                                        <span className="tw-font-bold tw-text-lg">$99.99</span>
                                        <button className="tw-px-3 tw-py-1 tw-bg-blue-600 tw-text-white tw-text-sm tw-rounded">
                                            Add to Cart
                                        </button>
                                    </Stack>
                                </Stack>
                            </Stack>
                        </div>
                    ))}
                </Stack>
            </div>
            
            {/* Sidebar Layout */}
            <div>
                <h4 className="tw-text-lg tw-font-medium tw-mb-4">Dashboard Layout</h4>
                <Stack direction="row" className="tw-h-96 tw-border tw-border-gray-200 tw-rounded-lg tw-overflow-hidden">
                    {/* Sidebar */}
                    <StackFactory.Sidebar className="tw-w-64 tw-bg-gray-800 tw-text-white tw-p-4">
                        <div className="tw-font-bold tw-text-lg tw-mb-4">Dashboard</div>
                        <Stack direction="column" spacing="xs">
                            <a href="#" className="tw-px-3 tw-py-2 tw-rounded tw-bg-gray-700">Overview</a>
                            <a href="#" className="tw-px-3 tw-py-2 tw-rounded hover:tw-bg-gray-700">Analytics</a>
                            <a href="#" className="tw-px-3 tw-py-2 tw-rounded hover:tw-bg-gray-700">Reports</a>
                            <a href="#" className="tw-px-3 tw-py-2 tw-rounded hover:tw-bg-gray-700">Settings</a>
                        </Stack>
                    </StackFactory.Sidebar>
                    
                    {/* Main Content */}
                    <Stack direction="column" fullWidth className="tw-p-6 tw-bg-white">
                        <h2 className="tw-text-2xl tw-font-bold tw-mb-4">Welcome back!</h2>
                        <div className="tw-grid tw-grid-cols-3 tw-gap-4 tw-mb-6">
                            <div className="tw-p-4 tw-bg-blue-50 tw-rounded-lg">
                                <div className="tw-text-2xl tw-font-bold tw-text-blue-600">1,234</div>
                                <div className="tw-text-sm tw-text-gray-600">Total Users</div>
                            </div>
                            <div className="tw-p-4 tw-bg-green-50 tw-rounded-lg">
                                <div className="tw-text-2xl tw-font-bold tw-text-green-600">$12,345</div>
                                <div className="tw-text-sm tw-text-gray-600">Revenue</div>
                            </div>
                            <div className="tw-p-4 tw-bg-purple-50 tw-rounded-lg">
                                <div className="tw-text-2xl tw-font-bold tw-text-purple-600">98.5%</div>
                                <div className="tw-text-sm tw-text-gray-600">Uptime</div>
                            </div>
                        </div>
                        <div className="tw-flex-1 tw-bg-gray-100 tw-rounded-lg tw-p-4">
                            <div className="tw-text-center tw-text-gray-500">Chart area</div>
                        </div>
                    </Stack>
                </Stack>
            </div>
            
            {/* Footer */}
            <div>
                <h4 className="tw-text-lg tw-font-medium tw-mb-4">Footer</h4>
                <Stack direction="column" spacing="lg" className="tw-p-6 tw-bg-gray-900 tw-text-white tw-rounded-lg">
                    <Stack direction="row" justify="between" wrap="wrap" spacing="lg">
                        <Stack direction="column" spacing="md">
                            <h5 className="tw-font-semibold">Company</h5>
                            <Stack direction="column" spacing="xs">
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">About</a>
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Careers</a>
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Contact</a>
                            </Stack>
                        </Stack>
                        <Stack direction="column" spacing="md">
                            <h5 className="tw-font-semibold">Product</h5>
                            <Stack direction="column" spacing="xs">
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Features</a>
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Pricing</a>
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">API</a>
                            </Stack>
                        </Stack>
                        <Stack direction="column" spacing="md">
                            <h5 className="tw-font-semibold">Support</h5>
                            <Stack direction="column" spacing="xs">
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Help Center</a>
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Documentation</a>
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Status</a>
                            </Stack>
                        </Stack>
                    </Stack>
                    <div className="tw-border-t tw-border-gray-700 tw-pt-4">
                        <Stack direction="row" justify="between" align="center">
                            <p className="tw-text-sm tw-text-gray-300">© 2024 MyApp. All rights reserved.</p>
                            <Stack direction="row" spacing="md">
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Privacy</a>
                                <a href="#" className="tw-text-gray-300 hover:tw-text-white tw-text-sm">Terms</a>
                            </Stack>
                        </Stack>
                    </div>
                </Stack>
            </div>
        </div>
    ),
};