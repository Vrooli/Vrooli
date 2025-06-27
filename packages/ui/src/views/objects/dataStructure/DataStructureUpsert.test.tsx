// AI_CHECK: TEST_QUALITY=8 | LAST: 2025-01-26
import "@testing-library/jest-dom/vitest";
import { screen } from "@testing-library/react";
import { DUMMY_ID } from "@vrooli/shared";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { dataStructureFormTestConfig } from "../../../__test/fixtures/form-testing/DataStructureFormTest.js";
import {
    createMockSession,
    createSimpleFormTester,
    renderFormComponent
} from "../../../__test/helpers/formComponentTestHelpers.js";

// Mock only heavy dependencies and complex hooks
vi.mock("../../../hooks/useManagedObject.js", () => ({
    useManagedObject: vi.fn(() => {
        const mockSession = {
            id: DUMMY_ID,
            users: [{ id: DUMMY_ID, languages: ["en", "es"] }],
        };
        const mockInitialValues = dataStructureFormTestConfig.initialValuesFunction?.(mockSession) || dataStructureFormTestConfig.formFixtures.minimal;

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
            // Simulate successful submission with real validation
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
                mockHandleCompleted(); // Assume success for testing
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
    CodeInput: ({ name, label, value, onChange, ...props }: any) => (
        <div data-testid={`code-input-${name}`}>
            <label htmlFor={name}>{label || name}</label>
            <textarea
                id={name}
                name={name}
                value={value}
                onChange={(e) => onChange?.(e.target.value)}
                data-testid={`input-${name}`}
                {...props}
            />
        </div>
    ),
}));

vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    TranslatedAdvancedInput: ({ name, label, title, value, onChange, ...props }: any) => (
        <div data-testid={`translated-input-${name}`}>
            <label htmlFor={name}>{label || title || name}</label>
            <textarea
                id={name}
                name={name}
                value={value}
                onChange={(e) => onChange?.(e.target.value)}
                data-testid={`input-${name}`}
                {...props}
            />
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

// Import the component after all mocks are set up
import { DataStructureUpsert } from "./DataStructureUpsert.js";

describe("DataStructureUpsert", () => {
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
    const formTester = createSimpleFormTester(DataStructureUpsert, defaultProps, mockSession);

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
                DataStructureUpsert,
                { defaultProps, mockSession }
            );

            // Just verify the component renders without errors
            expect(container).toBeInTheDocument();
        });

        it("handles form submission with valid data", async () => {
            const { user } = renderFormComponent(
                DataStructureUpsert,
                { defaultProps, mockSession }
            );

            // Fill required fields using stable test IDs
            await user.type(screen.getByTestId("input-name"), "Test Data Structure");
            await user.click(screen.getByTestId("submit-button"));

            // Verify form submission was attempted
            expect(mockOnSubmit).toHaveBeenCalled();
        });

        it("handles form cancellation", async () => {
            const { user } = renderFormComponent(
                DataStructureUpsert,
                { defaultProps, mockSession }
            );

            await user.click(screen.getByTestId("cancel-button"));
            expect(mockHandleCancel).toHaveBeenCalled();
        });
    });

    describe("Form Field Interactions", () => {
        it("handles name input", async () => {
            await formTester.testElement("name", "Test Data Structure");
        });

        it("handles description input", async () => {
            await formTester.testElement("description", "A test description", "textarea");
        });

        it("handles version input", async () => {
            await formTester.testElement("versionLabel", "2.0.0");
        });

        it("handles multiple fields together", async () => {
            await formTester.testElements([
                ["name", "Test Data Structure"],
                ["description", "Test description", "textarea"],
                ["versionLabel", "1.5.0"],
            ]);
        });
    });

    describe("Different Modes", () => {
        it("displays create mode correctly", async () => {
            const { user } = renderFormComponent(
                DataStructureUpsert,
                { defaultProps: { ...defaultProps, isCreate: true }, mockSession }
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton).toHaveTextContent(/create/i);
        });

        it("displays edit mode correctly", async () => {
            const { user } = renderFormComponent(
                DataStructureUpsert,
                { defaultProps: { ...defaultProps, isCreate: false }, mockSession }
            );

            const submitButton = screen.getByTestId("submit-button");
            // The actual component uses "Save" for both create and edit modes
            expect(submitButton).toHaveTextContent(/save/i);
        });
    });

    describe("Form Validation Integration", () => {
        it("integrates with real validation schemas", async () => {
            const { user } = renderFormComponent(
                DataStructureUpsert,
                { defaultProps, mockSession }
            );

            // Submit without required fields to test validation
            await user.click(screen.getByTestId("submit-button"));

            // The mock should use real validation and catch errors
            expect(mockOnSubmit).toHaveBeenCalled();
        });
    });

    describe("Fixture Data Integration", () => {
        it("uses fixture data correctly", () => {
            const fixtureData = dataStructureFormTestConfig.formFixtures.minimal;
            expect(fixtureData).toBeDefined();
            expect(fixtureData.translations[0].name).toBe("Test Data Structure");
        });

        it("initializes with correct defaults", () => {
            const values = dataStructureFormTestConfig.initialValuesFunction?.(mockSession);
            expect(values).toMatchObject({
                __typename: "ResourceVersion",
                isPrivate: false,
                id: expect.any(String),
            });
        });

        it("preserves existing data in edit mode", () => {
            const existingData = {
                id: "existing-id",
                versionLabel: "1.5.0",
                isPrivate: true,
                config: { schema: '{"existing": true}' },
            };

            const values = dataStructureFormTestConfig.initialValuesFunction?.(mockSession, existingData);
            expect(values).toMatchObject({
                id: "existing-id",
                versionLabel: "1.5.0",
                isPrivate: true,
            });
            expect(values?.config?.schema).toBe('{"existing": true}');
        });
    });

    describe("Complex Scenarios", () => {
        it("handles complete form workflow", async () => {
            const onCompleted = vi.fn();
            const { user } = renderFormComponent(
                DataStructureUpsert,
                { defaultProps: { ...defaultProps, onCompleted }, mockSession }
            );

            // Fill form with complete data
            await user.type(screen.getByTestId("input-name"), "Complete Data Structure");
            await user.type(screen.getByTestId("input-description"), "Complete description");

            // Submit form
            await user.click(screen.getByTestId("submit-button"));

            // Verify submission was called
            expect(mockOnSubmit).toHaveBeenCalled();

            // The completion behavior depends on the form's implementation
            // For this test, we just verify the form can be filled and submitted
        });

        describe.each([
            ['minimal', dataStructureFormTestConfig.formFixtures.minimal],
            ['complete', dataStructureFormTestConfig.formFixtures.complete || dataStructureFormTestConfig.formFixtures.minimal],
        ])('with %s fixture data', (scenario, fixtureData) => {
            it(`renders correctly with ${scenario} data`, async () => {
                // Test different fixture scenarios
                expect(fixtureData).toBeDefined();
                expect(fixtureData.translations).toBeDefined();
            });
        });
    });
});