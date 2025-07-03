import type { Meta, StoryObj } from "@storybook/react";
import { Formik } from "formik";
import { FormInputColorPicker } from "./FormInputColorPicker.js";

const meta: Meta<typeof FormInputColorPicker> = {
    title: "Components/Inputs/Form/FormInputColorPicker",
    component: FormInputColorPicker,
    tags: ["autodocs"],
    argTypes: {
        fieldName: {
            control: { type: "text" },
            description: "The name of the field for Formik",
        },
        label: {
            control: { type: "text" },
            description: "Label text for the color picker",
        },
        defaultValue: {
            control: { type: "color" },
            description: "Default color value",
        },
        disabled: {
            control: { type: "boolean" },
            description: "Whether the input is disabled",
        },
    },
    parameters: {
        layout: "padded",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default color picker
 */
export const Default: Story = {
    render: (args) => (
        <Formik
            initialValues={{ color: "#ff0000" }}
            onSubmit={() => {}}
        >
            <FormInputColorPicker {...args} />
        </Formik>
    ),
    args: {
        fieldName: "color",
        label: "Choose Color",
        defaultValue: "#ff0000",
    },
};

/**
 * With different default colors
 */
export const DifferentColors: Story = {
    render: () => (
        <div className="tw-space-y-4">
            <Formik
                initialValues={{ 
                    red: "#ff0000",
                    green: "#00ff00", 
                    blue: "#0000ff",
                    purple: "#9333EA",
                    orange: "#f97316",
                }}
                onSubmit={() => {}}
            >
                <div className="tw-space-y-4">
                    <FormInputColorPicker 
                        fieldName="red" 
                        label="Red" 
                        defaultValue="#ff0000" 
                    />
                    <FormInputColorPicker 
                        fieldName="green" 
                        label="Green" 
                        defaultValue="#00ff00" 
                    />
                    <FormInputColorPicker 
                        fieldName="blue" 
                        label="Blue" 
                        defaultValue="#0000ff" 
                    />
                    <FormInputColorPicker 
                        fieldName="purple" 
                        label="Purple" 
                        defaultValue="#9333EA" 
                    />
                    <FormInputColorPicker 
                        fieldName="orange" 
                        label="Orange" 
                        defaultValue="#f97316" 
                    />
                </div>
            </Formik>
        </div>
    ),
};

/**
 * Disabled state
 */
export const Disabled: Story = {
    render: () => (
        <Formik
            initialValues={{ color: "#6b7280" }}
            onSubmit={() => {}}
        >
            <div className="tw-space-y-4">
                <FormInputColorPicker 
                    fieldName="color" 
                    label="Enabled Color Picker" 
                    defaultValue="#6b7280" 
                />
                <FormInputColorPicker 
                    fieldName="color" 
                    label="Disabled Color Picker" 
                    defaultValue="#6b7280" 
                    disabled 
                />
            </div>
        </Formik>
    ),
};

/**
 * Without label
 */
export const WithoutLabel: Story = {
    render: () => (
        <Formik
            initialValues={{ color: "#8b5cf6" }}
            onSubmit={() => {}}
        >
            <FormInputColorPicker 
                fieldName="color" 
                defaultValue="#8b5cf6" 
            />
        </Formik>
    ),
};

/**
 * Custom styling
 */
export const CustomStyling: Story = {
    render: () => (
        <Formik
            initialValues={{ 
                theme: "#3b82f6",
                accent: "#ef4444",
            }}
            onSubmit={() => {}}
        >
            <div className="tw-p-4 tw-bg-gray-100 tw-rounded-lg tw-space-y-4">
                <FormInputColorPicker 
                    fieldName="theme" 
                    label="Theme Color" 
                    defaultValue="#3b82f6"
                    className="tw-p-2 tw-bg-white tw-rounded"
                />
                <FormInputColorPicker 
                    fieldName="accent" 
                    label="Accent Color" 
                    defaultValue="#ef4444"
                    className="tw-p-2 tw-bg-white tw-rounded"
                />
            </div>
        </Formik>
    ),
};

/**
 * Interactive example with preview
 */
export const WithPreview: Story = {
    render: () => {
        const initialValues = { 
            backgroundColor: "#f3f4f6",
            textColor: "#111827",
            borderColor: "#d1d5db",
        };

        return (
            <Formik
                initialValues={initialValues}
                onSubmit={() => {}}
            >
                {({ values }) => (
                    <div className="tw-space-y-6">
                        {/* Color Controls */}
                        <div className="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4">
                            <FormInputColorPicker 
                                fieldName="backgroundColor" 
                                label="Background Color" 
                                defaultValue={initialValues.backgroundColor}
                            />
                            <FormInputColorPicker 
                                fieldName="textColor" 
                                label="Text Color" 
                                defaultValue={initialValues.textColor}
                            />
                            <FormInputColorPicker 
                                fieldName="borderColor" 
                                label="Border Color" 
                                defaultValue={initialValues.borderColor}
                            />
                        </div>
                        
                        {/* Preview */}
                        <div 
                            className="tw-p-6 tw-rounded-lg tw-border-2"
                            style={{
                                backgroundColor: values.backgroundColor,
                                color: values.textColor,
                                borderColor: values.borderColor,
                            }}
                        >
                            <h3 className="tw-text-lg tw-font-semibold tw-mb-2">
                                Color Preview
                            </h3>
                            <p>
                                This is a preview of your selected colors. The background, 
                                text, and border colors are all customizable using the 
                                color pickers above.
                            </p>
                            <div className="tw-mt-4 tw-flex tw-space-x-2">
                                <div 
                                    className="tw-w-8 tw-h-8 tw-rounded tw-border"
                                    style={{ 
                                        backgroundColor: values.backgroundColor,
                                        borderColor: values.borderColor,
                                    }}
                                    title="Background"
                                />
                                <div 
                                    className="tw-w-8 tw-h-8 tw-rounded tw-border"
                                    style={{ 
                                        backgroundColor: values.textColor,
                                        borderColor: values.borderColor,
                                    }}
                                    title="Text"
                                />
                                <div 
                                    className="tw-w-8 tw-h-8 tw-rounded"
                                    style={{ 
                                        backgroundColor: values.borderColor,
                                    }}
                                    title="Border"
                                />
                            </div>
                        </div>
                    </div>
                )}
            </Formik>
        );
    },
};
