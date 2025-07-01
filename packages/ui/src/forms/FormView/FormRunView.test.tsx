import { act, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { FormStructureType, InputType, type FormSchema } from "@vrooli/shared";
import { Formik } from "formik";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../__test/testUtils.js";
import { FormRunView } from "./FormRunView.js";

// Mock dependencies
vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string, options?: object) => 
            options && typeof options === "object" && "count" in options 
                ? `${key}_${(options as any).count}` 
                : key,
    }),
}));

// Mock ContentCollapse to simplify testing
vi.mock("../../components/containers/ContentCollapse", () => ({
    ContentCollapse: ({ children, title, helpText }: any) => (
        <div data-testid="content-collapse">
            {title && <h3>{title}</h3>}
            {helpText && <p>{helpText}</p>}
            {children}
        </div>
    ),
}));

// Mock form components to avoid theme dependencies
vi.mock("../../components/inputs/form/FormHeader", () => ({
    FormHeader: ({ element }: any) => (
        <h2 data-testid="form-header">{element.label}</h2>
    ),
}));

vi.mock("../../components/inputs/form/FormDivider", () => ({
    FormDivider: () => (
        <div data-testid="form-divider" role="separator" aria-orientation="horizontal" />
    ),
}));

vi.mock("../../components/inputs/form/FormTip", () => ({
    FormTip: ({ element }: any) => (
        <div data-testid="form-tip" role="note">{element.label}</div>
    ),
}));

vi.mock("../../components/inputs/form/FormImage", () => ({
    FormImage: ({ element }: any) => (
        <div data-testid="form-image" role="img" aria-label={element.alt || element.label} />
    ),
}));

vi.mock("../../components/inputs/form/FormVideo", () => ({
    FormVideo: ({ element }: any) => (
        <div data-testid="form-video" role="application" aria-label={element.label} />
    ),
}));

vi.mock("../../components/inputs/form/FormQrCode", () => ({
    FormQrCode: ({ element }: any) => (
        <div data-testid="form-qr-code" role="img" aria-label={element.label} />
    ),
}));

vi.mock("../../components/inputs/form/FormInput", () => ({
    FormInput: ({ fieldData, disabled }: any) => {
        const inputType = fieldData.type;
        const baseProps = {
            id: fieldData.fieldName,
            name: fieldData.fieldName,
            "aria-label": fieldData.label,
            disabled,
        };

        switch (inputType) {
            case InputType.Switch:
                return (
                    <div>
                        <label htmlFor={fieldData.fieldName}>{fieldData.label}</label>
                        <input type="checkbox" role="switch" {...baseProps} />
                    </div>
                );
            case InputType.Radio:
                return (
                    <div>
                        <fieldset>
                            <legend>{fieldData.label}</legend>
                            {fieldData.props?.options?.map((opt: any) => (
                                <div key={opt.value}>
                                    <input 
                                        type="radio" 
                                        id={`${fieldData.fieldName}-${opt.value}`}
                                        name={fieldData.fieldName}
                                        value={opt.value}
                                        disabled={disabled}
                                    />
                                    <label htmlFor={`${fieldData.fieldName}-${opt.value}`}>{opt.label}</label>
                                </div>
                            ))}
                        </fieldset>
                    </div>
                );
            case InputType.Checkbox:
                return (
                    <div>
                        <fieldset>
                            <legend>{fieldData.label}</legend>
                            {fieldData.props?.options?.map((opt: any) => (
                                <div key={opt.value}>
                                    <input 
                                        type="checkbox" 
                                        id={`${fieldData.fieldName}-${opt.value}`}
                                        name={`${fieldData.fieldName}[]`}
                                        value={opt.value}
                                        disabled={disabled}
                                    />
                                    <label htmlFor={`${fieldData.fieldName}-${opt.value}`}>{opt.label}</label>
                                </div>
                            ))}
                        </fieldset>
                    </div>
                );
            default:
                return (
                    <div>
                        <label htmlFor={fieldData.fieldName}>{fieldData.label}</label>
                        <input type="text" {...baseProps} />
                        {fieldData.helpText && <span>{fieldData.helpText}</span>}
                    </div>
                );
        }
    },
}));

describe("FormRunView", () => {
    const basicSchema: FormSchema = {
        elements: [
            {
                id: "header1",
                type: FormStructureType.Header,
                tag: "h2",
                label: "Test Header",
            },
            {
                id: "field1",
                type: InputType.Text,
                fieldName: "textField",
                label: "Text Input",
                isRequired: true,
                helpText: "Enter some text",
            },
            {
                id: "divider1",
                type: FormStructureType.Divider,
                label: "",
            },
            {
                id: "tip1",
                type: FormStructureType.Tip,
                label: "This is a helpful tip",
                icon: "Info",
            },
        ],
        containers: [],
    };

    const emptySchema: FormSchema = {
        elements: [],
        containers: [],
    };

    describe("Rendering", () => {
        it("renders empty form message when no elements", () => {
            renderWithProviders(<FormRunView disabled={false} schema={emptySchema} />);
            
            expect(screen.getByText("The form is empty.")).toBeDefined();
        });

        it("renders form elements correctly", () => {
            renderWithProviders(
                <Formik initialValues={{ textField: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={false} schema={basicSchema} />
                </Formik>,
            );

            // Check header
            expect(screen.getByText("Test Header")).toBeDefined();
            
            // Check text input
            expect(screen.getByLabelText("Text Input")).toBeDefined();
            expect(screen.getByText("Enter some text")).toBeDefined();
            
            // Check divider exists
            expect(screen.getByTestId("form-divider")).toBeDefined();
            
            // Check tip exists
            expect(screen.getByTestId("form-tip")).toBeDefined();
            expect(screen.getByText("This is a helpful tip")).toBeDefined();
        });

        it("renders with fieldNamePrefix", () => {
            renderWithProviders(
                <Formik initialValues={{ "prefix-textField": "" }} onSubmit={vi.fn()}>
                    <FormRunView 
                        disabled={false} 
                        fieldNamePrefix="prefix" 
                        schema={basicSchema} 
                    />
                </Formik>,
            );

            const container = screen.getByTestId("prefix-form-run-view");
            expect(container).toBeDefined();
        });

        it("renders disabled inputs when disabled prop is true", () => {
            renderWithProviders(
                <Formik initialValues={{ textField: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={true} schema={basicSchema} />
                </Formik>,
            );

            const textInput = screen.getByLabelText("Text Input");
            expect(textInput.hasAttribute("disabled")).toBe(true);
        });
    });

    describe("Complex layouts", () => {
        it("renders elements in sections with containers", () => {
            const schemaWithContainers: FormSchema = {
                elements: [
                    {
                        id: "field1",
                        type: InputType.Text,
                        fieldName: "field1",
                        label: "Field 1",
                    },
                    {
                        id: "field2",
                        type: InputType.Text,
                        fieldName: "field2",
                        label: "Field 2",
                    },
                    {
                        id: "field3",
                        type: InputType.Text,
                        fieldName: "field3",
                        label: "Field 3",
                    },
                ],
                containers: [
                    {
                        totalItems: 2,
                        direction: "row",
                        xs: 1,
                        sm: 2,
                    },
                    {
                        totalItems: 1,
                        direction: "column",
                        xs: 1,
                    },
                ],
            };

            renderWithProviders(
                <Formik initialValues={{ field1: "", field2: "", field3: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={false} schema={schemaWithContainers} />
                </Formik>,
            );

            // Check that all fields are rendered
            expect(screen.getByLabelText("Field 1")).toBeDefined();
            expect(screen.getByLabelText("Field 2")).toBeDefined();
            expect(screen.getByLabelText("Field 3")).toBeDefined();
        });

        it("renders collapsible sections with titles", () => {
            const schemaWithTitledContainers: FormSchema = {
                elements: [
                    {
                        id: "field1",
                        type: InputType.Text,
                        fieldName: "field1",
                        label: "Field 1",
                    },
                    {
                        id: "field2",
                        type: InputType.Text,
                        fieldName: "field2",
                        label: "Field 2",
                    },
                ],
                containers: [
                    {
                        totalItems: 1,
                        title: "Section 1",
                        description: "First section description",
                        direction: "column",
                        xs: 1,
                    },
                    {
                        totalItems: 1,
                        title: "Section 2",
                        direction: "column",
                        xs: 1,
                        disableCollapse: true,
                    },
                ],
            };

            renderWithProviders(
                <Formik initialValues={{ field1: "", field2: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={false} schema={schemaWithTitledContainers} />
                </Formik>,
            );

            // Check section titles
            expect(screen.getByText("Section 1")).toBeDefined();
            expect(screen.getByText("Section 2")).toBeDefined();
            
            // Check description is rendered
            expect(screen.getByText("First section description")).toBeDefined();
        });
    });

    describe("Different element types", () => {
        it("renders all structural element types", () => {
            const structuralSchema: FormSchema = {
                elements: [
                    {
                        id: "image1",
                        type: FormStructureType.Image,
                        label: "Test Image",
                        url: "https://example.com/image.jpg",
                        alt: "Test alt text",
                    },
                    {
                        id: "video1",
                        type: FormStructureType.Video,
                        label: "Test Video",
                        url: "https://example.com/video.mp4",
                    },
                    {
                        id: "qr1",
                        type: FormStructureType.QrCode,
                        label: "Test QR",
                        url: "https://example.com",
                    },
                ],
                containers: [],
            };

            renderWithProviders(<FormRunView disabled={false} schema={structuralSchema} />);

            // Check image
            expect(screen.getByTestId("form-image")).toBeDefined();
            
            // Check video
            expect(screen.getByTestId("form-video")).toBeDefined();
            
            // Check QR code
            expect(screen.getByTestId("form-qr-code")).toBeDefined();
        });

        it("renders various input types", () => {
            const inputSchema: FormSchema = {
                elements: [
                    {
                        id: "switch1",
                        type: InputType.Switch,
                        fieldName: "switchField",
                        label: "Toggle Switch",
                    },
                    {
                        id: "radio1",
                        type: InputType.Radio,
                        fieldName: "radioField",
                        label: "Radio Options",
                        props: {
                            options: [
                                { label: "Option 1", value: "opt1" },
                                { label: "Option 2", value: "opt2" },
                            ],
                        },
                    },
                    {
                        id: "checkbox1",
                        type: InputType.Checkbox,
                        fieldName: "checkboxField",
                        label: "Checkbox Options",
                        props: {
                            options: [
                                { label: "Check 1", value: "check1" },
                                { label: "Check 2", value: "check2" },
                            ],
                        },
                    },
                ],
                containers: [],
            };

            renderWithProviders(
                <Formik 
                    initialValues={{ 
                        switchField: false, 
                        radioField: "", 
                        checkboxField: [], 
                    }} 
                    onSubmit={vi.fn()}
                >
                    <FormRunView disabled={false} schema={inputSchema} />
                </Formik>,
            );

            // Check switch
            expect(screen.getByLabelText("Toggle Switch")).toBeDefined();
            
            // Check radio options
            expect(screen.getByLabelText("Option 1")).toBeDefined();
            expect(screen.getByLabelText("Option 2")).toBeDefined();
            
            // Check checkbox options
            expect(screen.getByLabelText("Check 1")).toBeDefined();
            expect(screen.getByLabelText("Check 2")).toBeDefined();
        });
    });

    describe("User interactions", () => {
        it("allows input when not disabled", async () => {
            const user = userEvent.setup();
            
            renderWithProviders(
                <Formik initialValues={{ textField: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={false} schema={basicSchema} />
                </Formik>,
            );

            const textInput = screen.getByLabelText("Text Input");
            
            await act(async () => {
                await user.type(textInput, "Test input");
            });

            expect((textInput as HTMLInputElement).value).toBe("Test input");
        });

        it("prevents input when disabled", async () => {
            const user = userEvent.setup();
            
            renderWithProviders(
                <Formik initialValues={{ textField: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={true} schema={basicSchema} />
                </Formik>,
            );

            const textInput = screen.getByLabelText("Text Input");
            
            await act(async () => {
                await user.type(textInput, "Test input");
            });

            expect((textInput as HTMLInputElement).value).toBe("");
        });
    });

    describe("Error handling", () => {
        it("handles invalid element types gracefully", () => {
            const invalidSchema: FormSchema = {
                elements: [
                    {
                        id: "invalid1",
                        type: "InvalidType" as any,
                        fieldName: "invalidField",
                        label: "Invalid Element",
                    },
                ],
                containers: [],
            };

            // Should not throw an error
            expect(() => {
                renderWithProviders(
                    <Formik initialValues={{ invalidField: "" }} onSubmit={vi.fn()}>
                        <FormRunView disabled={false} schema={invalidSchema} />
                    </Formik>,
                );
            }).not.toThrow();
        });

        it("normalizes containers when not provided", () => {
            const schemaWithoutContainers: FormSchema = {
                elements: [
                    {
                        id: "field1",
                        type: InputType.Text,
                        fieldName: "field1",
                        label: "Field 1",
                    },
                    {
                        id: "field2",
                        type: InputType.Text,
                        fieldName: "field2",
                        label: "Field 2",
                    },
                ],
                containers: [],
            };

            renderWithProviders(
                <Formik initialValues={{ field1: "", field2: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={false} schema={schemaWithoutContainers} />
                </Formik>,
            );

            // Should render all elements despite no containers
            expect(screen.getByLabelText("Field 1")).toBeDefined();
            expect(screen.getByLabelText("Field 2")).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("maintains proper ARIA attributes", () => {
            renderWithProviders(
                <Formik initialValues={{ textField: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={false} schema={basicSchema} />
                </Formik>,
            );

            // Check form elements have proper labels
            const textInput = screen.getByLabelText("Text Input");
            expect(textInput).toBeDefined();
            
            // Check divider has proper role
            const divider = screen.getByTestId("form-divider");
            expect(divider.getAttribute("role")).toBe("separator");
        });

        it("associates help text with inputs", () => {
            renderWithProviders(
                <Formik initialValues={{ textField: "" }} onSubmit={vi.fn()}>
                    <FormRunView disabled={false} schema={basicSchema} />
                </Formik>,
            );

            // Help text should be visible and associated
            expect(screen.getByText("Enter some text")).toBeDefined();
        });
    });
});

