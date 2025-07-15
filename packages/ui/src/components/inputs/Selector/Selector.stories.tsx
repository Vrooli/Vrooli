import type { Meta, StoryObj } from "@storybook/react";
import { InputType, type FormSchema } from "@vrooli/shared";
import { Formik } from "formik";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../../__test/helpers/storybookDecorators.js";
import { Box } from "../../layout/Box.js";
import { Typography } from "../../text/Typography.js";
import { IconCommon } from "../../../icons/Icons.js";
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

const categoryOptions = [
    { id: "home", name: "Home" },
    { id: "work", name: "Work" },
    { id: "personal", name: "Personal" },
    { id: "shopping", name: "Shopping" },
];

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
                            { label: "Filled", value: "filled" },
                            { label: "Outline", value: "outline" },
                            { label: "Underline", value: "underline" },
                        ],
                        defaultValue: "filled",
                    },
                },
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
                    id: "fullWidth",
                    type: InputType.Switch,
                    fieldName: "fullWidth",
                    label: "Full Width",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "showIcons",
                    type: InputType.Switch,
                    fieldName: "showIcons",
                    label: "Show Icons",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "showDescriptions",
                    type: InputType.Switch,
                    fieldName: "showDescriptions",
                    label: "Show Descriptions",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "showNoneOption",
                    type: InputType.Switch,
                    fieldName: "showNoneOption",
                    label: "Show None Option",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
                {
                    id: "showAddOption",
                    type: InputType.Switch,
                    fieldName: "showAddOption",
                    label: "Show Add Option",
                    isRequired: false,
                    props: {
                        defaultValue: false,
                    },
                },
            ],
            containers: [{
                totalItems: 8,
            }],
        };

        // Initial values for the form
        const initialValues = {
            variant: "filled",
            size: "md",
            disabled: false,
            fullWidth: false,
            showIcons: false,
            showDescriptions: false,
            showNoneOption: false,
            showAddOption: false,
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "Selector",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { variant, size, disabled, fullWidth, showIcons, showDescriptions, showNoneOption, showAddOption } = values;

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

                const getCategoryIcon = showIcons ? (option: any) => {
                    const iconMap: Record<string, any> = {
                        home: "Home",
                        work: "Briefcase",
                        personal: "User",
                        shopping: "ShoppingCart",
                    };
                    return <IconCommon name={iconMap[option.id]} />;
                } : undefined;

                return (
                    <Box className="tw-grid tw-grid-cols-1 md:tw-grid-cols-2 tw-gap-6">
                        {/* Object Options with Descriptions */}
                        <Box>
                            <Typography variant="h6" className="tw-mb-3">
                                Object Options (with descriptions)
                            </Typography>
                            <Formik
                                initialValues={{ objectOption: null }}
                                onSubmit={() => {}}
                            >
                                <Selector
                                    name="objectOption"
                                    label="Select an option"
                                    placeholder="Choose an option..."
                                    options={sampleOptions}
                                    getOptionLabel={getOptionLabel}
                                    getOptionDescription={getOptionDescription}
                                    getOptionIcon={getOptionIcon}
                                    disabled={disabled}
                                    fullWidth={fullWidth}
                                    noneOption={showNoneOption}
                                    noneText="None"
                                    variant={variant as any}
                                    size={size as any}
                                    addOption={showAddOption ? {
                                        label: "Add new option",
                                        onSelect: () => console.log("Add option clicked"),
                                    } : undefined}
                                />
                            </Formik>
                        </Box>

                        {/* String Options */}
                        <Box>
                            <Typography variant="h6" className="tw-mb-3">
                                String Options
                            </Typography>
                            <Formik
                                initialValues={{ stringOption: "Apple" }}
                                onSubmit={() => {}}
                            >
                                <Selector
                                    name="stringOption"
                                    label="Select a fruit"
                                    placeholder="Pick your favorite fruit..."
                                    options={simpleOptions}
                                    getOptionLabel={getOptionLabel}
                                    getOptionIcon={getOptionIcon}
                                    disabled={disabled}
                                    fullWidth={fullWidth}
                                    noneOption={showNoneOption}
                                    noneText="None"
                                    variant={variant as any}
                                    size={size as any}
                                />
                            </Formik>
                        </Box>

                        {/* Category Options with Icons */}
                        <Box>
                            <Typography variant="h6" className="tw-mb-3">
                                Category Options (with icons)
                            </Typography>
                            <Formik
                                initialValues={{ categoryOption: null }}
                                onSubmit={() => {}}
                            >
                                <Selector
                                    name="categoryOption"
                                    label="Select a category"
                                    placeholder="Pick a category..."
                                    options={categoryOptions}
                                    getOptionLabel={getOptionLabel}
                                    getOptionIcon={getCategoryIcon}
                                    disabled={disabled}
                                    fullWidth={fullWidth}
                                    noneOption={showNoneOption}
                                    noneText="None"
                                    variant={variant as any}
                                    size={size as any}
                                />
                            </Formik>
                        </Box>

                        {/* SelectorBase (without Formik) */}
                        <Box>
                            <Typography variant="h6" className="tw-mb-3">
                                SelectorBase (without Formik)
                            </Typography>
                            {(() => {
                                const [value, setValue] = React.useState<string | null>("Banana");
                                return (
                                    <SelectorBase
                                        name="baseExample"
                                        label="Base selector example"
                                        placeholder="Select a fruit..."
                                        options={simpleOptions}
                                        getOptionLabel={getOptionLabel}
                                        getOptionIcon={getOptionIcon}
                                        value={value}
                                        onChange={(newValue) => setValue(newValue)}
                                        disabled={disabled}
                                        fullWidth={fullWidth}
                                        noneOption={showNoneOption}
                                        noneText="None"
                                        variant={variant as any}
                                        size={size as any}
                                    />
                                );
                            })()}
                        </Box>

                        {/* Placeholder Examples */}
                        <Box className="md:tw-col-span-2">
                            <Typography variant="h6" className="tw-mb-3">
                                Placeholder Examples
                            </Typography>
                            <Box className="tw-grid tw-grid-cols-1 md:tw-grid-cols-4 tw-gap-4">
                                <Box>
                                    <Typography variant="subtitle2" className="tw-mb-2">
                                        Custom Placeholder
                                    </Typography>
                                    <Formik initialValues={{ customPlaceholder: null }} onSubmit={() => {}}>
                                        <Selector
                                            name="customPlaceholder"
                                            label="Fruit selector"
                                            placeholder="Choose your favorite fruit..."
                                            options={simpleOptions}
                                            getOptionLabel={getOptionLabel}
                                            disabled={disabled}
                                            variant={variant as any}
                                            size={size as any}
                                        />
                                    </Formik>
                                </Box>
                                
                                <Box>
                                    <Typography variant="subtitle2" className="tw-mb-2">
                                        Default Placeholder
                                    </Typography>
                                    <Formik initialValues={{ defaultPlaceholder: null }} onSubmit={() => {}}>
                                        <Selector
                                            name="defaultPlaceholder"
                                            label="Fruit selector"
                                            options={simpleOptions}
                                            getOptionLabel={getOptionLabel}
                                            disabled={disabled}
                                            variant={variant as any}
                                            size={size as any}
                                        />
                                    </Formik>
                                </Box>
                                
                                <Box>
                                    <Typography variant="subtitle2" className="tw-mb-2">
                                        Empty Placeholder
                                    </Typography>
                                    <Formik initialValues={{ emptyPlaceholder: null }} onSubmit={() => {}}>
                                        <Selector
                                            name="emptyPlaceholder"
                                            label="Fruit selector"
                                            placeholder=""
                                            options={simpleOptions}
                                            getOptionLabel={getOptionLabel}
                                            disabled={disabled}
                                            variant={variant as any}
                                            size={size as any}
                                        />
                                    </Formik>
                                </Box>
                                
                                <Box>
                                    <Typography variant="subtitle2" className="tw-mb-2">
                                        Has Selection
                                    </Typography>
                                    <Formik initialValues={{ hasSelection: "Apple" }} onSubmit={() => {}}>
                                        <Selector
                                            name="hasSelection"
                                            label="Fruit selector"
                                            placeholder="Choose your favorite fruit..."
                                            options={simpleOptions}
                                            getOptionLabel={getOptionLabel}
                                            disabled={disabled}
                                            variant={variant as any}
                                            size={size as any}
                                        />
                                    </Formik>
                                </Box>
                            </Box>
                        </Box>
                    </Box>
                );
            },
        };

        return React.createElement(showcaseDecorator(showcaseConfig));
    },
};