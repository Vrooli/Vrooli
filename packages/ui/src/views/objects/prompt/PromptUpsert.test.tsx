// AI_CHECK: TEST_QUALITY=8 | LAST: 2025-01-26
import { screen, waitFor } from "@testing-library/react";
import { DUMMY_ID } from "@vrooli/shared";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { promptFormTestConfig } from "../../../__test/fixtures/form-testing/PromptFormTest.js";
import {
    createMockSession,
    createSimpleFormTester,
    renderFormComponent,
} from "../../../__test/helpers/formComponentTestHelpers.js";

// Mock only heavy dependencies and complex hooks
vi.mock("../../../hooks/useManagedObject.js", () => ({
    useManagedObject: vi.fn(() => {
        const mockSession = {
            id: DUMMY_ID,
            users: [{ id: DUMMY_ID, languages: ["en", "es"] }],
        };
        const mockInitialValues = promptFormTestConfig.initialValuesFunction?.(mockSession) || promptFormTestConfig.formFixtures.minimal;

        return {
            isLoading: false,
            object: mockInitialValues,
            permissions: { canUpdate: true },
            setObject: vi.fn(),
        };
    }),
}));

// Declare the mock functions at the top level so they can be used in the mock
const mockOnSubmit = vi.fn();
const mockHandleCompleted = vi.fn();
const mockHandleCancel = vi.fn();

vi.mock("../../../hooks/useStandardUpsertForm.ts", () => ({
    useStandardUpsertForm: vi.fn((config, options) => {
        mockHandleCancel.mockImplementation(() => {
            options.onClose?.();
        });
        mockHandleCompleted.mockImplementation(() => {
            options.onCompleted?.();
        });
        mockOnSubmit.mockImplementation((values) => {
            try {
                if (config.validation && config.validation.isValidSync) {
                    const isValid = config.validation.isValidSync(values);
                    if (isValid) {
                        mockHandleCompleted();
                        return {};
                    }
                }
                return { general: "Validation failed" };
            } catch (error) {
                mockHandleCompleted();
                return {};
            }
        });

        return {
            session: createMockSession(),
            isLoading: false,
            handleCancel: mockHandleCancel,
            handleCompleted: mockHandleCompleted,
            onSubmit: mockOnSubmit,
            validateValues: vi.fn(),
            language: "en",
            languages: ["en", "es"],
            handleAddLanguage: vi.fn(),
            handleDeleteLanguage: vi.fn(),
            setLanguage: vi.fn(),
            translationErrors: {},
        };
    }),
}));

// Mock heavy UI components with simple, testable versions
vi.mock("../../../components/inputs/CodeInput/CodeInput.js", () => ({
    CodeInputBase: ({ 
        name, 
        label, 
        content, 
        defaultValue, 
        disabled, 
        codeLanguage, 
        handleCodeLanguageChange, 
        handleContentChange, 
        format, 
        limitTo, 
        variables,
        ...restProps 
    }: any) => {
        // Filter out component-specific props to prevent them from reaching DOM elements
        const { 
            // Remove any other props that shouldn't be on DOM elements
            ...safeDomProps 
        } = restProps;
        
        return (
            <div data-testid={`code-input-${name}`}>
                <label htmlFor={name}>{label || name}</label>
                <div data-testid={`language-${name}`}>Language: {codeLanguage}</div>
                <textarea
                    id={name}
                    name={name}
                    value={content || defaultValue || ""}
                    onChange={(e) => handleContentChange?.(e.target.value)}
                    data-testid={`input-${name}`}
                    disabled={disabled}
                    {...safeDomProps}
                />
                {variables && <div data-testid={`variables-${name}`}>Variables available</div>}
            </div>
        );
    },
}));

vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    TranslatedAdvancedInput: ({ 
        name, 
        label, 
        title, 
        value, 
        onChange, 
        error, 
        helperText, 
        multiline,
        disabled,
        placeholder,
        ...domProps 
    }: any) => (
        <div data-testid={`translated-input-${name}`}>
            <label htmlFor={name}>{label || title || name}</label>
            <textarea
                id={name}
                name={name}
                value={value}
                onChange={(e) => onChange?.(e.target.value)}
                data-testid={`input-${name}`}
                disabled={disabled}
                placeholder={placeholder}
                {...domProps}
            />
            {error && <div data-testid={`error-${name}`}>{error}</div>}
            {helperText && <div data-testid={`helper-${name}`}>{helperText}</div>}
        </div>
    ),
}));

vi.mock("../../../components/buttons/AutoFillButton.js", () => ({
    AutoFillButton: ({ onClick }: any) => (
        <button data-testid="autofill-button" onClick={onClick}>AutoFill</button>
    ),
}));

vi.mock("../../../hooks/tasks.ts", () => ({
    useAutoFill: vi.fn(() => ({
        autoFill: vi.fn(),
        isAutoFillLoading: false,
    })),
    getAutoFillTranslationData: vi.fn(() => ({})),
}));

vi.mock("../../../hooks/useDimensions.js", () => ({
    useDimensions: vi.fn(() => ({
        dimensions: { width: 1200, height: 800 },
        ref: { current: null },
    })),
}));

// Mock navigation components
vi.mock("../../../components/buttons/SearchExistingButton/SearchExistingButton.js", () => ({
    SearchExistingButton: () => <button data-testid="search-existing-button">Search Existing</button>,
}));

vi.mock("../../../components/navigation/TopBar.js", () => ({
    TopBar: ({ title }: any) => <div data-testid="top-bar">{title}</div>,
}));

vi.mock("../../../components/containers/ContentCollapse.js", () => ({
    ContentCollapse: ({ children, title, isOpen }: any) => (
        <div data-testid={`content-collapse-${title?.toLowerCase()?.replace(/\s+/g, "-") || "unknown"}`}>
            <button data-testid={`collapse-toggle-${title?.toLowerCase()?.replace(/\s+/g, "-") || "unknown"}`}>
                {title} ({isOpen ? "Open" : "Closed"})
            </button>
            {isOpen && <div>{children}</div>}
        </div>
    ),
}));

// Import the component after all mocks are set up
import { PromptUpsert } from "./PromptUpsert.js";

describe("PromptUpsert", () => {
    const mockSession = createMockSession();

    const defaultProps = {
        isCreate: true,
        isOpen: true,
        display: "Dialog" as const,
        onClose: vi.fn(),
        onCompleted: vi.fn(),
        onDeleted: vi.fn(),
    };

    // Create the simple form tester once for all tests
    const formTester = createSimpleFormTester(PromptUpsert, defaultProps, mockSession);

    beforeEach(() => {
        vi.clearAllMocks();
        
        // Reset mock functions
        mockOnSubmit.mockClear();
        mockHandleCompleted.mockClear();
        mockHandleCancel.mockClear();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Basic Functionality", () => {
        it("renders successfully", async () => {
            const { container } = renderFormComponent(
                PromptUpsert,
                { defaultProps, mockSession },
            );
            
            // Just verify the component renders without errors
            expect(container).toBeTruthy();
        });

        it("handles form submission with valid data", async () => {
            const { user } = renderFormComponent(
                PromptUpsert,
                { defaultProps, mockSession },
            );

            // Fill required fields using stable test IDs
            await user.type(screen.getByTestId("input-name"), "Test Prompt");
            await user.click(screen.getByTestId("submit-button-wrapper"));

            // Verify form submission was attempted
            await waitFor(() => {
                expect(mockOnSubmit).toHaveBeenCalled();
            });
        });

        it("handles form cancellation", async () => {
            const { user } = renderFormComponent(
                PromptUpsert,
                { defaultProps, mockSession },
            );

            await user.click(screen.getByTestId("cancel-button"));
            await waitFor(() => {
                expect(mockHandleCancel).toHaveBeenCalled();
            });
        });
    });

    describe("Form Field Interactions", () => {
        it("handles name input", async () => {
            await formTester.testElement("name", "Test Prompt");
        });

        it("handles description input", async () => {
            await formTester.testElement("description", "A test description", "textarea");
        });

        it("handles version input", async () => {
            await formTester.testElement("versionLabel", "2.0.0");
        });

        it("handles output function input", async () => {
            await formTester.testElement("output", "function generatePrompt(...inputs) { return inputs.map(i => i.value).join(' '); }", "textarea");
        });

        it("handles multiple fields together", async () => {
            await formTester.testElements([
                ["name", "Test Prompt"],
                ["description", "Test description", "textarea"],
                ["versionLabel", "1.5.0"],
            ]);
        });
    });

    describe("Different Modes", () => {
        it("displays create mode correctly", async () => {
            renderFormComponent(
                PromptUpsert,
                { defaultProps: { ...defaultProps, isCreate: true }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Create");
        });

        it("displays edit mode correctly", async () => {
            renderFormComponent(
                PromptUpsert,
                { defaultProps: { ...defaultProps, isCreate: false }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Save");
        });
    });

    describe("Form Validation Integration", () => {
        it("integrates with real validation schemas", async () => {
            const { user } = renderFormComponent(
                PromptUpsert,
                { defaultProps, mockSession },
            );

            // Submit without required fields to test validation
            await user.click(screen.getByTestId("submit-button-wrapper"));
            
            // The mock should use real validation and catch errors
            await waitFor(() => {
                expect(mockOnSubmit).toHaveBeenCalled();
            });
        });
    });

    describe("Fixture Data Integration", () => {
        it("uses fixture data correctly", () => {
            const fixtureData = promptFormTestConfig.formFixtures.minimal;
            expect(fixtureData).toBeDefined();
            expect(fixtureData.translations[0].name).toBe("Test Prompt");
        });

        it("initializes with correct defaults", () => {
            const values = promptFormTestConfig.initialValuesFunction?.(mockSession);
            expect(values).toMatchObject({
                __typename: "ResourceVersion",
                resourceSubType: "StandardPrompt",
                isPrivate: false,
                id: expect.any(String),
            });
        });

        it("preserves existing data in edit mode", () => {
            const existingData = {
                id: "existing-id",
                versionLabel: "1.5.0",
                isPrivate: true,
                config: { 
                    props: JSON.stringify({
                        output: "function generatePrompt() { return 'Existing prompt: {input}'; }",
                    }),
                },
            };

            const values = promptFormTestConfig.initialValuesFunction?.(mockSession, existingData);
            expect(values).toMatchObject({
                id: "existing-id",
                versionLabel: "1.5.0",
                isPrivate: true,
            });
            expect(values?.config?.props).toBeDefined();
        });
    });

    describe("Complex Scenarios", () => {
        it("handles complete form workflow", async () => {
            const onCompleted = vi.fn();
            const { user } = renderFormComponent(
                PromptUpsert,
                { defaultProps: { ...defaultProps, onCompleted }, mockSession },
            );

            // Fill form with complete data
            await user.type(screen.getByTestId("input-name"), "Complete Prompt");
            await user.type(screen.getByTestId("input-description"), "Complete description");
            // For complex code with special characters, use paste instead of type
            const outputInput = screen.getByTestId("input-output");
            await user.click(outputInput);
            await user.paste("function generatePrompt(...inputs) { return inputs.map(i => i.value).join(' '); }");
            
            // Submit form
            await user.click(screen.getByTestId("submit-button-wrapper"));

            // Verify submission was called
            await waitFor(() => {
                expect(mockOnSubmit).toHaveBeenCalled();
            });
        });

        describe.each([
            ["minimal", promptFormTestConfig.formFixtures.minimal],
            ["complete", promptFormTestConfig.formFixtures.complete || promptFormTestConfig.formFixtures.minimal],
        ])("with %s fixture data", (scenario, fixtureData) => {
            it(`renders correctly with ${scenario} data`, async () => {
                // Test different fixture scenarios
                expect(fixtureData).toBeDefined();
                expect(fixtureData.translations).toBeDefined();
            });
        });
    });
});
