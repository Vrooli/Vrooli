import type { Meta, StoryObj } from "@storybook/react";
import { FormStructureType, InputType, type FormSchema } from "@vrooli/shared";
import { Formik } from "formik";
import React from "react";
import { maxWidthDecorator, paddedDecorator } from "../../__test/helpers/storybookDecorators.js";
import { FormView } from "./FormView.js";

const meta: Meta<typeof FormView> = {
    title: "Forms/FormView",
    component: FormView,
    parameters: {
        layout: "fullscreen",
    },
    decorators: [maxWidthDecorator(800)],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Sample schemas for different scenarios
const emptySchema: FormSchema = {
    elements: [],
    containers: [],
};

const basicInputSchema: FormSchema = {
    elements: [
        {
            id: "text-input",
            type: InputType.Text,
            fieldName: "name",
            label: "Full Name",
            isRequired: true,
            props: {
                placeholder: "Enter your full name",
            },
        },
        {
            id: "email-input",
            type: InputType.Text,
            fieldName: "email",
            label: "Email Address",
            isRequired: true,
            props: {
                placeholder: "Enter your email",
                type: "email",
            },
        },
        {
            id: "age-input",
            type: InputType.IntegerInput,
            fieldName: "age",
            label: "Age",
            isRequired: false,
            props: {
                min: 0,
                max: 120,
            },
        },
    ],
    containers: [{
        totalItems: 3,
        spacing: 2,
        xs: 1,
        sm: 1,
        md: 1,
        lg: 1,
    }],
};

const complexFormSchema: FormSchema = {
    elements: [
        {
            id: "header-1",
            type: FormStructureType.Header,
            label: "Personal Information",
            tag: "h2",
        },
        {
            id: "first-name",
            type: InputType.Text,
            fieldName: "firstName",
            label: "First Name",
            isRequired: true,
            props: {
                placeholder: "Enter first name",
            },
        },
        {
            id: "last-name",
            type: InputType.Text,
            fieldName: "lastName",
            label: "Last Name",
            isRequired: true,
            props: {
                placeholder: "Enter last name",
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
            id: "newsletter",
            type: InputType.Switch,
            fieldName: "newsletter",
            label: "Subscribe to Newsletter",
            isRequired: false,
            props: {
                defaultValue: false,
            },
        },
        {
            id: "interests",
            type: InputType.Checkbox,
            fieldName: "interests",
            label: "Interests",
            isRequired: false,
            props: {
                options: [
                    { label: "Technology", value: "tech" },
                    { label: "Sports", value: "sports" },
                    { label: "Music", value: "music" },
                    { label: "Travel", value: "travel" },
                ],
            },
        },
        {
            id: "tip-1",
            type: FormStructureType.Tip,
            label: "Choose all interests that apply to you. This helps us personalize your experience.",
            icon: "Info",
        },
    ],
    containers: [
        {
            totalItems: 1,
            spacing: 2,
            xs: 1,
            sm: 1,
            md: 1,
            lg: 1,
        },
        {
            totalItems: 2,
            spacing: 2,
            xs: 1,
            sm: 2,
            md: 2,
            lg: 2,
            direction: "row",
        },
        {
            totalItems: 5,
            spacing: 2,
            xs: 1,
            sm: 1,
            md: 1,
            lg: 1,
        },
    ],
};

const allInputTypesSchema: FormSchema = {
    elements: [
        {
            id: "header-inputs",
            type: FormStructureType.Header,
            label: "All Input Types Demo",
            tag: "h2",
        },
        {
            id: "text-field",
            type: InputType.Text,
            fieldName: "textField",
            label: "Text Input",
            isRequired: false,
            props: {
                placeholder: "Enter text here",
            },
        },
        {
            id: "json-field",
            type: InputType.JSON,
            fieldName: "jsonField",
            label: "JSON Input",
            isRequired: false,
            props: {
                placeholder: "{\"key\": \"value\"}",
            },
        },
        {
            id: "number-field",
            type: InputType.IntegerInput,
            fieldName: "numberField",
            label: "Number Input",
            isRequired: false,
            props: {
                min: 0,
                max: 100,
            },
        },
        {
            id: "slider-field",
            type: InputType.Slider,
            fieldName: "sliderField",
            label: "Slider Input",
            isRequired: false,
            props: {
                min: 0,
                max: 100,
                step: 10,
                defaultValue: 50,
            },
        },
        {
            id: "switch-field",
            type: InputType.Switch,
            fieldName: "switchField",
            label: "Switch Input",
            isRequired: false,
            props: {
                defaultValue: false,
            },
        },
        {
            id: "radio-field",
            type: InputType.Radio,
            fieldName: "radioField",
            label: "Radio Input",
            isRequired: false,
            props: {
                options: [
                    { label: "Option 1", value: "opt1" },
                    { label: "Option 2", value: "opt2" },
                    { label: "Option 3", value: "opt3" },
                ],
            },
        },
        {
            id: "checkbox-field",
            type: InputType.Checkbox,
            fieldName: "checkboxField",
            label: "Checkbox Input",
            isRequired: false,
            props: {
                options: [
                    { label: "Choice A", value: "a" },
                    { label: "Choice B", value: "b" },
                    { label: "Choice C", value: "c" },
                ],
            },
        },
        {
            id: "selector-field",
            type: InputType.Selector,
            fieldName: "selectorField",
            label: "Selector Input",
            isRequired: false,
            props: {
                options: [
                    { label: "First Option", value: "first" },
                    { label: "Second Option", value: "second" },
                    { label: "Third Option", value: "third" },
                ],
            },
        },
    ],
    containers: [{
        totalItems: 9,
        spacing: 2,
        xs: 1,
        sm: 1,
        md: 2,
        lg: 2,
    }],
};

export const EmptyForm: Story = {
    render: () => (
        <Formik initialValues={{}} onSubmit={() => {}}>
            <FormView
                disabled={false}
                isEditing={false}
                schema={emptySchema}
                onSchemaChange={() => {}}
                data-testid="empty-form-view"
            />
        </Formik>
    ),
};

export const BasicForm: Story = {
    render: () => (
        <Formik initialValues={{ name: "", email: "", age: "" }} onSubmit={() => {}}>
            <FormView
                disabled={false}
                isEditing={false}
                schema={basicInputSchema}
                onSchemaChange={() => {}}
                data-testid="basic-form-view"
            />
        </Formik>
    ),
};

export const EditMode: Story = {
    render: () => (
        <FormView
            disabled={false}
            isEditing={true}
            schema={basicInputSchema}
            onSchemaChange={(newSchema) => {
                console.log("Schema changed:", newSchema);
            }}
            data-testid="edit-form-view"
        />
    ),
};

export const DisabledForm: Story = {
    render: () => (
        <Formik initialValues={{ name: "John Doe", email: "john@example.com", age: "30" }} onSubmit={() => {}}>
            <FormView
                disabled={true}
                isEditing={false}
                schema={basicInputSchema}
                onSchemaChange={() => {}}
                data-testid="disabled-form-view"
            />
        </Formik>
    ),
};

export const ComplexForm: Story = {
    render: () => (
        <Formik 
            initialValues={{ 
                firstName: "", 
                lastName: "", 
                newsletter: false, 
                interests: [], 
            }} 
            onSubmit={() => {}}
        >
            <FormView
                disabled={false}
                isEditing={false}
                schema={complexFormSchema}
                onSchemaChange={() => {}}
                data-testid="complex-form-view"
            />
        </Formik>
    ),
};

export const AllInputTypes: Story = {
    render: () => (
        <Formik 
            initialValues={{
                textField: "",
                jsonField: "",
                numberField: "",
                sliderField: 50,
                switchField: false,
                radioField: "",
                checkboxField: [],
                selectorField: "",
            }} 
            onSubmit={() => {}}
        >
            <FormView
                disabled={false}
                isEditing={false}
                schema={allInputTypesSchema}
                onSchemaChange={() => {}}
                data-testid="all-types-form-view"
            />
        </Formik>
    ),
};

export const WithFieldNamePrefix: Story = {
    render: () => (
        <Formik initialValues={{ "prefix-name": "", "prefix-email": "" }} onSubmit={() => {}}>
            <FormView
                disabled={false}
                fieldNamePrefix="prefix"
                isEditing={false}
                schema={basicInputSchema}
                onSchemaChange={() => {}}
                data-testid="prefixed-form-view"
            />
        </Formik>
    ),
};

export const Interactive: Story = {
    decorators: [paddedDecorator(2)],
    render: () => {
        const [schema, setSchema] = React.useState(basicInputSchema);
        const [isEditing, setIsEditing] = React.useState(false);

        return (
            <div>
                <div style={{ marginBottom: "16px", display: "flex", gap: "8px" }}>
                    <button 
                        onClick={() => setIsEditing(!isEditing)}
                        style={{
                            padding: "8px 16px",
                            backgroundColor: isEditing ? "#f44336" : "#2196f3",
                            color: "white",
                            border: "none",
                            borderRadius: "4px",
                            cursor: "pointer",
                        }}
                    >
                        {isEditing ? "Stop Editing" : "Start Editing"}
                    </button>
                    <button 
                        onClick={() => setSchema(schema === basicInputSchema ? complexFormSchema : basicInputSchema)}
                        style={{
                            padding: "8px 16px",
                            backgroundColor: "#4caf50",
                            color: "white",
                            border: "none",
                            borderRadius: "4px",
                            cursor: "pointer",
                        }}
                    >
                        Switch Schema
                    </button>
                </div>
                
                {isEditing ? (
                    <FormView
                        disabled={false}
                        isEditing={true}
                        schema={schema}
                        onSchemaChange={setSchema}
                        data-testid="interactive-form-view"
                    />
                ) : (
                    <Formik initialValues={{}} onSubmit={() => {}}>
                        <FormView
                            disabled={false}
                            isEditing={false}
                            schema={schema}
                            onSchemaChange={() => {}}
                            data-testid="interactive-form-view"
                        />
                    </Formik>
                )}
            </div>
        );
    },
};
