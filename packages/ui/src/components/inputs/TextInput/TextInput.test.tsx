/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable react-perf/jsx-no-new-object-as-prop */
import { Formik } from "formik";
import { act } from "react";
import { fireEvent, render, screen } from "../../../__mocks__/testUtils";
import { TextInput, TranslatedTextInput } from "./TextInput";

describe("TextInput", () => {
    it("renders with label and placeholder", () => {
        render(<TextInput label="Test Label" placeholder="Test Placeholder" />);
        expect(screen.getByLabelText("Test Label")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Test Placeholder")).toBeInTheDocument();
    });

    it("shows required asterisk when isRequired is true", () => {
        render(<TextInput label="Required Field" isRequired={true} />);
        const asterisks = screen.getAllByText("*");
        expect(asterisks.length).toBeGreaterThan(0);
        expect(asterisks[0]).toBeInTheDocument();
    });

    it("calls onSubmit when Enter is pressed and enterWillSubmit is true", () => {
        const mockSubmit = jest.fn();
        render(<TextInput enterWillSubmit={true} onSubmit={mockSubmit} />);
        fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter", code: "Enter" });
        expect(mockSubmit).toHaveBeenCalled();
    });

    it("does not call onSubmit when Enter is pressed and enterWillSubmit is false", () => {
        const mockSubmit = jest.fn();
        render(<TextInput enterWillSubmit={false} onSubmit={mockSubmit} />);
        fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter", code: "Enter" });
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

    testCases.forEach(({ name, initialValues, language, newValue, expectedValue }) => {
        it(`renders TranslatedTextInput and handles change with ${name}`, async () => {
            render(
                <Formik initialValues={initialValues} onSubmit={jest.fn()}>
                    {({ values }) => (
                        <>
                            <TranslatedTextInput name="testName" language={language} />
                            <div data-testid="form-values">{JSON.stringify(values)}</div>
                        </>
                    )}
                </Formik>,
            );

            const input = screen.getByRole("textbox");
            // @ts-ignore Testing runtime scenario
            const initialInputValue = initialValues.translations?.find(t => t.language === language)?.testName || "";
            expect(input).toHaveValue(initialInputValue);

            await act(async () => {
                fireEvent.change(input, { target: { name: "testName", value: newValue } });
            });

            const formValues = JSON.parse(screen.getByTestId("form-values").textContent || "{}");
            const updatedTranslation = formValues.translations.find(t => t.language === language);
            expect(updatedTranslation?.testName).toBe(expectedValue);
        });
    });
});
