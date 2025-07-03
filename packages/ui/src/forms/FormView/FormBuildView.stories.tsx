import Typography from "@mui/material/Typography";
import type { Meta, StoryObj } from "@storybook/react";
import { FormStructureType, InputType, type FormSchema } from "@vrooli/shared";
import React from "react";
import { showcaseDecorator, type ShowcaseDecoratorConfig } from "../../__test/helpers/storybookDecorators.js";
import { FormBuildView } from "./FormBuildView.js";

const meta: Meta<typeof FormBuildView> = {
    title: "Forms/FormView/FormBuildView",
    component: FormBuildView,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Sample form schemas for different use cases
const emptySchema: FormSchema = {
    elements: [],
    containers: [],
};

const simpleSchema: FormSchema = {
    elements: [
        {
            id: "name-field",
            type: InputType.Text,
            fieldName: "name",
            label: "Name",
            isRequired: true,
            props: {
                placeholder: "Enter your name",
            },
        },
        {
            id: "email-field", 
            type: InputType.Text,
            fieldName: "email",
            label: "Email",
            isRequired: true,
            props: {
                placeholder: "Enter your email",
            },
        },
    ],
    containers: [{
        totalItems: 2,
        spacing: 2,
        xs: 1,
        sm: 1,
        md: 2,
        lg: 2,
    }],
};

const complexSchema: FormSchema = {
    elements: [
        {
            id: "header-1",
            type: FormStructureType.Header,
            label: "Personal Information",
            tag: "h2",
        },
        {
            id: "name-field",
            type: InputType.Text,
            fieldName: "name", 
            label: "Full Name",
            isRequired: true,
            props: {
                placeholder: "Enter your full name",
            },
        },
        {
            id: "age-field",
            type: InputType.IntegerInput,
            fieldName: "age",
            label: "Age",
            isRequired: false,
            props: {
                min: 0,
                max: 120,
            },
        },
        {
            id: "divider-1",
            type: FormStructureType.Divider,
            label: "",
        },
        {
            id: "header-2", 
            type: FormStructureType.Header,
            label: "Preferences",
            tag: "h3",
        },
        {
            id: "newsletter-field",
            type: InputType.Switch,
            fieldName: "newsletter",
            label: "Subscribe to newsletter",
            isRequired: false,
            props: {
                defaultValue: false,
            },
        },
        {
            id: "interests-field",
            type: InputType.Checkbox,
            fieldName: "interests",
            label: "Interests",
            isRequired: false,
            props: {
                options: [
                    { label: "Technology", value: "tech" },
                    { label: "Sports", value: "sports" },
                    { label: "Music", value: "music" },
                ],
            },
        },
    ],
    containers: [{
        totalItems: 7,
        spacing: 2,
        xs: 1,
        sm: 1,
        md: 1,
        lg: 1,
    }],
};

export const Showcase: Story = {
    render: () => {
        // Define the form schema for controls
        const controlsSchema: FormSchema = {
            elements: [
                {
                    id: "schema-type",
                    type: InputType.Radio,
                    fieldName: "schemaType",
                    label: "Form Schema",
                    isRequired: false,
                    props: {
                        options: [
                            { label: "Empty Form", value: "empty" },
                            { label: "Simple Form", value: "simple" },
                            { label: "Complex Form", value: "complex" },
                        ],
                        defaultValue: "empty",
                        row: true,
                    },
                },
                {
                    id: "field-prefix",
                    type: InputType.Text,
                    fieldName: "fieldNamePrefix",
                    label: "Field Name Prefix",
                    isRequired: false,
                    props: {
                        placeholder: "Optional prefix for field names",
                        defaultValue: "",
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
            schemaType: "empty",
            fieldNamePrefix: "",
        };

        // Showcase decorator configuration
        const showcaseConfig: ShowcaseDecoratorConfig = {
            componentName: "FormBuildView",
            controlsSchema,
            initialValues,
            renderShowcase: (values) => {
                const { schemaType, fieldNamePrefix } = values;

                let selectedSchema: FormSchema;
                switch (schemaType) {
                    case "simple":
                        selectedSchema = simpleSchema;
                        break;
                    case "complex":
                        selectedSchema = complexSchema;
                        break;
                    default:
                        selectedSchema = emptySchema;
                }

                return (
                    <div style={{ minHeight: "600px" }}>
                        <Typography variant="h6" sx={{ mb: 2 }}>
                            Form Builder
                        </Typography>
                        <Typography variant="body2" sx={{ mb: 3, color: "text.secondary" }}>
                            {schemaType === "empty" && "Start with an empty form. Use the toolbar to add elements."}
                            {schemaType === "simple" && "A simple form with text inputs. Click elements to edit them."}
                            {schemaType === "complex" && "A complex form with headers, dividers, and various input types."}
                        </Typography>
                        
                        <FormBuildView
                            fieldNamePrefix={fieldNamePrefix || undefined}
                            onSchemaChange={(newSchema) => {
                                console.log("Schema changed:", newSchema);
                            }}
                            schema={selectedSchema}
                        />
                    </div>
                );
            },
        };

        // Use the showcase decorator
        const ShowcaseComponent = showcaseDecorator(showcaseConfig);
        return <ShowcaseComponent />;
    },
};

export const EmptyForm: Story = {
    args: {
        schema: emptySchema,
        onSchemaChange: (schema) => console.log("Schema changed:", schema),
    },
    decorators: [
        (Story) => (
            <div style={{ padding: "20px", minHeight: "400px" }}>
                <Typography variant="h6" sx={{ mb: 2 }}>Empty Form Builder</Typography>
                <Story />
            </div>
        ),
    ],
};

export const SimpleForm: Story = {
    args: {
        schema: simpleSchema,
        onSchemaChange: (schema) => console.log("Schema changed:", schema),
    },
    decorators: [
        (Story) => (
            <div style={{ padding: "20px", minHeight: "500px" }}>
                <Typography variant="h6" sx={{ mb: 2 }}>Simple Form Builder</Typography>
                <Story />
            </div>
        ),
    ],
};

export const ComplexForm: Story = {
    args: {
        schema: complexSchema,
        onSchemaChange: (schema) => console.log("Schema changed:", schema),
        fieldNamePrefix: "example",
    },
    decorators: [
        (Story) => (
            <div style={{ padding: "20px", minHeight: "700px" }}>
                <Typography variant="h6" sx={{ mb: 2 }}>Complex Form Builder</Typography>
                <Story />
            </div>
        ),
    ],
};

export const WithLimits: Story = {
    args: {
        schema: emptySchema,
        onSchemaChange: (schema) => console.log("Schema changed:", schema),
        limits: {
            inputs: [InputType.Text, InputType.Switch],
            structures: [FormStructureType.Header, FormStructureType.Divider],
        },
    },
    decorators: [
        (Story) => (
            <div style={{ padding: "20px", minHeight: "400px" }}>
                <Typography variant="h6" sx={{ mb: 2 }}>Form Builder with Limited Options</Typography>
                <Typography variant="body2" sx={{ mb: 2, color: "text.secondary" }}>
                    Only Text inputs, Switches, Headers, and Dividers are available.
                </Typography>
                <Story />
            </div>
        ),
    ],
};
