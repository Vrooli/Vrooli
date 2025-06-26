import { type ListObject, type OrArray, type Session } from "@vrooli/shared";
import { useCallback, useContext, useMemo } from "react";
import { useSubmitHelper } from "../api/fetchWrapper.js";
import { SessionContext } from "../contexts/session.js";
import { useLocation } from "../route/router.js";
import { type FormProps, type ViewDisplayType } from "../types.js";
import { PubSub } from "../utils/pubsub.js";
import { useUpsertActions } from "./forms.js";
import { useUpsertFetch } from "./useUpsertFetch.js";

/**
 * Configuration for the standard batch upsert form hook
 */
export interface StandardBatchUpsertFormConfig<
    TShapeArray extends Array<{ __typename: string; id: string }>,
    TCreateInputArray extends Array<Record<string, any>>,
    TUpdateInputArray extends Array<Record<string, any>>,
    TResultArray extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
> {
    /** Object type for the entities being created/updated */
    objectType: ListObject["__typename"];
    /** Transform function to convert form values array to API input array */
    transformFunction: (values: TShapeArray, existing: TShapeArray, isCreate: boolean) => TCreateInputArray | TUpdateInputArray;
    /** Validation function for the array of values */
    validateFunction: (values: TShapeArray, existing: TShapeArray, isCreate: boolean) => Promise<Record<string, any>>;
    /** Endpoints for create and update operations */
    endpoints: {
        create: { endpoint: string; method: "POST" | "PUT" };
        update: { endpoint: string; method: "POST" | "PUT" };
    };
}

/**
 * Props for the standard batch upsert form hook
 */
export interface UseStandardBatchUpsertFormProps<
    TShapeArray extends Array<{ __typename: string; id: string }>,
    TResultArray extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
> extends Pick<FormProps<TResultArray, object>, "onCancel" | "onCompleted" | "onDeleted"> {
    /** Current form values array */
    values: TShapeArray;
    /** Existing entities array (for updates) */
    existing: TShapeArray;
    /** Whether this is a create operation */
    isCreate: boolean;
    /** Display type (Page or Dialog) */
    display: ViewDisplayType | `${ViewDisplayType}`;
    /** Whether the form is disabled */
    disabled?: boolean;
    /** Whether to mutate (save to database) or just return values */
    isMutate?: boolean;
    /** Whether this is a read-only loading state */
    isReadLoading?: boolean;
    /** Whether form is currently submitting */
    isSubmitting?: boolean;
    /** Function to update form values */
    handleUpdate: (values: Partial<TShapeArray>) => void;
    /** Function to set submitting state */
    setSubmitting: (submitting: boolean) => void;
    /** Additional action handler */
    onAction?: FormProps<TResultArray, object>["onAction"];
    /** Optional callback for close action */
    onClose?: () => void;
    /** Whether to suppress snack notifications */
    suppressSnack?: boolean;
}

/**
 * Standard batch upsert form hook for handling arrays of objects
 * 
 * This hook handles forms that work with arrays of objects instead of single entities,
 * such as multiple invites or bulk operations.
 * 
 * @template TShapeArray - Array of shape interfaces for form values
 * @template TCreateInputArray - Array of API input types for create operations
 * @template TUpdateInputArray - Array of API input types for update operations  
 * @template TResultArray - Array of API result types
 */
export function useStandardBatchUpsertForm<
    TShapeArray extends Array<{ __typename: string; id: string }>,
    TCreateInputArray extends Array<Record<string, any>>,
    TUpdateInputArray extends Array<Record<string, any>>,
    TResultArray extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
>(
    config: StandardBatchUpsertFormConfig<TShapeArray, TCreateInputArray, TUpdateInputArray, TResultArray>,
    props: UseStandardBatchUpsertFormProps<TShapeArray, TResultArray>
) {
    const session = useContext(SessionContext);
    const [location] = useLocation();
    const {
        values,
        existing,
        isCreate,
        display,
        disabled = false,
        isMutate = true,
        isReadLoading = false,
        isSubmitting = false,
        handleUpdate,
        setSubmitting,
        onAction,
        onClose,
        onCancel,
        onCompleted,
        onDeleted,
        suppressSnack = false,
    } = props;
    
    const pathname = location.pathname;

    // Standard form actions (cancel, complete, delete)
    const { handleCancel, handleCompleted, handleDeleted } = useUpsertActions<TResultArray>({
        display,
        isCreate,
        objectType: config.objectType,
        pathname,
        onAction,
        onCancel,
        onCompleted,
        onDeleted,
        suppressSnack,
    });

    // API fetch logic for create/update operations (only used if isMutate is true)
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<TResultArray, TCreateInputArray, TUpdateInputArray>({
        isCreate,
        isMutate,
        endpointCreate: config.endpoints.create,
        endpointUpdate: config.endpoints.update,
    });

    // Form validation using the provided validation function
    const validateValues = useCallback(async (values: TShapeArray) => {
        return config.validateFunction(values, existing, isCreate);
    }, [existing, isCreate, config]);

    // Determine if form is in a loading state
    const isLoading = useMemo(() => 
        (isMutate ? (isCreateLoading || isUpdateLoading) : false) || isReadLoading || isSubmitting,
        [isMutate, isCreateLoading, isUpdateLoading, isReadLoading, isSubmitting]
    );

    // Submit handler that supports both mutating and non-mutating forms
    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        
        if (isMutate) {
            // Standard mutation flow
            const submitHelper = useSubmitHelper<TCreateInputArray | TUpdateInputArray, TResultArray>({
                disabled,
                existing,
                fetch,
                inputs: config.transformFunction(values, existing, isCreate),
                isCreate,
                onSuccess: (data) => {
                    handleCompleted(data);
                },
                onCompleted: () => {
                    setSubmitting(false);
                },
            });
            submitHelper();
        } else {
            // Non-mutating flow - just return the values
            handleCompleted(values as TResultArray);
            setSubmitting(false);
        }
    }, [
        disabled,
        isCreate,
        existing,
        isMutate,
        fetch,
        values,
        config,
        handleCompleted,
        setSubmitting,
    ]);

    return {
        // Form state
        isLoading,
        isCreateLoading: isMutate ? isCreateLoading : false,
        isUpdateLoading: isMutate ? isUpdateLoading : false,
        
        // Form actions
        handleCancel,
        handleCompleted,
        handleDeleted,
        onSubmit,
        
        // Validation
        validateValues,
        
        // Utilities
        session,
        isMutate,
    };
}