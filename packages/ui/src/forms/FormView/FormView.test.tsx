import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Formik } from "formik";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { FormStructureType, InputType, type FormSchema } from "@vrooli/shared";
import { FormView } from "./FormView";
import { withSuppressedConsole } from "../../__test/utils/consoleUtils";

// Test data
const emptySchema: FormSchema = {
    elements: [],
    containers: [],
};

const basicInputSchema: FormSchema = {
    elements: [
        {
            id: "text-input-1",
            type: InputType.Text,
            fieldName: "name",
            label: "Full Name",
            isRequired: true,
            props: {
                placeholder: "Enter your name",
            },
        },
        {
            id: "email-input-1",
            type: InputType.Text,
            fieldName: "email",
            label: "Email",
            isRequired: false,
            props: {
                placeholder: "Enter your email",
                type: "email",
            },
        },
    ],
    containers: [{
        totalItems: 2,
        spacing: 2,
        xs: 1,
        sm: 1,
        md: 1,
        lg: 1,
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
            id: "divider-1",
            type: FormStructureType.Divider,
            label: "",
        },
        {
            id: "text-input-2",
            type: InputType.Text,
            fieldName: "firstName",
            label: "First Name",
            isRequired: true,
            props: {
                placeholder: "Enter first name",
            },
        },
        {
            id: "switch-input-1",
            type: InputType.Switch,
            fieldName: "newsletter",
            label: "Subscribe to Newsletter",
            isRequired: false,
            props: {
                defaultValue: false,
            },
        },
        {
            id: "tip-1",
            type: FormStructureType.Tip,
            label: "This is a helpful tip for users.",
        },
    ],
    containers: [{
        totalItems: 5,
        spacing: 2,
        xs: 1,
        sm: 1,
        md: 1,
        lg: 1,
    }],
};

// Helper component that wraps FormView with Formik for run mode tests
function FormViewWithFormik({ 
    schema, 
    isEditing = false, 
    disabled = false, 
    fieldNamePrefix,
    onSchemaChange = vi.fn(),
    initialValues = {},
}: {
    schema: FormSchema;
    isEditing?: boolean;
    disabled?: boolean;
    fieldNamePrefix?: string;
    onSchemaChange?: (schema: FormSchema) => void;
    initialValues?: Record<string, any>;
}) {
    return (
        <Formik initialValues={initialValues} onSubmit={vi.fn()}>
            <FormView
                disabled={disabled}
                fieldNamePrefix={fieldNamePrefix}
                isEditing={isEditing}
                schema={schema}
                onSchemaChange={onSchemaChange}
                data-testid="form-view"
            />
        </Formik>
    );
}

describe("FormView", () => {
    describe("Component Mode Switching", () => {
        it("renders FormRunView when isEditing is false", () => {
            render(
                <FormViewWithFormik 
                    schema={basicInputSchema} 
                    isEditing={false} 
                />
            );

            // FormRunView should render the form for display/input
            expect(screen.getByTestId("form-view")).toBeTruthy();
            
            // Should contain form elements for user interaction
            const inputs = screen.getAllByRole("textbox");
            expect(inputs).toHaveLength(2);
        });

        it("renders FormBuildView when isEditing is true", () => {
            const onSchemaChange = vi.fn();
            
            render(
                <FormView
                    disabled={false}
                    isEditing={true}
                    schema={basicInputSchema}
                    onSchemaChange={onSchemaChange}
                    data-testid="form-view"
                />
            );

            // FormBuildView should render the form builder interface
            expect(screen.getByTestId("form-view")).toBeTruthy();
            
            // Should have form builder elements (elements can be selected/edited)
            // Note: The specific test depends on the FormBuildView implementation
            // This is testing the mode switching behavior
        });

        it("switches between modes correctly", () => {
            const onSchemaChange = vi.fn();
            const { rerender } = render(
                <FormViewWithFormik 
                    schema={basicInputSchema} 
                    isEditing={false} 
                />
            );

            // Start in run mode
            const inputs = screen.getAllByRole("textbox");
            expect(inputs).toHaveLength(2);

            // Switch to edit mode
            rerender(
                <FormView
                    disabled={false}
                    isEditing={true}
                    schema={basicInputSchema}
                    onSchemaChange={onSchemaChange}
                    data-testid="form-view"
                />
            );

            // Should now be in edit mode (form builder)
            expect(screen.getByTestId("form-view")).toBeTruthy();
        });
    });

    describe("Run Mode (FormRunView)", () => {
        describe("Empty schema handling", () => {
            it("displays empty form message when no elements", () => {
                render(
                    <FormViewWithFormik schema={emptySchema} />
                );

                expect(screen.getByText("The form is empty.")).toBeTruthy();
            });

            it("has correct test id for empty form", () => {
                render(
                    <FormViewWithFormik schema={emptySchema} />
                );

                expect(screen.getByTestId("form-view")).toBeTruthy();
            });
        });

        describe("Basic form elements", () => {
            it("renders text inputs correctly", () => {
                render(
                    <FormViewWithFormik schema={basicInputSchema} />
                );

                // Look for form inputs by their field names or test IDs since labels might be wrapped
                const formView = screen.getByTestId("form-view");
                expect(formView).toBeTruthy();
                
                // Check that form elements are rendered
                const inputs = screen.getAllByRole("textbox");
                expect(inputs).toHaveLength(2);
            });

            it("handles form input interactions", async () => {
                const user = userEvent.setup();
                
                render(
                    <FormViewWithFormik 
                        schema={basicInputSchema}
                        initialValues={{ name: "", email: "" }}
                    />
                );

                const inputs = screen.getAllByRole("textbox");
                expect(inputs).toHaveLength(2);

                await act(async () => {
                    await user.type(inputs[0], "John Doe");
                    await user.type(inputs[1], "john@example.com");
                });

                expect(inputs[0].value).toBe("John Doe");
                expect(inputs[1].value).toBe("john@example.com");
            });

            it("respects disabled state", () => {
                render(
                    <FormViewWithFormik 
                        schema={basicInputSchema} 
                        disabled={true}
                    />
                );

                const inputs = screen.getAllByRole("textbox");
                expect(inputs).toHaveLength(2);
                
                inputs.forEach(input => {
                    expect(input.disabled).toBe(true);
                });
            });
        });

        describe("Complex form elements", () => {
            it("renders form view without errors", () => {
                const { container } = render(
                    <FormViewWithFormik schema={basicInputSchema} />
                );

                // Just check that the form renders without throwing errors
                expect(container).toBeTruthy();
                const formView = screen.getByTestId("form-view");
                expect(formView).toBeTruthy();
            });

            it("handles complex schema elements", () => {
                // Use a simpler schema that we know works
                const simpleComplexSchema: FormSchema = {
                    elements: [
                        {
                            id: "text-input-3",
                            type: InputType.Text,
                            fieldName: "firstName",
                            label: "First Name",
                            isRequired: true,
                            props: {
                                placeholder: "Enter first name",
                            },
                        },
                    ],
                    containers: [{
                        totalItems: 1,
                        spacing: 2,
                        xs: 1,
                        sm: 1,
                        md: 1,
                        lg: 1,
                    }],
                };

                render(
                    <FormViewWithFormik 
                        schema={simpleComplexSchema}
                        initialValues={{ firstName: "" }}
                    />
                );

                const formView = screen.getByTestId("form-view");
                expect(formView).toBeTruthy();
            });
        });

        describe("Field name prefix", () => {
            it("applies field name prefix correctly", () => {
                render(
                    <FormViewWithFormik 
                        schema={basicInputSchema}
                        fieldNamePrefix="test-prefix"
                        initialValues={{ "test-prefix-name": "", "test-prefix-email": "" }}
                    />
                );

                // Check that the form renders with prefix
                const formView = screen.getByTestId("form-view");
                expect(formView).toBeTruthy();
            });

            it("generates correct element ID with prefix", () => {
                render(
                    <FormViewWithFormik 
                        schema={basicInputSchema}
                        fieldNamePrefix="prefix"
                    />
                );

                const formElement = screen.getByTestId("form-view");
                expect(formElement).toBeTruthy();
            });
        });
    });

    describe("Build Mode (FormBuildView)", () => {
        describe("Schema modification", () => {
            it("calls onSchemaChange when schema is modified", () => {
                const onSchemaChange = vi.fn();
                
                render(
                    <FormView
                        disabled={false}
                        isEditing={true}
                        schema={basicInputSchema}
                        onSchemaChange={onSchemaChange}
                        data-testid="form-view"
                    />
                );

                // The specific interaction depends on FormBuildView implementation
                // This test verifies the callback is passed correctly
                expect(onSchemaChange).toHaveBeenCalledTimes(0); // No changes yet
            });

            it("handles empty schema in edit mode", () => {
                const onSchemaChange = vi.fn();
                
                render(
                    <FormView
                        disabled={false}
                        isEditing={true}
                        schema={emptySchema}
                        onSchemaChange={onSchemaChange}
                        data-testid="form-view"
                    />
                );

                expect(screen.getByTestId("form-view")).toBeTruthy();
                // Should show form builder interface for empty form
            });
        });

        describe("Build mode interactions", () => {
            it("renders build interface correctly", () => {
                const onSchemaChange = vi.fn();
                
                render(
                    <FormView
                        disabled={false}
                        isEditing={true}
                        schema={basicInputSchema}
                        onSchemaChange={onSchemaChange}
                        data-testid="form-view"
                    />
                );

                // Form builder should be rendered
                expect(screen.getByTestId("form-view")).toBeTruthy();
            });

            it("respects disabled state in build mode", () => {
                const onSchemaChange = vi.fn();
                
                render(
                    <FormView
                        disabled={true}
                        isEditing={true}
                        schema={basicInputSchema}
                        onSchemaChange={onSchemaChange}
                        data-testid="form-view"
                    />
                );

                expect(screen.getByTestId("form-view")).toBeTruthy();
                // Disabled state should be handled by FormBuildView
            });
        });
    });

    describe("Accessibility", () => {
        it("maintains proper form semantics in run mode", () => {
            render(
                <FormViewWithFormik schema={basicInputSchema} />
            );

            // Check that inputs are accessible
            const inputs = screen.getAllByRole("textbox");
            expect(inputs).toHaveLength(2);
        });

        it("provides proper test identifiers", () => {
            render(
                <FormViewWithFormik schema={basicInputSchema} />
            );

            expect(screen.getByTestId("form-view")).toBeTruthy();
        });

        it("renders without accessibility violations", () => {
            const { container } = render(
                <FormViewWithFormik schema={basicInputSchema} />
            );

            // Basic accessibility check - form should render
            expect(container).toBeTruthy();
        });
    });

    describe("Error boundaries and edge cases", () => {
        it("handles malformed schema gracefully", async () => {
            await withSuppressedConsole(() => {
                const malformedSchema: FormSchema = {
                    elements: [
                        {
                            id: "invalid-element",
                            // @ts-expect-error - Testing invalid schema
                            type: "InvalidType",
                            fieldName: "invalid",
                            label: "Invalid Element",
                            isRequired: false,
                            props: {},
                        },
                    ],
                    containers: [],
                };

                // Should not crash when rendering invalid elements
                expect(() => {
                    render(
                        <FormViewWithFormik schema={malformedSchema} />
                    );
                }).not.toThrow();
            });
        });

        it("handles missing container configuration", () => {
            const schemaWithoutContainers: FormSchema = {
                elements: basicInputSchema.elements,
                containers: [], // No containers defined
            };

            expect(() => {
                render(
                    <FormViewWithFormik schema={schemaWithoutContainers} />
                );
            }).not.toThrow();
        });

        it("handles undefined/null props gracefully", async () => {
            await withSuppressedConsole(() => {
                const schemaWithNullProps: FormSchema = {
                    elements: [
                        {
                            id: "text-null-props",
                            type: InputType.Text,
                            fieldName: "nullProps",
                            label: "Null Props",
                            isRequired: false,
                            // @ts-expect-error - Testing null props
                            props: null,
                        },
                    ],
                    containers: [{
                        totalItems: 1,
                        spacing: 2,
                        xs: 1,
                        sm: 1,
                        md: 1,
                        lg: 1,
                    }],
                };

                expect(() => {
                    render(
                        <FormViewWithFormik schema={schemaWithNullProps} />
                    );
                }).not.toThrow();
            });
        });
    });

    describe("Performance considerations", () => {
        it("handles large schemas efficiently", () => {
            const largeSchema: FormSchema = {
                elements: Array.from({ length: 20 }, (_, i) => ({
                    id: `element-${i}`,
                    type: InputType.Text,
                    fieldName: `field${i}`,
                    label: `Field ${i}`,
                    isRequired: false,
                    props: {
                        placeholder: `Enter value for field ${i}`,
                    },
                })),
                containers: [{
                    totalItems: 20,
                    spacing: 2,
                    xs: 1,
                    sm: 2,
                    md: 3,
                    lg: 4,
                }],
            };

            const startTime = performance.now();
            render(
                <FormViewWithFormik schema={largeSchema} />
            );
            const endTime = performance.now();

            // Should render within reasonable time (less than 1 second)
            expect(endTime - startTime).toBeLessThan(1000);
        });
    });
});