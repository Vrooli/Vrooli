/* c8 ignore start */
/**
 * Integration utilities for connecting UI fixtures with @vrooli/shared fixtures
 * 
 * This module provides helpers to leverage existing fixtures from the shared package
 * while maintaining type safety and avoiding duplication.
 */

import type { 
    ValidationResult as SharedValidationResult 
} from "@vrooli/shared/__test/fixtures/api/types.js";
import type { FormValidationResult } from "../types.js";

/**
 * Convert shared validation result to UI validation result
 */
export function convertValidationResult(
    sharedResult: SharedValidationResult
): FormValidationResult {
    const errors: Record<string, string> = {};
    
    if (!sharedResult.isValid && sharedResult.errors) {
        // Convert array of errors to field-based errors
        for (const error of sharedResult.errors) {
            // Try to extract field name from error message
            const fieldMatch = error.match(/^(\w+):/);
            if (fieldMatch) {
                const field = fieldMatch[1];
                errors[field] = error.substring(field.length + 2).trim();
            } else {
                // Generic form error
                errors._form = error;
            }
        }
    }
    
    return {
        isValid: sharedResult.isValid,
        errors: Object.keys(errors).length > 0 ? errors : undefined
    };
}

/**
 * Create a validation adapter that uses shared validation schemas
 */
export function createValidationAdapter<TFormData>(
    sharedValidate: (input: unknown) => Promise<SharedValidationResult>
): (data: TFormData) => Promise<FormValidationResult> {
    return async (data: TFormData) => {
        try {
            const result = await sharedValidate(data);
            return convertValidationResult(result);
        } catch (error) {
            return {
                isValid: false,
                errors: {
                    _form: error instanceof Error ? error.message : String(error)
                }
            };
        }
    };
}

/**
 * Helper to extract form data shape from API fixtures
 * 
 * This allows us to use the shared fixtures as a base and only
 * define UI-specific fields in our form fixtures.
 */
export function extractFormDataFromAPIFixture<TAPIInput, TFormData>(
    apiFixture: TAPIInput,
    transformer: (api: TAPIInput) => TFormData
): TFormData {
    return transformer(apiFixture);
}

/**
 * Create a form data factory that extends shared fixtures
 */
export function createFormDataFactory<TFormData, TAPIInput>(config: {
    baseFixture: TAPIInput;
    uiOnlyFields: Partial<TFormData>;
    transformer: (merged: any) => TFormData;
}): TFormData {
    const merged = {
        ...config.baseFixture,
        ...config.uiOnlyFields
    };
    
    return config.transformer(merged);
}

/**
 * Type guard to check if a value is a valid form data object
 */
export function isValidFormData<T extends Record<string, unknown>>(
    value: unknown,
    requiredFields: Array<keyof T>
): value is T {
    if (!value || typeof value !== "object") {
        return false;
    }
    
    const obj = value as Record<string, unknown>;
    
    return requiredFields.every(field => 
        field in obj && obj[field as string] !== undefined
    );
}

/**
 * Helper to generate form events from field changes
 */
export function generateFormEvents(
    fields: Array<{ name: string; value: unknown; delay?: number }>
): Array<{ type: string; field: string; value: unknown; timestamp: number }> {
    const events: Array<{ type: string; field: string; value: unknown; timestamp: number }> = [];
    let timestamp = Date.now();
    
    for (const field of fields) {
        // Focus event
        events.push({
            type: "focus",
            field: field.name,
            value: undefined,
            timestamp
        });
        timestamp += 50;
        
        // Change event
        events.push({
            type: "change",
            field: field.name,
            value: field.value,
            timestamp
        });
        timestamp += field.delay || 100;
        
        // Blur event
        events.push({
            type: "blur",
            field: field.name,
            value: field.value,
            timestamp
        });
        timestamp += 50;
    }
    
    return events;
}

/**
 * Create MSW response wrapper for consistent API responses
 */
export function createAPIResponse<T>(
    data: T,
    options?: {
        success?: boolean;
        message?: string;
        errors?: string[];
        pageInfo?: {
            hasMore: boolean;
            totalCount: number;
            cursor?: string;
        };
    }
): Record<string, unknown> {
    return {
        success: options?.success ?? true,
        data,
        ...(options?.message && { message: options.message }),
        ...(options?.errors && { errors: options.errors }),
        ...(options?.pageInfo && { pageInfo: options.pageInfo })
    };
}

/**
 * Helper to merge form data with defaults
 */
export function mergeWithDefaults<T extends Record<string, unknown>>(
    data: Partial<T>,
    defaults: T
): T {
    const result = { ...defaults };
    
    for (const [key, value] of Object.entries(data)) {
        if (value !== undefined) {
            result[key as keyof T] = value as T[keyof T];
        }
    }
    
    return result;
}

/**
 * Extract error messages from various error formats
 */
export function extractErrorMessages(error: unknown): string[] {
    if (!error) return [];
    
    // Handle string errors
    if (typeof error === "string") {
        return [error];
    }
    
    // Handle Error objects
    if (error instanceof Error) {
        return [error.message];
    }
    
    // Handle objects with message property
    if (typeof error === "object" && "message" in error) {
        return [String(error.message)];
    }
    
    // Handle objects with errors array
    if (typeof error === "object" && "errors" in error && Array.isArray(error.errors)) {
        return error.errors.map(e => 
            typeof e === "string" ? e : 
            e.message ? String(e.message) : 
            String(e)
        );
    }
    
    // Handle validation error format
    if (typeof error === "object" && "details" in error) {
        const details = error.details as Record<string, unknown>;
        if ("fields" in details && typeof details.fields === "object") {
            return Object.entries(details.fields as Record<string, string>)
                .map(([field, message]) => `${field}: ${message}`);
        }
    }
    
    return [String(error)];
}

/**
 * Create a delay promise for simulating async operations
 */
export function delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Batch multiple async operations with error handling
 */
export async function batchAsync<T>(
    operations: Array<() => Promise<T>>,
    options?: {
        continueOnError?: boolean;
        maxConcurrent?: number;
    }
): Promise<Array<{ success: boolean; result?: T; error?: Error }>> {
    const results: Array<{ success: boolean; result?: T; error?: Error }> = [];
    const { continueOnError = false, maxConcurrent = 5 } = options || {};
    
    // Process in batches
    for (let i = 0; i < operations.length; i += maxConcurrent) {
        const batch = operations.slice(i, i + maxConcurrent);
        const batchResults = await Promise.allSettled(
            batch.map(op => op())
        );
        
        for (const result of batchResults) {
            if (result.status === "fulfilled") {
                results.push({ success: true, result: result.value });
            } else {
                results.push({ success: false, error: result.reason });
                if (!continueOnError) {
                    throw result.reason;
                }
            }
        }
    }
    
    return results;
}
/* c8 ignore stop */