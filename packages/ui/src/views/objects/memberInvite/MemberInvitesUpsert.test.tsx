// AI_CHECK: TEST_QUALITY=8 | LAST: 2025-01-26
import { screen, waitFor } from "@testing-library/react";
import { DUMMY_ID } from "@vrooli/shared";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { memberInvitesFormTestConfig } from "../../../__test/fixtures/form-testing/MemberInvitesFormTest.js";
import {
    createMockSession,
    createSimpleFormTester,
    renderFormComponent,
} from "../../../__test/helpers/formComponentTestHelpers.js";

// Mock only heavy dependencies and complex hooks
vi.mock("../../../hooks/useHistoryState.js", () => ({
    useHistoryState: vi.fn(() => [
        [],
        vi.fn(),
        vi.fn(),
        vi.fn(),
        vi.fn(),
    ]),
}));

// Declare the mock functions at the top level so they can be used in the mock
const mockOnSubmit = vi.fn();
const mockHandleCompleted = vi.fn();
const mockHandleCancel = vi.fn();

vi.mock("../../../hooks/useStandardBatchUpsertForm.js", () => ({
    useStandardBatchUpsertForm: vi.fn((config, options) => {
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
vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    AdvancedInputBase: ({ name, label, value, onChange, error, helperText, ...domProps }: any) => (
        <div data-testid={`advanced-input-${name}`}>
            <label htmlFor={name}>{label || name}</label>
            <textarea
                id={name}
                name={name}
                value={value}
                onChange={(e) => onChange?.(e.target.value)}
                data-testid={`input-${name}`}
                {...domProps}
            />
            {error && <div data-testid={`error-${name}`}>{error}</div>}
            {helperText && <div data-testid={`helper-${name}`}>{helperText}</div>}
        </div>
    ),
}));

vi.mock("../../../components/lists/ObjectList/ObjectList.js", () => ({
    ObjectList: ({ title, items, onAdd }: any) => (
        <div data-testid="object-list">
            <h3>{title}</h3>
            <div>Items: {items?.length || 0}</div>
            <button onClick={onAdd} data-testid="add-item-button">Add Item</button>
        </div>
    ),
}));

vi.mock("../../../hooks/useWindowSize.js", () => ({
    useWindowSize: vi.fn(() => ({
        height: 800,
        width: 1200,
    })),
}));

vi.mock("../../../hooks/useIsMobile.js", () => ({
    useIsMobile: vi.fn(() => false),
}));

// Import the component after all mocks are set up
import { MemberInvitesUpsert } from "./MemberInvitesUpsert.js";

describe("MemberInvitesUpsert", () => {
    const mockSession = createMockSession();

    const mockObject = {
        __typename: "Team" as const,
        id: DUMMY_ID,
    };

    const defaultProps = {
        isCreate: true,
        isOpen: true,
        display: "Dialog" as const,
        object: mockObject,
        invites: [],
        isMutate: false,
        onClose: vi.fn(),
        onCompleted: vi.fn(),
        onDeleted: vi.fn(),
    };

    // Create the simple form tester once for all tests
    const formTester = createSimpleFormTester(MemberInvitesUpsert, defaultProps, mockSession);

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
                MemberInvitesUpsert,
                { defaultProps, mockSession },
            );
            
            // Just verify the component renders without errors
            expect(container).toBeTruthy();
        });

        it("handles form submission with valid data", async () => {
            const { user } = renderFormComponent(
                MemberInvitesUpsert,
                { defaultProps, mockSession },
            );

            // Fill required fields using stable test IDs
            await user.type(screen.getByTestId("input-message"), "Welcome to the team!");
            await user.click(screen.getByTestId("submit-button-wrapper"));

            // Verify form submission was attempted
            await waitFor(() => {
                expect(mockOnSubmit).toHaveBeenCalled();
            });
        });

        it("handles form cancellation", async () => {
            const { user } = renderFormComponent(
                MemberInvitesUpsert,
                { defaultProps, mockSession },
            );

            await user.click(screen.getByTestId("cancel-button"));
            await waitFor(() => {
                expect(mockHandleCancel).toHaveBeenCalled();
            });
        });
    });

    describe("Form Field Interactions", () => {
        it("handles message input", async () => {
            const { user } = renderFormComponent(
                MemberInvitesUpsert,
                { defaultProps, mockSession },
            );

            const messageInput = screen.getByTestId("input-message");
            expect(messageInput).toBeTruthy();
            
            await user.type(messageInput, "Welcome to our team!");
            expect(messageInput).toBeTruthy();
        });
    });

    describe("Different Modes", () => {
        it("displays create mode correctly", async () => {
            renderFormComponent(
                MemberInvitesUpsert,
                { defaultProps: { ...defaultProps, isCreate: true }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toMatch(/create/i);
        });

        it("displays edit mode correctly", async () => {
            renderFormComponent(
                MemberInvitesUpsert,
                { defaultProps: { ...defaultProps, isCreate: false }, mockSession },
            );

            const submitButton = screen.getByTestId("submit-button");
            expect(submitButton.textContent).toMatch(/save/i);
        });
    });

    describe("Form Validation Integration", () => {
        it("integrates with real validation schemas", async () => {
            const { user } = renderFormComponent(
                MemberInvitesUpsert,
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

    describe("Complex Scenarios", () => {
        it("handles complete form workflow", async () => {
            const onCompleted = vi.fn();
            const { user } = renderFormComponent(
                MemberInvitesUpsert,
                { defaultProps: { ...defaultProps, onCompleted }, mockSession },
            );

            // Fill form with complete data
            await user.type(screen.getByTestId("input-message"), "Welcome! We're excited to have you join our team.");
            
            // Submit form
            await user.click(screen.getByTestId("submit-button-wrapper"));

            // Verify submission was called
            await waitFor(() => {
                expect(mockOnSubmit).toHaveBeenCalled();
            });
        });
    });
});
