/* c8 ignore start */
/**
 * Base implementation for form fixture factories
 * 
 * This class provides a concrete implementation of the FormFixtureFactory interface
 * with integration points for validation schemas and shape functions from @vrooli/shared.
 */

import type { UseFormReturn } from "react-hook-form";
import { generatePK } from "@vrooli/shared";
import type {
    FormFixtureFactory,
    FormValidationResult,
    FormEvent
} from "./types.js";

/**
 * Configuration for creating a form fixture factory
 */
export interface FormFixtureFactoryConfig<TFormData, TAPIInput> {
    // Scenario data
    scenarios: {
        minimal: TFormData;
        complete: TFormData;
        invalid: Partial<TFormData>;
        [key: string]: TFormData | Partial<TFormData>;
    };
    
    // Validation function (from @vrooli/shared validation schemas)
    validate?: (data: TFormData) => Promise<FormValidationResult>;
    
    // Shape transformation function (from @vrooli/shared shape functions)
    shapeToAPI?: (formData: TFormData) => TAPIInput;
    
    // Event generation configuration
    eventConfig?: {
        submitDelay?: number;
        fieldChangeDelay?: number;
        includeBlurEvents?: boolean;
    };
}

/**
 * Base form fixture factory implementation
 * 
 * Provides a type-safe foundation for all form fixtures with integration
 * for validation schemas and shape functions from @vrooli/shared.
 * 
 * @template TFormData - The form data type as it appears in UI state
 * @template TAPIInput - The API input type after shape transformation
 */
export class BaseFormFixtureFactory<
    TFormData extends Record<string, unknown>,
    TAPIInput = unknown
> implements FormFixtureFactory<TFormData> {
    
    protected config: FormFixtureFactoryConfig<TFormData, TAPIInput>;
    
    constructor(config: FormFixtureFactoryConfig<TFormData, TAPIInput>) {
        this.config = config;
    }
    
    /**
     * Create form data for a specific scenario
     */
    createFormData(scenario: "minimal" | "complete" | "invalid" | string): TFormData {
        const data = this.config.scenarios[scenario];
        if (!data) {
            throw new Error(`Unknown scenario: ${scenario}`);
        }
        
        // For invalid scenarios, merge with minimal to ensure we have a base
        if (scenario === "invalid") {
            return {
                ...this.config.scenarios.minimal,
                ...data
            } as TFormData;
        }
        
        return { ...data } as TFormData;
    }
    
    /**
     * Simulate form events that would occur during user interaction
     */
    simulateFormEvents(formData: TFormData): FormEvent[] {
        const events: FormEvent[] = [];
        const { eventConfig = {} } = this.config;
        const {
            fieldChangeDelay = 100,
            includeBlurEvents = true
        } = eventConfig;
        
        let timestamp = Date.now();
        
        // Simulate field changes
        for (const [field, value] of Object.entries(formData)) {
            // Focus event
            events.push({
                type: "focus",
                field,
                timestamp
            });
            timestamp += 50;
            
            // Change event
            events.push({
                type: "change",
                field,
                value,
                timestamp
            });
            timestamp += fieldChangeDelay;
            
            // Blur event (if enabled)
            if (includeBlurEvents) {
                events.push({
                    type: "blur",
                    field,
                    timestamp
                });
                timestamp += 50;
            }
        }
        
        // Submit event
        events.push({
            type: "submit",
            timestamp
        });
        
        return events;
    }
    
    /**
     * Validate form data using the configured validation function
     */
    async validateFormData(formData: TFormData): Promise<FormValidationResult> {
        if (!this.config.validate) {
            // No validation configured, assume valid
            return { isValid: true };
        }
        
        try {
            return await this.config.validate(formData);
        } catch (error) {
            // Validation threw an error, convert to validation result
            return {
                isValid: false,
                errors: {
                    _form: error instanceof Error ? error.message : String(error)
                }
            };
        }
    }
    
    /**
     * Transform form data to API input using configured shape function
     */
    transformToAPIInput(formData: TFormData): TAPIInput {
        if (!this.config.shapeToAPI) {
            throw new Error("No shape transformation configured for this form fixture");
        }
        
        return this.config.shapeToAPI(formData);
    }
    
    /**
     * Create a copy of form data with a unique ID
     */
    createUniqueFormData(scenario: "minimal" | "complete" | string): TFormData {
        const baseData = this.createFormData(scenario);
        
        // Add unique ID if the form data has an id field
        if ("id" in baseData) {
            return {
                ...baseData,
                id: generatePK().toString()
            };
        }
        
        return baseData;
    }
    
    /**
     * Merge form data with overrides
     */
    mergeFormData(base: TFormData, overrides: Partial<TFormData>): TFormData {
        return {
            ...base,
            ...overrides
        };
    }
    
    /**
     * Test helper: Create invalid form data variations
     */
    createInvalidVariations(): Array<{ data: Partial<TFormData>; expectedError: string }> {
        const variations: Array<{ data: Partial<TFormData>; expectedError: string }> = [];
        
        // Start with the configured invalid scenario
        if (this.config.scenarios.invalid) {
            variations.push({
                data: this.config.scenarios.invalid,
                expectedError: "Validation failed"
            });
        }
        
        // Add common invalid patterns
        const minimalData = this.config.scenarios.minimal;
        
        // Empty required fields
        for (const field of Object.keys(minimalData)) {
            variations.push({
                data: {
                    ...minimalData,
                    [field]: ""
                },
                expectedError: `${field} is required`
            });
        }
        
        return variations;
    }
    
    /**
     * Test helper: Validate multiple scenarios
     */
    async validateScenarios(): Promise<Record<string, FormValidationResult>> {
        const results: Record<string, FormValidationResult> = {};
        
        for (const scenario of Object.keys(this.config.scenarios)) {
            const data = this.createFormData(scenario);
            results[scenario] = await this.validateFormData(data);
        }
        
        return results;
    }
}

/**
 * Enhanced form fixture factory with additional utilities
 */
export class EnhancedFormFixtureFactory<
    TFormData extends Record<string, unknown>,
    TAPIInput = unknown
> extends BaseFormFixtureFactory<TFormData, TAPIInput> {
    
    /**
     * Create form data with field-level generation
     */
    createWithFieldOverrides(
        scenario: "minimal" | "complete" | string,
        fieldOverrides: Partial<TFormData>
    ): TFormData {
        const base = this.createFormData(scenario);
        return this.mergeFormData(base, fieldOverrides);
    }
    
    /**
     * Create multiple unique instances
     */
    createMultiple(
        scenario: "minimal" | "complete" | string,
        count: number,
        customizer?: (data: TFormData, index: number) => TFormData
    ): TFormData[] {
        const results: TFormData[] = [];
        
        for (let i = 0; i < count; i++) {
            let data = this.createUniqueFormData(scenario);
            
            if (customizer) {
                data = customizer(data, i);
            }
            
            results.push(data);
        }
        
        return results;
    }
    
    /**
     * Test helper utilities
     */
    testHelpers = {
        expectValid: async (formData: TFormData): Promise<void> => {
            const result = await this.validateFormData(formData);
            if (!result.isValid) {
                throw new Error(`Expected form data to be valid, but got errors: ${JSON.stringify(result.errors)}`);
            }
        },
        
        expectInvalid: async (formData: Partial<TFormData>, expectedField?: string): Promise<void> => {
            const result = await this.validateFormData(formData as TFormData);
            if (result.isValid) {
                throw new Error("Expected form data to be invalid, but validation passed");
            }
            
            if (expectedField && result.errors && !result.errors[expectedField]) {
                throw new Error(`Expected error for field "${expectedField}", but got: ${JSON.stringify(result.errors)}`);
            }
        },
        
        expectTransformSuccess: (formData: TFormData): TAPIInput => {
            try {
                return this.transformToAPIInput(formData);
            } catch (error) {
                throw new Error(`Transform failed: ${error instanceof Error ? error.message : String(error)}`);
            }
        }
    };
}
/* c8 ignore stop */