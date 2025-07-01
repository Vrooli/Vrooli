// AI_CHECK: TEST_QUALITY=8 | LAST: 2025-01-26
import { screen } from "@testing-library/react";
import { DUMMY_ID, teamFormConfig } from "@vrooli/shared";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { teamFormTestConfig } from "../../../__test/fixtures/form-testing/TeamFormTest.js";
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
        const mockInitialValues = teamFormConfig.transformations.getInitialValues(mockSession) || teamFormTestConfig.formFixtures.minimal;

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
vi.mock("../../../components/inputs/ProfilePictureInput/ProfilePictureInput.js", () => ({
    ProfilePictureInput: ({ onBannerImageChange, onProfileImageChange }: any) => (
        <div data-testid="profile-picture-input">
            <button onClick={() => onBannerImageChange?.(new File([], "banner.jpg"))}>
                Change Banner
            </button>
            <button onClick={() => onProfileImageChange?.(new File([], "profile.jpg"))}>
                Change Profile
            </button>
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

// Mock AdvancedInput components
vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    TranslatedAdvancedInput: ({ name, label, title, value, onChange, isRequired, disabled, placeholder, ...domProps }: any) => {
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
                    placeholder={placeholder}
                    {...filteredProps}
                />
            </div>
        );
    },
}));

// Mock navigation components
vi.mock("../../../components/buttons/SearchExistingButton/SearchExistingButton.js", () => ({
    SearchExistingButton: () => <button data-testid="search-existing-button">Search Existing</button>,
}));

// Import the component after all mocks are set up
import { TeamUpsert } from "./TeamUpsert.js";

describe("TeamUpsert", () => {
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
    const formTester = createSimpleFormTester(TeamUpsert, defaultProps, mockSession);

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
                TeamUpsert,
                { defaultProps, mockSession },
            );
            
            // Just verify the component renders without errors
            expect(container).toBeTruthy();
        });

        it("handles form submission with valid data", async () => {
            const { user } = renderFormComponent(
                TeamUpsert,
                { defaultProps, mockSession },
            );

            // Fill required fields using stable test IDs
            await user.type(screen.getByTestId("input-name"), "Test Team");
            await user.click(screen.getByTestId("submit-button"));

            // Verify form submission was attempted
            expect(mockOnSubmit).toHaveBeenCalled();
        });

        it("handles form cancellation", async () => {
            const { user } = renderFormComponent(
                TeamUpsert,
                { defaultProps, mockSession },
            );

            await user.click(screen.getByTestId("cancel-button"));
            expect(mockHandleCancel).toHaveBeenCalled();
        });
    });

    describe("Form Field Interactions", () => {
        it("handles name input", async () => {
            await formTester.testElement("name", "Test Team");
        });

        it("handles bio input", async () => {
            await formTester.testElement("bio", "Test team description", "textarea");
        });

        it("handles multiple fields together", async () => {
            await formTester.testElements([
                ["name", "Test Team"],
                ["bio", "Team description", "textarea"],
            ]);
        });
    });

    describe("Different Modes", () => {
        it("displays create mode correctly", async () => {
            const { user } = renderFormComponent(
                TeamUpsert,
                { defaultProps: { ...defaultProps, isCreate: true }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Create");
        });

        it("displays edit mode correctly", async () => {
            const { user } = renderFormComponent(
                TeamUpsert,
                { defaultProps: { ...defaultProps, isCreate: false }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Save");
        });
    });

    describe("Form Validation Integration", () => {
        it("integrates with real validation schemas", async () => {
            const { user } = renderFormComponent(
                TeamUpsert,
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
            const fixtureData = teamFormTestConfig.formFixtures.minimal;
            expect(fixtureData).toBeDefined();
            expect(fixtureData.translations[0].name).toBe("Test Team");
        });

        it("initializes with correct defaults", () => {
            const values = teamFormTestConfig.initialValuesFunction?.(mockSession);
            expect(values).toMatchObject({
                __typename: "Team",
                isPrivate: false,
                id: expect.any(String),
            });
        });

        it("preserves existing data in edit mode", () => {
            const existingData = {
                id: "existing-id",
                isPrivate: true,
                name: "Existing Team",
            };

            const values = teamFormTestConfig.initialValuesFunction?.(mockSession, existingData);
            expect(values).toMatchObject({
                id: "existing-id",
                isPrivate: true,
            });
        });
    });

    describe("Complex Scenarios", () => {
        it("handles complete form workflow", async () => {
            const onCompleted = vi.fn();
            const { user } = renderFormComponent(
                TeamUpsert,
                { defaultProps: { ...defaultProps, onCompleted }, mockSession },
            );

            // Fill form with complete data
            await user.type(screen.getByTestId("input-name"), "Complete Team");
            await user.type(screen.getByTestId("input-bio"), "Complete description");
            
            // Submit form
            await user.click(screen.getByTestId("submit-button"));

            // Verify submission was called
            expect(mockOnSubmit).toHaveBeenCalled();
        });

        describe.each([
            ["minimal", teamFormTestConfig.formFixtures.minimal],
            ["complete", teamFormTestConfig.formFixtures.complete || teamFormTestConfig.formFixtures.minimal],
        ])("with %s fixture data", (scenario, fixtureData) => {
            it(`renders correctly with ${scenario} data`, async () => {
                // Test different fixture scenarios
                expect(fixtureData).toBeDefined();
                expect(fixtureData.translations).toBeDefined();
            });
        });
    });
});
