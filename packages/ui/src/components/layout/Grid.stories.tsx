import type { Meta, StoryObj } from "@storybook/react";
import { Box } from "./Box.js";
import { Grid, GridFactory } from "./Grid.js";

const meta: Meta<typeof Grid> = {
    title: "Components/Layout/Grid",
    component: Grid,
    tags: ["autodocs"],
    argTypes: {
        component: {
            control: { type: "select" },
            options: ["div", "section", "article", "main", "aside"],
            description: "HTML element type to render",
        },
        columns: {
            control: { type: "select" },
            options: ["none", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
            description: "Number of columns in the grid",
        },
        rows: {
            control: { type: "select" },
            options: ["none", 1, 2, 3, 4, 5, 6],
            description: "Number of rows in the grid",
        },
        gap: {
            control: { type: "select" },
            options: ["none", "xs", "sm", "md", "lg", "xl"],
            description: "Gap between grid items",
        },
        columnGap: {
            control: { type: "select" },
            options: ["none", "xs", "sm", "md", "lg", "xl"],
            description: "Column gap between grid items",
        },
        rowGap: {
            control: { type: "select" },
            options: ["none", "xs", "sm", "md", "lg", "xl"],
            description: "Row gap between grid items",
        },
        alignItems: {
            control: { type: "select" },
            options: ["start", "center", "end", "stretch"],
            description: "Align items along the cross axis",
        },
        justifyContent: {
            control: { type: "select" },
            options: ["start", "center", "end", "between", "around", "evenly"],
            description: "Justify content along the main axis",
        },
        flow: {
            control: { type: "select" },
            options: ["row", "col", "row-dense", "col-dense"],
            description: "Grid auto flow direction",
        },
        fullWidth: {
            control: { type: "boolean" },
            description: "Whether the grid should take full width",
        },
        fullHeight: {
            control: { type: "boolean" },
            description: "Whether the grid should take full height",
        },
    },
    parameters: {
        layout: "padded",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Basic 3-column grid
 */
export const Default: Story = {
    args: {
        columns: 3,
        gap: "md",
    },
    render: (args) => (
        <Grid {...args}>
            <Box variant="paper" padding="md" borderRadius="md">Item 1</Box>
            <Box variant="paper" padding="md" borderRadius="md">Item 2</Box>
            <Box variant="paper" padding="md" borderRadius="md">Item 3</Box>
            <Box variant="paper" padding="md" borderRadius="md">Item 4</Box>
            <Box variant="paper" padding="md" borderRadius="md">Item 5</Box>
            <Box variant="paper" padding="md" borderRadius="md">Item 6</Box>
        </Grid>
    ),
};

/**
 * Grid with different column counts
 */
export const ColumnVariations: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">2 Columns</h3>
                <Grid columns={2} gap="md">
                    <Box variant="outlined" padding="md" borderRadius="md">Item 1</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">Item 2</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">Item 3</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">Item 4</Box>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">4 Columns</h3>
                <Grid columns={4} gap="md">
                    <Box variant="outlined" padding="md" borderRadius="md">1</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">2</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">3</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">4</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">5</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">6</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">7</Box>
                    <Box variant="outlined" padding="md" borderRadius="md">8</Box>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">6 Columns</h3>
                <Grid columns={6} gap="md">
                    {Array.from({ length: 12 }, (_, i) => (
                        <Box key={i} variant="outlined" padding="md" borderRadius="md">{i + 1}</Box>
                    ))}
                </Grid>
            </div>
        </div>
    ),
};

/**
 * Gap size variations
 */
export const GapSizes: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">No Gap</h3>
                <Grid columns={3} gap="none">
                    <Box variant="elevated" padding="md">Item 1</Box>
                    <Box variant="elevated" padding="md">Item 2</Box>
                    <Box variant="elevated" padding="md">Item 3</Box>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Small Gap</h3>
                <Grid columns={3} gap="sm">
                    <Box variant="elevated" padding="md" borderRadius="md">Item 1</Box>
                    <Box variant="elevated" padding="md" borderRadius="md">Item 2</Box>
                    <Box variant="elevated" padding="md" borderRadius="md">Item 3</Box>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Large Gap</h3>
                <Grid columns={3} gap="lg">
                    <Box variant="elevated" padding="md" borderRadius="md">Item 1</Box>
                    <Box variant="elevated" padding="md" borderRadius="md">Item 2</Box>
                    <Box variant="elevated" padding="md" borderRadius="md">Item 3</Box>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Different Column and Row Gaps</h3>
                <Grid columns={3} columnGap="xl" rowGap="sm">
                    {Array.from({ length: 6 }, (_, i) => (
                        <Box key={i} variant="elevated" padding="md" borderRadius="md">Item {i + 1}</Box>
                    ))}
                </Grid>
            </div>
        </div>
    ),
};

/**
 * Grid with item components
 */
export const WithGridItems: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Column Spanning</h3>
                <Grid columns={4} gap="md">
                    <Grid item colSpan={2}>
                        <Box variant="paper" padding="md" borderRadius="md" className="tw-h-full">
                            Spans 2 columns
                        </Box>
                    </Grid>
                    <Grid item>
                        <Box variant="paper" padding="md" borderRadius="md" className="tw-h-full">
                            Normal
                        </Box>
                    </Grid>
                    <Grid item>
                        <Box variant="paper" padding="md" borderRadius="md" className="tw-h-full">
                            Normal
                        </Box>
                    </Grid>
                    <Grid item colSpan={4}>
                        <Box variant="elevated" padding="md" borderRadius="md">
                            Full width (spans 4 columns)
                        </Box>
                    </Grid>
                    <Grid item colSpan={3}>
                        <Box variant="paper" padding="md" borderRadius="md">
                            Spans 3 columns
                        </Box>
                    </Grid>
                    <Grid item>
                        <Box variant="paper" padding="md" borderRadius="md">
                            Normal
                        </Box>
                    </Grid>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Row and Column Spanning</h3>
                <Grid columns={3} rows={3} gap="md">
                    <Grid item colSpan={2} rowSpan={2}>
                        <Box variant="elevated" padding="lg" borderRadius="md" className="tw-h-full tw-flex tw-items-center tw-justify-center">
                            2x2 Grid Area
                        </Box>
                    </Grid>
                    <Grid item>
                        <Box variant="paper" padding="md" borderRadius="md">Item 1</Box>
                    </Grid>
                    <Grid item>
                        <Box variant="paper" padding="md" borderRadius="md">Item 2</Box>
                    </Grid>
                    <Grid item colSpan={3}>
                        <Box variant="outlined" padding="md" borderRadius="md">
                            Full width bottom row
                        </Box>
                    </Grid>
                </Grid>
            </div>
        </div>
    ),
};

/**
 * Grid positioning examples
 */
export const Positioning: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Column Positioning</h3>
                <Grid columns={6} gap="md" className="tw-bg-gray-100 tw-p-4 tw-rounded-lg">
                    <Grid item colStart={2} colEnd={4}>
                        <Box variant="elevated" padding="md" borderRadius="md">
                            Columns 2-3
                        </Box>
                    </Grid>
                    <Grid item colStart={5} colSpan={2}>
                        <Box variant="elevated" padding="md" borderRadius="md">
                            Starting at 5
                        </Box>
                    </Grid>
                    <Grid item colStart={1} colEnd={7}>
                        <Box variant="paper" padding="md" borderRadius="md">
                            Full width (1-6)
                        </Box>
                    </Grid>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Grid Area Positioning</h3>
                <Grid columns={4} rows={4} gap="md" className="tw-bg-gray-100 tw-p-4 tw-rounded-lg tw-h-96">
                    <Grid item colStart={1} colEnd={3} rowStart={1} rowEnd={3}>
                        <Box variant="elevated" padding="md" borderRadius="md" className="tw-h-full tw-flex tw-items-center tw-justify-center">
                            Main Area
                        </Box>
                    </Grid>
                    <Grid item colStart={3} colEnd={5} rowStart={1} rowEnd={2}>
                        <Box variant="paper" padding="md" borderRadius="md" className="tw-h-full tw-flex tw-items-center tw-justify-center">
                            Top Right
                        </Box>
                    </Grid>
                    <Grid item colStart={3} colEnd={5} rowStart={2} rowEnd={4}>
                        <Box variant="outlined" padding="md" borderRadius="md" className="tw-h-full tw-flex tw-items-center tw-justify-center">
                            Sidebar
                        </Box>
                    </Grid>
                    <Grid item colStart={1} colEnd={3} rowStart={3} rowEnd={5}>
                        <Box variant="subtle" padding="md" borderRadius="md" className="tw-h-full tw-flex tw-items-center tw-justify-center">
                            Bottom Left
                        </Box>
                    </Grid>
                    <Grid item colStart={3} colEnd={5} rowStart={4} rowEnd={5}>
                        <Box variant="paper" padding="md" borderRadius="md" className="tw-h-full tw-flex tw-items-center tw-justify-center">
                            Bottom Right
                        </Box>
                    </Grid>
                </Grid>
            </div>
        </div>
    ),
};

/**
 * Alignment options
 */
export const Alignment: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Align Items</h3>
                <div className="tw-grid tw-grid-cols-2 tw-gap-4">
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Start</p>
                        <Grid columns={3} gap="md" alignItems="start" className="tw-h-32 tw-bg-gray-100 tw-p-2 tw-rounded">
                            <Box variant="paper" padding="sm" borderRadius="sm">1</Box>
                            <Box variant="paper" padding="md" borderRadius="sm">2</Box>
                            <Box variant="paper" padding="sm" borderRadius="sm">3</Box>
                        </Grid>
                    </div>
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Center</p>
                        <Grid columns={3} gap="md" alignItems="center" className="tw-h-32 tw-bg-gray-100 tw-p-2 tw-rounded">
                            <Box variant="paper" padding="sm" borderRadius="sm">1</Box>
                            <Box variant="paper" padding="md" borderRadius="sm">2</Box>
                            <Box variant="paper" padding="sm" borderRadius="sm">3</Box>
                        </Grid>
                    </div>
                </div>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Justify Content</h3>
                <div className="tw-space-y-4">
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Space Between</p>
                        <Grid columns={3} gap="md" justifyContent="between" className="tw-bg-gray-100 tw-p-2 tw-rounded">
                            <Box variant="paper" padding="md" borderRadius="sm" className="tw-w-20">1</Box>
                            <Box variant="paper" padding="md" borderRadius="sm" className="tw-w-20">2</Box>
                            <Box variant="paper" padding="md" borderRadius="sm" className="tw-w-20">3</Box>
                        </Grid>
                    </div>
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Space Evenly</p>
                        <Grid columns={3} gap="md" justifyContent="evenly" className="tw-bg-gray-100 tw-p-2 tw-rounded">
                            <Box variant="paper" padding="md" borderRadius="sm" className="tw-w-20">1</Box>
                            <Box variant="paper" padding="md" borderRadius="sm" className="tw-w-20">2</Box>
                            <Box variant="paper" padding="md" borderRadius="sm" className="tw-w-20">3</Box>
                        </Grid>
                    </div>
                </div>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Item Self Alignment</h3>
                <Grid columns={3} gap="md" className="tw-h-32 tw-bg-gray-100 tw-p-2 tw-rounded">
                    <Grid item alignSelf="start">
                        <Box variant="paper" padding="md" borderRadius="sm">Start</Box>
                    </Grid>
                    <Grid item alignSelf="center">
                        <Box variant="paper" padding="md" borderRadius="sm">Center</Box>
                    </Grid>
                    <Grid item alignSelf="end">
                        <Box variant="paper" padding="md" borderRadius="sm">End</Box>
                    </Grid>
                </Grid>
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
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Pre-configured Grids</h3>
                
                <div className="tw-space-y-6">
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Two Column Grid</p>
                        <GridFactory.TwoColumn>
                            <Box variant="outlined" padding="md" borderRadius="md">Column 1</Box>
                            <Box variant="outlined" padding="md" borderRadius="md">Column 2</Box>
                        </GridFactory.TwoColumn>
                    </div>
                    
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Three Column Grid</p>
                        <GridFactory.ThreeColumn>
                            <Box variant="outlined" padding="md" borderRadius="md">1</Box>
                            <Box variant="outlined" padding="md" borderRadius="md">2</Box>
                            <Box variant="outlined" padding="md" borderRadius="md">3</Box>
                        </GridFactory.ThreeColumn>
                    </div>
                    
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Centered Grid</p>
                        <GridFactory.Centered columns={3} gap="md" className="tw-h-32 tw-bg-gray-100 tw-p-2 tw-rounded">
                            <Box variant="paper" padding="md" borderRadius="md">Centered</Box>
                            <Box variant="paper" padding="md" borderRadius="md">Items</Box>
                        </GridFactory.Centered>
                    </div>
                    
                    <div>
                        <p className="tw-text-sm tw-font-medium tw-mb-2">Responsive Grid (resize window to see)</p>
                        <GridFactory.Responsive gap="md">
                            {Array.from({ length: 8 }, (_, i) => (
                                <Box key={i} variant="elevated" padding="md" borderRadius="md">
                                    Item {i + 1}
                                </Box>
                            ))}
                        </GridFactory.Responsive>
                    </div>
                </div>
            </div>
            
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Pre-configured Grid Items</h3>
                <Grid columns={12} gap="md">
                    <GridFactory.ItemHalf>
                        <Box variant="paper" padding="md" borderRadius="md">Half (6/12)</Box>
                    </GridFactory.ItemHalf>
                    <GridFactory.ItemQuarter>
                        <Box variant="paper" padding="md" borderRadius="md">Quarter (3/12)</Box>
                    </GridFactory.ItemQuarter>
                    <GridFactory.ItemQuarter>
                        <Box variant="paper" padding="md" borderRadius="md">Quarter (3/12)</Box>
                    </GridFactory.ItemQuarter>
                    <GridFactory.ItemThird>
                        <Box variant="outlined" padding="md" borderRadius="md">Third (4/12)</Box>
                    </GridFactory.ItemThird>
                    <GridFactory.ItemTwoThirds>
                        <Box variant="outlined" padding="md" borderRadius="md">Two Thirds (8/12)</Box>
                    </GridFactory.ItemTwoThirds>
                    <GridFactory.ItemFull>
                        <Box variant="elevated" padding="md" borderRadius="md">Full Width</Box>
                    </GridFactory.ItemFull>
                </Grid>
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
            <div>
                <h3 className="tw-text-xl tw-font-bold tw-mb-6">Dashboard Layout</h3>
                <Grid columns={12} gap="lg">
                    {/* Header */}
                    <Grid item colSpan={12}>
                        <Box variant="paper" padding="lg" borderRadius="md">
                            <h2 className="tw-text-lg tw-font-semibold">Dashboard Header</h2>
                        </Box>
                    </Grid>
                    
                    {/* Stats Cards */}
                    <Grid item colSpan={3}>
                        <Box variant="elevated" padding="md" borderRadius="md">
                            <h3 className="tw-text-sm tw-font-medium tw-text-gray-600">Total Users</h3>
                            <p className="tw-text-2xl tw-font-bold">1,234</p>
                        </Box>
                    </Grid>
                    <Grid item colSpan={3}>
                        <Box variant="elevated" padding="md" borderRadius="md">
                            <h3 className="tw-text-sm tw-font-medium tw-text-gray-600">Revenue</h3>
                            <p className="tw-text-2xl tw-font-bold">$45,678</p>
                        </Box>
                    </Grid>
                    <Grid item colSpan={3}>
                        <Box variant="elevated" padding="md" borderRadius="md">
                            <h3 className="tw-text-sm tw-font-medium tw-text-gray-600">Orders</h3>
                            <p className="tw-text-2xl tw-font-bold">567</p>
                        </Box>
                    </Grid>
                    <Grid item colSpan={3}>
                        <Box variant="elevated" padding="md" borderRadius="md">
                            <h3 className="tw-text-sm tw-font-medium tw-text-gray-600">Growth</h3>
                            <p className="tw-text-2xl tw-font-bold">+12%</p>
                        </Box>
                    </Grid>
                    
                    {/* Main Content Area */}
                    <GridFactory.ItemMainContent>
                        <Box variant="paper" padding="lg" borderRadius="md" className="tw-h-64">
                            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Main Content Area</h3>
                            <p className="tw-text-gray-600">Charts, tables, or other main content goes here</p>
                        </Box>
                    </GridFactory.ItemMainContent>
                    
                    {/* Sidebar */}
                    <GridFactory.ItemSidebar>
                        <Box variant="outlined" padding="lg" borderRadius="md" className="tw-h-64">
                            <h3 className="tw-text-lg tw-font-semibold tw-mb-4">Sidebar</h3>
                            <div className="tw-space-y-2">
                                <Box variant="subtle" padding="sm" borderRadius="sm">Menu Item 1</Box>
                                <Box variant="subtle" padding="sm" borderRadius="sm">Menu Item 2</Box>
                                <Box variant="subtle" padding="sm" borderRadius="sm">Menu Item 3</Box>
                            </div>
                        </Box>
                    </GridFactory.ItemSidebar>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-xl tw-font-bold tw-mb-6">Blog Layout</h3>
                <Grid columns={12} gap="lg">
                    {/* Featured Post */}
                    <Grid item colSpan={12}>
                        <Box variant="elevated" padding="xl" borderRadius="lg">
                            <h2 className="tw-text-2xl tw-font-bold tw-mb-2">Featured Post Title</h2>
                            <p className="tw-text-gray-600 tw-mb-4">Published on January 1, 2024</p>
                            <p>This is the featured post content preview...</p>
                        </Box>
                    </Grid>
                    
                    {/* Post Grid */}
                    <Grid item colSpan={4}>
                        <Box variant="paper" padding="lg" borderRadius="md">
                            <h3 className="tw-font-semibold tw-mb-2">Post Title 1</h3>
                            <p className="tw-text-sm tw-text-gray-600">Post preview content...</p>
                        </Box>
                    </Grid>
                    <Grid item colSpan={4}>
                        <Box variant="paper" padding="lg" borderRadius="md">
                            <h3 className="tw-font-semibold tw-mb-2">Post Title 2</h3>
                            <p className="tw-text-sm tw-text-gray-600">Post preview content...</p>
                        </Box>
                    </Grid>
                    <Grid item colSpan={4}>
                        <Box variant="paper" padding="lg" borderRadius="md">
                            <h3 className="tw-font-semibold tw-mb-2">Post Title 3</h3>
                            <p className="tw-text-sm tw-text-gray-600">Post preview content...</p>
                        </Box>
                    </Grid>
                </Grid>
            </div>
            
            <div>
                <h3 className="tw-text-xl tw-font-bold tw-mb-6">Form Layout</h3>
                <Box variant="outlined" padding="xl" borderRadius="lg">
                    <Grid columns={12} gap="md">
                        <Grid item colSpan={6}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">First Name</label>
                            <input 
                                type="text" 
                                className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md" 
                                placeholder="John"
                            />
                        </Grid>
                        <Grid item colSpan={6}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">Last Name</label>
                            <input 
                                type="text" 
                                className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md" 
                                placeholder="Doe"
                            />
                        </Grid>
                        <Grid item colSpan={12}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">Email</label>
                            <input 
                                type="email" 
                                className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md" 
                                placeholder="john.doe@example.com"
                            />
                        </Grid>
                        <Grid item colSpan={8}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">Street Address</label>
                            <input 
                                type="text" 
                                className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md" 
                                placeholder="123 Main St"
                            />
                        </Grid>
                        <Grid item colSpan={4}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">Apt/Suite</label>
                            <input 
                                type="text" 
                                className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md" 
                                placeholder="Apt 4B"
                            />
                        </Grid>
                        <Grid item colSpan={5}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">City</label>
                            <input 
                                type="text" 
                                className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md" 
                                placeholder="New York"
                            />
                        </Grid>
                        <Grid item colSpan={3}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">State</label>
                            <select className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md">
                                <option>NY</option>
                                <option>CA</option>
                                <option>TX</option>
                            </select>
                        </Grid>
                        <Grid item colSpan={4}>
                            <label className="tw-block tw-text-sm tw-font-medium tw-mb-1">ZIP Code</label>
                            <input 
                                type="text" 
                                className="tw-w-full tw-px-3 tw-py-2 tw-border tw-rounded-md" 
                                placeholder="10001"
                            />
                        </Grid>
                        <Grid item colSpan={12} className="tw-mt-4">
                            <button className="tw-px-6 tw-py-2 tw-bg-blue-600 tw-text-white tw-rounded-md hover:tw-bg-blue-700">
                                Submit
                            </button>
                        </Grid>
                    </Grid>
                </Box>
            </div>
        </div>
    ),
};

/**
 * Responsive behavior
 */
export const ResponsiveBehavior: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <h3 className="tw-text-xl tw-font-bold tw-mb-6">Responsive Grid (resize window to see changes)</h3>
            
            <div>
                <p className="tw-text-sm tw-text-gray-600 tw-mb-4">
                    This grid changes from 1 column on mobile, to 2 on tablet, to 4 on desktop
                </p>
                <Grid 
                    columns={1} 
                    gap="md"
                    className="sm:tw-grid-cols-2 lg:tw-grid-cols-4"
                >
                    {Array.from({ length: 8 }, (_, i) => (
                        <Box key={i} variant="elevated" padding="lg" borderRadius="md">
                            <h3 className="tw-font-semibold">Card {i + 1}</h3>
                            <p className="tw-text-sm tw-text-gray-600 tw-mt-2">
                                Responsive card content
                            </p>
                        </Box>
                    ))}
                </Grid>
            </div>
            
            <div>
                <p className="tw-text-sm tw-text-gray-600 tw-mb-4">
                    Complex responsive layout with sidebar
                </p>
                <Grid columns={12} gap="lg">
                    <Grid item 
                        colSpan={12} 
                        className="lg:tw-col-span-3"
                    >
                        <Box variant="outlined" padding="lg" borderRadius="md" className="tw-h-full">
                            <h3 className="tw-font-semibold">Sidebar</h3>
                            <p className="tw-text-sm tw-text-gray-600 tw-mt-2">
                                Stacks on mobile, sidebar on desktop
                            </p>
                        </Box>
                    </Grid>
                    <Grid item 
                        colSpan={12} 
                        className="lg:tw-col-span-9"
                    >
                        <Grid columns={1} gap="md" className="sm:tw-grid-cols-2 xl:tw-grid-cols-3">
                            {Array.from({ length: 6 }, (_, i) => (
                                <Box key={i} variant="paper" padding="md" borderRadius="md">
                                    Content Item {i + 1}
                                </Box>
                            ))}
                        </Grid>
                    </Grid>
                </Grid>
            </div>
        </div>
    ),
};
