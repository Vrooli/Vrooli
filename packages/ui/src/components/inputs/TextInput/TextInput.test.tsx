/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable react-perf/jsx-no-new-object-as-prop */
import { describe, it, expect, vi } from "vitest";
import { Formik } from "formik";
import { act } from "react";
import { fireEvent, render, screen } from "../../../__test/testUtils.js";
import { TextInput, TranslatedTextInput } from "./TextInput.js";

describe("TextInput", () => {
    it("renders with label and placeholder", () => {
        render(<TextInput label="Test Label" placeholder="Test Placeholder" />);
        expect(screen.getByLabelText("Test Label")).toBeTruthy();
        expect(screen.getByPlaceholderText("Test Placeholder")).toBeTruthy();
    });

    it("shows required asterisk when isRequired is true", () => {
        render(<TextInput label="Required Field" isRequired={true} />);
        const asterisks = screen.getAllByText("*");
        expect(asterisks.length).toBeGreaterThan(0);
        expect(asterisks[0]).toBeTruthy();
    });

    it("calls onSubmit when Enter is pressed and enterWillSubmit is true", () => {
        const mockSubmit = vi.fn();
        render(<TextInput enterWillSubmit={true} onSubmit={mockSubmit} name="test" />);
        const textbox = screen.getByRole("textbox", { hidden: true });
        fireEvent.keyDown(textbox, { key: "Enter", code: "Enter" });
        expect(mockSubmit).toHaveBeenCalled();
    });

    it("does not call onSubmit when Enter is pressed and enterWillSubmit is false", () => {
        const mockSubmit = vi.fn();
        render(<TextInput enterWillSubmit={false} onSubmit={mockSubmit} name="test" />);
        const textbox = screen.getByRole("textbox", { hidden: true });
        fireEvent.keyDown(textbox, { key: "Enter", code: "Enter" });
        expect(mockSubmit).not.toHaveBeenCalled();
    });
});

describe("TranslatedTextInput", () => {
    const testCases = [
        {
            name: "empty initial values",
            initialValues: {
                translations: [
                    { language: "en", testName: "" },
                    { language: "fr", testName: "" },
                ],
            },
            language: "en",
            newValue: "New Value",
            expectedValue: "New Value",
        },
        {
            name: "existing values",
            initialValues: {
                translations: [
                    { language: "en", testName: "Existing English" },
                    { language: "fr", testName: "Existing French" },
                ],
            },
            language: "fr",
            newValue: "Nouvelle Valeur",
            expectedValue: "Nouvelle Valeur",
        },
        {
            name: "missing language",
            initialValues: {
                translations: [
                    { language: "en", testName: "English Only" },
                ],
            },
            language: "de",
            newValue: "Deutsch",
            expectedValue: "Deutsch",
        },
        {
            name: "missing field in translation",
            initialValues: {
                translations: [
                    { language: "en" },
                ],
            },
            language: "en",
            newValue: "Missing Field",
            expectedValue: "Missing Field",
        },
        {
            name: "missing translations",
            initialValues: {},
            language: "en",
            newValue: "New Value",
            expectedValue: "New Value",
        },
    ];

    it.each(testCases)(
        "renders TranslatedTextInput and handles change with $name",
        async ({ initialValues, language, newValue, expectedValue }) => {
            render(
                <Formik initialValues={initialValues} onSubmit={vi.fn()}>
                    {({ values }) => (
                        <>
                            <TranslatedTextInput name="testName" language={language} />
                            <div data-testid="form-values">{JSON.stringify(values)}</div>
                        </>
                    )}
                </Formik>,
            );

            const input = screen.getByRole("textbox", { hidden: true });
            
            await act(async () => {
                fireEvent.change(input, { target: { name: "testName", value: newValue } });
            });

            const formValues = JSON.parse(screen.getByTestId("form-values").textContent || "{}");
            const updatedTranslation = formValues.translations?.find(t => t.language === language);
            expect(updatedTranslation?.testName).toBe(expectedValue);
        }
    );
});
