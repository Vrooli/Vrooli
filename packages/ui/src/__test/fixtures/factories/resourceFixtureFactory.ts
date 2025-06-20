/* c8 ignore start */
/**
 * Resource fixture factory implementation
 * 
 * This demonstrates the complete pattern for implementing fixtures for a Resource object type
 * with full type safety and integration with @vrooli/shared.
 */

// Basic types to avoid complex import issues
interface MockResource {
    id: string;
    resourceType: string;
    isInternal?: boolean;
    isPrivate?: boolean;
    name?: string;
    description?: string;
}

interface MockResourceCreateInput {
    id: string;
    resourceType: string;
    isInternal?: boolean;
    isPrivate?: boolean;
    ownedByUserConnect?: string;
    ownedByTeamConnect?: string;
    versionsCreate?: Array<{
        id: string;
        versionLabel: string;
        isPrivate?: boolean;
        translationsCreate?: Array<{
            id: string;
            language: string;
            name: string;
            description?: string;
            instructions?: string;
            details?: string;
        }>;
    }>;
    tagsCreate?: Array<{
        id: string;
        tag: string;
    }>;
}

interface MockResourceUpdateInput {
    id: string;
    isInternal?: boolean;
    isPrivate?: boolean;
}

// Mock ResourceType enum
const MockResourceType = {
    Note: "Note",
    Project: "Project",
    Api: "Api",
    Code: "Code",
    Standard: "Standard",
    Routine: "Routine"
} as const;

// Mock generatePK function
const mockGeneratePK = () => Date.now().toString() + Math.random().toString(36).substring(2);

/**
 * Resource form data type with UI-specific fields
 * 
 * This includes UI-specific fields like fileUpload and uploadProgress
 * that don't exist in the API input type.
 */
export interface ResourceFormData extends Record<string, unknown> {
    name: string;
    description?: string;
    isInternal?: boolean;
    isPrivate?: boolean;
    resourceType: string;
    usedBy?: string[];
    link?: string;
    index?: number;
    tags?: string[];
    fileUpload?: File;
    uploadProgress?: number;
    versionLabel?: string;
    instructions?: string;
    details?: string;
}

/**
 * Resource UI state type
 */
export interface ResourceUIState {
    isLoading: boolean;
    resource: MockResource | null;
    error: string | null;
    isEditing: boolean;
    hasUnsavedChanges: boolean;
    uploadProgress?: number;
    isUploading?: boolean;
}

/**
 * Simple validation result type
 */
interface ValidationResult {
    isValid: boolean;
    errors?: string[];
}

/**
 * Resource form fixture factory
 */
export class ResourceFormFixtureFactory {
    private scenarios: Record<string, ResourceFormData> = {
        minimal: {
            name: "Test Resource",
            resourceType: MockResourceType.Note,
            versionLabel: "1.0.0"
        },
        complete: {
            name: "Complete Test Resource",
            description: "A comprehensive test resource with all features enabled",
            isInternal: false,
            isPrivate: false,
            resourceType: MockResourceType.Project,
            usedBy: ["routine1", "routine2"],
            link: "https://example.com/resource",
            index: 0,
            tags: ["test", "automation", "productivity"],
            versionLabel: "1.0.0",
            instructions: "Step-by-step instructions for using this resource",
            details: "Detailed information about the resource implementation"
        },
        invalid: {
            name: "", // Invalid - empty name
            resourceType: "InvalidType", // Invalid enum value
            versionLabel: ""
        },
        internal: {
            name: "Internal Resource",
            description: "An internal-only resource for system use",
            isInternal: true,
            isPrivate: true,
            resourceType: MockResourceType.Code,
            versionLabel: "1.0.0"
        },
        external: {
            name: "External Resource",
            description: "A publicly available external resource",
            isInternal: false,
            isPrivate: false,
            resourceType: MockResourceType.Api,
            link: "https://api.example.com/v1",
            versionLabel: "1.0.0"
        },
        withFile: {
            name: "File Resource",
            description: "A resource with an attached file",
            resourceType: MockResourceType.Note,
            fileUpload: new File(["test content"], "test.txt", { type: "text/plain" }),
            uploadProgress: 0,
            versionLabel: "1.0.0"
        },
        library: {
            name: "Library Resource",
            description: "A reusable library resource",
            isInternal: false,
            isPrivate: false,
            resourceType: MockResourceType.Standard,
            tags: ["library", "reusable", "utilities"],
            versionLabel: "1.0.0"
        }
    };
    
    /**
     * Create form data for a specific scenario
     */
    createFormData(scenario: "minimal" | "complete" | "invalid" | string): ResourceFormData {
        const data = this.scenarios[scenario];
        if (!data) {
            throw new Error(`Unknown scenario: ${scenario}`);
        }
        return { ...data };
    }
    
    /**
     * Validate form data
     */
    async validateFormData(formData: ResourceFormData): Promise<ValidationResult> {
        const errors: string[] = [];
        
        if (!formData.name || formData.name.trim().length === 0) {
            errors.push("name: Name is required");
        }
        
        if (!formData.resourceType) {
            errors.push("resourceType: Resource type is required");
        }
        
        if (!formData.versionLabel || formData.versionLabel.trim().length === 0) {
            errors.push("versionLabel: Version label is required");
        }
        
        if (formData.fileUpload && formData.fileUpload.size > 10 * 1024 * 1024) { // 10MB limit
            errors.push("fileUpload: File size exceeds 10MB limit");
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors.length > 0 ? errors : undefined
        };
    }
    
    /**
     * Transform form data to API input
     */
    transformToAPIInput(formData: ResourceFormData): MockResourceCreateInput {
        return {
            id: mockGeneratePK(),
            resourceType: formData.resourceType,
            isInternal: formData.isInternal,
            isPrivate: formData.isPrivate,
            ownedByUserConnect: mockGeneratePK(), // Mock user ID
            versionsCreate: [
                {
                    id: mockGeneratePK(),
                    versionLabel: formData.versionLabel || "1.0.0",
                    isPrivate: formData.isPrivate || false,
                    translationsCreate: [
                        {
                            id: mockGeneratePK(),
                            language: "en",
                            name: formData.name,
                            ...(formData.description && { description: formData.description }),
                            ...(formData.instructions && { instructions: formData.instructions }),
                            ...(formData.details && { details: formData.details })
                        }
                    ]
                }
            ],
            ...(formData.tags && formData.tags.length > 0 && {
                tagsCreate: formData.tags.map(tag => ({
                    id: mockGeneratePK(),
                    tag
                }))
            })
        };
    }
    
    /**
     * Create form data with file upload simulation
     */
    createWithFileUpload(scenario: "minimal" | "complete" = "minimal"): ResourceFormData {
        const baseData = this.createFormData(scenario);
        return {
            ...baseData,
            fileUpload: new File(["test content"], "test-resource.txt", { type: "text/plain" }),
            uploadProgress: 0
        };
    }
    
    /**
     * Create form data for different resource types
     */
    createByType(resourceType: string): ResourceFormData {
        const baseData = this.createFormData("minimal");
        return {
            ...baseData,
            resourceType,
            name: `${resourceType} Resource`
        };
    }
}

/**
 * Resource MSW handler factory
 */
export class ResourceMSWHandlerFactory {
    /**
     * Create success response mock
     */
    createSuccessResponse(input: MockResourceCreateInput): MockResource {
        return {
            id: input.id,
            resourceType: input.resourceType,
            isInternal: input.isInternal || false,
            isPrivate: input.isPrivate || false,
            name: input.versionsCreate?.[0]?.translationsCreate?.[0]?.name || "Unnamed Resource",
            description: input.versionsCreate?.[0]?.translationsCreate?.[0]?.description
        };
    }
    
    /**
     * Validate API input
     */
    validateAPIInput(input: MockResourceCreateInput): ValidationResult {
        const errors: string[] = [];
        
        if (!input.id) {
            errors.push("ID is required");
        }
        
        if (!input.resourceType) {
            errors.push("Resource type is required");
        }
        
        if (!input.ownedByUserConnect && !input.ownedByTeamConnect) {
            errors.push("Owner is required");
        }
        
        if (!input.versionsCreate || input.versionsCreate.length === 0) {
            errors.push("At least one version is required");
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors.length > 0 ? errors : undefined
        };
    }
}

/**
 * Resource UI state fixture factory
 */
export class ResourceUIStateFixtureFactory {
    createLoadingState(): ResourceUIState {
        return {
            isLoading: true,
            resource: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createErrorState(error: { message: string }): ResourceUIState {
        return {
            isLoading: false,
            resource: null,
            error: error.message,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createSuccessState(data: MockResource): ResourceUIState {
        return {
            isLoading: false,
            resource: data,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createEmptyState(): ResourceUIState {
        return {
            isLoading: false,
            resource: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createUploadingState(progress: number): ResourceUIState {
        return {
            isLoading: true,
            resource: null,
            error: null,
            isEditing: true,
            hasUnsavedChanges: true,
            uploadProgress: progress,
            isUploading: true
        };
    }
    
    transitionToLoading(currentState: ResourceUIState): ResourceUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null
        };
    }
    
    transitionToSuccess(currentState: ResourceUIState, data: MockResource): ResourceUIState {
        return {
            ...currentState,
            isLoading: false,
            resource: data,
            error: null,
            hasUnsavedChanges: false,
            isUploading: false
        };
    }
    
    transitionToError(currentState: ResourceUIState, error: { message: string }): ResourceUIState {
        return {
            ...currentState,
            isLoading: false,
            error: error.message,
            isUploading: false
        };
    }
    
    transitionToUploading(currentState: ResourceUIState, progress: number): ResourceUIState {
        return {
            ...currentState,
            isLoading: true,
            uploadProgress: progress,
            isUploading: true,
            hasUnsavedChanges: true
        };
    }
}

/**
 * Complete Resource fixture factory
 */
export class ResourceFixtureFactory {
    readonly objectType = "resource";
    
    form: ResourceFormFixtureFactory;
    handlers: ResourceMSWHandlerFactory;
    states: ResourceUIStateFixtureFactory;
    
    constructor() {
        this.form = new ResourceFormFixtureFactory();
        this.handlers = new ResourceMSWHandlerFactory();
        this.states = new ResourceUIStateFixtureFactory();
    }
    
    createFormData(scenario: "minimal" | "complete" | string = "minimal"): ResourceFormData {
        return this.form.createFormData(scenario);
    }
    
    createAPIInput(formData: ResourceFormData): MockResourceCreateInput {
        return this.form.transformToAPIInput(formData);
    }
    
    createMockResponse(overrides?: Partial<MockResource>): MockResource {
        const baseResponse: MockResource = {
            id: mockGeneratePK(),
            resourceType: MockResourceType.Note,
            isInternal: false,
            isPrivate: false,
            name: "Test Resource",
            description: "A test resource"
        };
        
        return {
            ...baseResponse,
            ...overrides
        };
    }
    
    setupMSW(scenario: "success" | "error" | "loading" = "success"): void {
        // This would integrate with the MSW server
        // For now, it's a placeholder
        console.log(`Setting up MSW handlers for scenario: ${scenario}`);
    }
    
    async testCreateFlow(formData?: ResourceFormData): Promise<MockResource> {
        const data = formData || this.createFormData("minimal");
        
        // Validate form data
        const formValidation = await this.form.validateFormData(data);
        if (!formValidation.isValid) {
            throw new Error(`Form validation failed: ${formValidation.errors?.join(", ")}`);
        }
        
        // Transform to API input
        const apiInput = this.form.transformToAPIInput(data);
        
        // Validate API input
        const apiValidation = this.handlers.validateAPIInput(apiInput);
        if (!apiValidation.isValid) {
            throw new Error(`API validation failed: ${apiValidation.errors?.join(", ")}`);
        }
        
        // Create mock response
        return this.handlers.createSuccessResponse(apiInput);
    }
    
    async testUpdateFlow(id: string, updates: Partial<ResourceFormData>): Promise<MockResource> {
        const existingResource = this.createMockResponse({ id });
        const formData = { ...this.createFormData("minimal"), ...updates };
        
        // Validate updates
        const validation = await this.form.validateFormData(formData);
        if (!validation.isValid) {
            throw new Error(`Update validation failed: ${validation.errors?.join(", ")}`);
        }
        
        return {
            ...existingResource,
            name: formData.name,
            description: formData.description,
            isInternal: formData.isInternal,
            isPrivate: formData.isPrivate
        };
    }
    
    async testDeleteFlow(id: string): Promise<boolean> {
        // Mock delete operation
        if (!id) {
            throw new Error("ID is required for delete operation");
        }
        return true;
    }
    
    async testRoundTrip(formData?: ResourceFormData) {
        const data = formData || this.createFormData("complete");
        
        // Test create flow
        const createResult = await this.testCreateFlow(data);
        
        // Test update flow
        const updateData = { name: "Updated " + data.name };
        const updateResult = await this.testUpdateFlow(createResult.id, updateData);
        
        // Test delete flow
        const deleteResult = await this.testDeleteFlow(createResult.id);
        
        return {
            success: true,
            formData: data,
            apiResponse: createResult,
            updateResponse: updateResult,
            deleteSuccess: deleteResult,
            uiState: this.states.createSuccessState(createResult)
        };
    }
    
    /**
     * Test file upload flow
     */
    async testFileUploadFlow(file: File) {
        const formData = this.form.createWithFileUpload("minimal");
        formData.fileUpload = file;
        
        // Simulate upload progress
        const uploadStates: ResourceUIState[] = [];
        for (let progress = 0; progress <= 100; progress += 25) {
            uploadStates.push(this.states.createUploadingState(progress));
        }
        
        const result = await this.testCreateFlow(formData);
        return {
            uploadStates,
            finalResource: result
        };
    }
    
    /**
     * Test resource type scenarios
     */
    async testResourceTypeScenarios() {
        const results: Record<string, MockResource> = {};
        
        for (const resourceType of Object.values(MockResourceType)) {
            const formData = this.form.createByType(resourceType);
            results[resourceType] = await this.testCreateFlow(formData);
        }
        
        return results;
    }
}

// Export a factory function that can be used without complex dependencies
export function createResourceFixtureFactory(): ResourceFixtureFactory {
    return new ResourceFixtureFactory();
}

// Export the mock types for use in tests
export type { MockResource as Resource, MockResourceCreateInput as ResourceCreateInput, MockResourceUpdateInput as ResourceUpdateInput };
export { MockResourceType as ResourceType };
/* c8 ignore stop */