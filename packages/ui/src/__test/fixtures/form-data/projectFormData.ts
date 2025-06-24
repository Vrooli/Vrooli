/**
 * Project Form Data Fixtures
 * 
 * Provides comprehensive form data fixtures for project creation and editing forms.
 * Projects are Resources with resourceType="Project" in the Vrooli system.
 */

import { useFormik, type FormikProps, type FormikErrors } from "formik";

import * as yup from "yup";
import { act, waitFor } from "@testing-library/react";
import type { 
    ResourceCreateInput,
    ResourceUpdateInput,
    Resource,
    ResourceFor,
    ResourceUsedFor,
} from "@vrooli/shared";
import { 
    resourceValidation,
    ResourceFor as ResourceForEnum,
    ResourceUsedFor as ResourceUsedForEnum,
} from "@vrooli/shared";

/**
 * UI-specific form data for project creation
 */
export interface ProjectFormData {
    // Basic info
    handle?: string;
    name: string;
    description?: string;
    
    // Project-specific fields
    projectType?: "software" | "documentation" | "research" | "other";
    tags?: string[];
    
    // Privacy & permissions
    isPrivate: boolean;
    isInternal?: boolean;
    
    // Completion tracking
    completedAt?: Date | null;
    percentComplete?: number;
    
    // Parent relationship (UI-specific)
    parentType?: "User" | "Team";
    parentId?: string;
    
    // Version info
    versionLabel?: string;
    versionNotes?: string;
    
    // Resources (links, files, etc.)
    resources?: Array<{
        name: string;
        link: string;
        usedFor: ResourceUsedFor;
        description?: string;
    }>;
}

/**
 * Extended form state with project-specific properties
 */
export interface ProjectFormState {
    values: ProjectFormData;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    isValidating?: boolean;
    
    // Project-specific state
    projectState?: {
        isCheckingHandleAvailability?: boolean;
        handleAvailable?: boolean;
        currentProject?: Partial<Resource>;
        suggestedTags?: string[];
        relatedProjects?: Array<{
            id: string;
            name: string;
            handle?: string;
        }>;
    };
    
    // Completion tracking
    completionState?: {
        isCompleted: boolean;
        canMarkComplete: boolean;
        blockers?: string[];
    };
}

/**
 * Project form data factory with React Hook Form integration
 */
export class ProjectFormDataFactory {
    /**
     * Create validation schema for projects
     */
    private createProjectSchema(): yup.ObjectSchema<ProjectFormData> {
        return yup.object({
            handle: yup
                .string()
                .optional()
                .nullable()
                .test("handle-validation", "Invalid handle format", function(value) {
                    if (!value) return true; // Optional
                    return /^[a-zA-Z0-9_-]+$/.test(value) && value.length >= 3 && value.length <= 50;
                }),
                
            name: yup
                .string()
                .required("Project name is required")
                .min(1, "Project name cannot be empty")
                .max(255, "Project name is too long"),
                
            description: yup
                .string()
                .max(5000, "Description is too long")
                .optional(),
                
            projectType: yup
                .string()
                .oneOf(["software", "documentation", "research", "other"])
                .optional(),
                
            tags: yup
                .array(
                    yup.string()
                        .min(2, "Tag must be at least 2 characters")
                        .max(30, "Tag is too long"),
                )
                .max(10, "Maximum 10 tags allowed")
                .optional(),
                
            isPrivate: yup
                .boolean()
                .required("Privacy setting is required"),
                
            isInternal: yup
                .boolean()
                .optional(),
                
            completedAt: yup
                .date()
                .nullable()
                .optional(),
                
            percentComplete: yup
                .number()
                .min(0, "Percentage cannot be negative")
                .max(100, "Percentage cannot exceed 100")
                .optional(),
                
            parentType: yup
                .string()
                .oneOf(["User", "Team"])
                .when("parentId", {
                    is: (val: string | undefined) => !!val,
                    then: (schema) => schema.required("Parent type is required when parent is selected"),
                    otherwise: (schema) => schema.optional(),
                }),
                
            parentId: yup
                .string()
                .optional(),
                
            versionLabel: yup
                .string()
                .max(50, "Version label is too long")
                .optional(),
                
            versionNotes: yup
                .string()
                .max(1000, "Version notes are too long")
                .optional(),
                
            resources: yup.array(
                yup.object({
                    name: yup
                        .string()
                        .required("Resource name is required")
                        .max(255, "Resource name is too long"),
                    link: yup
                        .string()
                        .required("Resource link is required")
                        .url("Invalid URL format"),
                    usedFor: yup
                        .mixed<ResourceUsedFor>()
                        .oneOf(Object.values(ResourceUsedForEnum))
                        .required("Resource purpose is required"),
                    description: yup
                        .string()
                        .max(500, "Resource description is too long")
                        .optional(),
                }),
            ).optional(),
        }).defined();
    }
    
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `proj_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create project form data for different scenarios
     */
    createFormData(
        scenario: "empty" | "minimal" | "complete" | "invalid" | "privateProject" | 
                 "teamProject" | "completedProject" | "withResources" | "partiallyCompleted",
    ): ProjectFormData {
        switch (scenario) {
            case "empty":
                return {
                    name: "",
                    isPrivate: false,
                };
                
            case "minimal":
                return {
                    name: "My Project",
                    isPrivate: false,
                };
                
            case "complete":
                return {
                    handle: "awesome-project",
                    name: "Awesome Project",
                    description: "A comprehensive project showcasing best practices in software development",
                    projectType: "software",
                    tags: ["opensource", "typescript", "react", "nodejs"],
                    isPrivate: false,
                    isInternal: false,
                    percentComplete: 75,
                    parentType: "User",
                    parentId: "user_123",
                    versionLabel: "v2.0.0",
                    versionNotes: "Major update with new features and improvements",
                    resources: [
                        {
                            name: "GitHub Repository",
                            link: "https://github.com/user/awesome-project",
                            usedFor: ResourceUsedForEnum.Context,
                            description: "Main source code repository",
                        },
                        {
                            name: "Documentation",
                            link: "https://docs.awesome-project.com",
                            usedFor: ResourceUsedForEnum.Context,
                            description: "Project documentation and guides",
                        },
                    ],
                };
                
            case "invalid":
                return {
                    handle: "a", // Too short
                    name: "", // Empty
                    isPrivate: false,
                    tags: ["a", "this-tag-is-way-too-long-and-exceeds-the-maximum-allowed-length"], // Invalid tags
                    percentComplete: 150, // Over 100
                    resources: [
                        {
                            name: "", // Empty name
                            link: "not-a-url", // Invalid URL
                            usedFor: "" as any, // Invalid type
                        },
                    ],
                };
                
            case "privateProject":
                return {
                    handle: "internal-project",
                    name: "Internal Project",
                    description: "Private project for internal use only",
                    projectType: "research",
                    isPrivate: true,
                    isInternal: true,
                    parentType: "Team",
                    parentId: "team_456",
                };
                
            case "teamProject":
                return {
                    handle: "team-collaboration",
                    name: "Team Collaboration Project",
                    description: "A project managed by our development team",
                    projectType: "software",
                    tags: ["teamwork", "agile", "scrum"],
                    isPrivate: false,
                    parentType: "Team",
                    parentId: "team_789",
                    percentComplete: 45,
                };
                
            case "completedProject":
                return {
                    handle: "finished-project",
                    name: "Completed Project",
                    description: "This project has been successfully completed",
                    projectType: "documentation",
                    tags: ["completed", "archived"],
                    isPrivate: false,
                    completedAt: new Date("2024-01-15"),
                    percentComplete: 100,
                    versionLabel: "v1.0.0-final",
                };
                
            case "withResources":
                return {
                    name: "Resource-Rich Project",
                    description: "Project with multiple external resources",
                    isPrivate: false,
                    resources: [
                        {
                            name: "API Documentation",
                            link: "https://api.example.com/docs",
                            usedFor: ResourceUsedForEnum.Context,
                            description: "REST API documentation",
                        },
                        {
                            name: "Design System",
                            link: "https://design.example.com",
                            usedFor: ResourceUsedForEnum.Display,
                            description: "UI/UX design guidelines",
                        },
                        {
                            name: "Project Wiki",
                            link: "https://wiki.example.com/project",
                            usedFor: ResourceUsedForEnum.Context,
                            description: "Detailed project information",
                        },
                    ],
                };
                
            case "partiallyCompleted":
                return {
                    name: "Work in Progress",
                    description: "A project that is partially complete",
                    isPrivate: false,
                    percentComplete: 35,
                    tags: ["wip", "development"],
                    resources: [
                        {
                            name: "Roadmap",
                            link: "", // Missing URL
                            usedFor: ResourceUsedForEnum.Context,
                        },
                    ],
                };
                
            default:
                throw new Error(`Unknown project scenario: ${scenario}`);
        }
    }
    
    /**
     * Create form state for different scenarios
     */
    createFormState(
        scenario: "pristine" | "dirty" | "submitting" | "withErrors" | "valid" | "checkingHandle" | "nearCompletion",
    ): ProjectFormState {
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
                    touched: { name: true, description: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "submitting":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: true,
                    isValid: true,
                };
                
            case "withErrors":
                return {
                    values: this.createFormData("invalid"),
                    errors: {
                        handle: "Handle must be at least 3 characters",
                        name: "Project name is required",
                        percentComplete: "Percentage cannot exceed 100",
                        "resources[0].name": "Resource name is required",
                        "resources[0].link": "Invalid URL format",
                    },
                    touched: { 
                        handle: true, 
                        name: true, 
                        percentComplete: true,
                        resources: true,
                    },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "valid":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "checkingHandle":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { handle: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                    isValidating: true,
                    projectState: {
                        isCheckingHandleAvailability: true,
                    },
                };
                
            case "nearCompletion":
                return {
                    values: {
                        ...baseFormData,
                        percentComplete: 95,
                    },
                    errors: {},
                    touched: { percentComplete: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    completionState: {
                        isCompleted: false,
                        canMarkComplete: true,
                        blockers: [],
                    },
                };
                
            default:
                throw new Error(`Unknown form state scenario: ${scenario}`);
        }
    }
    
    /**
     * Create React Hook Form instance
     */
    createFormInstance(
        initialData?: Partial<ProjectFormData>,
    ): UseFormReturn<ProjectFormData> {
        const defaultValues: ProjectFormData = {
            name: "",
            isPrivate: false,
            tags: [],
            resources: [],
            ...initialData,
        };
        
        return useForm<ProjectFormData>({
            mode: "onChange",
            reValidateMode: "onChange",
            shouldFocusError: true,
            defaultValues,
            resolver: yupResolver(this.createProjectSchema()),
        });
    }
    
    /**
     * Validate form data using real validation
     */
    async validateFormData(
        formData: ProjectFormData,
    ): Promise<{
        isValid: boolean;
        errors?: Record<string, string>;
        apiInput?: ResourceCreateInput;
    }> {
        try {
            // Validate with form schema
            await this.createProjectSchema().validate(formData, { abortEarly: false });
            
            // Transform to API input
            const apiInput = this.transformToAPIInput(formData);
            
            // Validate with real API validation
            await resourceValidation.create.validate(apiInput);
            
            return {
                isValid: true,
                apiInput,
            };
        } catch (error: any) {
            const errors: Record<string, string> = {};
            
            if (error.inner) {
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
    private transformToAPIInput(formData: ProjectFormData): ResourceCreateInput {
        const input: ResourceCreateInput = {
            id: this.generateId(),
            index: 0,
            isPrivate: formData.isPrivate,
            isInternal: formData.isInternal,
            completedAt: formData.completedAt?.toISOString(),
            versionsCreate: [{
                id: this.generateId(),
                isComplete: formData.percentComplete === 100,
                versionLabel: formData.versionLabel || "1.0.0",
                translationsCreate: [{
                    id: this.generateId(),
                    language: "en",
                    name: formData.name,
                    description: formData.description,
                    versionNotes: formData.versionNotes,
                }],
            }],
        };
        
        // Set parent relationship
        if (formData.parentId) {
            if (formData.parentType === "User") {
                input.forConnect = formData.parentId;
                input.resourceFor = ResourceForEnum.User;
            } else if (formData.parentType === "Team") {
                input.forConnect = formData.parentId;
                input.resourceFor = ResourceForEnum.Team;
            }
        }
        
        // Add resources as version resources
        if (formData.resources && formData.resources.length > 0 && input.versionsCreate?.[0]) {
            input.versionsCreate[0].resourcesCreate = formData.resources.map((resource, index) => ({
                id: this.generateId(),
                index,
                link: resource.link,
                usedFor: resource.usedFor,
                translationsCreate: [{
                    id: this.generateId(),
                    language: "en",
                    name: resource.name,
                    description: resource.description,
                }],
            }));
        }
        
        return input;
    }
    
    /**
     * Suggest tags based on project content
     */
    suggestTags(formData: Partial<ProjectFormData>): string[] {
        const suggestions: string[] = [];
        
        if (formData.projectType) {
            suggestions.push(formData.projectType);
        }
        
        if (formData.name?.toLowerCase().includes("api")) {
            suggestions.push("api");
        }
        
        if (formData.name?.toLowerCase().includes("app")) {
            suggestions.push("mobile", "webapp");
        }
        
        if (formData.description?.toLowerCase().includes("opensource")) {
            suggestions.push("opensource", "community");
        }
        
        return suggestions.slice(0, 5); // Return top 5 suggestions
    }
}

/**
 * Form interaction simulator for project forms
 */
export class ProjectFormInteractionSimulator {
    private interactionDelay = 100;
    
    constructor(delay?: number) {
        this.interactionDelay = delay || 100;
    }
    
    /**
     * Simulate project creation flow
     */
    async simulateProjectCreationFlow(
        formInstance: UseFormReturn<ProjectFormData>,
        formData: ProjectFormData,
    ): Promise<void> {
        // Type project name
        await this.simulateTyping(formInstance, "name", formData.name);
        await new Promise(resolve => setTimeout(resolve, 200));
        
        // Type handle if present
        if (formData.handle) {
            for (let i = 1; i <= formData.handle.length; i++) {
                const partialHandle = formData.handle.substring(0, i);
                await this.fillField(formInstance, "handle", partialHandle);
                
                // Simulate availability check after 3 characters
                if (i >= 3) {
                    await new Promise(resolve => setTimeout(resolve, 300));
                }
            }
        }
        
        // Type description
        if (formData.description) {
            await this.simulateTyping(formInstance, "description", formData.description);
        }
        
        // Select project type
        if (formData.projectType) {
            await this.fillField(formInstance, "projectType", formData.projectType);
        }
        
        // Add tags
        if (formData.tags) {
            for (const tag of formData.tags) {
                await this.addTag(formInstance, tag);
            }
        }
        
        // Set privacy
        await this.fillField(formInstance, "isPrivate", formData.isPrivate);
        
        // Set completion percentage
        if (formData.percentComplete !== undefined) {
            await this.simulateSliderChange(formInstance, "percentComplete", formData.percentComplete);
        }
        
        // Add resources
        if (formData.resources) {
            for (let i = 0; i < formData.resources.length; i++) {
                await this.addResource(formInstance, i, formData.resources[i]);
            }
        }
    }
    
    /**
     * Add a tag to the project
     */
    private async addTag(
        formInstance: UseFormReturn<ProjectFormData>,
        tag: string,
    ): Promise<void> {
        const currentTags = formInstance.getValues("tags") || [];
        await this.fillField(formInstance, "tags", [...currentTags, tag]);
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    /**
     * Add a resource to the project
     */
    private async addResource(
        formInstance: UseFormReturn<ProjectFormData>,
        index: number,
        resource: ProjectFormData["resources"][0],
    ): Promise<void> {
        await this.simulateTyping(formInstance, `resources.${index}.name` as any, resource.name);
        await this.simulateTyping(formInstance, `resources.${index}.link` as any, resource.link);
        await this.fillField(formInstance, `resources.${index}.usedFor` as any, resource.usedFor);
        
        if (resource.description) {
            await this.simulateTyping(formInstance, `resources.${index}.description` as any, resource.description);
        }
        
        await new Promise(resolve => setTimeout(resolve, 200));
    }
    
    /**
     * Simulate slider change for percentage
     */
    private async simulateSliderChange(
        formInstance: UseFormReturn<ProjectFormData>,
        fieldName: string,
        targetValue: number,
    ): Promise<void> {
        const steps = 10;
        const currentValue = formInstance.getValues(fieldName as any) || 0;
        const increment = (targetValue - currentValue) / steps;
        
        for (let i = 1; i <= steps; i++) {
            const newValue = Math.round(currentValue + (increment * i));
            await this.fillField(formInstance, fieldName, newValue);
            await new Promise(resolve => setTimeout(resolve, 50));
        }
    }
    
    /**
     * Simulate typing
     */
    private async simulateTyping(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        text: string,
    ): Promise<void> {
        for (let i = 1; i <= text.length; i++) {
            await this.fillField(formInstance, fieldName, text.substring(0, i));
            await new Promise(resolve => setTimeout(resolve, 50));
        }
    }
    
    /**
     * Fill field helper
     */
    private async fillField(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        value: any,
    ): Promise<void> {
        act(() => {
            formInstance.setValue(fieldName, value, {
                shouldDirty: true,
                shouldTouch: true,
                shouldValidate: true,
            });
        });
        
        await waitFor(() => {
            expect(formInstance.formState.isValidating).toBe(false);
        });
    }
}

// Export factory instances
export const projectFormFactory = new ProjectFormDataFactory();
export const projectFormSimulator = new ProjectFormInteractionSimulator();

// Export pre-configured scenarios
export const projectFormScenarios = {
    // Basic scenarios
    emptyProject: () => projectFormFactory.createFormState("pristine"),
    validProject: () => projectFormFactory.createFormState("valid"),
    projectWithErrors: () => projectFormFactory.createFormState("withErrors"),
    submittingProject: () => projectFormFactory.createFormState("submitting"),
    
    // Project types
    minimalProject: () => projectFormFactory.createFormData("minimal"),
    completeProject: () => projectFormFactory.createFormData("complete"),
    privateProject: () => projectFormFactory.createFormData("privateProject"),
    teamProject: () => projectFormFactory.createFormData("teamProject"),
    completedProject: () => projectFormFactory.createFormData("completedProject"),
    resourceRichProject: () => projectFormFactory.createFormData("withResources"),
    
    // Interactive workflows
    async completeProjectWorkflow(formInstance: UseFormReturn<ProjectFormData>) {
        const simulator = new ProjectFormInteractionSimulator();
        const formData = projectFormFactory.createFormData("complete");
        await simulator.simulateProjectCreationFlow(formInstance, formData);
        return formData;
    },
    
    async quickProjectSetup(formInstance: UseFormReturn<ProjectFormData>) {
        const simulator = new ProjectFormInteractionSimulator(25); // Faster
        const formData = projectFormFactory.createFormData("minimal");
        await simulator.simulateProjectCreationFlow(formInstance, formData);
        return formData;
    },
    
    // Tag suggestions
    getTagSuggestions(formData: Partial<ProjectFormData>) {
        return projectFormFactory.suggestTags(formData);
    },
};
