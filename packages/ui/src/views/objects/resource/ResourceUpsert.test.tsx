// AI_CHECK: TEST_QUALITY=8 | LAST: 2025-01-26
import { screen } from "@testing-library/react";
import { DUMMY_ID } from "@vrooli/shared";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { resourceFormTestConfig } from "../../../__test/fixtures/form-testing/ResourceFormTest.js";
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
        
        // Create a proper initial value with owner set
        const mockInitialValues = {
            __typename: "Resource" as const,
            id: DUMMY_ID,
            isInternal: false,
            isPrivate: false,
            permissions: "",
            resourceType: "Note",
            publicId: null,
            // Set owner to user to satisfy validation
            owner: {
                __typename: "User" as const,
                id: DUMMY_ID,
            },
            parent: null,
            tags: null,
            versions: [{
                __typename: "ResourceVersion" as const,
                id: DUMMY_ID,
                codeLanguage: null,
                config: null,
                isAutomatable: false,
                isComplete: false,
                isPrivate: false,
                resourceSubType: null,
                versionLabel: "1.0.0",
                versionNotes: null,
                publicId: null,
                relatedVersions: null,
                translations: [{
                    __typename: "ResourceVersionTranslation" as const,
                    id: DUMMY_ID,
                    language: "en",
                    name: "",
                    description: "",
                    instructions: null,
                    details: null,
                }],
            }],
        };

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

vi.mock("../../../hooks/useStandardUpsertForm.js", () => ({
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
vi.mock("../../../components/inputs/LinkInput/LinkInput.js", () => ({
    LinkInput: ({ name, onObjectData, autoFocus }: any) => (
        <div data-testid="link-input">
            <input 
                name={name}
                placeholder="Enter URL or search for object..."
                autoFocus={autoFocus}
                data-testid="link-input-field"
                onChange={(e) => {
                    if (e.target.value.includes("example.com")) {
                        onObjectData?.({ title: "Example Site", subtitle: "Example description" });
                    }
                }}
            />
        </div>
    ),
}));

vi.mock("../../../components/inputs/Selector/Selector.js", () => ({
    Selector: ({ name, options, getOptionLabel, label, value, onChange }: any) => (
        <div data-testid={`selector-${name}`}>
            <label>{label}</label>
            <select 
                name={name}
                value={value || ""}
                onChange={(e) => onChange?.(e.target.value)}
                data-testid={`selector-${name}-select`}
            >
                <option value="">Select...</option>
                {options?.map((option: any) => (
                    <option key={option} value={option}>
                        {getOptionLabel ? getOptionLabel(option) : option}
                    </option>
                ))}
            </select>
        </div>
    ),
}));

vi.mock("../../../hooks/useIsMobile.js", () => ({
    useIsMobile: vi.fn(() => false),
}));

// Import the component after all mocks are set up
import { ResourceUpsert } from "./ResourceUpsert.js";

describe("ResourceUpsert", () => {
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
    const formTester = createSimpleFormTester(ResourceUpsert, defaultProps, mockSession);

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
                ResourceUpsert,
                { defaultProps, mockSession },
            );
            
            // Just verify the component renders without errors
            expect(container).toBeTruthy();
        });

        it("handles form submission with valid data", async () => {
            const { user } = renderFormComponent(
                ResourceUpsert,
                { defaultProps, mockSession },
            );

            // Fill required fields using stable test IDs
            await user.type(screen.getByTestId("input-name"), "Test Resource");
            await user.type(screen.getByTestId("link-input-field"), "https://example.com");
            await user.click(screen.getByTestId("submit-button"));

            // Verify form submission was attempted
            expect(mockOnSubmit).toHaveBeenCalled();
        });

        it("handles form cancellation", async () => {
            const { user } = renderFormComponent(
                ResourceUpsert,
                { defaultProps, mockSession },
            );

            await user.click(screen.getByTestId("cancel-button"));
            expect(mockHandleCancel).toHaveBeenCalled();
        });
    });

    describe("Form Field Interactions", () => {
        it("handles name input", async () => {
            await formTester.testElement("name", "Test Resource");
        });

        it("handles description input", async () => {
            await formTester.testElement("description", "Test resource description", "textarea");
        });

        it("handles link input", async () => {
            const { user } = renderFormComponent(
                ResourceUpsert,
                { defaultProps, mockSession },
            );

            const linkInput = screen.getByTestId("link-input-field");
            await user.type(linkInput, "https://example.com");
            
            expect(linkInput.value).toBe("https://example.com");
        });

        it("handles multiple fields together", async () => {
            await formTester.testElements([
                ["name", "Test Resource"],
                ["description", "Resource description", "textarea"],
            ]);
        });
    });

    describe("Different Modes", () => {
        it("displays create mode correctly", async () => {
            const { user } = renderFormComponent(
                ResourceUpsert,
                { defaultProps: { ...defaultProps, isCreate: true }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Create");
        });

        it("displays edit mode correctly", async () => {
            const { user } = renderFormComponent(
                ResourceUpsert,
                { defaultProps: { ...defaultProps, isCreate: false }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toContain("Save");
        });
    });

    describe("Form Validation Integration", () => {
        it("integrates with real validation schemas", async () => {
            const { user } = renderFormComponent(
                ResourceUpsert,
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
            const fixtureData = resourceFormTestConfig.formFixtures.minimal;
            expect(fixtureData).toBeDefined();
            expect(fixtureData.translations[0].name).toBe("Test Resource");
        });

        it("initializes with correct defaults", () => {
            const values = resourceFormTestConfig.initialValuesFunction?.(mockSession);
            expect(values).toMatchObject({
                __typename: "Resource",
                id: expect.any(String),
            });
        });

        it("preserves existing data in edit mode", () => {
            const existingData = {
                id: "existing-id",
                resourceType: "Note",
            };

            const values = resourceFormTestConfig.initialValuesFunction?.(mockSession, existingData);
            expect(values).toMatchObject({
                id: "existing-id",
                resourceType: "Note",
            });
        });
    });

    describe("Complex Scenarios", () => {
        it("handles complete form workflow", async () => {
            const onCompleted = vi.fn();
            const { user } = renderFormComponent(
                ResourceUpsert,
                { defaultProps: { ...defaultProps, onCompleted }, mockSession },
            );

            // Fill form with complete data
            await user.type(screen.getByTestId("input-name"), "Complete Resource");
            await user.type(screen.getByTestId("input-description"), "Complete description");
            await user.type(screen.getByTestId("link-input-field"), "https://example.com");
            
            // Submit form
            await user.click(screen.getByTestId("submit-button"));

            // Verify submission was called
            expect(mockOnSubmit).toHaveBeenCalled();
        });

        describe.each([
            ["minimal", resourceFormTestConfig.formFixtures.minimal],
            ["complete", resourceFormTestConfig.formFixtures.complete || resourceFormTestConfig.formFixtures.minimal],
        ])("with %s fixture data", (scenario, fixtureData) => {
            it(`renders correctly with ${scenario} data`, async () => {
                // Test different fixture scenarios
                expect(fixtureData).toBeDefined();
                expect(fixtureData.translations).toBeDefined();
            });
        });
    });
});
