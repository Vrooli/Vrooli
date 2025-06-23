/**
 * Bookmark Form Data Fixtures
 * 
 * This file provides comprehensive form data fixtures for bookmark forms,
 * including React Hook Form integration and user interaction simulation.
 */

import { useForm, type UseFormReturn, type FieldErrors } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import * as yup from "yup";
import { act, waitFor } from "@testing-library/react";
import type { 
    BookmarkFor,
    BookmarkCreateInput,
    BookmarkUpdateInput, 
} from "@vrooli/shared";
import { 
    BookmarkFor as BookmarkForEnum,
    bookmarkValidation, 
} from "@vrooli/shared";
import type { BookmarkFormData } from "../types.js";

/**
 * Extended form state with additional UI-specific properties
 */
export interface BookmarkFormState {
    values: BookmarkFormData;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    isValidating?: boolean;
    
    // UI-specific state
    autoSaveStatus?: "idle" | "saving" | "saved" | "error";
    lastAutoSave?: Date;
    hasUnsavedChanges?: boolean;
    
    // List management state
    listState?: {
        isLoadingLists: boolean;
        availableLists: Array<{ id: string; label: string }>;
        isCreatingNewList: boolean;
        newListValidation?: {
            isChecking: boolean;
            isUnique?: boolean;
        };
    };
}

/**
 * Form validation configuration
 */
interface FormValidationConfig {
    mode: "onChange" | "onBlur" | "onSubmit";
    reValidateMode: "onChange" | "onBlur" | "onSubmit";
    shouldFocusError: boolean;
    delayError?: number;
}

/**
 * User interaction event types
 */
export type FormInteractionEvent = 
    | { type: "FIELD_FOCUS"; field: keyof BookmarkFormData }
    | { type: "FIELD_BLUR"; field: keyof BookmarkFormData }
    | { type: "FIELD_CHANGE"; field: keyof BookmarkFormData; value: any }
    | { type: "FORM_SUBMIT" }
    | { type: "FORM_RESET" }
    | { type: "AUTO_SAVE_TRIGGER" };

/**
 * Bookmark form data factory with React Hook Form integration
 */
export class BookmarkFormDataFactory {
    /**
     * Create validation schema using yup (compatible with React Hook Form)
     */
    private createValidationSchema(): yup.ObjectSchema<BookmarkFormData> {
        return yup.object({
            bookmarkFor: yup
                .mixed<BookmarkFor>()
                .oneOf(Object.values(BookmarkForEnum))
                .required("Bookmark type is required"),
                
            forConnect: yup
                .string()
                .required("Target object is required")
                .min(1, "Target object ID cannot be empty"),
                
            listId: yup
                .string()
                .when("createNewList", {
                    is: (val: boolean) => !val,
                    then: (schema) => schema.required("Please select a list or create a new one"),
                    otherwise: (schema) => schema.optional(),
                }),
                
            createNewList: yup
                .boolean()
                .optional(),
                
            newListLabel: yup
                .string()
                .when("createNewList", {
                    is: true,
                    then: (schema) => schema
                        .required("List name is required")
                        .min(1, "List name cannot be empty")
                        .max(50, "List name cannot exceed 50 characters")
                        .matches(/^[a-zA-Z0-9\s\-_]+$/, "List name contains invalid characters"),
                    otherwise: (schema) => schema.optional(),
                }),
                
            isPrivate: yup
                .boolean()
                .optional(),
        }).defined();
    }
    
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `form_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create form data for different scenarios
     */
    createFormData(scenario: "empty" | "minimal" | "complete" | "invalid" | "withNewList" | "withExistingList" | "partiallyCompleted"): BookmarkFormData {
        switch (scenario) {
            case "empty":
                return {
                    bookmarkFor: BookmarkForEnum.Resource,
                    forConnect: "",
                };
                
            case "minimal":
                return {
                    bookmarkFor: BookmarkForEnum.Resource,
                    forConnect: this.generateId(),
                };
                
            case "complete":
                return {
                    bookmarkFor: BookmarkForEnum.Resource,
                    forConnect: this.generateId(),
                    listId: this.generateId(),
                    isPrivate: false,
                };
                
            case "invalid":
                return {
                    bookmarkFor: "" as any, // Invalid bookmark type
                    forConnect: "", // Missing required field
                    createNewList: true,
                    newListLabel: "", // Missing when createNewList is true
                };
                
            case "withNewList":
                return {
                    bookmarkFor: BookmarkForEnum.User,
                    forConnect: this.generateId(),
                    createNewList: true,
                    newListLabel: "My New Bookmarks",
                    isPrivate: true,
                };
                
            case "withExistingList":
                return {
                    bookmarkFor: BookmarkForEnum.Team,
                    forConnect: this.generateId(),
                    listId: this.generateId(),
                    isPrivate: false,
                };
                
            case "partiallyCompleted":
                return {
                    bookmarkFor: BookmarkForEnum.Comment,
                    forConnect: this.generateId(),
                    createNewList: true,
                    newListLabel: "", // Incomplete - missing list name
                };
                
            default:
                throw new Error(`Unknown form scenario: ${scenario}`);
        }
    }
    
    /**
     * Create form state for different scenarios
     */
    createFormState(scenario: "pristine" | "dirty" | "submitting" | "withErrors" | "valid" | "autoSaving"): BookmarkFormState {
        const baseFormData = this.createFormData("complete");
        
        switch (scenario) {
            case "pristine":
                return {
                    values: this.createFormData("empty"),
                    errors: {},
                    touched: {},
                    isDirty: false,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "dirty":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { forConnect: true, listId: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    hasUnsavedChanges: true,
                };
                
            case "submitting":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { forConnect: true, listId: true },
                    isDirty: true,
                    isSubmitting: true,
                    isValid: true,
                };
                
            case "withErrors":
                return {
                    values: this.createFormData("invalid"),
                    errors: {
                        bookmarkFor: "Bookmark type is required",
                        forConnect: "Target object is required",
                        newListLabel: "List name is required",
                    },
                    touched: { bookmarkFor: true, forConnect: true, newListLabel: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "valid":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { forConnect: true, listId: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "autoSaving":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { forConnect: true, listId: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    autoSaveStatus: "saving",
                    hasUnsavedChanges: true,
                };
                
            default:
                throw new Error(`Unknown form state scenario: ${scenario}`);
        }
    }
    
    /**
     * Create React Hook Form instance with pre-configured data
     */
    createFormInstance(
        initialData?: Partial<BookmarkFormData>,
        config?: Partial<FormValidationConfig>,
    ): UseFormReturn<BookmarkFormData> {
        const defaultValues: BookmarkFormData = {
            bookmarkFor: BookmarkForEnum.Resource,
            forConnect: "",
            ...initialData,
        };
        
        const validationConfig: FormValidationConfig = {
            mode: "onChange",
            reValidateMode: "onChange",
            shouldFocusError: true,
            ...config,
        };
        
        return useForm<BookmarkFormData>({
            mode: validationConfig.mode,
            reValidateMode: validationConfig.reValidateMode,
            shouldFocusError: validationConfig.shouldFocusError,
            defaultValues,
            resolver: yupResolver(this.createValidationSchema()),
        });
    }
    
    /**
     * Validate form data using real validation from @vrooli/shared
     */
    async validateFormData(formData: BookmarkFormData): Promise<{
        isValid: boolean;
        errors?: Record<string, string>;
        apiInput?: BookmarkCreateInput;
    }> {
        try {
            // First validate with our form schema
            await this.createValidationSchema().validate(formData, { abortEarly: false });
            
            // Then validate with the actual API validation
            const apiInput = this.transformToAPIInput(formData);
            await bookmarkValidation.create.validate(apiInput);
            
            return {
                isValid: true,
                apiInput,
            };
        } catch (error: any) {
            const errors: Record<string, string> = {};
            
            if (error.inner) {
                // Yup validation errors
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        errors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                errors.general = error.message;
            }
            
            return {
                isValid: false,
                errors,
            };
        }
    }
    
    /**
     * Transform form data to API input
     */
    private transformToAPIInput(formData: BookmarkFormData): BookmarkCreateInput {
        const apiInput: BookmarkCreateInput = {
            id: this.generateId(),
            bookmarkFor: formData.bookmarkFor,
            forConnect: formData.forConnect,
        };
        
        if (formData.createNewList && formData.newListLabel) {
            apiInput.listCreate = {
                id: this.generateId(),
                label: formData.newListLabel,
            };
        } else if (formData.listId) {
            apiInput.listConnect = formData.listId;
        }
        
        return apiInput;
    }
}

/**
 * Form interaction simulator for testing user behavior
 */
export class BookmarkFormInteractionSimulator {
    private interactionDelay = 100;
    
    constructor(delay?: number) {
        this.interactionDelay = delay || 100;
    }
    
    /**
     * Simulate filling a form field
     */
    async fillField(
        formInstance: UseFormReturn<BookmarkFormData>,
        fieldName: keyof BookmarkFormData,
        value: any,
        options?: { shouldValidate?: boolean; shouldTouch?: boolean },
    ): Promise<void> {
        const { shouldValidate = true, shouldTouch = true } = options || {};
        
        act(() => {
            formInstance.setValue(fieldName, value, {
                shouldDirty: true,
                shouldTouch,
                shouldValidate,
            });
        });
        
        // Wait for validation to complete
        if (shouldValidate) {
            await waitFor(() => {
                expect(formInstance.formState.isValidating).toBe(false);
            });
        }
    }
    
    /**
     * Simulate typing in a text field with realistic delays
     */
    async simulateTyping(
        formInstance: UseFormReturn<BookmarkFormData>,
        fieldName: keyof BookmarkFormData,
        text: string,
        options?: { charDelay?: number; shouldValidateOnChange?: boolean },
    ): Promise<void> {
        const { charDelay = 50, shouldValidateOnChange = true } = options || {};
        
        // Clear field first
        await this.fillField(formInstance, fieldName, "", { shouldValidate: false });
        
        // Type each character
        for (let i = 1; i <= text.length; i++) {
            const partialText = text.substring(0, i);
            await this.fillField(formInstance, fieldName, partialText, { 
                shouldValidate: shouldValidateOnChange && i === text.length, 
            });
            
            if (charDelay > 0) {
                await new Promise(resolve => setTimeout(resolve, charDelay));
            }
        }
    }
    
    /**
     * Simulate field focus and blur events
     */
    async simulateFieldFocus(
        formInstance: UseFormReturn<BookmarkFormData>,
        fieldName: keyof BookmarkFormData,
        duration = 1000,
    ): Promise<void> {
        // Simulate focus
        act(() => {
            formInstance.setFocus(fieldName);
        });
        
        await new Promise(resolve => setTimeout(resolve, duration));
        
        // Simulate blur by triggering validation
        act(() => {
            formInstance.trigger(fieldName);
        });
    }
    
    /**
     * Simulate form submission
     */
    async simulateFormSubmission(
        formInstance: UseFormReturn<BookmarkFormData>,
        onSubmit?: (data: BookmarkFormData) => Promise<void> | void,
        onError?: (errors: FieldErrors<BookmarkFormData>) => void,
    ): Promise<{ success: boolean; data?: BookmarkFormData; errors?: FieldErrors<BookmarkFormData> }> {
        return new Promise((resolve) => {
            const handleSubmit = async (data: BookmarkFormData) => {
                try {
                    if (onSubmit) {
                        await onSubmit(data);
                    }
                    resolve({ success: true, data });
                } catch (error) {
                    resolve({ success: false, errors: { root: { type: "submit", message: "Submission failed" } } });
                }
            };
            
            const handleError = (errors: FieldErrors<BookmarkFormData>) => {
                if (onError) {
                    onError(errors);
                }
                resolve({ success: false, errors });
            };
            
            act(() => {
                formInstance.handleSubmit(handleSubmit, handleError)();
            });
        });
    }
    
    /**
     * Simulate complete form filling workflow
     */
    async simulateCompleteFormFilling(
        formInstance: UseFormReturn<BookmarkFormData>,
        formData: BookmarkFormData,
        options?: {
            withTypingDelay?: boolean;
            withValidationPauses?: boolean;
            withFieldFocus?: boolean;
        },
    ): Promise<void> {
        const { 
            withTypingDelay = true, 
            withValidationPauses = true, 
            withFieldFocus = true, 
        } = options || {};
        
        // Fill bookmark type
        if (withFieldFocus) {
            await this.simulateFieldFocus(formInstance, "bookmarkFor", 200);
        }
        await this.fillField(formInstance, "bookmarkFor", formData.bookmarkFor);
        
        if (withValidationPauses) {
            await new Promise(resolve => setTimeout(resolve, 200));
        }
        
        // Fill target object
        if (withFieldFocus) {
            await this.simulateFieldFocus(formInstance, "forConnect", 300);
        }
        
        if (withTypingDelay && typeof formData.forConnect === "string") {
            await this.simulateTyping(formInstance, "forConnect", formData.forConnect);
        } else {
            await this.fillField(formInstance, "forConnect", formData.forConnect);
        }
        
        if (withValidationPauses) {
            await new Promise(resolve => setTimeout(resolve, 300));
        }
        
        // Handle list selection or creation
        if (formData.createNewList) {
            await this.fillField(formInstance, "createNewList", true);
            
            if (formData.newListLabel) {
                if (withFieldFocus) {
                    await this.simulateFieldFocus(formInstance, "newListLabel", 250);
                }
                
                if (withTypingDelay) {
                    await this.simulateTyping(formInstance, "newListLabel", formData.newListLabel);
                } else {
                    await this.fillField(formInstance, "newListLabel", formData.newListLabel);
                }
            }
        } else if (formData.listId) {
            await this.fillField(formInstance, "listId", formData.listId);
        }
        
        // Set privacy if specified
        if (formData.isPrivate !== undefined) {
            await this.fillField(formInstance, "isPrivate", formData.isPrivate);
        }
        
        // Final validation pause
        if (withValidationPauses) {
            await new Promise(resolve => setTimeout(resolve, 200));
        }
    }
    
    /**
     * Simulate error recovery workflow
     */
    async simulateErrorRecovery(
        formInstance: UseFormReturn<BookmarkFormData>,
        errorField: keyof BookmarkFormData,
        correctValue: any,
    ): Promise<void> {
        // First, ensure the field has an error
        await this.fillField(formInstance, errorField, "");
        
        // Wait for error to appear
        await waitFor(() => {
            expect(formInstance.formState.errors[errorField]).toBeDefined();
        });
        
        // Simulate user noticing error and correcting it
        await new Promise(resolve => setTimeout(resolve, 500));
        
        // Focus on the error field
        await this.simulateFieldFocus(formInstance, errorField, 300);
        
        // Correct the value
        await this.fillField(formInstance, errorField, correctValue);
        
        // Wait for error to clear
        await waitFor(() => {
            expect(formInstance.formState.errors[errorField]).toBeUndefined();
        });
    }
}

/**
 * Pre-configured form scenarios for common testing patterns
 */
export const bookmarkFormScenarios = {
    // Basic scenarios
    emptyForm: () => new BookmarkFormDataFactory().createFormState("pristine"),
    validForm: () => new BookmarkFormDataFactory().createFormState("valid"),
    formWithErrors: () => new BookmarkFormDataFactory().createFormState("withErrors"),
    submittingForm: () => new BookmarkFormDataFactory().createFormState("submitting"),
    
    // Specific data scenarios
    minimalBookmark: () => new BookmarkFormDataFactory().createFormData("minimal"),
    completeBookmark: () => new BookmarkFormDataFactory().createFormData("complete"),
    bookmarkWithNewList: () => new BookmarkFormDataFactory().createFormData("withNewList"),
    bookmarkWithExistingList: () => new BookmarkFormDataFactory().createFormData("withExistingList"),
    invalidBookmark: () => new BookmarkFormDataFactory().createFormData("invalid"),
    
    // Interactive scenarios
    autoSavingForm: () => new BookmarkFormDataFactory().createFormState("autoSaving"),
    partiallyCompletedForm: () => {
        const factory = new BookmarkFormDataFactory();
        return {
            ...factory.createFormState("dirty"),
            values: factory.createFormData("partiallyCompleted"),
        };
    },
    
    // Form interaction workflows
    async completeFormFillWorkflow(formInstance: UseFormReturn<BookmarkFormData>) {
        const simulator = new BookmarkFormInteractionSimulator(50);
        const factory = new BookmarkFormDataFactory();
        const formData = factory.createFormData("complete");
        
        await simulator.simulateCompleteFormFilling(formInstance, formData, {
            withTypingDelay: true,
            withValidationPauses: true,
            withFieldFocus: true,
        });
        
        return formData;
    },
    
    async errorRecoveryWorkflow(formInstance: UseFormReturn<BookmarkFormData>) {
        const simulator = new BookmarkFormInteractionSimulator();
        
        // Simulate error in forConnect field
        await simulator.simulateErrorRecovery(
            formInstance, 
            "forConnect", 
            "resource-123",
        );
        
        // Simulate error in newListLabel field
        await simulator.fillField(formInstance, "createNewList", true);
        await simulator.simulateErrorRecovery(
            formInstance, 
            "newListLabel", 
            "My Corrected List Name",
        );
    },
};

// Export factory instances for easy use
export const bookmarkFormFactory = new BookmarkFormDataFactory();
export const bookmarkFormSimulator = new BookmarkFormInteractionSimulator();
