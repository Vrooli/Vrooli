import type { Meta, StoryObj } from "@storybook/react";
import { FormStructureType, InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../__test/helpers/storybookDecorators.js";
import { Box } from "../../components/layout/Box.js";
import { FormRunView } from "./FormRunView.js";

const meta: Meta<typeof FormRunView> = {
    title: "Forms/FormView/FormRunView",
    component: FormRunView,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Example form schemas for showcasing different form configurations
const basicFormSchema: FormSchema = {
    elements: [
        {
            id: "header1",
            type: FormStructureType.Header,
            tag: "h2",
            label: "User Information",
        },
        {
            id: "name",
            type: InputType.Text,
            fieldName: "name",
            label: "Full Name",
            isRequired: true,
            helpText: "Enter your full name",
        },
        {
            id: "email",
            type: InputType.Text,
            fieldName: "email",
            label: "Email Address",
            isRequired: true,
            props: {
                type: "email",
            },
        },
        {
            id: "divider1",
            type: FormStructureType.Divider,
            label: "",
        },
        {
            id: "preferences",
            type: FormStructureType.Header,
            tag: "h3",
            label: "Preferences",
        },
        {
            id: "newsletter",
            type: InputType.Switch,
            fieldName: "newsletter",
            label: "Subscribe to newsletter",
            props: {
                defaultValue: false,
            },
        },
    ],
    containers: [
        {
            totalItems: 3,
            direction: "column",
            xs: 1,
        },
        {
            totalItems: 1,
            direction: "column",
            xs: 1,
        },
        {
            totalItems: 2,
            direction: "column",
            xs: 1,
        },
    ],
};

const complexFormSchema: FormSchema = {
    elements: [
        {
            id: "tip1",
            type: FormStructureType.Tip,
            label: "Please fill out all required fields before submitting",
        },
        {
            id: "header1",
            type: FormStructureType.Header,
            tag: "h1",
            label: "Advanced Form Example",
        },
        {
            id: "image1",
            type: FormStructureType.Image,
            label: "Company Logo",
            url: "https://via.placeholder.com/300x200",
        },
        {
            id: "firstName",
            type: InputType.Text,
            fieldName: "firstName",
            label: "First Name",
            isRequired: true,
        },
        {
            id: "lastName",
            type: InputType.Text,
            fieldName: "lastName",
            label: "Last Name",
            isRequired: true,
        },
        {
            id: "age",
            type: InputType.IntegerInput,
            fieldName: "age",
            label: "Age",
            props: {
                min: 0,
                max: 120,
            },
        },
        {
            id: "divider2",
            type: FormStructureType.Divider,
            label: "",
        },
        {
            id: "interests",
            type: InputType.Checkbox,
            fieldName: "interests",
            label: "Interests",
            props: {
                options: [
                    { label: "Sports", value: "sports" },
                    { label: "Music", value: "music" },
                    { label: "Reading", value: "reading" },
                    { label: "Travel", value: "travel" },
                ],
            },
        },
        {
            id: "experience",
            type: InputType.Slider,
            fieldName: "experience",
            label: "Years of Experience",
            props: {
                min: 0,
                max: 50,
                defaultValue: 5,
            },
        },
        {
            id: "qrCode1",
            type: FormStructureType.QrCode,
            label: "Scan for More Info",
            data: "https://vrooli.com",
        },
    ],
    containers: [
        {
            totalItems: 3,
            direction: "column",
            xs: 1,
        },
        {
            totalItems: 3,
            direction: "row",
            xs: 1,
            sm: 2,
            md: 3,
        },
        {
            totalItems: 1,
            direction: "column",
            xs: 1,
        },
        {
            totalItems: 3,
            direction: "column",
            xs: 1,
            title: "Additional Information",
            description: "Optional fields for more details",
            disableCollapse: false,
        },
    ],
};

const emptyFormSchema: FormSchema = {
    elements: [],
    containers: [],
};

export const Showcase: Story = {
    render: () => {
        // Define the form schema for controls
        const controlsSchema: FormSchema = {
            elements: [
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
                    id: "formType",
                    type: InputType.Radio,
                    fieldName: "formType",
                    label: "Form Type",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Basic Form", value: "basic" },
                            { label: "Complex Form", value: "complex" },
                            { label: "Empty Form", value: "empty" },
                        ],
                        defaultValue: "basic",
                        row: true,
                    },
                },
            ],
            containers: [
                {
                    totalItems: 2,
                    spacing: 3,
                    xs: 1,
                    sm: 2,
                },
            ],
        };

        // Initial values for the form
        const initialValues = {
            disabled: false,
            formType: "basic",
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "FormRunView",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { disabled, formType } = values;

                let schema: FormSchema;
                switch (formType) {
                    case "complex":
                        schema = complexFormSchema;
                        break;
                    case "empty":
                        schema = emptyFormSchema;
                        break;
                    default:
                        schema = basicFormSchema;
                }

                return (
                    <Box className="tw-max-w-4xl tw-mx-auto tw-p-6">
                        <FormRunView
                            disabled={disabled}
                            fieldNamePrefix="showcase"
                            schema={schema}
                        />
                    </Box>
                );
            },
        };

        // Use the showcase decorator
        const ShowcaseComponent = showcaseDecorator(showcaseConfig);
        return <ShowcaseComponent />;
    },
};

export const BasicForm: Story = {
    args: {
        disabled: false,
        fieldNamePrefix: "basic",
        schema: basicFormSchema,
    },
};

export const ComplexForm: Story = {
    args: {
        disabled: false,
        fieldNamePrefix: "complex",
        schema: complexFormSchema,
    },
};

export const DisabledForm: Story = {
    args: {
        disabled: true,
        fieldNamePrefix: "disabled",
        schema: basicFormSchema,
    },
};

export const EmptyForm: Story = {
    args: {
        disabled: false,
        fieldNamePrefix: "empty",
        schema: emptyFormSchema,
    },
};

export const FormWithSections: Story = {
    args: {
        disabled: false,
        fieldNamePrefix: "sections",
        schema: {
            elements: [
                {
                    id: "section1-field1",
                    type: InputType.Text,
                    fieldName: "field1",
                    label: "Field 1",
                },
                {
                    id: "section1-field2",
                    type: InputType.Text,
                    fieldName: "field2",
                    label: "Field 2",
                },
                {
                    id: "section2-field3",
                    type: InputType.Radio,
                    fieldName: "field3",
                    label: "Choose Option",
                    props: {
                        options: [
                            { label: "Option A", value: "a" },
                            { label: "Option B", value: "b" },
                        ],
                    },
                },
                {
                    id: "section3-field4",
                    type: InputType.Checkbox,
                    fieldName: "field4",
                    label: "Select Multiple",
                    props: {
                        options: [
                            { label: "Item 1", value: "item1" },
                            { label: "Item 2", value: "item2" },
                            { label: "Item 3", value: "item3" },
                        ],
                    },
                },
            ],
            containers: [
                {
                    totalItems: 2,
                    title: "Section 1",
                    description: "Basic information",
                    direction: "column",
                    xs: 1,
                },
                {
                    totalItems: 1,
                    title: "Section 2",
                    description: "Preferences",
                    direction: "column",
                    xs: 1,
                    disableCollapse: true,
                },
                {
                    totalItems: 1,
                    title: "Section 3",
                    direction: "column",
                    xs: 1,
                },
            ],
        },
    },
};

