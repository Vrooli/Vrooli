import { type ListObject, type OrArray, type Session, type YupModel } from "@vrooli/shared";
import { useCallback, useContext, useMemo } from "react";
import { useSubmitHelper } from "../api/fetchWrapper.js";
import { SessionContext } from "../contexts/session.js";
import { useLocation } from "../route/router.js";
import { type FormProps, type ViewDisplayType } from "../types.js";
import { getUserLanguages } from "../utils/display/translationTools.js";
import { PubSub } from "../utils/pubsub.js";
import { validateFormValues } from "../utils/validateFormValues.js";
import { useSaveToCache, useUpsertActions } from "./forms.js";
import { useTranslatedFields } from "./useTranslatedFields.js";
import { useUpsertFetch } from "./useUpsertFetch.js";

/**
 * Configuration for the standard upsert form hook
 */
export interface StandardUpsertFormConfig<
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
    TResult extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
> {
    /** Object type for the entity being created/updated */
    objectType: ListObject["__typename"];
    /** Root object type (for versioned objects) */
    rootObjectType?: ListObject["__typename"];
    /** Validation schemas for create and update operations */
    validation: YupModel<["create", "update"]>;
    /** Translation validation schema (optional) */
    translationValidation?: YupModel<["create", "update"]>;
    /** Transform function to convert form values to API input */
    transformFunction: (values: TShape, existing: TShape, isCreate: boolean) => TCreateInput | TUpdateInput;
    /** Endpoints for create and update operations - same structure as FormConfig */
    endpoints?: {
        findOne?: { endpoint: string; method: string };
        findMany?: { endpoint: string; method: string };
        createOne?: { endpoint: string; method: string };
        createMany?: { endpoint: string; method: string };
        updateOne?: { endpoint: string; method: string };
        updateMany?: { endpoint: string; method: string };
        deleteOne?: { endpoint: string; method: string };
        deleteMany?: { endpoint: string; method: string };
        [key: string]: { endpoint: string; method: string } | undefined;
    };
    /** Default language provider */
    getDefaultLanguage?: (session?: Session) => string;
}

/**
 * Props for the standard upsert form hook
 */
export interface UseStandardUpsertFormProps<
    TShape extends { __typename: string; id: string },
    TResult extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
> extends Pick<FormProps<TResult, object>, "onCancel" | "onCompleted" | "onDeleted"> {
    /** Current form values */
    values: TShape;
    /** Existing entity (for updates) */
    existing: TShape;
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
    handleUpdate: (values: Partial<TShape>) => void;
    /** Function to set submitting state */
    setSubmitting: (submitting: boolean) => void;
    /** Additional action handler */
    onAction?: FormProps<TResult, object>["onAction"];
    /** Optional callback for close action */
    onClose?: () => void;
    /** Whether to suppress snack notifications */
    suppressSnack?: boolean;
}

/**
 * Standard upsert form hook that combines common form logic
 * 
 * This hook extracts the repeated patterns from form components:
 * - Form validation using real schemas
 * - API fetch logic for create/update
 * - Caching with automatic save/clear
 * - Translation field management
 * - Standard navigation and notifications
 * - Submit handling with error management
 * 
 * @template TShape - The shape interface for form values
 * @template TCreateInput - API input type for create operations
 * @template TUpdateInput - API input type for update operations  
 * @template TResult - API result type
 */
export function useStandardUpsertForm<
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
    TResult extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
>(
    config: StandardUpsertFormConfig<TShape, TCreateInput, TUpdateInput, TResult>,
    props: UseStandardUpsertFormProps<TShape, TResult>,
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

    // Get the object ID and root object ID for navigation and caching
    const objectId = values.id;
    const rootObjectId = config.rootObjectType && "root" in values && values.root && typeof values.root === "object" && "id" in values.root 
        ? (values.root as { id: string }).id 
        : undefined;

    // Standard form actions (cancel, complete, delete)
    const { handleCancel, handleCompleted, handleDeleted } = useUpsertActions<TResult>({
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
    } = useUpsertFetch<TResult, TCreateInput, TUpdateInput>({
        isCreate,
        isMutate,
        endpointCreate: config.endpoints?.createOne,
        endpointUpdate: config.endpoints?.updateOne,
    });

    // Automatic form caching (only if isMutate is true)
    if (isMutate) {
        useSaveToCache({
            isCreate,
            pathname,
            values,
        });
    }

    // Translation field management (if translation validation is provided)
    const translationFields = useTranslatedFields({
        defaultLanguage: config.getDefaultLanguage?.(session) ?? getUserLanguages(session)[0] ?? "en",
        validationSchema: config.translationValidation?.create({ env: process.env.NODE_ENV }),
    });

    // Form validation using real schemas
    const validateValues = useCallback(async (values: TShape) => {
        return validateFormValues(
            values,
            existing,
            isCreate,
            config.transformFunction,
            config.validation,
        );
    }, [existing, isCreate, config.transformFunction, config.validation]);

    // Determine if form is in a loading state
    const isLoading = useMemo(() => 
        (isMutate ? (isCreateLoading || isUpdateLoading) : false) || isReadLoading || isSubmitting,
        [isMutate, isCreateLoading, isUpdateLoading, isReadLoading, isSubmitting],
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
            const submitHelper = useSubmitHelper<TCreateInput | TUpdateInput, TResult>({
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
            handleCompleted({
                ...values,
                createdAt: (existing as any)?.createdAt ?? new Date().toISOString(),
                updatedAt: (existing as any)?.updatedAt ?? new Date().toISOString(),
            } as TResult);
            setSubmitting(false);
        }
    }, [
        disabled,
        isCreate,
        existing,
        isMutate,
        fetch,
        values,
        config.transformFunction,
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
        
        // Translation support (only if translation validation provided)
        ...(config.translationValidation ? {
            translationFields,
            language: translationFields.language,
            languages: translationFields.languages,
            handleAddLanguage: translationFields.handleAddLanguage,
            handleDeleteLanguage: translationFields.handleDeleteLanguage,
            setLanguage: translationFields.setLanguage,
            translationErrors: translationFields.translationErrors,
        } : {}),
        
        // Utilities
        session,
        objectId,
        rootObjectId,
        isMutate,
    };
}
