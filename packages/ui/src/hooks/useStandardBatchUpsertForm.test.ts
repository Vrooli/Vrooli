// AI_CHECK: TEST_QUALITY=4 | LAST: 2025-06-24
import { act, renderHook } from "@testing-library/react";
import React from "react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { SessionContext } from "../contexts/session.js";
import { useStandardBatchUpsertForm, type StandardBatchUpsertFormConfig, type UseStandardBatchUpsertFormProps } from "./useStandardBatchUpsertForm.js";
import { PubSub } from "../utils/pubsub.js";

// Mock dependencies
vi.mock("../api/fetchWrapper.js", () => ({
    useSubmitHelper: vi.fn(),
}));

vi.mock("./forms.js", () => ({
    useUpsertActions: vi.fn(() => ({
        handleCancel: vi.fn(),
        handleCompleted: vi.fn(),
        handleDeleted: vi.fn(),
    })),
}));

vi.mock("./useUpsertFetch.js", () => ({
    useUpsertFetch: vi.fn(() => ({
        fetch: vi.fn(),
        isCreateLoading: false,
        isUpdateLoading: false,
    })),
}));

vi.mock("../utils/pubsub.js", () => ({
    PubSub: {
        get: () => ({
            publish: vi.fn(),
        }),
    },
}));

// Test types for batch operations
interface TestInviteShape {
    __typename: "TestInvite";
    id: string;
    userEmail: string;
    willBeAdmin: boolean;
    message: string;
}

interface TestInviteCreateInput {
    userEmail: string;
    willBeAdmin: boolean;
    message: string;
}

interface TestInviteUpdateInput {
    id: string;
    userEmail?: string;
    willBeAdmin?: boolean;
    message?: string;
}

interface TestInviteResult {
    __typename: "TestInvite";
    id: string;
    userEmail: string;
    willBeAdmin: boolean;
    message: string;
    createdAt: string;
    updatedAt: string;
}

type TestInviteShapeArray = TestInviteShape[];
type TestInviteCreateInputArray = TestInviteCreateInput[];
type TestInviteUpdateInputArray = TestInviteUpdateInput[];
type TestInviteResultArray = TestInviteResult[];

// Test utilities
const createMockConfig = (): StandardBatchUpsertFormConfig<
    TestInviteShapeArray,
    TestInviteCreateInputArray,
    TestInviteUpdateInputArray,
    TestInviteResultArray
> => ({
    objectType: "TestInvite",
    transformFunction: vi.fn((values: TestInviteShapeArray, existing: TestInviteShapeArray, isCreate: boolean) => {
        if (isCreate) {
            return values.map(item => ({
                userEmail: item.userEmail,
                willBeAdmin: item.willBeAdmin,
                message: item.message,
            }));
        }
        return values.map(item => ({
            id: item.id,
            userEmail: item.userEmail,
            willBeAdmin: item.willBeAdmin,
            message: item.message,
        }));
    }),
    validateFunction: vi.fn(async (values: TestInviteShapeArray, existing: TestInviteShapeArray, isCreate: boolean) => {
        // Mock validation - return empty errors object for valid data
        const errors: Record<string, any> = {};
        values.forEach((item, index) => {
            if (!item.userEmail) {
                errors[`${index}.userEmail`] = "Email is required";
            }
        });
        return errors;
    }),
    endpoints: {
        create: { endpoint: "/api/test/invites", method: "POST" as const },
        update: { endpoint: "/api/test/invites", method: "PUT" as const },
    },
});

const createMockValuesArray = (): TestInviteShapeArray => [
    {
        __typename: "TestInvite",
        id: "invite-1",
        userEmail: "user1@example.com",
        willBeAdmin: true,
        message: "Welcome to the team!",
    },
    {
        __typename: "TestInvite",
        id: "invite-2",
        userEmail: "user2@example.com",
        willBeAdmin: false,
        message: "Join our project!",
    },
];

const createMockProps = (overrides: Partial<UseStandardBatchUpsertFormProps<TestInviteShapeArray, TestInviteResultArray>> = {}): UseStandardBatchUpsertFormProps<TestInviteShapeArray, TestInviteResultArray> => {
    const values = createMockValuesArray();
    return {
        values,
        existing: values,
        isCreate: false,
        display: "Page",
        handleUpdate: vi.fn(),
        setSubmitting: vi.fn(),
        onCompleted: vi.fn(),
        ...overrides,
    };
};

const createMockSession = () => ({
    id: "session-id",
    languages: ["en"],
    theme: "light",
});

// Helper to render hook with providers
const renderHookWithProviders = <T extends TestInviteShapeArray, R extends TestInviteResultArray>(
    config: StandardBatchUpsertFormConfig<T, any, any, R>,
    props: UseStandardBatchUpsertFormProps<T, R>,
    sessionOverride?: any,
) => {
    const session = sessionOverride !== undefined ? sessionOverride : createMockSession();
    
    const wrapper = ({ children }: { children: React.ReactNode }) => 
        React.createElement(SessionContext.Provider, { value: session }, children);

    return renderHook(
        ({ config: hookConfig, props: hookProps }) => useStandardBatchUpsertForm(hookConfig, hookProps), 
        { 
            wrapper,
            initialProps: { config, props }
        }
    );
};

// Import mocked functions
import { useSubmitHelper } from "../api/fetchWrapper.js";
import { useUpsertActions } from "./forms.js";
import { useUpsertFetch } from "./useUpsertFetch.js";

describe("useStandardBatchUpsertForm", () => {
    const mockUseSubmitHelper = vi.mocked(useSubmitHelper);
    const mockUseUpsertActions = vi.mocked(useUpsertActions);
    const mockUseUpsertFetch = vi.mocked(useUpsertFetch);

    beforeEach(() => {
        vi.clearAllMocks();
        
        // Setup default mock returns that maintain the function chain
        mockUseUpsertActions.mockImplementation(({ onCompleted, onCancel, onDeleted }) => ({
            handleCancel: vi.fn(() => onCancel?.()),
            handleCompleted: vi.fn((data) => onCompleted?.(data)),
            handleDeleted: vi.fn((data) => onDeleted?.(data)),
        }));

        mockUseUpsertFetch.mockReturnValue({
            fetch: vi.fn(),
            isCreateLoading: false,
            isUpdateLoading: false,
        });
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("basic functionality", () => {
        it("should initialize with correct default state", () => {
            const config = createMockConfig();
            const props = createMockProps();

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(false);
            expect(result.current.isCreateLoading).toBe(false);
            expect(result.current.isUpdateLoading).toBe(false);
            expect(typeof result.current.onSubmit).toBe("function");
            expect(typeof result.current.validateValues).toBe("function");
            expect(result.current.session).toBeDefined();
            expect(result.current.isMutate).toBe(true);
        });

        it("should initialize for create operation with array data", () => {
            const config = createMockConfig();
            const props = createMockProps({ isCreate: true });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(false);
            expect(typeof result.current.onSubmit).toBe("function");
        });

        it("should handle empty array of values", () => {
            const config = createMockConfig();
            const props = createMockProps({ 
                values: [],
                existing: [],
            });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(false);
        });
    });

    describe("isMutate pattern testing", () => {
        it("should handle isMutate=true (default) correctly", () => {
            const config = createMockConfig();
            const props = createMockProps({ isMutate: true });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isMutate).toBe(true);
            expect(mockUseUpsertFetch).toHaveBeenCalledWith(
                expect.objectContaining({ isMutate: true }),
            );
        });

        it("should handle isMutate=false correctly", () => {
            const config = createMockConfig();
            const props = createMockProps({ isMutate: false });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isMutate).toBe(false);
            expect(result.current.isCreateLoading).toBe(false);
            expect(result.current.isUpdateLoading).toBe(false);
            expect(mockUseUpsertFetch).toHaveBeenCalledWith(
                expect.objectContaining({ isMutate: false }),
            );
        });

        it("should handle non-mutating submission (isMutate=false)", async () => {
            const config = createMockConfig();
            const mockOnCompleted = vi.fn();
            const mockSetSubmitting = vi.fn();
            const props = createMockProps({
                isMutate: false,
                onCompleted: mockOnCompleted,
                setSubmitting: mockSetSubmitting,
            });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                result.current.onSubmit();
            });

            expect(mockOnCompleted).toHaveBeenCalledWith(props.values);
            expect(mockSetSubmitting).toHaveBeenCalledWith(false);
        });
    });

    describe("array handling logic", () => {
        it("should handle multiple items in the array", () => {
            const config = createMockConfig();
            const multipleItems = [
                ...createMockValuesArray(),
                {
                    __typename: "TestInvite" as const,
                    id: "invite-3",
                    userEmail: "user3@example.com",
                    willBeAdmin: false,
                    message: "Another invite",
                },
            ];
            const props = createMockProps({ 
                values: multipleItems,
                existing: multipleItems,
            });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(false);
        });

        it("should call transform function with array data", async () => {
            const config = createMockConfig();
            const mockSetSubmitting = vi.fn();
            const mockSubmitHelper = vi.fn();
            
            // Mock useSubmitHelper to return a function that simulates submission
            mockUseSubmitHelper.mockReturnValue(mockSubmitHelper);
            
            const props = createMockProps({
                isMutate: true, // Transform function only called when mutating
                setSubmitting: mockSetSubmitting,
            });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                result.current.onSubmit();
            });

            expect(config.transformFunction).toHaveBeenCalledWith(
                props.values,
                props.existing,
                false, // isCreate
            );
        });
    });

    describe("batch validation", () => {
        it("should call validateFunction with array data", async () => {
            const config = createMockConfig();
            const props = createMockProps();

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                await result.current.validateValues(props.values);
            });

            expect(config.validateFunction).toHaveBeenCalledWith(
                props.values,
                props.existing,
                false, // isCreate
            );
        });

        it("should handle validation for create operation with arrays", async () => {
            const config = createMockConfig();
            const props = createMockProps({ isCreate: true });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                await result.current.validateValues(props.values);
            });

            expect(config.validateFunction).toHaveBeenCalledWith(
                props.values,
                props.existing,
                true, // isCreate
            );
        });

        it("should handle validation errors for array items", async () => {
            const config = createMockConfig();
            const validationErrors = {
                "0.userEmail": "Email is required",
                "1.message": "Message is required",
            };
            config.validateFunction = vi.fn().mockResolvedValue(validationErrors);

            const props = createMockProps();

            const { result } = renderHookWithProviders(config, props);

            const errors = await act(async () => {
                return await result.current.validateValues(props.values);
            });

            expect(errors).toEqual(validationErrors);
        });
    });

    describe("loading states", () => {
        it("should reflect API loading states when isMutate=true", () => {
            mockUseUpsertFetch.mockReturnValue({
                fetch: vi.fn(),
                isCreateLoading: true,
                isUpdateLoading: false,
            });

            const config = createMockConfig();
            const props = createMockProps({ isMutate: true });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(true);
            expect(result.current.isCreateLoading).toBe(true);
            expect(result.current.isUpdateLoading).toBe(false);
        });

        it("should not show API loading states when isMutate=false", () => {
            mockUseUpsertFetch.mockReturnValue({
                fetch: vi.fn(),
                isCreateLoading: true,
                isUpdateLoading: true,
            });

            const config = createMockConfig();
            const props = createMockProps({ isMutate: false });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isCreateLoading).toBe(false);
            expect(result.current.isUpdateLoading).toBe(false);
        });

        it("should handle isReadLoading state", () => {
            const config = createMockConfig();
            const props = createMockProps({ isReadLoading: true });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(true);
        });

        it("should handle isSubmitting state", () => {
            const config = createMockConfig();
            const props = createMockProps({ isSubmitting: true });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(true);
        });
    });

    describe("batch submission", () => {
        it("should handle successful mutating submission with arrays", async () => {
            const mockSubmitHelper = vi.fn();
            mockUseSubmitHelper.mockReturnValue(mockSubmitHelper);

            const config = createMockConfig();
            const mockSetSubmitting = vi.fn();
            const props = createMockProps({
                isMutate: true,
                setSubmitting: mockSetSubmitting,
            });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                result.current.onSubmit();
            });

            expect(mockUseSubmitHelper).toHaveBeenCalledWith(
                expect.objectContaining({
                    disabled: false,
                    existing: props.existing,
                    inputs: expect.any(Array),
                    isCreate: false,
                    onSuccess: expect.any(Function),
                    onCompleted: expect.any(Function),
                }),
            );
            expect(mockSubmitHelper).toHaveBeenCalled();
        });

        it("should handle disabled form submission", async () => {
            const mockPublish = vi.fn();
            vi.spyOn(PubSub, 'get').mockReturnValue({ publish: mockPublish } as any);

            const config = createMockConfig();
            const props = createMockProps({ disabled: true });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                result.current.onSubmit();
            });

            expect(mockPublish).toHaveBeenCalledWith("snack", {
                messageKey: "Unauthorized",
                severity: "Error",
            });
        });

        it("should handle submission when existing array is missing for update", async () => {
            const mockPublish = vi.fn();
            vi.spyOn(PubSub, 'get').mockReturnValue({ publish: mockPublish } as any);

            const config = createMockConfig();
            const props = createMockProps({
                isCreate: false,
                existing: null as any,
            });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                result.current.onSubmit();
            });

            expect(mockPublish).toHaveBeenCalledWith("snack", {
                messageKey: "CouldNotReadObject",
                severity: "Error",
            });
        });
    });

    describe("array transformation", () => {
        it("should transform array for create operation", async () => {
            const config = createMockConfig();
            const mockSubmitHelper = vi.fn();
            
            // Mock useSubmitHelper to return a function that simulates submission
            mockUseSubmitHelper.mockReturnValue(mockSubmitHelper);
            
            const props = createMockProps({ 
                isCreate: true,
                isMutate: true, // Transform function only called when mutating
            });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                result.current.onSubmit();
            });

            expect(config.transformFunction).toHaveBeenCalledWith(
                props.values,
                props.existing,
                true, // isCreate
            );
        });

        it("should transform array for update operation", async () => {
            const config = createMockConfig();
            const mockSubmitHelper = vi.fn();
            
            // Mock useSubmitHelper to return a function that simulates submission
            mockUseSubmitHelper.mockReturnValue(mockSubmitHelper);
            
            const props = createMockProps({ 
                isCreate: false,
                isMutate: true, // Transform function only called when mutating
            });

            const { result } = renderHookWithProviders(config, props);

            await act(async () => {
                result.current.onSubmit();
            });

            expect(config.transformFunction).toHaveBeenCalledWith(
                props.values,
                props.existing,
                false, // isCreate
            );
        });
    });

    describe("error handling for partial failures", () => {
        it("should handle transform function errors gracefully", async () => {
            const config = createMockConfig();
            config.transformFunction = vi.fn(() => {
                throw new Error("Transform error");
            });

            const props = createMockProps();

            // Should not throw when rendering
            expect(() => {
                renderHookWithProviders(config, props);
            }).not.toThrow();
        });

        it("should handle validation errors for arrays", async () => {
            const config = createMockConfig();
            const validationError = new Error("Validation failed");
            config.validateFunction = vi.fn().mockRejectedValue(validationError);

            const props = createMockProps();

            const { result } = renderHookWithProviders(config, props);

            await expect(
                result.current.validateValues(props.values),
            ).rejects.toThrow("Validation failed");
        });

        it("should handle mixed valid and invalid items in array", async () => {
            const config = createMockConfig();
            const partialErrors = {
                "0.userEmail": "Email is required", // First item has error
                // Second item is valid (no error)
            };
            config.validateFunction = vi.fn().mockResolvedValue(partialErrors);

            const props = createMockProps();

            const { result } = renderHookWithProviders(config, props);

            const errors = await act(async () => {
                return await result.current.validateValues(props.values);
            });

            expect(errors).toEqual(partialErrors);
        });
    });

    describe("form actions", () => {
        it("should provide form action handlers", () => {
            const config = createMockConfig();
            const props = createMockProps();

            const { result } = renderHookWithProviders(config, props);

            expect(typeof result.current.handleCancel).toBe("function");
            expect(typeof result.current.handleCompleted).toBe("function");
            expect(typeof result.current.handleDeleted).toBe("function");
        });

        it("should call useUpsertActions with correct parameters", () => {
            const config = createMockConfig();
            const props = createMockProps({
                display: "Dialog",
                isCreate: true,
                onAction: vi.fn(),
                onCancel: vi.fn(),
                onCompleted: vi.fn(),
                onDeleted: vi.fn(),
                suppressSnack: true,
            });

            renderHookWithProviders(config, props);

            expect(mockUseUpsertActions).toHaveBeenCalledWith({
                display: "Dialog",
                isCreate: true,
                objectType: "TestInvite",
                pathname: expect.any(String),
                onAction: props.onAction,
                onCancel: props.onCancel,
                onCompleted: props.onCompleted,
                onDeleted: props.onDeleted,
                suppressSnack: true,
            });
        });
    });

    describe("edge cases", () => {
        it("should handle missing session gracefully", () => {
            const config = createMockConfig();
            const props = createMockProps();

            const { result } = renderHookWithProviders(config, props, null);

            expect(result.current.session).toBeNull();
        });

        it("should handle empty array of values", () => {
            const config = createMockConfig();
            const props = createMockProps({ 
                values: [],
                existing: [],
            });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(false);
        });

        it("should handle array with single item", () => {
            const config = createMockConfig();
            const singleItem = [createMockValuesArray()[0]];
            const props = createMockProps({ 
                values: singleItem,
                existing: singleItem,
            });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(false);
        });

        it("should handle large arrays", () => {
            const config = createMockConfig();
            const largeArray = Array.from({ length: 100 }, (_, index) => ({
                __typename: "TestInvite" as const,
                id: `invite-${index}`,
                userEmail: `user${index}@example.com`,
                willBeAdmin: index % 5 === 0, // Every 5th user is admin
                message: `Invite message ${index}`,
            }));
            const props = createMockProps({ 
                values: largeArray,
                existing: largeArray,
            });

            const { result } = renderHookWithProviders(config, props);

            expect(result.current.isLoading).toBe(false);
        });
    });

    describe("hook dependencies", () => {
        it("should update when props change", () => {
            const config = createMockConfig();
            const initialProps = createMockProps({ isCreate: false });

            const { result, rerender } = renderHookWithProviders(config, initialProps);

            expect(result.current.isMutate).toBe(true);

            const newValues = [
                {
                    __typename: "TestInvite" as const,
                    id: "new-invite-1",
                    userEmail: "newuser@example.com",
                    willBeAdmin: true,
                    message: "Updated invite",
                },
            ];
            const updatedProps = createMockProps({ 
                values: newValues,
                existing: newValues,
                isCreate: true,
                isMutate: false,
            });

            rerender({ config, props: updatedProps });

            expect(result.current.isMutate).toBe(false);
        });

        it("should maintain referential stability for callbacks", () => {
            const config = createMockConfig();
            const props = createMockProps();

            const { result, rerender } = renderHookWithProviders(config, props);

            const initialOnSubmit = result.current.onSubmit;
            const initialValidateValues = result.current.validateValues;

            // Rerender with exactly the same objects (same references)
            rerender({ config, props });

            // Note: This test may fail if config object contains functions that break referential equality
            // In practice, this is often acceptable as callbacks typically depend on props/config content
            expect(result.current.onSubmit).toBe(initialOnSubmit);
            expect(result.current.validateValues).toBe(initialValidateValues);
        });

        it("should update validation when config changes", async () => {
            const config = createMockConfig();
            const props = createMockProps();

            const { result, rerender } = renderHookWithProviders(config, props);

            // Call validation with original config
            await act(async () => {
                await result.current.validateValues(props.values);
            });

            expect(config.validateFunction).toHaveBeenCalledTimes(1);

            // Update config with new validation function
            const newConfig = createMockConfig();
            newConfig.validateFunction = vi.fn().mockResolvedValue({});

            rerender({ config: newConfig, props });

            // Call validation with new config
            await act(async () => {
                await result.current.validateValues(props.values);
            });

            expect(newConfig.validateFunction).toHaveBeenCalledTimes(1);
        });
    });
});
