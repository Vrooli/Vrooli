import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import MenuItem from "@mui/material/MenuItem";
import { Formik } from "formik";
import { IconCommon } from "../../../icons/Icons.js";
import { Switch } from "../Switch/Switch.js";
import { Selector, SelectorBase } from "./Selector.js";

const meta: Meta<typeof Selector> = {
    title: "Components/Inputs/Selector",
    component: Selector,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Sample data for the selector
const sampleOptions = [
    { id: "1", name: "Option 1", description: "First option description" },
    { id: "2", name: "Option 2", description: "Second option description" },
    { id: "3", name: "Option 3", description: "Third option description" },
    { id: "4", name: "Option 4", description: "Fourth option description" },
];

const simpleOptions = ["Apple", "Banana", "Cherry", "Date", "Elderberry"];

const numberOptions = [1, 2, 3, 4, 5];

export const Showcase: Story = {
    render: () => {
        const [variant, setVariant] = useState<"outline" | "filled" | "underline">("filled");
        const [size, setSize] = useState<"sm" | "md" | "lg">("md");
        const [disabled, setDisabled] = useState(false);
        const [fullWidth, setFullWidth] = useState(false);
        const [showNoneOption, setShowNoneOption] = useState(false);
        const [showAddOption, setShowAddOption] = useState(false);
        const [showIcons, setShowIcons] = useState(false);
        const [showDescriptions, setShowDescriptions] = useState(false);
        const [labelText, setLabelText] = useState("Select an option");
        const [noneText, setNoneText] = useState("None");

        const getOptionLabel = (option: any) => {
            if (typeof option === "string") return option;
            if (typeof option === "number") return option.toString();
            return option.name;
        };

        const getOptionDescription = showDescriptions ? (option: any) => {
            if (typeof option === "object" && option.description) return option.description;
            return null;
        } : undefined;

        const getOptionIcon = showIcons ? (option: any) => {
            const iconNames = ["Home", "Star", "Heart", "Settings", "User"];
            const index = typeof option === "string" ? simpleOptions.indexOf(option) :
                         typeof option === "number" ? option - 1 :
                         parseInt(option.id) - 1;
            const iconName = iconNames[index % iconNames.length];
            return <IconCommon name={iconName as any} />;
        } : undefined;

        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default", 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: "column",
                    maxWidth: 1200, 
                    mx: "auto", 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Selector Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Variant</FormLabel>
                                <TextField
                                    select
                                    value={variant}
                                    onChange={(e) => setVariant(e.target.value as any)}
                                    size="small"
                                    fullWidth
                                >
                                    <MenuItem value="outline">Outline</MenuItem>
                                    <MenuItem value="filled">Filled</MenuItem>
                                    <MenuItem value="underline">Underline</MenuItem>
                                </TextField>
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size</FormLabel>
                                <TextField
                                    select
                                    value={size}
                                    onChange={(e) => setSize(e.target.value as any)}
                                    size="small"
                                    fullWidth
                                >
                                    <MenuItem value="sm">Small</MenuItem>
                                    <MenuItem value="md">Medium</MenuItem>
                                    <MenuItem value="lg">Large</MenuItem>
                                </TextField>
                            </Box>

                            <Switch
                                checked={disabled}
                                onChange={(checked) => setDisabled(checked)}
                                size="sm"
                                label="Disabled"
                                labelPosition="right"
                            />

                            <Switch
                                checked={fullWidth}
                                onChange={(checked) => setFullWidth(checked)}
                                size="sm"
                                label="Full Width"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showNoneOption}
                                onChange={(checked) => setShowNoneOption(checked)}
                                size="sm"
                                label="Show None Option"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showAddOption}
                                onChange={(checked) => setShowAddOption(checked)}
                                size="sm"
                                label="Show Add Option"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showIcons}
                                onChange={(checked) => setShowIcons(checked)}
                                size="sm"
                                label="Show Icons"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showDescriptions}
                                onChange={(checked) => setShowDescriptions(checked)}
                                size="sm"
                                label="Show Descriptions"
                                labelPosition="right"
                            />

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label Text</FormLabel>
                                <TextField
                                    value={labelText}
                                    onChange={(e) => setLabelText(e.target.value)}
                                    size="small"
                                    fullWidth
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>None Text</FormLabel>
                                <TextField
                                    value={noneText}
                                    onChange={(e) => setNoneText(e.target.value)}
                                    size="small"
                                    fullWidth
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Examples Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Selector Examples</Typography>
                        
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                            {/* Object Options Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Object Options</Typography>
                                <Formik
                                    initialValues={{ objectOption: null }}
                                    onSubmit={() => {}}
                                >
                                    <Selector
                                        name="objectOption"
                                        label={labelText}
                                        options={sampleOptions}
                                        getOptionLabel={getOptionLabel}
                                        getOptionDescription={getOptionDescription}
                                        getOptionIcon={getOptionIcon}
                                        disabled={disabled}
                                        fullWidth={fullWidth}
                                        noneOption={showNoneOption}
                                        noneText={noneText}
                                        variant={variant}
                                        size={size}
                                        addOption={showAddOption ? {
                                            label: "Add new option",
                                            onSelect: () => console.log("Add option clicked"),
                                        } : undefined}
                                    />
                                </Formik>
                            </Box>

                            {/* String Options Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>String Options</Typography>
                                <Formik
                                    initialValues={{ stringOption: "Apple" }}
                                    onSubmit={() => {}}
                                >
                                    <Selector
                                        name="stringOption"
                                        label="Select a fruit"
                                        options={simpleOptions}
                                        getOptionLabel={getOptionLabel}
                                        getOptionIcon={getOptionIcon}
                                        disabled={disabled}
                                        fullWidth={fullWidth}
                                        noneOption={showNoneOption}
                                        noneText={noneText}
                                        variant={variant}
                                        size={size}
                                    />
                                </Formik>
                            </Box>

                            {/* Number Options Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Number Options</Typography>
                                <Formik
                                    initialValues={{ numberOption: 3 }}
                                    onSubmit={() => {}}
                                >
                                    <Selector
                                        name="numberOption"
                                        label="Select a number"
                                        options={numberOptions}
                                        getOptionLabel={getOptionLabel}
                                        disabled={disabled}
                                        fullWidth={fullWidth}
                                        noneOption={showNoneOption}
                                        noneText={noneText}
                                        variant={variant}
                                        size={size}
                                    />
                                </Formik>
                            </Box>

                            {/* SelectorBase Example (without Formik) */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>SelectorBase (without Formik)</Typography>
                                {(() => {
                                    const [value, setValue] = useState<string | null>("Banana");
                                    return (
                                        <SelectorBase
                                            name="baseExample"
                                            label="Base selector example"
                                            options={simpleOptions}
                                            getOptionLabel={getOptionLabel}
                                            getOptionIcon={getOptionIcon}
                                            value={value}
                                            onChange={(newValue) => setValue(newValue)}
                                            disabled={disabled}
                                            fullWidth={fullWidth}
                                            noneOption={showNoneOption}
                                            noneText={noneText}
                                            variant={variant}
                                            size={size}
                                        />
                                    );
                                })()}
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

export const BasicUsage: Story = {
    render: () => (
        <Box sx={{ p: 3 }}>
            <Formik
                initialValues={{ fruit: "Apple" }}
                onSubmit={(values) => console.log("Selected:", values)}
            >
                <Selector
                    name="fruit"
                    label="Select a fruit"
                    options={["Apple", "Banana", "Cherry", "Date", "Elderberry"]}
                    getOptionLabel={(option) => option}
                />
            </Formik>
        </Box>
    ),
};

export const WithIcons: Story = {
    render: () => (
        <Box sx={{ p: 3 }}>
            <Formik
                initialValues={{ category: null }}
                onSubmit={(values) => console.log("Selected:", values)}
            >
                <Selector
                    name="category"
                    label="Select a category"
                    options={[
                        { id: "home", name: "Home" },
                        { id: "work", name: "Work" },
                        { id: "personal", name: "Personal" },
                        { id: "shopping", name: "Shopping" },
                    ]}
                    getOptionLabel={(option) => option.name}
                    getOptionIcon={(option) => {
                        const iconMap: Record<string, any> = {
                            home: "Home",
                            work: "Briefcase",
                            personal: "User",
                            shopping: "ShoppingCart",
                        };
                        return <IconCommon name={iconMap[option.id]} />;
                    }}
                />
            </Formik>
        </Box>
    ),
};

export const WithDescriptions: Story = {
    render: () => (
        <Box sx={{ p: 3 }}>
            <Formik
                initialValues={{ plan: null }}
                onSubmit={(values) => console.log("Selected:", values)}
            >
                <Selector
                    name="plan"
                    label="Select a plan"
                    options={[
                        { id: "free", name: "Free", description: "Basic features for personal use" },
                        { id: "pro", name: "Pro", description: "Advanced features for professionals" },
                        { id: "enterprise", name: "Enterprise", description: "Full features for teams" },
                    ]}
                    getOptionLabel={(option) => option.name}
                    getOptionDescription={(option) => option.description}
                />
            </Formik>
        </Box>
    ),
};

export const WithAddOption: Story = {
    render: () => (
        <Box sx={{ p: 3 }}>
            <Formik
                initialValues={{ tag: null }}
                onSubmit={(values) => console.log("Selected:", values)}
            >
                <Selector
                    name="tag"
                    label="Select a tag"
                    options={["JavaScript", "TypeScript", "React", "Node.js"]}
                    getOptionLabel={(option) => option}
                    addOption={{
                        label: "Create new tag",
                        onSelect: () => console.log("Add new tag clicked"),
                    }}
                />
            </Formik>
        </Box>
    ),
};

export const FullWidth: Story = {
    render: () => (
        <Box sx={{ p: 3 }}>
            <Formik
                initialValues={{ option: null }}
                onSubmit={(values) => console.log("Selected:", values)}
            >
                <Selector
                    name="option"
                    label="Full width selector"
                    options={["Option 1", "Option 2", "Option 3"]}
                    getOptionLabel={(option) => option}
                    fullWidth
                />
            </Formik>
        </Box>
    ),
};

export const Disabled: Story = {
    render: () => (
        <Box sx={{ p: 3 }}>
            <Formik
                initialValues={{ option: "Option 2" }}
                onSubmit={(values) => console.log("Selected:", values)}
            >
                <Selector
                    name="option"
                    label="Disabled selector"
                    options={["Option 1", "Option 2", "Option 3"]}
                    getOptionLabel={(option) => option}
                    disabled
                />
            </Formik>
        </Box>
    ),
};

export const WithNoneOption: Story = {
    render: () => (
        <Box sx={{ p: 3 }}>
            <Formik
                initialValues={{ option: null }}
                onSubmit={(values) => console.log("Selected:", values)}
            >
                <Selector
                    name="option"
                    label="Optional selector"
                    options={["Option 1", "Option 2", "Option 3"]}
                    getOptionLabel={(option) => option}
                    noneOption
                    noneText="Select later"
                />
            </Formik>
        </Box>
    ),
};

export const Variants: Story = {
    render: () => (
        <Box sx={{ p: 3, display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h6">Selector Variants</Typography>
            
            <Formik initialValues={{ filled: "Apple", outline: "Banana", underline: "Cherry" }} onSubmit={() => {}}>
                <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                    <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Filled Variant (Default)</Typography>
                        <Selector
                            name="filled"
                            label="Filled selector"
                            options={["Apple", "Banana", "Cherry", "Date"]}
                            getOptionLabel={(option) => option}
                            variant="filled"
                        />
                    </Box>
                    
                    <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Outline Variant</Typography>
                        <Selector
                            name="outline"
                            label="Outline selector"
                            options={["Apple", "Banana", "Cherry", "Date"]}
                            getOptionLabel={(option) => option}
                            variant="outline"
                        />
                    </Box>
                    
                    <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Underline Variant</Typography>
                        <Selector
                            name="underline"
                            label="Underline selector"
                            options={["Apple", "Banana", "Cherry", "Date"]}
                            getOptionLabel={(option) => option}
                            variant="underline"
                        />
                    </Box>
                </Box>
            </Formik>
        </Box>
    ),
};

export const Sizes: Story = {
    render: () => (
        <Box sx={{ p: 3, display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h6">Selector Sizes</Typography>
            
            <Formik initialValues={{ small: "Apple", medium: "Banana", large: "Cherry" }} onSubmit={() => {}}>
                <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                    <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Small Size</Typography>
                        <Selector
                            name="small"
                            label="Small selector"
                            options={["Apple", "Banana", "Cherry", "Date"]}
                            getOptionLabel={(option) => option}
                            size="sm"
                        />
                    </Box>
                    
                    <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Medium Size (Default)</Typography>
                        <Selector
                            name="medium"
                            label="Medium selector"
                            options={["Apple", "Banana", "Cherry", "Date"]}
                            getOptionLabel={(option) => option}
                            size="md"
                        />
                    </Box>
                    
                    <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Large Size</Typography>
                        <Selector
                            name="large"
                            label="Large selector"
                            options={["Apple", "Banana", "Cherry", "Date"]}
                            getOptionLabel={(option) => option}
                            size="lg"
                        />
                    </Box>
                </Box>
            </Formik>
        </Box>
    ),
};
