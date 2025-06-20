/* c8 ignore start */
/**
 * Base implementation for round-trip testing orchestrators
 * 
 * This class coordinates the complete data flow from UI forms through API calls
 * to database storage and back, ensuring data integrity at every step.
 */

import type {
    RoundTripOrchestrator,
    RoundTripConfig,
    RoundTripResult,
    TestResult,
    DataIntegrityReport,
    TestAPIClient,
    DatabaseVerifier,
    FormFixtureFactory
} from "./types.js";

/**
 * Configuration for creating a round-trip orchestrator
 */
export interface RoundTripOrchestratorConfig<TFormData, TAPIResponse, TAPIInput> {
    // API client for making real requests
    apiClient: TestAPIClient;
    
    // Database verifier for checking persistence
    dbVerifier: DatabaseVerifier;
    
    // Form fixture factory for data transformation
    formFixture: FormFixtureFactory<TFormData>;
    
    // API endpoints
    endpoints: {
        create: string;
        update: string;
        delete: string;
        find: string;
    };
    
    // Table name for database verification
    tableName: string;
    
    // Optional shape functions if not in form fixture
    shapeToAPI?: (formData: TFormData) => TAPIInput;
    shapeFromAPI?: (apiResponse: TAPIResponse) => unknown;
    
    // Field mapping for integrity checking
    fieldMappings?: Record<string, string>;
}

/**
 * Base round-trip orchestrator implementation
 * 
 * Provides comprehensive testing of data flow through all application layers
 * with real API calls and database verification.
 * 
 * @template TFormData - The form data type
 * @template TAPIResponse - The API response type
 * @template TAPIInput - The API input type
 */
export class BaseRoundTripOrchestrator<
    TFormData extends Record<string, unknown>,
    TAPIResponse extends { id: string },
    TAPIInput = unknown
> implements RoundTripOrchestrator<TFormData, TAPIResponse> {
    
    protected config: RoundTripOrchestratorConfig<TFormData, TAPIResponse, TAPIInput>;
    
    constructor(config: RoundTripOrchestratorConfig<TFormData, TAPIResponse, TAPIInput>) {
        this.config = config;
    }
    
    /**
     * Execute complete round-trip test cycle
     */
    async executeFullCycle(config: RoundTripConfig<TFormData>): Promise<RoundTripResult<TAPIResponse>> {
        const stages: Record<string, "pending" | "completed" | "failed" | "skipped"> = {
            validation: "pending",
            transformation: "pending",
            apiCall: "pending",
            dbVerification: "pending",
            dataRetrieval: "pending",
            integrityCheck: "pending"
        };
        
        try {
            // Stage 1: Validate form data
            if (config.validateEachStep) {
                const validation = await this.config.formFixture.validateFormData(config.formData);
                if (!validation.isValid) {
                    stages.validation = "failed";
                    return {
                        success: false,
                        errors: Object.values(validation.errors || {}),
                        stages
                    };
                }
            }
            stages.validation = "completed";
            
            // Stage 2: Transform to API input
            let apiInput: TAPIInput;
            try {
                apiInput = this.getAPIInput(config.formData);
                stages.transformation = "completed";
            } catch (error) {
                stages.transformation = "failed";
                return {
                    success: false,
                    errors: [`Transformation failed: ${error instanceof Error ? error.message : String(error)}`],
                    stages
                };
            }
            
            // Stage 3: Make API call
            let apiResponse: TAPIResponse;
            try {
                const response = await this.config.apiClient.post<TAPIResponse>(
                    this.config.endpoints.create,
                    apiInput
                );
                apiResponse = response.data;
                stages.apiCall = "completed";
            } catch (error) {
                stages.apiCall = "failed";
                return {
                    success: false,
                    errors: [`API call failed: ${error instanceof Error ? error.message : String(error)}`],
                    stages
                };
            }
            
            // Stage 4: Verify database state
            const dbRecord = await this.config.dbVerifier.verifyCreated<TAPIResponse>(
                this.config.tableName,
                apiResponse.id
            );
            if (!dbRecord) {
                stages.dbVerification = "failed";
                return {
                    success: false,
                    errors: ["Database record not found"],
                    stages
                };
            }
            stages.dbVerification = "completed";
            
            // Stage 5: Retrieve via API
            let fetchedData: TAPIResponse;
            try {
                const fetchResponse = await this.config.apiClient.get<TAPIResponse>(
                    `${this.config.endpoints.find}/${apiResponse.id}`
                );
                fetchedData = fetchResponse.data;
                stages.dataRetrieval = "completed";
            } catch (error) {
                stages.dataRetrieval = "failed";
                return {
                    success: false,
                    errors: [`Data retrieval failed: ${error instanceof Error ? error.message : String(error)}`],
                    stages
                };
            }
            
            // Stage 6: Verify data integrity
            const integrity = this.verifyDataIntegrity(config.formData, fetchedData);
            stages.integrityCheck = integrity.isValid ? "completed" : "failed";
            
            // Calculate UI display data
            const uiDisplay = this.config.shapeFromAPI 
                ? this.config.shapeFromAPI(fetchedData)
                : fetchedData;
            
            return {
                success: integrity.isValid,
                data: {
                    formData: config.formData,
                    apiInput,
                    apiResponse,
                    dbRecord,
                    fetchedData,
                    uiDisplay
                },
                stages,
                dataIntegrity: integrity.isValid,
                canDisplay: true,
                errors: integrity.isValid ? undefined : integrity.warnings
            };
            
        } catch (error) {
            return {
                success: false,
                errors: [`Unexpected error: ${error instanceof Error ? error.message : String(error)}`],
                stages
            };
        }
    }
    
    /**
     * Test create flow
     */
    async testCreateFlow(formData: TFormData): Promise<TestResult> {
        const result = await this.executeFullCycle({
            formData,
            validateEachStep: true
        });
        
        return {
            success: result.success,
            errors: result.errors,
            metadata: {
                id: result.data?.apiResponse?.id,
                stages: result.stages
            }
        };
    }
    
    /**
     * Test update flow
     */
    async testUpdateFlow(id: string, formData: Partial<TFormData>): Promise<TestResult> {
        try {
            // Get current data
            const currentResponse = await this.config.apiClient.get<TAPIResponse>(
                `${this.config.endpoints.find}/${id}`
            );
            
            // Merge with updates
            const updateInput = {
                id,
                ...this.getAPIInput(formData as TFormData)
            };
            
            // Make update call
            const updateResponse = await this.config.apiClient.put<TAPIResponse>(
                this.config.endpoints.update,
                updateInput
            );
            
            // Verify database was updated
            const wasUpdated = await this.config.dbVerifier.verifyUpdated(
                this.config.tableName,
                id,
                updateInput
            );
            
            return {
                success: wasUpdated,
                metadata: {
                    id,
                    previousData: currentResponse.data,
                    updatedData: updateResponse.data
                }
            };
            
        } catch (error) {
            return {
                success: false,
                errors: [`Update failed: ${error instanceof Error ? error.message : String(error)}`]
            };
        }
    }
    
    /**
     * Test delete flow
     */
    async testDeleteFlow(id: string): Promise<TestResult> {
        try {
            // Verify exists before delete
            const exists = await this.config.dbVerifier.verifyCreated(
                this.config.tableName,
                id
            );
            
            if (!exists) {
                return {
                    success: false,
                    errors: ["Record does not exist"]
                };
            }
            
            // Delete via API
            await this.config.apiClient.delete(
                `${this.config.endpoints.delete}/${id}`
            );
            
            // Verify deleted from database
            const wasDeleted = await this.config.dbVerifier.verifyDeleted(
                this.config.tableName,
                id
            );
            
            return {
                success: wasDeleted,
                metadata: { id }
            };
            
        } catch (error) {
            return {
                success: false,
                errors: [`Delete failed: ${error instanceof Error ? error.message : String(error)}`]
            };
        }
    }
    
    /**
     * Verify data integrity between original form data and final result
     */
    verifyDataIntegrity(original: TFormData, result: TAPIResponse): DataIntegrityReport {
        const mismatches: Array<{
            field: string;
            original: unknown;
            result: unknown;
            path: string;
        }> = [];
        const warnings: string[] = [];
        
        // Get field mappings
        const mappings = this.config.fieldMappings || this.getDefaultFieldMappings();
        
        // Check each mapped field
        for (const [formField, apiField] of Object.entries(mappings)) {
            const originalValue = this.getNestedValue(original, formField);
            const resultValue = this.getNestedValue(result, apiField);
            
            if (!this.valuesMatch(originalValue, resultValue)) {
                mismatches.push({
                    field: formField,
                    original: originalValue,
                    result: resultValue,
                    path: apiField
                });
            }
        }
        
        // Check for data that might have been lost
        const formFields = Object.keys(original);
        const mappedFields = Object.keys(mappings);
        const unmappedFields = formFields.filter(f => !mappedFields.includes(f));
        
        if (unmappedFields.length > 0) {
            warnings.push(`Unmapped form fields: ${unmappedFields.join(", ")}`);
        }
        
        return {
            isValid: mismatches.length === 0,
            mismatches,
            warnings
        };
    }
    
    /**
     * Get API input from form data
     */
    protected getAPIInput(formData: TFormData): TAPIInput {
        if (this.config.shapeToAPI) {
            return this.config.shapeToAPI(formData);
        }
        
        if (this.config.formFixture.transformToAPIInput) {
            return this.config.formFixture.transformToAPIInput(formData) as TAPIInput;
        }
        
        throw new Error("No shape transformation configured");
    }
    
    /**
     * Get default field mappings (assumes same field names)
     */
    protected getDefaultFieldMappings(): Record<string, string> {
        const mappings: Record<string, string> = {};
        
        // Map common fields
        const commonFields = ["name", "title", "description", "email", "handle"];
        for (const field of commonFields) {
            mappings[field] = field;
        }
        
        return mappings;
    }
    
    /**
     * Get nested value from object using dot notation
     */
    protected getNestedValue(obj: unknown, path: string): unknown {
        if (!obj || typeof obj !== "object") return undefined;
        
        const parts = path.split(".");
        let current: unknown = obj;
        
        for (const part of parts) {
            if (current && typeof current === "object" && part in current) {
                current = (current as Record<string, unknown>)[part];
            } else {
                return undefined;
            }
        }
        
        return current;
    }
    
    /**
     * Check if two values match (with type coercion for common cases)
     */
    protected valuesMatch(val1: unknown, val2: unknown): boolean {
        // Exact match
        if (val1 === val2) return true;
        
        // Both null/undefined
        if (val1 == null && val2 == null) return true;
        
        // String comparison (case-insensitive)
        if (typeof val1 === "string" && typeof val2 === "string") {
            return val1.toLowerCase() === val2.toLowerCase();
        }
        
        // Number comparison (handle string numbers)
        if ((typeof val1 === "number" || typeof val1 === "string") && 
            (typeof val2 === "number" || typeof val2 === "string")) {
            return Number(val1) === Number(val2);
        }
        
        // Array comparison (shallow)
        if (Array.isArray(val1) && Array.isArray(val2)) {
            return val1.length === val2.length && 
                   val1.every((v, i) => this.valuesMatch(v, val2[i]));
        }
        
        // Object comparison (shallow)
        if (typeof val1 === "object" && typeof val2 === "object" && 
            val1 !== null && val2 !== null) {
            const keys1 = Object.keys(val1);
            const keys2 = Object.keys(val2);
            
            return keys1.length === keys2.length &&
                   keys1.every(key => this.valuesMatch(
                       (val1 as Record<string, unknown>)[key],
                       (val2 as Record<string, unknown>)[key]
                   ));
        }
        
        return false;
    }
}
/* c8 ignore stop */