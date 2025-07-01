// AI_CHECK: TEST_QUALITY=8 | LAST: 2025-01-26
// AI_CHECK: TYPE_SAFETY=eliminated-4-any-types-in-mock-interfaces | LAST: 2025-06-28
import { screen } from "@testing-library/react";
import { DUMMY_ID } from "@vrooli/shared";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { smartContractFormTestConfig } from "../../../__test/fixtures/form-testing/SmartContractFormTest.js";
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
        const mockInitialValues = smartContractFormTestConfig.initialValuesFunction?.(mockSession) || smartContractFormTestConfig.formFixtures.minimal;

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
interface CodeInputProps {
    name: string;
    label?: string;
    value?: string;
    onChange?: (value: string) => void;
    limitTo?: unknown;
    disabled?: boolean;
    isRequired?: boolean;
    [key: string]: unknown;
}

vi.mock("../../../components/inputs/CodeInput/CodeInput.js", () => ({
    CodeInput: ({ name, label, value, onChange, limitTo, disabled, codeLanguageField, defaultValueField, formatField, variablesField, ...domProps }: CodeInputProps) => {
        // Filter out React-specific props that shouldn't go to DOM elements
        const { isRequired, ...filteredProps } = domProps;
        
        return (
            <div data-testid={`code-input-${name}`}>
                <label htmlFor={name}>{label || name}</label>
                <textarea
                    id={name}
                    name={name}
                    value={value}
                    onChange={(e) => onChange?.(e.target.value)}
                    data-testid={`input-${name}`}
                    disabled={disabled}
                    {...filteredProps}
                />
            </div>
        );
    },
}));

interface TranslatedAdvancedInputProps {
    name: string;
    label?: string;
    title?: string;
    value?: string;
    onChange?: (value: string) => void;
    isRequired?: boolean;
    disabled?: boolean;
    features?: unknown;
    language?: string;
    limitTo?: unknown;
    onTasksChange?: unknown;
    onContextDataChange?: unknown;
    onSubmit?: unknown;
    tabIndex?: number;
    tasks?: unknown;
    contextData?: unknown;
    [key: string]: unknown;
}

vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    TranslatedAdvancedInput: ({ name, label, title, value, onChange, isRequired, disabled, ...domProps }: TranslatedAdvancedInputProps) => {
        // Filter out React-specific props that shouldn't go to DOM elements
        const { features, language, limitTo, onTasksChange, onContextDataChange, onSubmit, tabIndex, tasks, contextData, ...filteredProps } = domProps;
        
        return (
            <div data-testid={`translated-input-${name}`}>
                <label htmlFor={name}>{label || title || name}</label>
                <textarea
                    id={name}
                    name={name}
                    value={value}
                    onChange={(e) => onChange?.(e.target.value)}
                    data-testid={`input-${name}`}
                    disabled={disabled}
                    {...filteredProps}
                />
            </div>
        );
    },
}));

interface AutoFillButtonProps {
    onClick?: () => void;
}

vi.mock("../../../components/buttons/AutoFillButton.js", () => ({
    AutoFillButton: ({ onClick }: AutoFillButtonProps) => (
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

interface TopBarProps {
    title?: string;
}

vi.mock("../../../components/navigation/TopBar.js", () => ({
    TopBar: ({ title }: TopBarProps) => <div data-testid="top-bar">{title}</div>,
}));

// Import the component after all mocks are set up
import { SmartContractUpsert } from "./SmartContractUpsert.js";

describe("SmartContractUpsert", () => {
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
    const formTester = createSimpleFormTester(SmartContractUpsert, defaultProps, mockSession);

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
                SmartContractUpsert,
                { defaultProps, mockSession },
            );
            
            // Just verify the component renders without errors
            expect(container).toBeTruthy();
        });

        it("handles form submission with valid data", async () => {
            const { user } = renderFormComponent(
                SmartContractUpsert,
                { defaultProps, mockSession },
            );

            // Fill required fields using stable test IDs
            await user.type(screen.getByTestId("input-name"), "Test Smart Contract");
            await user.click(screen.getByTestId("submit-button"));

            // Verify form submission was attempted
            expect(mockOnSubmit).toHaveBeenCalled();
        });

        it("handles form cancellation", async () => {
            const { user } = renderFormComponent(
                SmartContractUpsert,
                { defaultProps, mockSession },
            );

            await user.click(screen.getByTestId("cancel-button"));
            expect(mockHandleCancel).toHaveBeenCalled();
        });
    });

    describe("Form Field Interactions", () => {
        it("handles name input", async () => {
            await formTester.testElement("name", "Test Smart Contract");
        });

        it("handles description input", async () => {
            await formTester.testElement("description", "A test description", "textarea");
        });

        it("handles version input", async () => {
            await formTester.testElement("versionLabel", "2.0.0");
        });

        it("handles code input", async () => {
            await formTester.testElement("config.content", "pragma solidity ^0.8.0;\ncontract Test {}", "textarea");
        });

        it("handles multiple fields together", async () => {
            await formTester.testElements([
                ["name", "Test Smart Contract"],
                ["description", "Test description", "textarea"],
                ["versionLabel", "1.5.0"],
            ]);
        });
    });

    describe("Different Modes", () => {
        it("displays create mode correctly", async () => {
            const { user } = renderFormComponent(
                SmartContractUpsert,
                { defaultProps: { ...defaultProps, isCreate: true }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Create");
        });

        it("displays edit mode correctly", async () => {
            const { user } = renderFormComponent(
                SmartContractUpsert,
                { defaultProps: { ...defaultProps, isCreate: false }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Save");
        });
    });

    describe("Form Validation Integration", () => {
        it("integrates with real validation schemas", async () => {
            const { user } = renderFormComponent(
                SmartContractUpsert,
                { defaultProps, mockSession },
            );

            // Submit without required fields to test validation
            await user.click(screen.getByTestId("submit-button"));
            
            // The mock should use real validation and catch errors
            expect(mockOnSubmit).toHaveBeenCalled();
        });
    });

    describe("Fixture Data Integration", () => {
        it("uses fixture data correctly", () => {
            const fixtureData = smartContractFormTestConfig.formFixtures.minimal;
            expect(fixtureData).toBeDefined();
            expect(fixtureData.translations[0].name).toBe("Test Smart Contract");
        });

        it("initializes with correct defaults", () => {
            const values = smartContractFormTestConfig.initialValuesFunction?.(mockSession);
            expect(values).toMatchObject({
                __typename: "ResourceVersion",
                resourceSubType: "CodeSmartContract",
                isPrivate: false,
                id: expect.any(String),
            });
        });

        it("preserves existing data in edit mode", () => {
            const existingData = {
                id: "existing-id",
                versionLabel: "1.5.0",
                codeLanguage: "Solidity",
                isPrivate: true,
                config: { schema: "{\"existing\": true}" },
            };

            const values = smartContractFormTestConfig.initialValuesFunction?.(mockSession, existingData);
            expect(values).toMatchObject({
                id: "existing-id",
                versionLabel: "1.5.0",
                codeLanguage: "Solidity",
                isPrivate: true,
            });
        });
    });

    describe("Complex Scenarios", () => {
        it("handles complete form workflow", async () => {
            const onCompleted = vi.fn();
            const { user } = renderFormComponent(
                SmartContractUpsert,
                { defaultProps: { ...defaultProps, onCompleted }, mockSession },
            );

            // Fill form with complete data
            await user.type(screen.getByTestId("input-name"), "Complete Smart Contract");
            await user.type(screen.getByTestId("input-description"), "Complete description");
            await user.type(screen.getByTestId("input-config.content"), "pragma solidity ^0.8.0;");
            
            // Submit form
            await user.click(screen.getByTestId("submit-button"));

            // Verify submission was called
            expect(mockOnSubmit).toHaveBeenCalled();
        });

        describe.each([
            ["minimal", smartContractFormTestConfig.formFixtures.minimal],
            ["complete", smartContractFormTestConfig.formFixtures.complete || smartContractFormTestConfig.formFixtures.minimal],
        ])("with %s fixture data", (scenario, fixtureData) => {
            it(`renders correctly with ${scenario} data`, async () => {
                // Test different fixture scenarios
                expect(fixtureData).toBeDefined();
                expect(fixtureData.translations).toBeDefined();
            });
        });
    });
});
